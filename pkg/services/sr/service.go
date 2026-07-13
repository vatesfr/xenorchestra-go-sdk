package sr

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/core"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/tagger"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/tasker"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/zap"
)

type Service struct {
	client      *client.Client
	log         *logger.Logger
	taskService library.Task
	tagService  *tagger.Tagger
}

func New(client *client.Client, taskService library.Task, log *logger.Logger) library.SR {
	return &Service{
		client:      client,
		log:         log,
		taskService: taskService,
		tagService:  tagger.New(client, log, payloads.ResourceTypeSR),
	}
}

func (s *Service) AddTag(ctx context.Context, id uuid.UUID, tag string) error {
	return s.tagService.Add(ctx, id, tag)
}

func (s *Service) RemoveTag(ctx context.Context, id uuid.UUID, tag string) error {
	return s.tagService.Remove(ctx, id, tag)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*payloads.StorageRepository, error) {
	var result payloads.StorageRepository
	path := core.NewPathBuilder().Resource(payloads.ResourceTypeSR.Path()).ID(id).Build()
	err := client.TypedGet(
		ctx,
		s.client,
		path,
		core.EmptyParams,
		&result,
	)
	if err != nil {
		s.log.Error("Failed to get SR by ID", zap.String("srID", id.String()), zap.Error(err))
		return nil, err
	}
	return &result, nil
}

func (s *Service) GetAll(ctx context.Context, limit int, filter string) ([]*payloads.StorageRepository, error) {
	path := core.NewPathBuilder().Resource(payloads.ResourceTypeSR.Path()).Build()
	params := make(map[string]any)
	if limit > 0 {
		params["limit"] = limit
	}
	// Get all fields to retrieve complete SR objects
	params["fields"] = "*"

	if filter != "" {
		params["filter"] = filter
	}

	var result []*payloads.StorageRepository
	if err := client.TypedGet(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to get all SRs", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (s *Service) GetTasks(ctx context.Context, id uuid.UUID, limit int, filter string) ([]*payloads.Task, error) {
	return tasker.GetTasks(ctx, s.client, s.log, payloads.ResourceTypeSR, id, limit, filter)
}

func (s *Service) ReclaimSpace(ctx context.Context, id uuid.UUID) (string, error) {
	path := core.NewPathBuilder().Resource(payloads.ResourceTypeSR.Path()).
		ID(id).ActionsGroup().Action("reclaim_space").Build()

	var result payloads.TaskIDResponse

	err := client.TypedPost(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to reclaim space on SR", zap.String("srID", id.String()), zap.Error(err))
		return "", err
	}

	taskResult, err := s.taskService.HandleTaskResponse(ctx, result, false)
	if err != nil {
		s.log.Error("Task handling failed for SR reclaim_space", zap.String("srID", id.String()), zap.Error(err))
		return "", fmt.Errorf("SR reclaim_space failed: %w", err)
	}

	return taskResult.ID, nil
}

func (s *Service) Scan(ctx context.Context, id uuid.UUID) (string, error) {
	path := core.NewPathBuilder().Resource(payloads.ResourceTypeSR.Path()).ID(id).ActionsGroup().Action("scan").Build()

	var result payloads.TaskIDResponse

	err := client.TypedPost(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to scan SR", zap.String("srID", id.String()), zap.Error(err))
		return "", err
	}

	taskResult, err := s.taskService.HandleTaskResponse(ctx, result, false)
	if err != nil {
		s.log.Error("Task handling failed for SR scan", zap.String("srID", id.String()), zap.Error(err))
		return "", fmt.Errorf("SR scan failed: %w", err)
	}

	return taskResult.ID, nil
}
