package payloads

import "github.com/gofrs/uuid"

type NetworkOperation string

const (
	NetworkOperationAttaching NetworkOperation = "attaching"
)

// Network represents a Xen Orchestra Network object.
type Network struct {
	Pool              uuid.UUID                   `json:"$pool"`
	XAPIRef           string                      `json:"_xapiRef"`
	UUID              string                      `json:"uuid"`
	MTU               uint                        `json:"MTU"`
	PIFs              []uuid.UUID                 `json:"PIFs,omitempty"`
	VIFs              []uuid.UUID                 `json:"VIFs,omitempty"`
	Automatic         bool                        `json:"automatic"`
	Bridge            string                      `json:"bridge"`
	CurrentOperations map[string]NetworkOperation `json:"current_operations,omitempty"`
	DefaultIsLocked   bool                        `json:"defaultIsLocked"`
	ID                uuid.UUID                   `json:"id"`
	InsecureNBD       *bool                       `json:"insecureNbd,omitempty"`
	IsBonded          bool                        `json:"isBonded"`
	NameDescription   string                      `json:"name_description"`
	NameLabel         string                      `json:"name_label"`
	NBD               *bool                       `json:"nbd,omitempty"`
	OtherConfig       map[string]string           `json:"other_config,omitempty"`
	Tags              []string                    `json:"tags,omitempty"`
	Type              ResourceType                `json:"type"`
}
