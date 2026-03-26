package pbd

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

func New(client *client.Client, taskService library.Task, log *logger.Logger) library.PBD {
	return &Service{
		client:      client,
		log:         log,
		taskService: taskService,
	}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*payloads.PBD, error) {
	var result payloads.PBD
	path := core.NewPathBuilder().Resource("pbds").ID(id).Build()
	err := client.TypedGet(
		ctx,
		s.client,
		path,
		core.EmptyParams,
		&result,
	)
	if err != nil {
		s.log.Error("Failed to get PBD by ID", zap.String("pbdID", id.String()), zap.Error(err))
		return nil, err
	}
	return &result, nil
}

func (s *Service) GetAll(ctx context.Context, limit int, filter string) ([]*payloads.PBD, error) {
	path := core.NewPathBuilder().Resource("pbds").Build()
	params := make(map[string]any)
	if limit > 0 {
		params["limit"] = limit
	}
	// Get all fields to retrieve complete PBD objects
	params["fields"] = "*"

	if filter != "" {
		params["filter"] = filter
	}

	var result []*payloads.PBD
	if err := client.TypedGet(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to get all PBDs", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (s *Service) Plug(ctx context.Context, id uuid.UUID) (string, error) {
	path := core.NewPathBuilder().Resource("pbds").ID(id).ActionsGroup().Action("plug").Build()

	var result payloads.TaskIDResponse

	err := client.TypedPost(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to plug PBD", zap.String("pbdID", id.String()), zap.Error(err))
		return "", err
	}

	taskResult, err := s.taskService.HandleTaskResponse(ctx, result, false)
	if err != nil {
		s.log.Error("Task handling failed for PBD plug", zap.String("pbdID", id.String()), zap.Error(err))
		return "", fmt.Errorf("PBD plug failed: %w", err)
	}

	return taskResult.ID, nil
}

func (s *Service) Unplug(ctx context.Context, id uuid.UUID) (string, error) {
	path := core.NewPathBuilder().Resource("pbds").ID(id).ActionsGroup().Action("unplug").Build()

	var result payloads.TaskIDResponse

	err := client.TypedPost(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to unplug PBD", zap.String("pbdID", id.String()), zap.Error(err))
		return "", err
	}

	taskResult, err := s.taskService.HandleTaskResponse(ctx, result, false)
	if err != nil {
		s.log.Error("Task handling failed for PBD unplug", zap.String("pbdID", id.String()), zap.Error(err))
		return "", fmt.Errorf("PBD unplug failed: %w", err)
	}

	return taskResult.ID, nil
}
