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
