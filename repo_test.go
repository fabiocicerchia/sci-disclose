package main

import (
	"testing"
)

func TestDetectWorkloadPrefersTheMakefile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "Makefile", "test:\n\tpytest -q\n")
	writeFile(t, dir, "go.mod", "module demo\n")
	argv, why, ok := DetectWorkload(dir)
	if !ok || argv[0] != "make" || argv[1] != "test" {
		t.Fatalf("got %v (%s)", argv, why)
	}
}

func TestDetectWorkloadFallsThroughToLanguageDefaults(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module demo\n")
	if argv, _, _ := DetectWorkload(dir); argv[0] != "go" {
		t.Fatalf("go.mod: %v", argv)
	}
	writeFile(t, dir, "package.json", `{"scripts": {"test": "vitest"}}`)
	if argv, _, _ := DetectWorkload(dir); argv[0] != "npm" || argv[1] != "test" {
		t.Fatalf("package.json: %v", argv)
	}
}

func TestDetectWorkloadReturnsNothingForAnEmptyRepo(t *testing.T) {
	if _, _, ok := DetectWorkload(t.TempDir()); ok {
		t.Fatal("an empty directory has no workload to run")
	}
}

func TestKubernetesQuantitiesAndInstanceSizes(t *testing.T) {
	cpu, err := ParseCPUQuantity("500m")
	if err != nil || cpu != 0.5 {
		t.Errorf("500m: %g (%v)", cpu, err)
	}
	if cpu, _ := ParseCPUQuantity("2"); cpu != 2 {
		t.Errorf("2: %g", cpu)
	}
	if memory, _ := ParseMemoryQuantity("512Mi"); memory != 0.5 {
		t.Errorf("512Mi: %g", memory)
	}
	if memory, _ := ParseMemoryQuantity("2Gi"); memory != 2 {
		t.Errorf("2Gi: %g", memory)
	}
	if vcpus, ok := InstanceVCPUs("m5.2xlarge"); !ok || vcpus != 8 {
		t.Errorf("m5.2xlarge: %d", vcpus)
	}
	if _, ok := InstanceVCPUs("weird"); ok {
		t.Error("an unknown size should not be guessed silently")
	}
}

func TestScanRepoReadsDeploymentsAndTerraform(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "deploy.yaml", `apiVersion: apps/v1
kind: Deployment
metadata:
  name: api
spec:
  replicas: 3
  template:
    spec:
      containers:
        - name: api
          resources:
            requests:
              cpu: 500m
              memory: 512Mi
`)
	writeFile(t, dir, "cron.yaml", `apiVersion: batch/v1
kind: CronJob
metadata:
  name: nightly
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: job
              resources:
                requests:
                  cpu: "2"
`)
	writeFile(t, dir, "main.tf", "  instance_type = \"c6i.4xlarge\"\n")
	writeFile(t, dir, "node_modules/pkg/deploy.yaml", "kind: Deployment\nmetadata:\n  name: skipme\n")

	found, _ := ScanRepo(dir)
	byName := map[string]Discovered{}
	for _, component := range found {
		byName[component.Name] = component
	}
	api, ok := byName["api"]
	if !ok || api.VCPUs != 0.5 || api.Replicas != 3 || api.MemoryGB != 0.5 {
		t.Fatalf("api: %+v", api)
	}
	nightly, ok := byName["nightly"]
	if !ok || nightly.VCPUs != 2 {
		t.Fatalf("cronjob: %+v", nightly)
	}
	if _, skipped := byName["skipme"]; skipped {
		t.Error("node_modules should not be scanned")
	}
	var sawInstance bool
	for _, component := range found {
		if component.VCPUs == 16 {
			sawInstance = true
		}
	}
	if !sawInstance {
		t.Error("the Terraform instance type was not picked up")
	}
}
