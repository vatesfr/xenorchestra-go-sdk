package payloads

import (
	"encoding/json"
	"strconv"

	"github.com/gofrs/uuid"
)

/*
Videoram is represented as an integer, but sometimes comes as a string in the API response.
Therefore, we need to handle both formats by parsing it as a string when necessary and
converting it to an integer.
*/
type Videoram int

func (v *Videoram) UnmarshalJSON(data []byte) error {
	var intValue int
	if err := json.Unmarshal(data, &intValue); err == nil {
		*v = Videoram(intValue)
		return nil
	}

	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err != nil {
		return err
	}

	if stringValue == "" {
		*v = 0
		return nil
	}

	intValue, err := strconv.Atoi(stringValue)
	if err != nil {
		return err
	}

	*v = Videoram(intValue)
	return nil
}

// Introducing stronger type for UUID by using a package rather than a string.
type VM struct {
	ID                 uuid.UUID              `json:"id,omitempty"`
	Template           uuid.UUID              `json:"template,omitempty"`
	NameLabel          string                 `json:"name_label"`
	NameDescription    string                 `json:"name_description"`
	PowerState         PowerState             `json:"power_state,omitempty"`
	Memory             Memory                 `json:"memory"`
	CPUs               CPUs                   `json:"CPUs"`
	VIFs               []string               `json:"VIFs,omitempty"`
	VBDs               []uuid.UUID            `json:"$VBDs,omitempty"`
	Tags               []string               `json:"tags,omitempty"`
	AutoPoweron        bool                   `json:"auto_poweron"`
	HA                 string                 `json:"high_availability,omitempty"`
	VirtualizationMode string                 `json:"virtualizationMode,omitempty"`
	StartDelay         int                    `json:"startDelay,omitempty"`
	ExpNestedHvm       bool                   `json:"expNestedHvm,omitempty"`
	Boot               Boot                   `json:"boot"`
	Videoram           Videoram               `json:"videoram,omitempty"`
	Vga                string                 `json:"vga,omitempty"`
	XenstoreData       map[string]string      `json:"xenStoreData,omitempty"`
	BlockedOperations  map[VMOperation]string `json:"blockedOperations,omitempty"`
	MainIpAddress      string                 `json:"mainIpAddress,omitempty"`
	PoolID             uuid.UUID              `json:"$poolId,omitempty"`
	Container          uuid.UUID              `json:"$container,omitempty"`
	Snapshots          []uuid.UUID            `json:"snapshots,omitempty"`
	Type               ResourceType           `json:"type"`
	Creation           Creation               `json:"creation,omitempty"`
	CurrentOperations  map[string]VMOperation `json:"current_operations,omitempty"`
}

type Creation struct {
	Date     string                 `json:"date,omitempty"`
	Template uuid.UUID              `json:"template,omitempty"`
	User     string                 `json:"user,omitempty"`
	Raw      map[string]interface{} `json:"-"`
}

func (c *Creation) UnmarshalJSON(data []byte) error {
	type creationAlias Creation

	if err := json.Unmarshal(data, (*creationAlias)(c)); err != nil {
		return err
	}

	return json.Unmarshal(data, &c.Raw)
}

func (v *VM) UnmarshalJSON(data []byte) error {
	type vmAlias VM

	if err := json.Unmarshal(data, (*vmAlias)(v)); err != nil {
		return err
	}

	if v.Template == uuid.Nil {
		v.Template = v.Creation.Template
	}

	return nil
}

type Memory struct {
	Dynamic []int64 `json:"dynamic,omitempty"`
	Static  []int64 `json:"static,omitempty"`
	Size    int64   `json:"size,omitempty"`
}

type CPUs struct {
	Number int `json:"number"`
	Max    int `json:"max,omitempty"`
}

type Boot struct {
	Firmware string `json:"firmware,omitempty"`
	Order    string `json:"order,omitempty"`
}

type VMFilter struct {
	PowerState string `json:"power_state,omitempty"`
	NameLabel  string `json:"name_label,omitempty"`
	PoolID     string `json:"$poolId,omitempty"`
	Tags       string `json:"tags,omitempty"`
}

// PowerState represents the power state of a VM or a VM template.
type PowerState string

const (
	PowerStateHalted    PowerState = "Halted"
	PowerStateRunning   PowerState = "Running"
	PowerStatePaused    PowerState = "Paused"
	PowerStateSuspended PowerState = "Suspended"
)

// DomainType represents the virtualization mode (domain type) of a VM or a
// VM template.
type DomainType string

