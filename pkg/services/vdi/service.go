package vdi

import (
	"context"
	"fmt"
	"io"

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

func New(client *client.Client, taskService library.Task, log *logger.Logger) library.VDI {
	return &Service{
		client:      client,
		log:         log,
		taskService: taskService,
	}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*payloads.VDI, error) {
	var result payloads.VDI
	path := core.NewPathBuilder().Resource("vdis").ID(id).Build()
	err := client.TypedGet(
		ctx,
		s.client,
		path,
		core.EmptyParams,
		&result,
	)
	if err != nil {
		s.log.Error("Failed to get VDI by ID", zap.String("vdiID", id.String()), zap.Error(err))
		return nil, err
	}
	return &result, nil
}

func (s *Service) GetAll(ctx context.Context, limit int, filter string) ([]*payloads.VDI, error) {
	path := core.NewPathBuilder().Resource("vdis").Build()
	params := make(map[string]any)
	if limit > 0 {
		params["limit"] = limit
	}
	// Get all fields to retrieve complete VDI objects
	params["fields"] = "*"

	if filter != "" {
		params["filter"] = filter
	}

	var result []*payloads.VDI
	if err := client.TypedGet(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to get all VDIs", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (s *Service) AddTag(ctx context.Context, id uuid.UUID, tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	path := core.NewPathBuilder().Resource("vdis").ID(id).Resource("tags").IDString(tag).Build()

	var result struct{}

	if err := client.TypedPut(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to add tag to VDI", zap.String("vdiID", id.String()), zap.String("tag", tag), zap.Error(err))
		return err
	}

	return nil
}

func (s *Service) RemoveTag(ctx context.Context, id uuid.UUID, tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	path := core.NewPathBuilder().Resource("vdis").ID(id).Resource("tags").IDString(tag).Build()

	var result struct{}

	if err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to remove tag from VDI", zap.String("vdiID", id.String()),
			zap.String("tag", tag), zap.Error(err))
		return err
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	path := core.NewPathBuilder().Resource("vdis").ID(id).Build()

	var result struct{}

	if err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to delete VDI", zap.String("vdiID", id.String()), zap.Error(err))
		return err
	}

	return nil
}

func (s *Service) Migrate(ctx context.Context, id uuid.UUID, srId uuid.UUID) (string, error) {

	path := core.NewPathBuilder().Resource("vdis").ID(id).ActionsGroup().Action("migrate").Build()

	var result payloads.TaskIDResponse

	params := map[string]string{
		"srId": srId.String(),
	}

	err := client.TypedPost(ctx, s.client, path, params, &result)
	if err != nil {
		s.log.Error("failed to migrate VDI", zap.String("vdiID", id.String()), zap.Error(err))
		return "", err
	}

	taskResult, err := s.taskService.HandleTaskResponse(ctx, result, false)
	if err != nil {
		s.log.Error("Task handling failed", zap.Error(err))
		return "", fmt.Errorf("VDI migration failed: %w", err)
	}

	if taskResult != nil {
		return taskResult.ID, nil
	}

	return "", fmt.Errorf("unexpected response from API call: %v", result)
}

func (s *Service) GetTasks(ctx context.Context, id uuid.UUID, limit int, filter string) ([]*payloads.Task, error) {
	path := core.NewPathBuilder().Resource("vdis").ID(id).Resource("tasks").Build()

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
		s.log.Error("Failed to get tasks for VDI", zap.String("vdiID", id.String()), zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *Service) Export(ctx context.Context, id uuid.UUID, format payloads.VDIFormat) (io.ReadCloser, error) {
	if format == "" {
		return nil, fmt.Errorf("format cannot be empty")
	}

	path := core.NewPathBuilder().Resource("vdis").ID(id).Build()
	endpoint := fmt.Sprintf("%s.%s", path, format)

	resp, err := client.RawGet(ctx, s.client, endpoint)
	if err != nil {
		s.log.Error("Failed to export VDI content", zap.String("vdiID", id.String()),
			zap.String("format", string(format)), zap.Error(err))
		return nil, err
	}

	return resp.Body, nil
}

func (s *Service) Import(
	ctx context.Context, id uuid.UUID, format payloads.VDIFormat, content io.Reader, size int64) error {
	if format == "" {
		return fmt.Errorf("format cannot be empty")
	}
	if content == nil {
		return fmt.Errorf("content cannot be nil")
	}
	if size <= 0 {
		return fmt.Errorf("size must be greater than 0")
	}

	path := core.NewPathBuilder().Resource("vdis").ID(id).Build()
	endpoint := fmt.Sprintf("%s.%s", path, format)

	resp, err := client.RawPut(ctx, s.client, endpoint, content, "application/octet-stream", size)
	if err != nil {
		s.log.Error("Failed to import VDI content", zap.String("vdiID", id.String()),
			zap.String("format", string(format)), zap.Error(err))
		return err
	}
	_ = resp.Body.Close()

	return nil
}
