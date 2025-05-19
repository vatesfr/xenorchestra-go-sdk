package payloads

import "github.com/gofrs/uuid"

type PoolCPUs struct {
	Cores   int `json:"cores"`
	Sockets int `json:"sockets"`
}

type Pool struct {
	AutoPoweron               bool              `json:"auto_poweron"`
	CurrentOperations         map[string]any    `json:"current_operations"`
	DefaultSR                 string            `json:"default_SR"`
	HAEnabled                 bool              `json:"HA_enabled"`
	HASRs                     []string          `json:"haSrs"`
	Master                    string            `json:"master"`
	Tags                      []string          `json:"tags"`
	NameDescription           string            `json:"name_description"`
	NameLabel                 string            `json:"name_label"`
	MigrationCompression      bool              `json:"migrationCompression"`
	XOSANPackInstallationTime *string           `json:"xosanPackInstallationTime,omitempty"`
	OtherConfig               map[string]string `json:"otherConfig"`
	CPUs                      PoolCPUs          `json:"cpus"`
	ZSTDSupported             bool              `json:"zstdSupported"`
	VTPMSupported             bool              `json:"vtpmSupported"`
	PlatformVersion           string            `json:"platform_version"`
	ID                        uuid.UUID         `json:"id"`
	Type                      string            `json:"type"`
	UUID                      uuid.UUID         `json:"uuid"`
	PoolRef                   string            `json:"$pool"`
	PoolID                    string            `json:"$poolId"`
	XAPIRef                   string            `json:"_xapiRef"`
}

type InstallParams struct {
	Method     string `json:"method,omitempty"`
	Repository string `json:"repository,omitempty"`
}

type VDIParams struct {
	Destroy         *bool   `json:"destroy,omitempty"`
	UserDevice      *string `json:"userdevice,omitempty"`
	Size            *int64  `json:"size,omitempty"` // Using int64 for size as it can be large
	SR              *string `json:"sr,omitempty"`
	NameDescription *string `json:"name_description,omitempty"`
	NameLabel       *string `json:"name_label,omitempty"`
}

type VIFParams struct {
	Destroy     *bool    `json:"destroy,omitempty"`
	Device      *string  `json:"device,omitempty"`
	IPV4Allowed []string `json:"ipv4_allowed,omitempty"`
	IPV6Allowed []string `json:"ipv6_allowed,omitempty"`
	MAC         *string  `json:"mac,omitempty"`
	Network     *string  `json:"network,omitempty"`
}

type CreateVMParams struct {
	Affinity              *string        `json:"affinity,omitempty"`
	AutoPoweron           *bool          `json:"auto_poweron,omitempty"`
	Boot                  bool           `json:"boot,omitempty"`
	Clone                 bool           `json:"clone,omitempty"`
	CloudConfig           *string        `json:"cloud_config,omitempty"`
	DestroyCloudConfigVDI bool           `json:"destroy_cloud_config_vdi,omitempty"`
	Install               *InstallParams `json:"install,omitempty"`
	Memory                *int           `json:"memory,omitempty"`
	NameDescription       *string        `json:"name_description,omitempty"`
	NameLabel             string         `json:"name_label"`
	NetworkConfig         *string        `json:"network_config,omitempty"`
	Template              string         `json:"template"`
	VDIs                  []VDIParams    `json:"vdis,omitempty"`
	VIFs                  []VIFParams    `json:"vifs,omitempty"`
}
