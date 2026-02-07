package network

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/core"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/zap"
)

type NetworkService struct {
	client *client.Client
	log    *logger.Logger
}

func New(client *client.Client, log *logger.Logger) library.Network {
	return &NetworkService{
		client: client,
		log:    log,
	}
}

func (s *NetworkService) Get(ctx context.Context, id uuid.UUID) (*payloads.Network, error) {
	path := core.NewPathBuilder().Resource("networks").ID(id).Build()
	var result payloads.Network
	if err := client.TypedGet(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to get network by ID", zap.String("networkID", id.String()), zap.Error(err))
		return nil, err
	}
	return &result, nil
}

func (s *NetworkService) GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Network, error) {
	path := core.NewPathBuilder().Resource("networks").Build()
	params := make(map[string]any)
	if limit > 0 {
		params["limit"] = limit
	}
	params["fields"] = "*"

	if filter != "" {
		params["filter"] = filter
	}

	var result []*payloads.Network
	if err := client.TypedGet(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to get all networks", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (s *NetworkService) Delete(ctx context.Context, id uuid.UUID) error {
	path := core.NewPathBuilder().Resource("networks").ID(id).Build()

	var result struct{}

	if err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to delete network", zap.String("networkID", id.String()), zap.Error(err))
		return err
	}
	return nil
}

func (s *NetworkService) AddTag(ctx context.Context, id uuid.UUID, tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	path := core.NewPathBuilder().Resource("networks").ID(id).Resource("tags").IDString(tag).Build()

	var result struct{}

	if err := client.TypedPut(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to add tag to network", zap.String("networkID", id.String()), zap.String("tag", tag), zap.Error(err))
		return err
	}

	return nil
}

func (s *NetworkService) RemoveTag(ctx context.Context, id uuid.UUID, tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	path := core.NewPathBuilder().Resource("networks").ID(id).Resource("tags").IDString(tag).Build()

	var result struct{}

	if err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to remove tag from network", zap.String("networkID", id.String()),
			zap.String("tag", tag), zap.Error(err))
		return err
	}

	return nil
}
