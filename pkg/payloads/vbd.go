package payloads

import (
	"github.com/gofrs/uuid"
)

// VBD represents a Virtual Block Device in Xen Orchestra.
// A VBD is an attachment between a VDI (Virtual Disk Image) and a VM.
type VBD struct {
	ID   uuid.UUID    `json:"id,omitempty"`
	UUID uuid.UUID    `json:"uuid"`
	Type ResourceType `json:"type"`
	// Pool is the ID of the pool this VBD belongs to.
	Pool     uuid.UUID `json:"$pool"`
	XapiRef  string    `json:"_xapiRef"`
	Attached bool      `json:"attached"`
	Bootable bool      `json:"bootable"`
	// Device is the device name (e.g. "xvda"), null when not plugged in.
	Device    *string    `json:"device"`
	IsCDDrive bool       `json:"is_cd_drive"`
	Position  string     `json:"position"`
	ReadOnly  bool       `json:"read_only"`
	VDI       *uuid.UUID `json:"VDI,omitempty"`
	// VM is the ID of the VM this VBD is attached to.
	VM uuid.UUID `json:"VM"`
}

// VBDType represents the type of a VBD
type VBDType string

const (
	VBDTypeCD     VBDType = "CD"
	VBDTypeDisk   VBDType = "Disk"
	VBDTypeFloppy VBDType = "Floppy"
)

// VBDMode represents the access mode of a VBD
type VBDMode string

const (
	VBDModeRO VBDMode = "RO"
	VBDModeRW VBDMode = "RW"
)

// CreateVBDParams contains the parameters for creating a new VBD.
// It attaches a VDI (or VDI-snapshot) to a VM.
type CreateVBDParams struct {
	// VM is the ID of the VM to attach to (required)
	VM uuid.UUID `json:"VM"`
	// VDI is the ID of the VDI or VDI-snapshot to attach (required)
	VDI uuid.UUID `json:"VDI"`
	// Type is the device type (CD, Disk, Floppy)
	Type VBDType `json:"type,omitempty"`
	// Mode is the access mode (RO, RW)
	Mode VBDMode `json:"mode,omitempty"`
	// Bootable indicates whether this VBD is bootable
	Bootable bool `json:"bootable,omitempty"`
	// Userdevice is the position/slot for the device
	Userdevice string `json:"userdevice,omitempty"`
	// Unpluggable indicates whether the VBD can be hot-unplugged
	Unpluggable *bool `json:"unpluggable,omitempty"`
	// Empty indicates whether the CD drive is empty (only relevant for CD type)
	Empty *bool `json:"empty,omitempty"`
	// OtherConfig holds additional configuration key-value pairs
	OtherConfig map[string]string `json:"other_config,omitempty"`
	// QosAlgorithmType is the QoS algorithm type
	QosAlgorithmType string `json:"qos_algorithm_type,omitempty"`
	// QosAlgorithmParams holds QoS algorithm parameters
	QosAlgorithmParams map[string]string `json:"qos_algorithm_params,omitempty"`
}
