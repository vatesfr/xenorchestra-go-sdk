package payloads

import (
	"github.com/gofrs/uuid"
)

type StorageRepository struct {
	ID              uuid.UUID `json:"id"`
	UUID            string    `json:"uuid"`
	NameLabel       string    `json:"name_label"`
	NameDescription string    `json:"name_description,omitempty"`
	PoolID          uuid.UUID `json:"$poolId"`
	SRType          string    `json:"SR_type"`
	Container       string    `json:"$container,omitempty"`
	PhysicalUsage   int64     `json:"physical_usage"`
	Size            int64     `json:"size"`
	Usage           int64     `json:"usage"`
	Tags            []string  `json:"tags,omitempty"`
}

type StorageRepositoryFilter struct {
	NameLabel string    `json:"name_label,omitempty"`
	PoolID    uuid.UUID `json:"$poolId,omitempty"`
	SRType    string    `json:"SR_type,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
}
