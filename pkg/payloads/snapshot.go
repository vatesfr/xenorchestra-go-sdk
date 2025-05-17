package payloads

import (
	"github.com/gofrs/uuid"
)

// NOTE: We could share the same struct for both snapshots and VMs.
// Like VM in Snaphot with his specific fields and avoid repetition.
type Snapshot struct {
	ID                 uuid.UUID         `json:"id"`
	UUID               string            `json:"uuid"`
	Type               string            `json:"type,omitempty"`
	NameLabel          string            `json:"name_label"`
	NameDescription    string            `json:"name_description,omitempty"`
	PowerState         string            `json:"power_state,omitempty"`
	Memory             *Memory           `json:"memory,omitempty"`
	CPUs               *CPUs             `json:"CPUs,omitempty"`
	CoresPerSocket     int               `json:"coresPerSocket,omitempty"`
	VIFs               []string          `json:"VIFs,omitempty"`
	VBDs               []string          `json:"$VBDs,omitempty"`
	VGPUs              []string          `json:"VGPUs,omitempty"`
	VTPMs              []string          `json:"VTPMs,omitempty"`
	Tags               []string          `json:"tags,omitempty"`
	AutoPoweron        bool              `json:"auto_poweron"`
	HA                 string            `json:"high_availability,omitempty"`
	VirtualizationMode string            `json:"virtualizationMode,omitempty"`
	StartDelay         int               `json:"startDelay,omitempty"`
	ExpNestedHvm       bool              `json:"expNestedHvm,omitempty"`
	Boot               *Boot             `json:"boot,omitempty"`
	SecureBoot         bool              `json:"secureBoot,omitempty"`
	Videoram           Videoram          `json:"videoram,omitempty"`
	Vga                string            `json:"vga,omitempty"`
	XenstoreData       map[string]string `json:"xenStoreData,omitempty"`
	BlockedOperations  map[string]string `json:"blockedOperations,omitempty"`
	Addresses          map[string]string `json:"addresses,omitempty"`
	BiosStrings        map[string]string `json:"bios_strings,omitempty"`
	OsVersion          *OsVersion        `json:"os_version,omitempty"`
	InstallTime        int64             `json:"installTime,omitempty"`
	StartTime          *int64            `json:"startTime,omitempty"`
	CurrentOperations  map[string]string `json:"current_operations,omitempty"`
	OtherConfig        map[string]string `json:"other,omitempty"`
	PoolID             uuid.UUID         `json:"$poolId,omitempty"`
	Container          string            `json:"$container,omitempty"`
	XapiRef            string            `json:"_xapiRef,omitempty"`

	// Snapshot specific fields
	SnapshotTime int64     `json:"snapshot_time,omitempty"`
	SnapshotOf   uuid.UUID `json:"$snapshot_of,omitempty"`
}
