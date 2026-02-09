package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate mockgen --build_flags=--mod=mod --destination mock/vdi.go . VDI
type VDI interface {
	// Get retrieves a VDI by its ID.
	// Parameters:
	//   - id: ID of the VDI to retrieve
	// Returns the VDI details or an error if the operation fails.
	Get(ctx context.Context, id uuid.UUID) (*payloads.VDI, error)
	// GetAll retrieves VDIs with configurable limit and filtering.
	// Parameters:
	//   - limit: maximum number of VDIs to return (0 for no limit)
	//   - filter: filter string for VDI selection (empty for no filter)
	// Returns all matching VDIs or an error if the operation fails.
	GetAll(ctx context.Context, limit int, filter string) ([]*payloads.VDI, error)
	AddTag(ctx context.Context, id uuid.UUID, tag string) error
	RemoveTag(ctx context.Context, id uuid.UUID, tag string) error
	// Delete removes a VDI by its ID.
	// Parameters:
	//   - id: ID of the VDI to delete
	// Returns an error if the operation fails.
	Delete(ctx context.Context, id uuid.UUID) error
}
