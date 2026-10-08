package payloads

import "github.com/gofrs/uuid"

// Template represents a VM template as exposed by the REST resource
// "vm-templates". Unlike VMs (and most other resources), a template's REST id
// is not always a plain UUID:
//   - for default templates (the "built-in" install templates),
//     id = "<poolUuid>-<templateUuid>" (a composite, 73-character string);
//   - for non-default templates, id is the bare template UUID.
//
// That is why the service that wraps this resource takes string ids. The bare
// template UUID (the one create_vm expects) is available in UUID.
type Template struct {
	// ID is the template's REST id. For default templates it is the composite
	// "<poolUuid>-<templateUuid>" value, e.g.
	// "b7569d99-30f8-178a-7d94-801de3e29b5b-f873abe0-b138-4995-8f6f-498b423d234d";
	// for non-default templates it is the bare template UUID.
	ID string `json:"id"`
	// UUID is the bare template UUID. It is the value create_vm (the
	// "template" parameter) expects.
	UUID uuid.UUID `json:"uuid"`
	// Type is the resource type, "VM-template" for a template.
	Type ResourceType `json:"type"`
	// XapiRef is the internal XAPI reference of the underlying object.
	XapiRef string `json:"_xapiRef"`

	// Pool is the pool the template belongs to.
	Pool uuid.UUID `json:"$pool"`
	// Container is the container the template is bound to (the pool, for a
	// template).
	Container uuid.UUID `json:"$container"`
	// VBDs are the IDs of the VBDs of the template.
	VBDs []uuid.UUID `json:"$VBDs"`
	// VGPUs are the IDs of the GPUs attached to the template.
	VGPUs []uuid.UUID `json:"$VGPUs"`
	// VIFs are the IDs of the VIFs of the template.
	VIFs []uuid.UUID `json:"VIFs"`
	// VTPMs are the IDs of the virtual TPMs of the template.
	VTPMs []uuid.UUID `json:"VTPMs"`
	// Snapshots are the IDs of the snapshots of the template.
	Snapshots []uuid.UUID `json:"snapshots"`

	// IsDefaultTemplate is true when the template is a built-in (default)
	// template of the pool, and false for templates created by the user.
	IsDefaultTemplate bool `json:"isDefaultTemplate"`

	NameLabel           string                 `json:"name_label"`
	NameDescription     string                 `json:"name_description"`
	PowerState          PowerState             `json:"power_state"`
	VirtualizationMode  DomainType             `json:"virtualizationMode"`
	HighAvailability    HighAvailability       `json:"high_availability"`
	Memory              Memory                 `json:"memory"`
	CPUs                CPUs                   `json:"CPUs"`
	Boot                Boot                   `json:"boot"`
	AutoPoweron         bool                   `json:"auto_poweron"`
	StartDelay          int                    `json:"startDelay"`
	Tags                []string               `json:"tags"`
	SecureBoot          bool                   `json:"secureBoot"`
	Viridian            bool                   `json:"viridian"`
	IsFirmwareSupported bool                   `json:"isFirmwareSupported"`
	IsNestedVirtEnabled bool                   `json:"isNestedVirtEnabled"`
	HasVendorDevice     bool                   `json:"hasVendorDevice"`
	NeedsVtpm           bool                   `json:"needsVtpm"`
	Addresses           map[string]string      `json:"addresses"`
	BIOSStrings         map[string]string      `json:"bios_strings"`
	BlockedOperations   map[VMOperation]string `json:"blockedOperations"`
	CurrentOperations   map[string]VMOperation `json:"current_operations"`
	Other               map[string]string      `json:"other"`
	XenStoreData        map[string]string      `json:"xenStoreData"`
	TemplateInfo        TemplateInfo           `json:"template_info"`
	Creation            Creation               `json:"creation"`

	// PVArgs are the PV boot arguments of the template.
	PVArgs string `json:"PV_args,omitempty"`
	// AffinityHost is the ID of the host the template is pinned to.
	AffinityHost *uuid.UUID `json:"affinityHost,omitempty"`
	// AttachedPcis are the IDs of the PCI devices attached to the template.
	AttachedPcis []string `json:"attachedPcis,omitempty"`
	// CPUCap is the maximum fraction of a physical CPU the template can use
	// (1.0 = full CPU).
	CPUCap float64 `json:"cpuCap,omitempty"`
	// CPUMask is the list of physical CPUs the template can run on.
	CPUMask []int `json:"cpuMask,omitempty"`
	// CPUWeight is the scheduling weight of the template (relative CPU time).
	CPUWeight int `json:"cpuWeight,omitempty"`
	// CoresPerSocket is the number of cores per socket, when set.
	CoresPerSocket int `json:"coresPerSocket,omitempty"`
	// InstallTime is the Unix timestamp of the OS installation, when known.
	InstallTime *int64 `json:"installTime,omitempty"`
	// StartTime is the Unix timestamp of the last power-on, when set.
	StartTime *int64 `json:"startTime,omitempty"`
	// MainIpAddress is the main IP address of the template.
	MainIpAddress string `json:"mainIpAddress,omitempty"`
	// ManagementAgentDetected is true when a management agent is detected.
	ManagementAgentDetected bool `json:"managementAgentDetected,omitempty"`
	// NicType is the type of the virtual network interface, when set.
	NicType string `json:"nicType,omitempty"`
	// Notes are free-form notes attached to the template.
	Notes string `json:"notes,omitempty"`
	// OsVersion is the OS version reported by the template. It is empty when
	// the OS has not been detected.
	OsVersion map[string]string `json:"os_version"`
	// Parent is the ID of the snapshot the template was created from, when set.
	Parent *uuid.UUID `json:"parent,omitempty"`
	// PvDriversDetected is true when PV drivers (Xen tools) are detected.
	PvDriversDetected bool `json:"pvDriversDetected,omitempty"`
	// PvDriversUpToDate is true when the detected PV drivers are up to date.
	PvDriversUpToDate bool `json:"pvDriversUpToDate,omitempty"`
	// PvDriversVersion is the version of the detected PV drivers.
	PvDriversVersion string `json:"pvDriversVersion,omitempty"`
	// ResourceSet is the ID of the resource set the template belongs to, when set.
	ResourceSet string `json:"resourceSet,omitempty"`
	// SuspendSr is the ID of the SR used to suspend the template.
	SuspendSr *uuid.UUID `json:"suspendSr,omitempty"`
	// Vga is the VGA type of the template (e.g. "std").
	Vga string `json:"vga,omitempty"`
	// Videoram is the amount of video memory, in MiB.
	Videoram Videoram `json:"videoram,omitempty"`
	// Docker holds the Docker configuration of the template, when enabled.
	Docker Docker `json:"docker,omitempty"`
}

