/*
TODO: REMOVE THIS COMMENT. Since the v1 has a huge interface for the client,
I decided to separate all the different components into interface, then we
use a library to register all the components.

For example we can have a task service, a vm service, a pool service, etc.

We only have to create the interface contract, add the interface to library,
then the xo client that is acting as a registry will register the component
and we can use it as an abstraction layer. Can see the v2 provided example.

I also introduce a way to generate the mock for the interface, it will reduce
the headache of having to implement all the methods for the interface when an
interface needs another interface to be implemented.
*/
package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate mockgen --build_flags=--mod=mod --destination mock/vm.go . VM

type VM interface {
	GetByID(ctx context.Context, id uuid.UUID) (*payloads.VM, error)
	List(ctx context.Context) ([]*payloads.VM, error)
	Create(ctx context.Context, vm *payloads.VM) (*payloads.VM, error)
	Update(ctx context.Context, vm *payloads.VM) (*payloads.VM, error)
	Delete(ctx context.Context, id uuid.UUID) error

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
