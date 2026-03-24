package payloads

import "github.com/gofrs/uuid"

// PBD represents a Physical Block Device in Xen Orchestra.
// A PBD is the connection between a host and a Storage Repository (SR).
type PBD struct {
	ID   uuid.UUID    `json:"id"`
	UUID uuid.UUID    `json:"uuid"`
	Type ResourceType `json:"type"`
	// Pool is the ID of the pool this PBD belongs to.
	Pool    uuid.UUID `json:"$pool"`
	XapiRef string    `json:"_xapiRef"`
	// Attached indicates whether this PBD is currently connected.
	Attached bool `json:"attached"`
	// Host is the ID of the host this PBD belongs to.
	Host uuid.UUID `json:"host"`
	// SR is the ID of the Storage Repository this PBD connects to.
	SR uuid.UUID `json:"SR"`
	// DeviceConfig holds the SR-type-specific configuration key-value pairs
	// (e.g. {"device": "/dev/sda"}, {"server": "nfs-host", "serverpath": "/export"}, etc.)
	DeviceConfig map[string]string `json:"device_config"`
	// OtherConfig holds additional configuration key-value pairs.
	OtherConfig map[string]string `json:"otherConfig"`
}
