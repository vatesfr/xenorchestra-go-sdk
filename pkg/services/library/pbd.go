package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate go run go.uber.org/mock/mockgen --build_flags=--mod=mod --destination mock/pbd.go . PBD
type PBD interface {
	// Get retrieves a PBD by its ID.
	// Parameters:
	//   - id: ID of the PBD to retrieve
	// Returns the PBD details or an error if the operation fails.
	Get(ctx context.Context, id uuid.UUID) (*payloads.PBD, error)

	// GetAll retrieves PBDs with configurable limit and filtering.
	// Parameters:
	//   - limit: maximum number of PBDs to return (0 for no limit)
	//   - filter: filter string for PBD selection (empty for no filter)
	// Returns all matching PBDs or an error if the operation fails.
	GetAll(ctx context.Context, limit int, filter string) ([]*payloads.PBD, error)

	// PBDActions is a group of actions that can be performed on a PBD.
	PBDActions
}

type PBDActions interface {
	// Plug connects the PBD, attaching the SR to its host.
	// Parameters:
	//   - id: ID of the PBD to plug
	// Returns an optional task ID or an error.
	// TODO: This task is asynchronous but the API offers a way to make it synchronous.
	Plug(ctx context.Context, id uuid.UUID) (string, error)

	// Unplug disconnects the PBD, detaching the SR from its host.
	// Parameters:
	//   - id: ID of the PBD to unplug
	// Returns an optional task ID or an error.
	// TODO: This task is asynchronous but the API offers a way to make it synchronous.
	Unplug(ctx context.Context, id uuid.UUID) (string, error)
}