const (
	DomainTypeHVM         DomainType = "hvm"
	DomainTypePV          DomainType = "pv"
	DomainTypePVH         DomainType = "pvh"
	DomainTypePVInPVH     DomainType = "pv_in_pvh"
	DomainTypeUnspecified DomainType = "unspecified"
)

type VMOperation string

const (
	VMOperationAssertOperationValid     VMOperation = "assert_operation_valid"
	VMOperationAwaitingMemoryLive       VMOperation = "awaiting_memory_live"
	VMOperationCallPlugin               VMOperation = "call_plugin"
	VMOperationChangingDynamicRange     VMOperation = "changing_dynamic_range"
	VMOperationChangingMemoryLimits     VMOperation = "changing_memory_limits"
	VMOperationChangingMemoryLive       VMOperation = "changing_memory_live"
	VMOperationChangingNVRAM            VMOperation = "changing_NVRAM"
	VMOperationChangingShadowMemory     VMOperation = "changing_shadow_memory"
	VMOperationChangingShadowMemoryLive VMOperation = "changing_shadow_memory_live"
	VMOperationChangingStaticRange      VMOperation = "changing_static_range"
	VMOperationChangingVCPUs            VMOperation = "changing_VCPUs"
	VMOperationChangingVCPUsLive        VMOperation = "changing_VCPUs_live"
	VMOperationCheckpoint               VMOperation = "checkpoint"
	VMOperationCleanReboot              VMOperation = "clean_reboot"
	VMOperationCleanShutdown            VMOperation = "clean_shutdown"
	VMOperationClone                    VMOperation = "clone"
	VMOperationCopy                     VMOperation = "copy"
	VMOperationCreateTemplate           VMOperation = "create_template"
	VMOperationCreateVTPM               VMOperation = "create_vtpm"
	VMOperationCSV                      VMOperation = "csvm"
	VMOperationDataSourceOp             VMOperation = "data_source_op"
	VMOperationDestroy                  VMOperation = "destroy"
	VMOperationExport                   VMOperation = "export"
	VMOperationGetBootRecord            VMOperation = "get_boot_record"
	VMOperationHardReboot               VMOperation = "hard_reboot"
	VMOperationHardShutdown             VMOperation = "hard_shutdown"
	VMOperationImport                   VMOperation = "import"
	VMOperationMakeIntoTemplate         VMOperation = "make_into_template"
	VMOperationMetadataExport           VMOperation = "metadata_export"
	VMOperationMigrateSend              VMOperation = "migrate_send"
	VMOperationPause                    VMOperation = "pause"
	VMOperationPoolMigrate              VMOperation = "pool_migrate"
	VMOperationPowerStateReset          VMOperation = "power_state_reset"
	VMOperationProvision                VMOperation = "provision"
	VMOperationQueryServices            VMOperation = "query_services"
	VMOperationResume                   VMOperation = "resume"
	VMOperationResumeOn                 VMOperation = "resume_on"
	VMOperationRevert                   VMOperation = "revert"
	VMOperationReverting                VMOperation = "reverting"
	VMOperationSendSysrq                VMOperation = "send_sysrq"
	VMOperationSendTrigger              VMOperation = "send_trigger"
	VMOperationShutdown                 VMOperation = "shutdown"
	VMOperationSnapshot                 VMOperation = "snapshot"
	VMOperationSnapshotWithQuiesce      VMOperation = "snapshot_with_quiesce"
	VMOperationStart                    VMOperation = "start"
	VMOperationStartOn                  VMOperation = "start_on"
	VMOperationSuspend                  VMOperation = "suspend"
	VMOperationUnpause                  VMOperation = "unpause"
	VMOperationUpdateAllowedOperations  VMOperation = "update_allowed_operations"
)

func (v *VM) hasOperation(ops ...VMOperation) bool {
	for _, op := range v.CurrentOperations {
		for _, want := range ops {
			if op == want {
				return true
			}
		}
	}
	return false
}

func (v *VM) IsShuttingDown() bool {
	return v.hasOperation(VMOperationSuspend, VMOperationCleanShutdown, VMOperationHardShutdown)
}

func (v *VM) IsStarting() bool {
	return v.hasOperation(VMOperationStart, VMOperationStartOn, VMOperationResume, VMOperationResumeOn, VMOperationUnpause)
}

func (v *VM) IsRebooting() bool {
	return v.hasOperation(VMOperationCleanReboot, VMOperationHardReboot)
}
