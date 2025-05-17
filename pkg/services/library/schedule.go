package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -destination=mock/schedule.go -package=mock_library Schedule
type Schedule interface {
	Get(ctx context.Context, id uuid.UUID) (*payloads.Schedule, error)
	GetAll(ctx context.Context) ([]*payloads.Schedule, error)

	Create(ctx context.Context, schedule *payloads.Schedule) (*payloads.Schedule, error)
	Update(ctx context.Context, id uuid.UUID, schedule *payloads.Schedule) (*payloads.Schedule, error)
	Delete(ctx context.Context, id uuid.UUID) error
}
