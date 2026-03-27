package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate go run go.uber.org/mock/mockgen --build_flags=--mod=mod --destination mock/sr.go . SR
type SR interface {
	// Get retrieves a Storage Repository by its ID.
	// Parameters:
	//   - id: ID of the SR to retrieve
	// Returns the SR details or an error if the operation fails.
	Get(ctx context.Context, id uuid.UUID) (*payloads.SR, error)

	// GetAll retrieves Storage Repositories with configurable limit and filtering.
	// Parameters:
	//   - limit: maximum number of SRs to return (0 for no limit)
	//   - filter: filter string for SR selection (empty for no filter)
	// Returns all matching SRs or an error if the operation fails.
	GetAll(ctx context.Context, limit int, filter string) ([]*payloads.SR, error)

	// GetTasks retrieves tasks associated with an SR, with optional limit and filtering.
	// Parameters:
	//   - id: ID of the SR whose tasks to retrieve
	//   - limit: maximum number of tasks to return (0 for no limit)
	//   - filter: filter string for task selection (empty for no filter)
	// Returns all matching tasks or an error if the operation fails.
	GetTasks(ctx context.Context, id uuid.UUID, limit int, filter string) ([]*payloads.Task, error)

	// AddTag adds a tag to the SR.
	// Parameters:
	//   - id: ID of the SR
	//   - tag: tag string to add
	// Returns an error if the operation fails.
	AddTag(ctx context.Context, id uuid.UUID, tag string) error

	// RemoveTag removes a tag from the SR.
	// Parameters:
	//   - id: ID of the SR
	//   - tag: tag string to remove
	// Returns an error if the operation fails.
	RemoveTag(ctx context.Context, id uuid.UUID, tag string) error

	SRActions
}

type SRActions interface {

	// ReclaimSpace triggers the reclaim_space action on the SR.
	// This reclaims unused space on the storage repository.
	// Parameters:
	//   - id: ID of the SR on which to reclaim space
	// Returns the resulting task ID or an error if the operation fails.
	// TODO: This task is asynchronous but the API offers a way to make it synchronous.
	ReclaimSpace(ctx context.Context, id uuid.UUID) (string, error)

	// Scan triggers the scan action on the SR.
	// This rescans the storage repository to detect changes.
	// Parameters:
	//   - id: ID of the SR to scan
	// Returns the resulting task ID or an error if the operation fails.
	// TODO: This task is asynchronous but the API offers a way to make it synchronous.
	Scan(ctx context.Context, id uuid.UUID) (string, error)
}
