package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

// This interface will be embedded in the VM interface.
// It's related to the VM however it's a different concept.
//
//go:generate mockgen --build_flags=--mod=mod --destination mock/snapshot.go . Snapshot
type Snapshot interface {
	GetByID(ctx context.Context, id uuid.UUID) (*payloads.Snapshot, error)
	ListByVM(ctx context.Context, vmID uuid.UUID, limit int) ([]*payloads.Snapshot, error)
	Create(ctx context.Context, vmID uuid.UUID, name string) (*payloads.Snapshot, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Revert(ctx context.Context, vmID uuid.UUID, snapshotID uuid.UUID) error
}
