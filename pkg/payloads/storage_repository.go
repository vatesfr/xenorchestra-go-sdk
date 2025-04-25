package payloads

import (
	"github.com/gofrs/uuid"
)

type StorageRepository struct {
	ID              uuid.UUID         `json:"id"`
	UUID            string            `json:"uuid"`
	Type            string            `json:"type,omitempty"`
	NameLabel       string            `json:"name_label"`
	NameDescription string            `json:"name_description,omitempty"`
	PoolID          uuid.UUID         `json:"$poolId"`
	XapiRef         string            `json:"_xapiRef,omitempty"`
	SRType          string            `json:"SR_type"`
	Container       string            `json:"$container,omitempty"`
	ContentType     string            `json:"content_type,omitempty"`
	Shared          bool              `json:"shared,omitempty"`
	OtherConfig     map[string]string `json:"other_config,omitempty"`
	SmConfig        map[string]string `json:"sm_config,omitempty"`
	PBDs            []string          `json:"$PBDs,omitempty"`
	PhysicalUsage   int64             `json:"physical_usage"`
	Size            int64             `json:"size"`
	Usage           int64             `json:"usage"`
	Tags            []string          `json:"tags,omitempty"`
}

type StorageRepositoryFilter struct {
	NameLabel string    `json:"name_label,omitempty"`
	PoolID    uuid.UUID `json:"$poolId,omitempty"`
	SRType    string    `json:"SR_type,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
}
