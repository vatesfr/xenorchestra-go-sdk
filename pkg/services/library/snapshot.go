package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

// This interface will be embedded in the VM interface.
// It's related to the VM however it's a different concept.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -destination=mock/snapshot.go -package=mock_library Snapshot
type Snapshot interface {
	GetByID(ctx context.Context, id uuid.UUID) (*payloads.Snapshot, error)
	List(ctx context.Context, limit int) ([]*payloads.Snapshot, error)
	Create(ctx context.Context, vmID uuid.UUID, name string) (payloads.TaskID, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Revert(ctx context.Context, vmID uuid.UUID, snapshotID uuid.UUID) error
}
