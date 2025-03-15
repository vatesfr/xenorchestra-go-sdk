package payloads

import (
	"time"

	"github.com/gofrs/uuid"
)

type Snapshot struct {
	ID              uuid.UUID `json:"id"`
	NameLabel       string    `json:"name_label"`
	NameDescription string    `json:"name_description,omitempty"`
	VmID            uuid.UUID `json:"vmid"`
	CreationDate    time.Time `json:"created"`
	IsAutomatic     bool      `json:"is_automatic"`
	Size            int64     `json:"size,omitempty"`
	Tags            []string  `json:"tags,omitempty"`
}
