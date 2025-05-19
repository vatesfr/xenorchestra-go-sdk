package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -destination=mock/pool.go -package=mock_library Pool
type Pool interface {
	Get(ctx context.Context, id uuid.UUID) (*payloads.Pool, error)
	GetAll(ctx context.Context, limit int) ([]*payloads.Pool, error)

	PoolAction
}

type PoolAction interface {
	CreateVM(ctx context.Context, poolID string, params payloads.CreateVMParams) (string, error)
	EmergencyShutdown(ctx context.Context) (string, error)
	RollingReboot(ctx context.Context) (string, error)
	RollingUpdate(ctx context.Context) (string, error)
}
