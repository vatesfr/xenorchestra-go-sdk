package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate mockgen --build_flags=--mod=mod --destination mock/host.go . Host
type Host interface {
	Get(ctx context.Context, id uuid.UUID) (*payloads.Host, error)
	GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Host, error)
	AddTag(ctx context.Context, id uuid.UUID, tag string) error
	RemoveTag(ctx context.Context, id uuid.UUID, tag string) error
}
