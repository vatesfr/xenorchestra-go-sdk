package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate mockgen --build_flags=--mod=mod --destination mock/storage_repository.go . StorageRepository
type StorageRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*payloads.StorageRepository, error)

	List(ctx context.Context, filter *payloads.StorageRepositoryFilter) ([]*payloads.StorageRepository, error)

	ListByPool(ctx context.Context, poolID uuid.UUID) ([]*payloads.StorageRepository, error)

	AddTag(ctx context.Context, id uuid.UUID, tag string) error

	RemoveTag(ctx context.Context, id uuid.UUID, tag string) error
}
