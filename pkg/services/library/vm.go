package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate mockgen --build_flags=--mod=mod --destination mock/vm.go . VM,VMActions

type VM interface {
	GetByID(ctx context.Context, id uuid.UUID) (*payloads.VM, error)
	// Deprecated: Use GetAll instead (List limits results to 10 VMs)
	List(ctx context.Context) ([]*payloads.VM, error)
	// GetAll retrieves VMs with configurable limit and filtering.
	// Parameters:
	//   - limit: maximum number of VMs to return (0 for no limit)
	//   - filter: filter string for VM selection (empty for no filter)
	// Returns all matching VMs or an error if the operation fails.
	GetAll(ctx context.Context, limit int, filter string) ([]*payloads.VM, error)
	// Create creates a new VM in the specified pool.
	// Note: VM creation is primarily handled by the Pool service; this method is provided for convenience.
	// Parameters:
	//   - poolID: ID of the pool where the VM will be created
	//   - vm: parameters for the VM to be created
	// Returns the created VM or an error if the operation fails.
	Create(ctx context.Context, poolID uuid.UUID, vm *payloads.CreateVMParams) (*payloads.VM, error)
	Update(ctx context.Context, vm *payloads.VM) (*payloads.VM, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// VMActions is a group of actions that can be performed on a VM.
	VMActions
}

type VMActions interface {
	Start(ctx context.Context, id uuid.UUID) error
	CleanShutdown(ctx context.Context, id uuid.UUID) error
	HardShutdown(ctx context.Context, id uuid.UUID) error
	CleanReboot(ctx context.Context, id uuid.UUID) error
	HardReboot(ctx context.Context, id uuid.UUID) error
	Snapshot(ctx context.Context, id uuid.UUID, name string) error
	Restart(ctx context.Context, id uuid.UUID) error
	Suspend(ctx context.Context, id uuid.UUID) error
	Resume(ctx context.Context, id uuid.UUID) error
}
