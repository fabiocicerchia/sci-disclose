package coefficients

// Coefficients. Every number here is a published estimate, not a measurement
// of your hardware; `sci coefficients` prints this table with its sources and
// every one of them is overridable from the command line.

// Version is the tool version, reported in every disclosure.
const Version = "0.2.0" // x-release-please-version

// CPUProfile is the per-vCPU power draw at idle and at full load, plus the
// datacentre PUE. Source: Cloud Carbon Footprint methodology (provider-averaged
// SPECpower fleets); PUE as published by each provider. On-prem PUE is the
// Uptime Institute global average.
type CPUProfile struct {
	MinW float64 `json:"min_w"`
	MaxW float64 `json:"max_w"`
	PUE  float64 `json:"pue"`
}

// CPUProfiles is keyed by the --provider flag.
var CPUProfiles = map[string]CPUProfile{
	"aws":    {MinW: 0.74, MaxW: 3.50, PUE: 1.135},
	"gcp":    {MinW: 0.71, MaxW: 4.26, PUE: 1.100},
	"azure":  {MinW: 0.78, MaxW: 3.76, PUE: 1.185},
	"onprem": {MinW: 0.74, MaxW: 3.50, PUE: 1.580},
	"laptop": {MinW: 0.30, MaxW: 2.00, PUE: 1.000},
}

// ProviderNames is CPUProfiles in a stable order, for help text and listings.
var ProviderNames = []string{"aws", "gcp", "azure", "onprem", "laptop"}

const (
	// MemoryWPerGB is Etsy's "Cloud Jewels" coefficient as carried by Cloud
	// Carbon Footprint: 0.000392 kWh/GBh, written here as watts per GB.
	MemoryWPerGB = 0.392
	// NetworkKWhPerGB is the Cloud Carbon Footprint networking coefficient.
	NetworkKWhPerGB = 0.001
	// HoursPerYear amortises embodied emissions over a device lifespan.
	HoursPerYear = 8760.0
	// DefaultIntensity is the world average, used when no region is given.
	DefaultIntensity = 481.0
)

// StorageWPerTB is watts per provisioned terabyte, from Cloud Carbon Footprint.
var StorageWPerTB = map[string]float64{"ssd": 1.2, "hdd": 0.65}

// Device is the embodied emissions of a whole device and the lifespan it is
// amortised over. The server figure is a mid-range 2U rack server from
// published LCAs (Dell PAIA, Boavizta: 1000-1600 kgCO2e); laptop and phone are
// vendor LCA midpoints.
type Device struct {
	EmbodiedKg    float64 `json:"embodied_kg"`
	LifespanYears float64 `json:"lifespan_years"`
}

// Hardware is keyed by the --hardware flag.
var Hardware = map[string]Device{
	"server": {EmbodiedKg: 1200, LifespanYears: 4},
	"laptop": {EmbodiedKg: 250, LifespanYears: 4},
	"phone":  {EmbodiedKg: 60, LifespanYears: 3},
}

// HardwareNames is Hardware in a stable order.
var HardwareNames = []string{"server", "laptop", "phone"}

// GridZones holds yearly-average grid intensity (gCO2e/kWh) per Electricity
// Maps zone. Yearly averages, not the marginal real-time figure the spec
// prefers: the intensity API or --intensity get you closer to that.
var GridZones = map[string]float64{
	"SE": 25, "NO": 30, "FR": 56, "CA-QC": 30, "CH": 45, "FI": 80,
	"IE": 290, "GB": 230, "NL": 330, "DE": 380, "ES": 170, "IT": 330, "BE": 160,
	"US": 380, "CA": 120, "AU": 550, "IN": 650,
	"US-OR": 120, "US-CA": 230, "US-VA": 390, "US-OH": 480, "US-TX": 400,
	"BR": 100, "JP": 480, "SG": 480, "AU-NSW": 550, "IN-WE": 650, "ZA": 700,
	"WORLD": 481,
}

