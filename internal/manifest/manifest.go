package manifest

import (
	"cmp"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/fabiocicerchia/sci-disclose/internal/coefficients"
	"github.com/fabiocicerchia/sci-disclose/internal/config"
	"github.com/fabiocicerchia/sci-disclose/internal/energy"
	"github.com/fabiocicerchia/sci-disclose/internal/grid"
	"github.com/fabiocicerchia/sci-disclose/internal/sci"
)

// Declared deployments: the SCI of software you are not running right now. A
// manifest describes the boundary (what runs, on what, for how long) and the
// functional unit; nothing is executed.

// Manifest is the sci.yaml document. JSON is accepted too, since YAML is a
// superset of it.
type Manifest struct {
	Name           string      `yaml:"name"`
	FunctionalUnit UnitSpec    `yaml:"functional-unit"`
	Defaults       Defaults    `yaml:"defaults"`
	PeriodHours    float64     `yaml:"period-hours"`
	Components     []Component `yaml:"components"`
	Notes          []string    `yaml:"notes"`
}

// UnitSpec is R as declared.
type UnitSpec struct {
	Label    string  `yaml:"label"`
	Quantity float64 `yaml:"quantity"`
}

// Defaults apply to every component that does not override them.
type Defaults struct {
	Provider    string  `yaml:"provider"`
	PUE         float64 `yaml:"pue"`
	Region      string  `yaml:"region"`
	Country     string  `yaml:"country"`
	Zone        string  `yaml:"zone"`
	Intensity   float64 `yaml:"intensity"`
	PeriodHours float64 `yaml:"period-hours"`
}

// Component is one part of the software boundary as declared in the manifest.
type Component struct {
	Name        string    `yaml:"name"`
	Type        string    `yaml:"type"`
	VCPUs       float64   `yaml:"vcpus"`
	Replicas    float64   `yaml:"replicas"`
	Utilisation *float64  `yaml:"utilisation"`
	MemoryGB    float64   `yaml:"memory-gb"`
	StorageGB   float64   `yaml:"storage-gb"`
	Medium      string    `yaml:"medium"`
	NetworkGB   float64   `yaml:"network-gb"`
	Watts       float64   `yaml:"watts"`
	Hours       float64   `yaml:"hours"`
	Provider    string    `yaml:"provider"`
	PUE         float64   `yaml:"pue"`
	Region      string    `yaml:"region"`
	Country     string    `yaml:"country"`
	Zone        string    `yaml:"zone"`
	Intensity   float64   `yaml:"intensity"`
	Embodied    *Embodied `yaml:"embodied"`
}

// Embodied is the hardware share a component occupies.
type Embodied struct {
	DeviceKg      float64  `yaml:"device-kg"`
	LifespanYears float64  `yaml:"lifespan-years"`
	ResourceShare *float64 `yaml:"resource-share"`
	TotalVCPUs    float64  `yaml:"total-vcpus"`
}

// LoadManifest reads a manifest, accepting hyphenated or underscored keys and
// either spelling of "utilisation".
func LoadManifest(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, err
	}
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return Manifest{}, fmt.Errorf("%s is not valid YAML or JSON: %w", path, err)
	}
	if len(root.Content) == 0 {
		return Manifest{}, fmt.Errorf("%s is empty", path)
	}
	normaliseKeys(&root)
	var manifest Manifest
	if err := root.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("%s is not an SCI manifest: %w", path, err)
	}
	return manifest, nil
}

// normaliseKeys rewrites mapping keys so `memory_gb`, `memory-gb` and
// `utilization` all reach the same field.
func normaliseKeys(node *yaml.Node) {
	if node.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			key.Value = strings.ToLower(strings.ReplaceAll(key.Value, "_", "-"))
			if key.Value == "utilization" {
				key.Value = "utilisation"
			}
		}
	}
	for _, child := range node.Content {
		normaliseKeys(child)
	}
}

