package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate mockgen --build_flags=--mod=mod --destination mock/pool.go . Pool,PoolAction
type Pool interface {
	Get(ctx context.Context, id uuid.UUID) (*payloads.Pool, error)
	GetAll(ctx context.Context, limit int) ([]*payloads.Pool, error)

	PoolAction
}

type PoolAction interface {
	CreateVM(ctx context.Context, poolID uuid.UUID, params payloads.CreateVMParams) (uuid.UUID, error)
	CreateNetwork(ctx context.Context, poolID uuid.UUID, params payloads.CreateNetworkParams) (uuid.UUID, error)
	EmergencyShutdown(ctx context.Context, poolID uuid.UUID) error
	RollingReboot(ctx context.Context, poolID uuid.UUID) error
	RollingUpdate(ctx context.Context, poolID uuid.UUID) error
}
