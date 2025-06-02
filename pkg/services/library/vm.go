package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -destination=mock/vm.go -package=mock_library VM

type VM interface {
	GetByID(ctx context.Context, id uuid.UUID) (*payloads.VM, error)

	// List retrieves VMs with optional query parameters using VMQueryOptions for type safety
	// Examples:
	//   - List(ctx, nil) // All VMs
	//   - List(ctx, core.QueryRunningVMs()) // Running VMs with basic fields
	//   - List(ctx, core.QueryVMsByPool("pool-uuid")) // VMs in specific pool
	//   - List(ctx, core.WithLimit(core.NewVMQuery(), 10)) // First 10 VMs
	// You can also build custom queries:
	//   query := core.NewVMQuery()
	//   core.WithFields(query, payloads.VMFieldNameLabel, payloads.VMFieldPowerState)
	//   core.WithFilter(query, core.FilterByPowerState(payloads.PowerStateRunning))
	//   vms, err := vmService.List(ctx, query)
	List(ctx context.Context, query *payloads.VMQueryOptions) ([]*payloads.VM, error)

	Create(ctx context.Context, vm *payloads.VM) (payloads.TaskID, error)
	Update(ctx context.Context, vm *payloads.VM) (*payloads.VM, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// VMActions is a group of actions that can be performed on a VM.
	// I added this type to avoid having huge intefaces.
	VMActions

	// Same here, however this is also related to the VM.
	// I also want to keep the method chaining approach.
	// This is about VM but not about the actions.
	Snapshot() Snapshot
}

/*
TODO: Wait restart, suspend and resume actions to be added in the REST API.
They are available in the JSON-RPC API with client v1 so no need to add them here.
*/
type VMActions interface {
	Start(ctx context.Context, id uuid.UUID) error
	CleanShutdown(ctx context.Context, id uuid.UUID) error
	HardShutdown(ctx context.Context, id uuid.UUID) error
	CleanReboot(ctx context.Context, id uuid.UUID) error
	HardReboot(ctx context.Context, id uuid.UUID) error
}
