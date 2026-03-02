package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate mockgen --build_flags=--mod=mod --destination mock/vbd.go . VBD
type VBD interface {
	// Get retrieves a VBD by its ID.
	// Parameters:
	//   - id: ID of the VBD to retrieve
	// Returns the VBD details or an error if the operation fails.
	Get(ctx context.Context, id uuid.UUID) (*payloads.VBD, error)

	// GetAll retrieves VBDs with configurable limit and filtering.
	// Parameters:
	//   - limit: maximum number of VBDs to return (0 for no limit)
	//   - filter: filter string for VBD selection (empty for no filter)
	// Returns all matching VBDs or an error if the operation fails.
	GetAll(ctx context.Context, limit int, filter string) ([]*payloads.VBD, error)

	// Create attaches a VDI to a VM by creating a new VBD.
	// Returns the ID of the newly created VBD or an error if the operation fails.
	Create(ctx context.Context, params *payloads.CreateVBDParams) (uuid.UUID, error)

	// Delete removes a VBD, detaching the VDI from the VM.
	// Note: the VDI itself is NOT deleted.
	// Parameters:
	//   - id: ID of the VBD to delete
	// Returns an error if the operation fails.
	Delete(ctx context.Context, id uuid.UUID) error

	// VBDActions is a group of actions that can be performed on a VBD.
	VBDActions
}

type VBDActions interface {
	// Connect hotplugs the VBD, dynamically attaching it to the running VM.
	// Parameters:
	//   - id: ID of the VBD to connect
	// Returns an optional task ID (empty string if synchronous) or an error.
	// TODO: This task is asynchronous but the API offers a way to mark it as synchronous.
	Connect(ctx context.Context, id uuid.UUID) (string, error)

	// Disconnect hot-unplugs the VBD, dynamically detaching it from the running VM.
	// Parameters:
	//   - id: ID of the VBD to disconnect
	// Returns an optional task ID (empty string if synchronous) or an error.
	// TODO: This task is asynchronous but the API offers a way to mark it as synchronous.
	Disconnect(ctx context.Context, id uuid.UUID) (string, error)
}
