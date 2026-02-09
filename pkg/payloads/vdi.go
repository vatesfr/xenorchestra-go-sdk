package payloads

import (
	"github.com/gofrs/uuid"
)

// VDI represents a Virtual Disk Image in Xen Orchestra
type VDI struct {
	ID                uuid.UUID               `json:"id,omitempty"`
	UUID              uuid.UUID               `json:"uuid"`
	Type              ResourceType            `json:"type"`
	NameLabel         string                  `json:"name_label"`
	NameDescription   string                  `json:"name_description"`
	Size              int64                   `json:"size"`
	Usage             int64                   `json:"usage"`
	VDIType           VDIType                 `json:"VDI_type"`
	CBTEnabled        *bool                   `json:"cbt_enabled,omitempty"`
	Missing           bool                    `json:"missing"`
	Parent            *uuid.UUID              `json:"parent,omitempty"`
	ImageFormat       *string                 `json:"image_format,omitempty"`
	Snapshots         []uuid.UUID             `json:"snapshots"`
	Tags              []string                `json:"tags"`
	CurrentOperations map[string]VDIOperation `json:"current_operations"`
	OtherConfig       map[string]string       `json:"other_config"`
	SR                uuid.UUID               `json:"$SR"`
	VBDs              []uuid.UUID             `json:"$VBDs"`
	PoolID            uuid.UUID               `json:"$poolId"`
	XapiRef           string                  `json:"_xapiRef"`
}

type ResourceType string

// VDI resource type identifier
const VDIResourceType ResourceType = "VDI"

type VDIType string

// VDI type constants
const (
	VDITypeUser        VDIType = "user"
	VDITypeSystem      VDIType = "system"
	VDITypeSuspend     VDIType = "suspend"
	VDITypeRRD         VDIType = "rrd"
	VDITypeRedoLog     VDIType = "redo_log"
	VDITypePVSCache    VDIType = "pvs_cache"
	VDITypeMetadata    VDIType = "metadata"
	VDITypeHAStatefile VDIType = "ha_statefile"
	VDITypeEphemeral   VDIType = "ephemeral"
	VDITypeCrashdump   VDIType = "crashdump"
	VDITypeCBTMetadata VDIType = "cbt_metadata"
)

type VDIOperation string

// VDI operation constants
const (
	VDIOperationBlocked           VDIOperation = "blocked"
	VDIOperationClone             VDIOperation = "clone"
	VDIOperationCopy              VDIOperation = "copy"
	VDIOperationDataDestroy       VDIOperation = "data_destroy"
	VDIOperationDestroy           VDIOperation = "destroy"
	VDIOperationDisableCBT        VDIOperation = "disable_cbt"
	VDIOperationEnableCBT         VDIOperation = "enable_cbt"
	VDIOperationForceUnlock       VDIOperation = "force_unlock"
	VDIOperationForget            VDIOperation = "forget"
	VDIOperationGenerateConfig    VDIOperation = "generate_config"
	VDIOperationListChangedBlocks VDIOperation = "list_changed_blocks"
	VDIOperationMirror            VDIOperation = "mirror"
	VDIOperationResize            VDIOperation = "resize"
	VDIOperationResizeOnline      VDIOperation = "resize_online"
	VDIOperationSetOnBoot         VDIOperation = "set_on_boot"
	VDIOperationSnapshot          VDIOperation = "snapshot"
	VDIOperationUpdate            VDIOperation = "update"
)
