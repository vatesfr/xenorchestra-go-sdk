package payloads

import "github.com/gofrs/uuid"

// HostCPUInfo represents CPU information for a Host.
type HostCPUInfo struct {
	CPUCount        string `json:"cpu_count"`
	SocketCount     string `json:"socket_count"`
	ThreadsPerCore  string `json:"threads_per_core"`
	Vendor          string `json:"vendor"`
	Speed           string `json:"speed"`
	ModelName       string `json:"modelname"`
	Family          string `json:"family"`
	Model           string `json:"model"`
	Stepping        string `json:"stepping"`
	Flags           string `json:"flags"`
	FeaturesPV      string `json:"features_pv"`
	FeaturesHVM     string `json:"features_hvm"`
	FeaturesHVMHost string `json:"features_hvm_host"`
	FeaturesPVHost  string `json:"features_pv_host"`
}

// HostBIOSStrings represents BIOS information for a Host.
type HostBIOSStrings struct {
	BIOSVendor            string `json:"bios-vendor"`
	BIOSVersion           string `json:"bios-version"`
	SystemManufacturer    string `json:"system-manufacturer"`
	SystemProductName     string `json:"system-product-name"`
	SystemVersion         string `json:"system-version"`
	SystemSerialNumber    string `json:"system-serial-number"`
	BaseboardManufacturer string `json:"baseboard-manufacturer"`
	BaseboardProductName  string `json:"baseboard-product-name"`
	BaseboardVersion      string `json:"baseboard-version"`
	BaseboardSerialNumber string `json:"baseboard-serial-number"`
	OEM1                  string `json:"oem-1"`
	OEM2                  string `json:"oem-2"`
	OEM3                  string `json:"oem-3"`
	OEM4                  string `json:"oem-4"`
	OEM5                  string `json:"oem-5"`
	OEM6                  string `json:"oem-6"`
	HPRomBios             string `json:"hp-rombios"`
}

// HostChipsetInfo represents chipset information for a Host.
type HostChipsetInfo struct {
	IOMMU bool `json:"iommu"`
}

// HostCPUCores represents CPU core counts for a Host.
type HostCPUCores struct {
	Cores   int `json:"cores"`
	Sockets int `json:"sockets"`
}

// HostMemory represents memory information for a Host.
type HostMemory struct {
	Usage int64 `json:"usage"`
	Size  int64 `json:"size"`
}

// HostLicenseServer represents license server configuration for a Host.
type HostLicenseServer struct {
	Address string `json:"address"`
	Port    string `json:"port"`
}

// HostCertificate represents a certificate on a Host.
type HostCertificate struct {
	Fingerprint string `json:"fingerprint"`
	NotAfter    int64  `json:"notAfter"`
}

// Host represents a Xen Orchestra Host object.
// Based on Partial_Unbrand_XoHost_ from swagger.
type Host struct {
	ID                uuid.UUID              `json:"id"`
	Uuid              string                 `json:"uuid"`
	Type              string                 `json:"type"`
	NameLabel         string                 `json:"name_label"`
	NameDescription   string                 `json:"name_description"`
	Address           string                 `json:"address"`
	Hostname          string                 `json:"hostname"`
	Build             string                 `json:"build"`
	Version           string                 `json:"version"`
	ProductBrand      string                 `json:"productBrand"`
	PowerState        string                 `json:"power_state"`
	PowerOnMode       string                 `json:"powerOnMode"`
	IscsiIqn          string                 `json:"iscsiIqn"`
	XapiRef           string                 `json:"_xapiRef"`
	CPUs              *HostCPUInfo           `json:"CPUs,omitempty"`
	BIOSStrings       *HostBIOSStrings       `json:"bios_strings,omitempty"`
	ChipsetInfo       *HostChipsetInfo       `json:"chipset_info,omitempty"`
	HostCPUCores      *HostCPUCores          `json:"cpus,omitempty"`
	Memory            *HostMemory            `json:"memory,omitempty"`
	LicenseServer     *HostLicenseServer     `json:"license_server,omitempty"`
	Enabled           bool                   `json:"enabled"`
	HVMCapable        bool                   `json:"hvmCapable"`
	Multipathing      bool                   `json:"multipathing"`
	ZstdSupported     bool                   `json:"zstdSupported"`
	RebootRequired    bool                   `json:"rebootRequired"`
	ControlDomain     uuid.UUID              `json:"controlDomain"`
	Pool              uuid.UUID              `json:"$pool"`
	PoolID            uuid.UUID              `json:"$poolId"`
	StartTime         int64                  `json:"startTime"`
	AgentStartTime    int64                  `json:"agentStartTime"`
	LicenseExpiry     *int64                 `json:"license_expiry"`
	CurrentOperations map[string]interface{} `json:"current_operations,omitempty"`
	OtherConfig       map[string]interface{} `json:"otherConfig,omitempty"`
	LicenseParams     map[string]interface{} `json:"license_params,omitempty"`
	Logging           map[string]interface{} `json:"logging,omitempty"`
	ResidentVMs       []uuid.UUID            `json:"residentVms,omitempty"`
	PIFs              []uuid.UUID            `json:"PIFs,omitempty"`
	PIFsRef           []uuid.UUID            `json:"$PIFs,omitempty"`
	PCIs              []uuid.UUID            `json:"PCIs,omitempty"`
	PCIsRef           []uuid.UUID            `json:"$PCIs,omitempty"`
	PGPUs             []uuid.UUID            `json:"PGPUs,omitempty"`
	PGPUsRef          []uuid.UUID            `json:"$PGPUs,omitempty"`
	PBDsRef           []uuid.UUID            `json:"$PBDs,omitempty"`
	Patches           []interface{}          `json:"patches,omitempty"`
	SupplementalPacks []interface{}          `json:"supplementalPacks,omitempty"`
	Tags              []string               `json:"tags,omitempty"`
	Certificates      []HostCertificate      `json:"certificates,omitempty"`
}
