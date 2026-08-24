package discover

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Repo targets: find the workload a repository already has, and scaffold the
// manifest for the workload it deploys.

var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, ".venv": true,
	"venv": true, "dist": true, "build": true, "target": true, ".tox": true,
	"__pycache__": true, ".terraform": true,
}

// DetectWorkload returns the command a contributor would run, and where that
// choice came from. Nothing is invented: an empty repo returns false.
func DetectWorkload(root string) ([]string, string, bool) {
	for _, name := range []string{"Makefile", "makefile", "GNUmakefile"} {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		for _, target := range []string{"test", "check", "bench"} {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, target+":") {
					return []string{"make", target}, name + " `" + target + "` target", true
				}
			}
		}
		break
	}
	if data, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil {
		var pkg struct {
			Scripts map[string]string `json:"scripts"`
		}
		if json.Unmarshal(data, &pkg) == nil {
			for _, script := range []string{"test", "bench", "build"} {
				if _, ok := pkg.Scripts[script]; ok {
					if script == "test" {
						return []string{"npm", "test"}, "package.json `test` script", true
					}
					return []string{"npm", "run", script},
						"package.json `" + script + "` script", true
				}
			}
		}
	}
	if exists(filepath.Join(root, "go.mod")) {
		return []string{"go", "test", "./..."}, "go.mod", true
	}
	if exists(filepath.Join(root, "Cargo.toml")) {
		return []string{"cargo", "test"}, "Cargo.toml", true
	}
	for _, name := range []string{"pytest.ini", "tox.ini", "pyproject.toml", "tests"} {
		if exists(filepath.Join(root, name)) {
			return []string{"pytest", "-q"}, "Python test layout", true
		}
	}
	return nil, "", false
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// InstanceVCPUs is the vCPU count for an EC2-style instance name, from its
// size suffix. Unknown sizes return false rather than a guess.
func InstanceVCPUs(instanceType string) (int, bool) {
	sizes := map[string]int{
		"nano": 2, "micro": 2, "small": 2, "medium": 2, "large": 2,
		"xlarge": 4, "2xlarge": 8, "3xlarge": 12, "4xlarge": 16,
		"6xlarge": 24, "8xlarge": 32, "9xlarge": 36, "12xlarge": 48,
		"16xlarge": 64, "18xlarge": 72, "24xlarge": 96, "32xlarge": 128,
	}
	parts := strings.Split(instanceType, ".")
	vcpus, ok := sizes[parts[len(parts)-1]]
	return vcpus, ok
}

// ParseCPUQuantity converts a Kubernetes CPU quantity ("500m", "2") to vCPUs.
func ParseCPUQuantity(value string) (float64, error) {
	text := strings.TrimSpace(value)
	if strings.HasSuffix(text, "m") {
		milli, err := strconv.ParseFloat(strings.TrimSuffix(text, "m"), 64)
		return milli / 1000, err
	}
	return strconv.ParseFloat(text, 64)
}

// ParseMemoryQuantity converts a Kubernetes memory quantity to GB.
func ParseMemoryQuantity(value string) (float64, error) {
	text := strings.TrimSpace(value)
	units := []struct {
		suffix string
		factor float64
	}{
		{"Ki", 1.0 / (1024 * 1024)}, {"Mi", 1.0 / 1024}, {"Gi", 1}, {"Ti", 1024},
		{"K", 1e-6}, {"M", 1e-3}, {"G", 1}, {"T", 1e3},
	}
	for _, unit := range units {
		if strings.HasSuffix(text, unit.suffix) {
			amount, err := strconv.ParseFloat(strings.TrimSuffix(text, unit.suffix), 64)
			return amount * unit.factor, err
		}
	}
	bytes, err := strconv.ParseFloat(text, 64)
	return bytes / (1024 * 1024 * 1024), err
}

// Discovered is a component found in the repo, with the note explaining where
// it came from and what had to be assumed.
type Discovered struct {
	Name     string
	VCPUs    float64
	Replicas int
	MemoryGB float64
	Comment  string
}

type podTemplate struct {
	Spec struct {
		Containers []struct {
			Resources struct {
				Requests struct {
					CPU    yaml.Node `yaml:"cpu"`
					Memory yaml.Node `yaml:"memory"`
				} `yaml:"requests"`
			} `yaml:"resources"`
		} `yaml:"containers"`
	} `yaml:"spec"`
}

type workloadDoc struct {
	Kind     string `yaml:"kind"`
	Metadata struct {
		Name string `yaml:"name"`
	} `yaml:"metadata"`
	Spec struct {
		Replicas    *int        `yaml:"replicas"`
		Template    podTemplate `yaml:"template"`
		JobTemplate struct {
			Spec struct {
				Template podTemplate `yaml:"template"`
			} `yaml:"spec"`
		} `yaml:"jobTemplate"`
	} `yaml:"spec"`
}

var workloadKinds = map[string]bool{
	"Deployment": true, "StatefulSet": true, "DaemonSet": true,
	"ReplicaSet": true, "Job": true, "CronJob": true,
}

// ScanRepo discovers the deployed boundary: Kubernetes workloads first, then
// Terraform instance types.
func ScanRepo(root string) ([]Discovered, []string) {
	var found []Discovered
	var notes []string

	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if skipDirs[entry.Name()] {
				return fs.SkipDir
			}
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			relative = path
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".yaml", ".yml":
			found = append(found, scanKubernetes(path, relative)...)
		case ".tf":
			found = append(found, scanTerraform(path, relative)...)
		}
		return nil
	})
	return found, notes
}