// componentConfig folds the manifest defaults and a component's overrides into
// the same config.Config the measured targets use, so both paths resolve I identically.
func componentConfig(component Component, defaults Defaults, base config.Config) config.Config {
	cfg := base
	cfg.Provider = cmp.Or(component.Provider, defaults.Provider, "onprem")
	cfg.Region = cmp.Or(component.Region, defaults.Region)
	cfg.Country = cmp.Or(component.Country, defaults.Country)
	cfg.Zone = cmp.Or(component.Zone, defaults.Zone)
	cfg.Intensity = firstPositive(component.Intensity, defaults.Intensity, base.Intensity)
	cfg.PUE = firstPositive(component.PUE, defaults.PUE)
	return cfg
}

func firstPositive(values ...float64) float64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}

// EstimateComponent computes energy and carbon for one declared component.
func EstimateComponent(component Component, defaults Defaults, base config.Config,
	cache map[string]grid.Intensity) (sci.ComponentResult, error) {
	kind := component.Type
	if kind == "" {
		kind = "compute"
	}
	hours := firstPositive(component.Hours, defaults.PeriodHours, 1)
	replicas := component.Replicas
	if replicas <= 0 {
		replicas = 1
	}
	cfg := componentConfig(component, defaults, base)
	if err := cfg.Validate(); err != nil {
		return sci.ComponentResult{}, err
	}
	profile := cfg.Profile()

	var parts []energy.EnergyPart
	switch kind {
	case "compute":
		vcpus := component.VCPUs
		if vcpus <= 0 {
			vcpus = 1
		}
		util := 0.5
		if component.Utilisation != nil {
			util = *component.Utilisation
		}
		watts := profile.MinW + util*(profile.MaxW-profile.MinW)
		parts = append(parts, energy.EnergyPart{Name: "cpu", KWh: vcpus * replicas * watts * hours / 1000})
		if component.MemoryGB > 0 {
			parts = append(parts,
				energy.EnergyPart{Name: "memory", KWh: energy.MemorykWh(component.MemoryGB*replicas, hours)})
		}
	case "storage":
		medium := component.Medium
		if medium == "" {
			medium = "ssd"
		}
		if _, ok := coefficients.StorageWPerTB[medium]; !ok {
			return sci.ComponentResult{}, fmt.Errorf("unknown storage medium %q (ssd or hdd)", medium)
		}
		parts = append(parts,
			energy.EnergyPart{Name: "storage", KWh: energy.StoragekWh(component.StorageGB*replicas, hours, medium)})
	case "network":
		parts = append(parts, energy.EnergyPart{Name: "network", KWh: energy.NetworkkWh(component.NetworkGB)})
	case "device":
		// End-user devices draw from the grid directly: no datacentre PUE.
		parts = append(parts, energy.EnergyPart{Name: "device", KWh: component.Watts * hours * replicas / 1000})
	default:
		return sci.ComponentResult{}, fmt.Errorf("unknown component type %q in %q",
			kind, componentName(component, kind))
	}

	var subtotal float64
	for _, part := range parts {
		subtotal += part.KWh
	}
	if kind != "device" {
		if overhead := subtotal * (profile.PUE - 1); overhead != 0 {
			parts = append(parts, energy.EnergyPart{Name: "datacentre_overhead", KWh: overhead})
			subtotal += overhead
		}
	}

	intensity, err := resolveCached(cfg, cache)
	if err != nil {
		return sci.ComponentResult{}, err
	}
	embodied := componentEmbodied(component, hours, replicas)
	return sci.ComponentResult{
		Name:            componentName(component, kind),
		Type:            kind,
		Hours:           hours,
		Replicas:        replicas,
		EnergyKWh:       subtotal,
		EnergyBreakdown: parts,
		Intensity:       intensity,
		Operational:     subtotal * intensity.Value,
		Embodied:        embodied,
		Total:           subtotal*intensity.Value + embodied,
	}, nil
}

func componentName(component Component, kind string) string {
	if component.Name != "" {
		return component.Name
	}
	return kind
}

