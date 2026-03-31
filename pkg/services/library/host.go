package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate go run go.uber.org/mock/mockgen --build_flags=--mod=mod --destination mock/host.go . Host
type Host interface {
	Get(ctx context.Context, id uuid.UUID) (*payloads.Host, error)
	GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Host, error)

	Taggable
}
