package payloads

import "github.com/gofrs/uuid"

// StorageOperation represents an in-progress operation on an SR.
type StorageOperation string

const (
	StorageOperationDestroy              StorageOperation = "destroy"
	StorageOperationForget               StorageOperation = "forget"
	StorageOperationPBDCreate            StorageOperation = "pbd_create"
	StorageOperationPBDDestroy           StorageOperation = "pbd_destroy"
	StorageOperationPlug                 StorageOperation = "plug"
	StorageOperationScan                 StorageOperation = "scan"
	StorageOperationUnplug               StorageOperation = "unplug"
	StorageOperationUpdate               StorageOperation = "update"
	StorageOperationVDIClone             StorageOperation = "vdi_clone"
	StorageOperationVDICreate            StorageOperation = "vdi_create"
	StorageOperationVDIDataDestroy       StorageOperation = "vdi_data_destroy"
	StorageOperationVDIDestroy           StorageOperation = "vdi_destroy"
	StorageOperationVDIDisableCBT        StorageOperation = "vdi_disable_cbt"
	StorageOperationVDIEnableCBT         StorageOperation = "vdi_enable_cbt"
	StorageOperationVDIIntroduce         StorageOperation = "vdi_introduce"
	StorageOperationVDIListChangedBlocks StorageOperation = "vdi_list_changed_blocks"
	StorageOperationVDIMirror            StorageOperation = "vdi_mirror"
	StorageOperationVDIResize            StorageOperation = "vdi_resize"
	StorageOperationVDISetOnBoot         StorageOperation = "vdi_set_on_boot"
	StorageOperationVDISnapshot          StorageOperation = "vdi_snapshot"
)

// AllocationStrategy represents the disk allocation strategy for an SR.
type AllocationStrategy string

const (
	// AllocationStrategyUnknown indicates the allocation strategy is unknown.
	AllocationStrategyUnknown AllocationStrategy = "unknown"
	// AllocationStrategyThin indicates thin provisioning.
	AllocationStrategyThin AllocationStrategy = "thin"
	// AllocationStrategyThick indicates thick provisioning.
	AllocationStrategyThick AllocationStrategy = "thick"
)

// SR represents a Storage Repository in Xen Orchestra.
// An SR is a storage container that holds VDIs (Virtual Disk Images).
type SR struct {
	ID   uuid.UUID    `json:"id"`
	UUID uuid.UUID    `json:"uuid"`
	Type ResourceType `json:"type"`
	// Pool is the ID of the pool this SR belongs to.
	Pool    uuid.UUID `json:"$pool"`
	XapiRef string    `json:"_xapiRef"`
	// PBDs is the list of Physical Block Device IDs connecting hosts to this SR.
	PBDs []uuid.UUID `json:"$PBDs"`
	// Container is the ID of the container (pool or host) that owns this SR.
	Container uuid.UUID `json:"$container"`
	// VDIs is the list of Virtual Disk Image IDs stored in this SR.
	VDIs []uuid.UUID `json:"VDIs"`
	// AllocationStrategy indicates the disk allocation strategy (unknown, thin, thick).
	// This field is optional and may be absent for some SR types.
	AllocationStrategy AllocationStrategy `json:"allocationStrategy,omitempty"`
	// ContentType describes the type of content stored (e.g. "user", "iso", "metadata").
	ContentType string `json:"content_type"`
	// CurrentOperations contains any in-progress XAPI operations on this SR,
	CurrentOperations map[string]StorageOperation `json:"current_operations"`
	// InMaintenanceMode indicates whether this SR is currently in maintenance mode.
	InMaintenanceMode bool `json:"inMaintenanceMode"`
	// NameLabel is the human-readable name of the SR.
	NameLabel string `json:"name_label"`
	// NameDescription is the human-readable description of the SR.
	NameDescription string `json:"name_description"`
	// OtherConfig holds additional configuration key-value pairs.
	OtherConfig map[string]string `json:"other_config"`
	// PhysicalUsage is the number of bytes physically used on the underlying storage.
	PhysicalUsage float64 `json:"physical_usage"`
	// Shared indicates whether the SR is shared across multiple hosts in the pool.
	Shared bool `json:"shared"`
	// Size is the total capacity of the SR in bytes.
	Size float64 `json:"size"`
	// SmConfig holds Storage Manager plugin configuration key-value pairs.
	SmConfig map[string]string `json:"sm_config"`
	// SRType is the XAPI SR type (e.g. "lvm", "nfs", "ext", "iso").
	SRType string `json:"SR_type"`
	// Tags is the list of user-defined tags on this SR.
	Tags []string `json:"tags"`
	// Usage is the number of bytes allocated (virtual size of all VDIs).
	Usage float64 `json:"usage"`
}
