package network

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/core"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/tagger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/zap"
)

type NetworkService struct {
	client      *client.Client
	log         *logger.Logger
	taskService library.Task
	tagService  *tagger.Tagger
	poolService library.Pool
}

func New(
	client *client.Client,
	taskService library.Task,
	poolService library.Pool,
	log *logger.Logger,
) library.Network {
	return &NetworkService{
		client:      client,
		log:         log,
		taskService: taskService,
		tagService:  tagger.New(client, log, payloads.ResourceTypeNetwork),
		poolService: poolService,
	}
}

func (s *NetworkService) Get(ctx context.Context, id uuid.UUID) (*payloads.Network, error) {
	path := core.NewPathBuilder().Resource(payloads.ResourceTypeNetwork.Path()).ID(id).Build()
	var result payloads.Network
	if err := client.TypedGet(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to get network by ID", zap.String("networkID", id.String()), zap.Error(err))
		return nil, err
	}
	return &result, nil
}

func (s *NetworkService) GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Network, error) {
	path := core.NewPathBuilder().Resource(payloads.ResourceTypeNetwork.Path()).Build()
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
	path := core.NewPathBuilder().Resource(payloads.ResourceTypeNetwork.Path()).ID(id).Build()

	var result struct{}

	if err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to delete network", zap.String("networkID", id.String()), zap.Error(err))
		return err
	}
	return nil
}

func (s *NetworkService) AddTag(ctx context.Context, id uuid.UUID, tag string) error {
	return s.tagService.Add(ctx, id, tag)
}

func (s *NetworkService) RemoveTag(ctx context.Context, id uuid.UUID, tag string) error {
	return s.tagService.Remove(ctx, id, tag)
}

func (s *NetworkService) GetTasks(
	ctx context.Context, id uuid.UUID, limit int, filter string) ([]*payloads.Task, error) {
	path := core.NewPathBuilder().Resource(payloads.ResourceTypeNetwork.Path()).ID(id).Resource("tasks").Build()

	params := make(map[string]any)
	params["fields"] = "*"
	if limit > 0 {
		params["limit"] = limit
	}
	if filter != "" {
		params["filter"] = filter
	}

	var result []*payloads.Task

	err := client.TypedGet(ctx, s.client, path, params, &result)
	if err != nil {
		s.log.Error("Failed to get tasks for Network", zap.String("networkID", id.String()), zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *NetworkService) Create(ctx context.Context, poolID uuid.UUID, params payloads.CreateNetworkParams) (uuid.UUID, error) {
	return s.poolService.CreateNetwork(ctx, poolID, params)
}

func (s *NetworkService) CreateInternal(ctx context.Context, poolID uuid.UUID, params payloads.CreateInternalNetworkParams) (uuid.UUID, error) {
	return s.poolService.CreateInternalNetwork(ctx, poolID, params)
}