// HighAvailability represents the high-availability mode of a VM template.
type HighAvailability string

const (
	HighAvailabilityBestEffort HighAvailability = "best-effort"
	HighAvailabilityRestart    HighAvailability = "restart"
)

// Docker is the Docker configuration attached to a template
// (the "docker" field).
type Docker struct {
	Enabled    bool     `json:"enabled"`
	Containers []string `json:"containers,omitempty"`
	Info       string   `json:"info,omitempty"`
	Version    string   `json:"version,omitempty"`
}

// TemplateInfo is the provisioning/install information attached to a template
// (the "template_info" field).
type TemplateInfo struct {
	// Arch is the CPU architecture the template targets (e.g. "x86_64").
	Arch string `json:"arch,omitempty"`
	// Disks are the disks to be provisioned for the template.
	Disks []TemplateDisk `json:"disks"`
	// InstallMethods are the installation methods supported by the template
	// (e.g. "cdrom", "http"). It is empty for templates that do not describe
	// installation methods (e.g. disk-based templates).
	InstallMethods []string `json:"install_methods"`
	// InstallRepository is the URL of the OS repository used to install the
	// template.
	InstallRepository string `json:"install_repository,omitempty"`
}

// TemplateDisk describes one disk of a template to be provisioned.
type TemplateDisk struct {
	Bootable bool   `json:"bootable"`
	Size     int64  `json:"size"`
	SR       string `json:"SR"`
	Type     string `json:"type"`
}
