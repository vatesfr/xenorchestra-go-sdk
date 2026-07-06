package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate go run go.uber.org/mock/mockgen  --build_flags=--mod=mod --destination mock/network.go . Network
type Network interface {
	// Get retrieves a Network by its ID.
	// Parameters:
	//   - id: ID of the Network to retrieve
	// Returns the Network details or an error if the operation fails.
	Get(ctx context.Context, id uuid.UUID) (*payloads.Network, error)

	// GetAll retrieves all Networks with optional filtering and pagination.
	// Parameters:
	//   - limit: Maximum number of Networks to retrieve
	//   - filter: Filter criteria for Networks
	// Returns a list of Networks or an error if the operation fails.
	GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Network, error)

	// GetTasks retrieves tasks associated with a Network, with optional limit and filtering.
	// Parameters:
	//   - id: ID of the Network whose tasks to retrieve
	//   - limit: maximum number of tasks to return (0 for no limit)
	//   - filter: filter string for task selection (empty for no filter)
	// Returns all matching tasks or an error if the operation fails.
	GetTasks(ctx context.Context, id uuid.UUID, limit int, filter string) ([]*payloads.Task, error)

	// Delete removes a Network by its ID.
	// Parameters:
	//   - id: ID of the Network to delete
	// Returns an error if the operation fails.
	Delete(ctx context.Context, id uuid.UUID) error

	NetworkActions

	Taggable
}

type NetworkActions interface {

	// Create creates a new Network within a specified Pool.
	// This is an alias of Pool.CreateNetwork, but it is provided here for convenience.
	// Parameters:
	//   - poolID: ID of the Pool where the Network will be created
	//   - params: parameters for creating the Network
	// Returns the ID of the newly created Network or an error if the operation fails.
	Create(ctx context.Context, poolID uuid.UUID, params payloads.CreateNetworkParams) (uuid.UUID, error)

	// CreateInternal creates a new internal Network within a specified Pool.
	// This is an alias of Pool.CreateInternalNetwork, but it is provided here for convenience.
	// Parameters:
	//   - poolID: ID of the Pool where the internal Network will be created
	//   - params: parameters for creating the internal Network
	// Returns the ID of the newly created internal Network or an error if the operation fails.
	CreateInternal(ctx context.Context, poolID uuid.UUID, params payloads.CreateInternalNetworkParams) (uuid.UUID, error)
}