func scanKubernetes(path, relative string) []Discovered {
	file, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer file.Close()

	var found []Discovered
	decoder := yaml.NewDecoder(file)
	for {
		var doc workloadDoc
		if err := decoder.Decode(&doc); err != nil {
			break // end of stream, or YAML this tool has no business reading
		}
		if !workloadKinds[doc.Kind] {
			continue
		}
		pod := doc.Spec.Template
		if doc.Kind == "CronJob" {
			pod = doc.Spec.JobTemplate.Spec.Template
		}
		var vcpus, memory float64
		for _, container := range pod.Spec.Containers {
			requests := container.Resources.Requests
			if requests.CPU.Value != "" {
				if value, err := ParseCPUQuantity(requests.CPU.Value); err == nil {
					vcpus += value
				}
			}
			if requests.Memory.Value != "" {
				if value, err := ParseMemoryQuantity(requests.Memory.Value); err == nil {
					memory += value
				}
			}
		}
		name := doc.Metadata.Name
		if name == "" {
			name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		}
		replicas := 1
		if doc.Spec.Replicas != nil && *doc.Spec.Replicas > 0 {
			replicas = *doc.Spec.Replicas
		}
		comment := fmt.Sprintf("%s from %s", doc.Kind, relative)
		if vcpus == 0 {
			comment += "; no CPU requests set, assumed 1 vCPU"
			vcpus = 1
		}
		found = append(found, Discovered{
			Name: name, VCPUs: round(vcpus, 3), Replicas: replicas,
			MemoryGB: round(memory, 3), Comment: comment,
		})
	}
	return found
}

func scanTerraform(path, relative string) []Discovered {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var found []Discovered
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.Contains(line, "instance_type") || !strings.Contains(line, `"`) {
			continue
		}
		parts := strings.Split(line, `"`)
		if len(parts) < 2 {
			continue
		}
		name := parts[1]
		vcpus, known := InstanceVCPUs(name)
		comment := "from Terraform"
		if !known {
			comment += "; vCPUs unknown, guessed"
			vcpus = 2
		}
		found = append(found, Discovered{
			Name: fmt.Sprintf("%s (%s)", name, relative), VCPUs: float64(vcpus),
			Replicas: 1, Comment: comment,
		})
	}
	return found
}

func round(value float64, places int) float64 {
	factor := 1.0
	for range places {
		factor *= 10
	}
	return float64(int64(value*factor+0.5)) / factor
}

// RenderManifest emits sci.yaml by hand, so the scaffold can carry its own
// commentary about what still has to be filled in.
func RenderManifest(name string, components []Discovered, notes []string) string {
	lines := []string{
		fmt.Sprintf("# SCI manifest for %s, scaffolded by `sci init`.", name),
		"# Every number below is a starting point: utilisation and the",
		"# functional unit are yours to fill in, and the SCI is only as",
		"# honest as they are. Compute it with `sci estimate -f sci.yaml`.",
		"name: " + name,
		"",
		"functional-unit:",
		"  # What one unit of useful work is. Requests, jobs, users, minutes.",
		"  label: request",
		"  quantity: 1000000  # units delivered over the period below",
		"",
		"defaults:",
		"  provider: aws        # aws | gcp | azure | onprem | laptop",
		"  region: eu-west-1    # or: country: IE / zone: IT/SICI / intensity: 290",
		"  period-hours: 720    # the window the quantity above covers",
		"",
		"components:",
	}
	if len(components) == 0 {
		lines = append(lines,
			"  # Nothing was discovered: no Kubernetes or Terraform files found.",
			"  - name: api",
			"    type: compute",
			"    vcpus: 2",
			"    replicas: 1",
			"    utilisation: 0.4",
			"    memory-gb: 4",
			"    embodied:",
			"      device-kg: 1200",
			"      lifespan-years: 4",
			"      total-vcpus: 64")
	}
	for _, component := range components {
		if component.Comment != "" {
			lines = append(lines, "  # "+component.Comment)
		}
		lines = append(lines,
			fmt.Sprintf("  - name: %s", component.Name),
			"    type: compute",
			fmt.Sprintf("    vcpus: %g", component.VCPUs),
			fmt.Sprintf("    replicas: %d", component.Replicas),
			"    utilisation: 0.5")
		if component.MemoryGB > 0 {
			lines = append(lines, fmt.Sprintf("    memory-gb: %g", component.MemoryGB))
		}
		lines = append(lines, "    embodied:", "      device-kg: 1200",
			"      lifespan-years: 4", "      total-vcpus: 64")
	}
	lines = append(lines,
		"",
		"  # Uncomment what else is inside your software boundary.",
		"  # - name: object storage",
		"  #   type: storage",
		"  #   storage-gb: 500",
		"  #   medium: ssd",
		"  # - name: egress",
		"  #   type: network",
		"  #   network-gb: 2000",
		"  # - name: end-user phones",
		"  #   type: device",
		"  #   watts: 2",
		"  #   replicas: 5000",
		"  #   hours: 0.05",
		"")
	for _, note := range notes {
		lines = append(lines, "# note: "+note)
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}
