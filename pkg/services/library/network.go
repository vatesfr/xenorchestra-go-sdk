package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate mockgen --build_flags=--mod=mod --destination mock/network.go . Network
type Network interface {
	Get(ctx context.Context, id uuid.UUID) (*payloads.Network, error)
	GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Network, error)
	Delete(ctx context.Context, id uuid.UUID) error
	AddTag(ctx context.Context, id uuid.UUID, tag string) error
	RemoveTag(ctx context.Context, id uuid.UUID, tag string) error
}
