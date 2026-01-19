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
	// Not implemented yet
	Update(ctx context.Context, vm *payloads.VM) (*payloads.VM, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// VMActions is a group of actions that can be performed on a VM.
	VMActions
}

type VMActions interface {
	// Start powers on the specified VM.
	// Parameters:
	//   - id: ID of the VM to start
	//   - hostID: optional ID of the host where the VM should be started (nil for automatic selection)
	// Returns the task ID associated with the start operation or an error if the operation fails.
	Start(ctx context.Context, id uuid.UUID, hostID *uuid.UUID) (string, error)
	// CleanShutdown gracefully shuts down the specified VM.
	// Parameters:
	//   - id: ID of the VM to shut down
	// Returns the task ID associated with the shutdown operation or an error if the operation fails.
	CleanShutdown(ctx context.Context, id uuid.UUID) (string, error)
	// HardShutdown forcefully powers off the specified VM.
	// Parameters:
	//   - id: ID of the VM to power off
	// Returns the task ID associated with the hard shutdown operation or an error if the operation fails.
	HardShutdown(ctx context.Context, id uuid.UUID) (string, error)
	// CleanReboot gracefully reboots the specified VM.
	// Parameters:
	//   - id: ID of the VM to reboot
	// Returns the task ID associated with the reboot operation or an error if the operation fails.
	CleanReboot(ctx context.Context, id uuid.UUID) (string, error)
	// HardReboot forcefully reboots the specified VM.
	// Parameters:
	//   - id: ID of the VM to reboot
	// Returns the task ID associated with the hard reboot operation or an error if the operation fails.
	HardReboot(ctx context.Context, id uuid.UUID) (string, error)
	// Snapshot creates a snapshot of the specified VM.
	// Parameters:
	//   - id: ID of the VM to snapshot
	//   - name: name of the snapshot
	// Returns the task ID associated with the snapshot operation or an error if the operation fails.
	Snapshot(ctx context.Context, id uuid.UUID, name string) (string, error)
	// Restart restarts the specified VM.
	// Parameters:
	//   - id: ID of the VM to restart
	// Returns the task ID associated with the restart operation or an error if the operation fails.
	Restart(ctx context.Context, id uuid.UUID) (string, error)
	// Suspend suspends the specified VM.
	// Parameters:
	//   - id: ID of the VM to suspend
	// Returns the task ID associated with the suspend operation or an error if the operation fails.
	Suspend(ctx context.Context, id uuid.UUID) (string, error)
	// Resume resumes the specified VM.
	// Parameters:
	//   - id: ID of the VM to resume
	// Returns the task ID associated with the resume operation or an error if the operation fails.
	Resume(ctx context.Context, id uuid.UUID) (string, error)
	// Pause pauses the specified VM.
	// Parameters:
	//   - id: ID of the VM to pause
	// Returns the task ID associated with the pause operation or an error if the operation fails.
	Pause(ctx context.Context, id uuid.UUID) (string, error)
	// Unpause unpauses the specified VM.
	// Parameters:
	//   - id: ID of the VM to unpause
	// Returns the task ID associated with the unpause operation or an error if the operation fails.
	Unpause(ctx context.Context, id uuid.UUID) (string, error)
}