// resolveCached keeps one manifest from making the same API call per component.
func resolveCached(cfg config.Config, cache map[string]grid.Intensity) (grid.Intensity, error) {
	key := strings.Join([]string{cfg.Region, cfg.Country, cfg.Zone,
		fmt.Sprintf("%g", cfg.Intensity), cfg.IntensityBasis}, "|")
	if hit, ok := cache[key]; ok {
		return hit, nil
	}
	intensity, err := grid.ResolveIntensity(cfg)
	if err != nil {
		return grid.Intensity{}, err
	}
	cache[key] = intensity
	return intensity, nil
}

func componentEmbodied(component Component, hours, replicas float64) float64 {
	spec := component.Embodied
	if spec == nil {
		return 0
	}
	deviceKg := spec.DeviceKg
	if deviceKg <= 0 {
		deviceKg = coefficients.Hardware["server"].EmbodiedKg
	}
	lifespan := spec.LifespanYears
	if lifespan <= 0 {
		lifespan = coefficients.Hardware["server"].LifespanYears
	}
	resourceShare := 1.0
	switch {
	case spec.ResourceShare != nil:
		resourceShare = *spec.ResourceShare
	case spec.TotalVCPUs > 0 && component.VCPUs > 0:
		resourceShare = component.VCPUs / spec.TotalVCPUs
	}
	timeShare := hours / (lifespan * coefficients.HoursPerYear)
	return deviceKg * 1000 * timeShare * resourceShare * replicas
}

// EstimateManifest turns a declared boundary into one disclosure.
func EstimateManifest(manifest Manifest, base config.Config, path string) (*sci.Report, error) {
	defaults := manifest.Defaults
	if defaults.PeriodHours == 0 {
		defaults.PeriodHours = manifest.PeriodHours
	}
	if len(manifest.Components) == 0 {
		return nil, fmt.Errorf("manifest declares no components")
	}

	quantity := manifest.FunctionalUnit.Quantity
	if quantity <= 0 {
		quantity = 1
	}
	label := manifest.FunctionalUnit.Label
	if label == "" {
		label = "run"
	}

	cache := map[string]grid.Intensity{}
	var rows []sci.ComponentResult
	var energyKWh, operational, embodied float64
	var breakdown []energy.EnergyPart
	var boundary []string
	for _, component := range manifest.Components {
		row, err := EstimateComponent(component, defaults, base, cache)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
		energyKWh += row.EnergyKWh
		operational += row.Operational
		embodied += row.Embodied
		breakdown = append(breakdown, energy.EnergyPart{Name: row.Name, KWh: row.EnergyKWh})
		boundary = append(boundary, fmt.Sprintf("%s (%s)", row.Name, row.Type))
	}

	name := manifest.Name
	if name == "" {
		name = "software system"
	}
	where := path
	if where == "" {
		where = "manifest"
	}
	total := operational + embodied
	report := &sci.Report{
		Tool:    "sci-disclose",
		Version: coefficients.Version,
		Target: sci.Target{Kind: "manifest", Description: fmt.Sprintf("%s (%s)", name, where),
			Path: path},
		Components:      rows,
		EnergyKWh:       energyKWh,
		EnergySource:    "declared (manifest)",
		EnergyBreakdown: breakdown,
		Operational:     operational,
		Embodied:        embodied,
		Total:           total,
		FunctionalUnit:  sci.FunctionalUnit{Label: label, Quantity: quantity},
		SCI:             total / quantity,
		SCIUnit:         "gCO2e per " + label,
		Boundary:        boundary,
		Assumptions: sci.Assumptions{
			Provider:    defaults.Provider,
			Components:  len(rows),
			PeriodHours: defaults.PeriodHours,
		},
		Notes: append(manifest.Notes, "declared, not measured: the SCI is only as good "+
			"as the utilisation and functional unit in the manifest"),
	}
	// A single headline I only makes sense when every component shares one.
	if same := sameIntensity(rows); same != nil {
		report.Intensity = same
	}
	return report, nil
}

func sameIntensity(rows []sci.ComponentResult) *grid.Intensity {
	if len(rows) == 0 {
		return nil
	}
	first := rows[0].Intensity
	for _, row := range rows[1:] {
		if row.Intensity.Value != first.Value || row.Intensity.Source != first.Source {
			return nil
		}
	}
	return &first
}
