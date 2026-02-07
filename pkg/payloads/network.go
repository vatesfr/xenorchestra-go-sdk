package payloads

import "github.com/gofrs/uuid"

// Network represents a Xen Orchestra Network object.
type Network struct {
	ID                uuid.UUID         `json:"id"`
	UUID              string            `json:"uuid"`
	Type              string            `json:"type"`
	NameLabel         string            `json:"name_label"`
	NameDescription   string            `json:"name_description"`
	Bridge            string            `json:"bridge"`
	MTU               int               `json:"MTU"`
	Automatic         bool              `json:"automatic"`
	DefaultIsLocked   bool              `json:"defaultIsLocked"`
	NBD               *bool             `json:"nbd,omitempty"`
	InsecureNBD       *bool             `json:"insecureNbd,omitempty"`
	CurrentOperations map[string]string `json:"current_operations,omitempty"`
	OtherConfig       map[string]string `json:"other_config,omitempty"`
	Tags              []string          `json:"tags,omitempty"`
	PIFs              []uuid.UUID       `json:"PIFs,omitempty"`
	VIFs              []uuid.UUID       `json:"VIFs,omitempty"`
	PoolRef           uuid.UUID         `json:"$pool"`
	PoolID            uuid.UUID         `json:"$poolId"`
	XAPIRef           string            `json:"_xapiRef"`
}
