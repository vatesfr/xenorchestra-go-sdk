package vbd

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

func New(client *client.Client, taskService library.Task, log *logger.Logger) library.VBD {
	return &Service{
		client:      client,
		log:         log,
		taskService: taskService,
	}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*payloads.VBD, error) {
	var result payloads.VBD
	path := core.NewPathBuilder().Resource("vbds").ID(id).Build()
	err := client.TypedGet(
		ctx,
		s.client,
		path,
		core.EmptyParams,
		&result,
	)
	if err != nil {
		s.log.Error("Failed to get VBD by ID", zap.String("vbdID", id.String()), zap.Error(err))
		return nil, err
	}
	return &result, nil
}

func (s *Service) GetAll(ctx context.Context, limit int, filter string) ([]*payloads.VBD, error) {
	path := core.NewPathBuilder().Resource("vbds").Build()
	params := make(map[string]any)
	if limit > 0 {
		params["limit"] = limit
	}
	// Get all fields to retrieve complete VBD objects
	params["fields"] = "*"

	if filter != "" {
		params["filter"] = filter
	}

	var result []*payloads.VBD
	if err := client.TypedGet(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to get all VBDs", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (s *Service) Create(ctx context.Context, params *payloads.CreateVBDParams) (uuid.UUID, error) {
	if params == nil {
		return uuid.Nil, fmt.Errorf("params cannot be nil")
	}
	if params.VM == uuid.Nil {
		return uuid.Nil, fmt.Errorf("VM ID cannot be empty")
	}
	if params.VDI == uuid.Nil {
		return uuid.Nil, fmt.Errorf("VDI ID cannot be empty")
	}

	path := core.NewPathBuilder().Resource("vbds").Build()

	var result struct {
		ID uuid.UUID `json:"id"`
	}

	if err := client.TypedPost(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to create VBD",
			zap.String("vmID", params.VM.String()),
			zap.String("vdiID", params.VDI.String()),
			zap.Error(err))
		return uuid.Nil, err
	}

	return result.ID, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	path := core.NewPathBuilder().Resource("vbds").ID(id).Build()

	var result struct{}

	if err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to delete VBD", zap.String("vbdID", id.String()), zap.Error(err))
		return err
	}

	return nil
}

func (s *Service) GetTasks(ctx context.Context, id uuid.UUID, limit int, filter string) ([]*payloads.Task, error) {
	path := core.NewPathBuilder().Resource("vbds").ID(id).Resource("tasks").Build()

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
		s.log.Error("Failed to get tasks for VBD", zap.String("vbdID", id.String()), zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *Service) Connect(ctx context.Context, id uuid.UUID) (string, error) {
	path := core.NewPathBuilder().Resource("vbds").ID(id).ActionsGroup().Action("connect").Build()

	var result payloads.TaskIDResponse

	err := client.TypedPost(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to connect VBD", zap.String("vbdID", id.String()), zap.Error(err))
		return "", err
	}

	taskResult, err := s.taskService.HandleTaskResponse(ctx, result, false)
	if err != nil {
		s.log.Error("Task handling failed for VBD connect", zap.String("vbdID", id.String()), zap.Error(err))
		return "", fmt.Errorf("VBD connect failed: %w", err)
	}

	return taskResult.ID, nil
}

func (s *Service) Disconnect(ctx context.Context, id uuid.UUID) (string, error) {
	path := core.NewPathBuilder().Resource("vbds").ID(id).ActionsGroup().Action("disconnect").Build()

	var result payloads.TaskIDResponse

	err := client.TypedPost(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to disconnect VBD", zap.String("vbdID", id.String()), zap.Error(err))
		return "", err
	}

	taskResult, err := s.taskService.HandleTaskResponse(ctx, result, false)
	if err != nil {
		s.log.Error("Task handling failed for VBD disconnect", zap.String("vbdID", id.String()), zap.Error(err))
		return "", fmt.Errorf("VBD disconnect failed: %w", err)
	}

	return taskResult.ID, nil
}
