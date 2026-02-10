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
	// GetTasks retrieves tasks associated with a VDI, with optional limit and filtering.
	GetTasks(ctx context.Context, id uuid.UUID, limit int, filter string) ([]*payloads.Task, error)

	// VDIActions is a group of actions that can be performed on a VDI.
	VDIActions
}

type VDIActions interface {
	// Migrate a VDI to another SR
	// Note: After migration, the VDI will have a new ID.
	// 		 Clients should retrieve the new VDI details once the task is complete.
	// Parameters:
	//   - id: ID of the VDI to migrate
	//   - srId: ID of the target SR for migration
	// Returns a task ID or an error if the operation fails.
	// TODO: This task is asynchronous but the API offers a way to mark it as synchronous.
	Migrate(ctx context.Context, id uuid.UUID, srId uuid.UUID) (string, error)
}
