package host

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/core"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/zap"
)

type HostService struct {
	client *client.Client
	log    *logger.Logger
}

func New(client *client.Client, log *logger.Logger) library.Host {
	return &HostService{
		client: client,
		log:    log,
	}
}

func (s *HostService) Get(ctx context.Context, id uuid.UUID) (*payloads.Host, error) {
	path := core.NewPathBuilder().Resource("hosts").ID(id).Build()
	var result payloads.Host
	if err := client.TypedGet(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to get host by ID", zap.String("hostID", id.String()), zap.Error(err))
		return nil, err
	}
	return &result, nil
}

func (s *HostService) GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Host, error) {
	path := core.NewPathBuilder().Resource("hosts").Build()
	params := make(map[string]any)
	if limit > 0 {
		params["limit"] = limit
	}
	// Get all fields to retrieve complete host objects
	params["fields"] = "*"

	if filter != "" {
		params["filter"] = filter
	}

	// Make the request
	var result []*payloads.Host
	if err := client.TypedGet(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to get all hosts", zap.Error(err))
		return nil, err
	}
	return result, nil
}
