package vdi

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
	client *client.Client
	log    *logger.Logger
}

func New(client *client.Client, log *logger.Logger) library.VDI {
	return &Service{
		client: client,
		log:    log,
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
