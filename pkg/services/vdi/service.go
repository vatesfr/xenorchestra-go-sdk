package vdi

import (
	"context"
	"fmt"
	"io"

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

const (
	vdiResourcePath = "vdis"
)

type Service struct {
	client      *client.Client
	log         *logger.Logger
	taskService library.Task
	tagService  *tagger.Tagger
}

func New(client *client.Client, taskService library.Task, log *logger.Logger) library.VDI {
	return &Service{
		client:      client,
		log:         log,
		taskService: taskService,
		tagService:  tagger.New(client, log, payloads.ResourceTypeVDI),
	}
}

func (s *Service) AddTag(ctx context.Context, id uuid.UUID, tag string) error {
	return s.tagService.Add(ctx, id, tag)
}

func (s *Service) RemoveTag(ctx context.Context, id uuid.UUID, tag string) error {
	return s.tagService.Remove(ctx, id, tag)
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*payloads.VDI, error) {
	var result payloads.VDI
	path := core.NewPathBuilder().Resource(vdiResourcePath).ID(id).Build()
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
	path := core.NewPathBuilder().Resource(vdiResourcePath).Build()
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

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	path := core.NewPathBuilder().Resource(vdiResourcePath).ID(id).Build()

	var result struct{}

	if err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to delete VDI", zap.String("vdiID", id.String()), zap.Error(err))
		return err
	}

	return nil
}

func (s *Service) Migrate(ctx context.Context, id uuid.UUID, srId uuid.UUID) (string, error) {
	path := core.NewPathBuilder().Resource(vdiResourcePath).ID(id).ActionsGroup().Action("migrate").Build()

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

	return taskResult.ID, nil
}

func (s *Service) GetTasks(ctx context.Context, id uuid.UUID, limit int, filter string) ([]*payloads.Task, error) {
	return tasker.GetTasks(ctx, s.client, s.log, payloads.ResourceTypeVDI, id, limit, filter)
}

func (s *Service) Export(ctx context.Context, id uuid.UUID, format payloads.VDIFormat, fn func(io.Reader) error) error {
	if format == "" {
		return fmt.Errorf("format cannot be empty")
	}
	if fn == nil {
		return fmt.Errorf("callback function cannot be nil")
	}

	path := core.NewPathBuilder().Resource(vdiResourcePath).ID(id).Build()
	endpoint := fmt.Sprintf("%s.%s", path, format)

	resp, err := client.RawGet(ctx, s.client, endpoint)
	if err != nil {
		s.log.Error("Failed to export VDI content", zap.String("vdiID", id.String()),
			zap.String("format", string(format)), zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	return fn(resp.Body)
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

	path := core.NewPathBuilder().Resource(vdiResourcePath).ID(id).Build()
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

func (s *Service) Create(ctx context.Context, params payloads.VDICreateParams) (uuid.UUID, error) {
	path := core.NewPathBuilder().Resource(vdiResourcePath).Build()

	var result payloads.CreateResponse

	err := client.TypedPost(ctx, s.client, path, params, &result)
	if err != nil {
		s.log.Error("Failed to create VDI", zap.Any("params", params), zap.Error(err))
		return uuid.Nil, err
	}

	return result.ID, nil
}
