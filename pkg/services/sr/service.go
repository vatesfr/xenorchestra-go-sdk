package sr

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

type Service struct {
	client      *client.Client
	log         *logger.Logger
	taskService library.Task
}

func New(client *client.Client, taskService library.Task, log *logger.Logger) library.SR {
	return &Service{
		client:      client,
		log:         log,
		taskService: taskService,
	}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*payloads.SR, error) {
	var result payloads.SR
	path := core.NewPathBuilder().Resource("srs").ID(id).Build()
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

func (s *Service) GetAll(ctx context.Context, limit int, filter string) ([]*payloads.SR, error) {
	path := core.NewPathBuilder().Resource("srs").Build()
	params := make(map[string]any)
	if limit > 0 {
		params["limit"] = limit
	}
	// Get all fields to retrieve complete SR objects
	params["fields"] = "*"

	if filter != "" {
		params["filter"] = filter
	}

	var result []*payloads.SR
	if err := client.TypedGet(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to get all SRs", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (s *Service) GetTasks(ctx context.Context, id uuid.UUID, limit int, filter string) ([]*payloads.Task, error) {
	path := core.NewPathBuilder().Resource("srs").ID(id).Resource("tasks").Build()

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
		s.log.Error("Failed to get tasks for SR", zap.String("srID", id.String()), zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *Service) ReclaimSpace(ctx context.Context, id uuid.UUID) (string, error) {
	path := core.NewPathBuilder().Resource("srs").ID(id).ActionsGroup().Action("reclaim_space").Build()

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
	path := core.NewPathBuilder().Resource("srs").ID(id).ActionsGroup().Action("scan").Build()

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

func (s *Service) AddTag(ctx context.Context, id uuid.UUID, tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	path := core.NewPathBuilder().Resource("srs").ID(id).Resource("tags").IDString(tag).Build()

	var result struct{}

	if err := client.TypedPut(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to add tag to SR", zap.String("srID", id.String()), zap.String("tag", tag), zap.Error(err))
		return err
	}

	return nil
}

func (s *Service) RemoveTag(ctx context.Context, id uuid.UUID, tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	path := core.NewPathBuilder().Resource("srs").ID(id).Resource("tags").IDString(tag).Build()

	var result struct{}

	if err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to remove tag from SR", zap.String("srID", id.String()), zap.String("tag", tag), zap.Error(err))
		return err
	}

	return nil
}