// RegionZone maps a cloud region onto the grid zone it sits in.
var RegionZone = map[string]string{
	// AWS
	"eu-north-1": "SE", "eu-west-1": "IE", "eu-west-2": "GB", "eu-west-3": "FR",
	"eu-central-1": "DE", "eu-south-1": "IT", "us-east-1": "US-VA",
	"us-east-2": "US-OH", "us-west-1": "US-CA", "us-west-2": "US-OR",
	"ca-central-1": "CA-QC", "sa-east-1": "BR", "ap-northeast-1": "JP",
	"ap-southeast-1": "SG", "ap-southeast-2": "AU-NSW", "ap-south-1": "IN-WE",
	// GCP
	"europe-north1": "FI", "europe-west1": "BE", "europe-west2": "GB",
	"europe-west3": "DE", "europe-west4": "NL", "us-central1": "US-OH",
	"us-east4": "US-VA", "us-west1": "US-OR", "northamerica-northeast1": "CA-QC",
	// Azure
	"swedencentral": "SE", "northeurope": "IE", "westeurope": "NL",
	"uksouth": "GB", "francecentral": "FR", "germanywestcentral": "DE",
	"eastus": "US-VA", "westus2": "US-OR", "canadaeast": "CA-QC",
}

// CoefficientSources documents where each constant above comes from.
var CoefficientSources = [][2]string{
	{"CPU min/max watts per vCPU", "Cloud Carbon Footprint methodology, " +
		"provider-averaged SPECpower fleet data"},
	{"PUE (aws/gcp/azure)", "provider-published fleet averages"},
	{"PUE (onprem 1.58)", "Uptime Institute global datacentre survey"},
	{"Memory 0.392 W/GB", "Etsy Cloud Jewels, via Cloud Carbon Footprint " +
		"(0.000392 kWh/GBh)"},
	{"Network 0.001 kWh/GB", "Cloud Carbon Footprint networking coefficient"},
	{"Storage 1.2 W/TB SSD, 0.65 W/TB HDD", "Cloud Carbon Footprint"},
	{"Embodied server 1200 kgCO2e / 4y", "mid-range 2U server LCAs " +
		"(Dell PAIA, Boavizta: 1000-1600 kgCO2e)"},
	{"Embodied laptop 250 kgCO2e / 4y, phone 60 kgCO2e / 3y", "vendor LCAs"},
	{"Grid intensity table", "Electricity Maps yearly zone averages; " +
		"world average 481 gCO2e/kWh (Ember). Superseded at runtime by the " +
		"last-hour figure from the intensity API when a zone is known."},
}

// DefaultIntensityAPI is overridable with --intensity-api or $SCI_INTENSITY_API.
const DefaultIntensityAPI = "https://ci-api.fabiocicerchia.it"

// IntensityBases are the four figures every reading carries, in the order the
// tool falls back through them. consumption_lifecycle is the default: it counts
// the whole supply chain of the electricity actually consumed in that country,
// which is the figure the API's own guidance says to report a footprint with.
var IntensityBases = []string{"consumption_lifecycle", "lifecycle", "consumption_direct", "direct"}

// RegionCountry maps a cloud region onto the ISO-3166 alpha-2 country the API
// is keyed by. Sub-national bidding zones are never guessed from a region —
// pass --zone COUNTRY/ZONE for those.
var RegionCountry = map[string]string{
	// AWS
	"eu-north-1": "SE", "eu-west-1": "IE", "eu-west-2": "GB", "eu-west-3": "FR",
	"eu-central-1": "DE", "eu-south-1": "IT", "us-east-1": "US", "us-east-2": "US",
	"us-west-1": "US", "us-west-2": "US", "ca-central-1": "CA", "sa-east-1": "BR",
	"ap-northeast-1": "JP", "ap-southeast-1": "SG", "ap-southeast-2": "AU",
	"ap-south-1": "IN",
	// GCP
	"europe-north1": "FI", "europe-west1": "BE", "europe-west2": "GB",
	"europe-west3": "DE", "europe-west4": "NL", "us-central1": "US",
	"us-east4": "US", "us-west1": "US", "northamerica-northeast1": "CA",
	// Azure
	"swedencentral": "SE", "northeurope": "IE", "westeurope": "NL",
	"uksouth": "GB", "francecentral": "FR", "germanywestcentral": "DE",
	"eastus": "US", "westus2": "US", "canadaeast": "CA",
}
