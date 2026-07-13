package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

// Taskable is implemented by all resources that support task operations.
type Taskable interface {
	// GetTasks retrieves tasks associated with a resource, with optional limit and filtering.
	// Parameters:
	//   - id: ID of the resource whose tasks to retrieve
	//   - limit: maximum number of tasks to return (0 for no limit)
	//   - filter: filter string for task selection (empty for no filter)
	// Returns all matching tasks or an error if the operation fails.
	GetTasks(ctx context.Context, id uuid.UUID, limit int, filter string) ([]*payloads.Task, error)
}
