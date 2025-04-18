package storage_repository

import (
	"context"
	"fmt"
	"strings"

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

func New(client *client.Client, log *logger.Logger) library.StorageRepository {
	return &Service{
		client: client,
		log:    log,
	}
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*payloads.StorageRepository, error) {
	var result payloads.StorageRepository
	path := core.NewPathBuilder().
		Resource("srs").
		ID(id).
		Build()

	err := client.TypedGet(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to get storage repository",
			zap.String("id", id.String()),
			zap.Error(err))
		return nil, err
	}

	return &result, nil
}

func (s *Service) List(ctx context.Context, filter *payloads.StorageRepositoryFilter, limit int) ([]*payloads.StorageRepository, error) {
	var urlPaths []string
	path := core.NewPathBuilder().
		Resource("srs").
		Build()

	var params map[string]any
	if filter != nil {
		params = make(map[string]any)
		if filter.NameLabel != "" {
			params["name_label"] = filter.NameLabel
		}
		if filter.PoolID != uuid.Nil {
			params["$poolId"] = filter.PoolID.String()
		}
		if filter.SRType != "" {
			params["SR_type"] = filter.SRType
		}
	} else {
		params = make(map[string]any)
	}

	if limit > 0 {
		params["limit"] = limit
	}

	err := client.TypedGet(ctx, s.client, path, params, &urlPaths)
	if err != nil {
		s.log.Error("Failed to get storage repository URLs", zap.Error(err))
		return nil, err
	}

	var result []*payloads.StorageRepository
	for _, urlPath := range urlPaths {
		parts := strings.Split(urlPath, "/")
		idStr := parts[len(parts)-1]

		id, err := uuid.FromString(idStr)
		if err != nil {
			s.log.Warn("Failed to parse UUID from URL path",
				zap.String("path", urlPath),
				zap.String("id", idStr),
				zap.Error(err))
			continue
		}

		sr, err := s.GetByID(ctx, id)
		if err != nil {
			s.log.Warn("Failed to get storage repository details",
				zap.String("id", id.String()),
				zap.Error(err))
			continue
		}

		result = append(result, sr)
	}

	if filter != nil && len(filter.Tags) > 0 {
		var filtered []*payloads.StorageRepository
		for _, sr := range result {
			if containsAllTags(sr.Tags, filter.Tags) {
				filtered = append(filtered, sr)
			}
		}
		result = filtered
	}

	return result, nil
}

func (s *Service) ListByPool(ctx context.Context, poolID uuid.UUID, limit int) ([]*payloads.StorageRepository, error) {
	filter := &payloads.StorageRepositoryFilter{
		PoolID: poolID,
	}
	return s.List(ctx, filter, limit)
}

func (s *Service) AddTag(ctx context.Context, id uuid.UUID, tag string) error {
	path := core.NewPathBuilder().
		Resource("srs").
		ID(id).
		Resource("tags").
		Build()

	payload := map[string]string{
		"tag": tag,
	}

	var result struct {
		Success bool `json:"success"`
	}

	err := client.TypedPost(ctx, s.client, path, payload, &result)
	if err != nil {
		s.log.Error("Failed to add tag to storage repository",
			zap.String("id", id.String()),
			zap.String("tag", tag),
			zap.Error(err))
		return err
	}

	if !result.Success {
		return fmt.Errorf("failed to add tag %s to storage repository %s", tag, id.String())
	}

	return nil
}

func (s *Service) RemoveTag(ctx context.Context, id uuid.UUID, tag string) error {
	path := core.NewPathBuilder().
		Resource("srs").
		ID(id).
		Resource("tags").
		IDString(tag).
		Build()

	var result struct {
		Success bool `json:"success"`
	}

	err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to remove tag from storage repository",
			zap.String("id", id.String()),
			zap.String("tag", tag),
			zap.Error(err))
		return err
	}

	if !result.Success {
		return fmt.Errorf("failed to remove tag %s from storage repository %s", tag, id.String())
	}

	return nil
}

func containsAllTags(haystack, needles []string) bool {
	if len(needles) == 0 {
		return true
	}

	if len(haystack) == 0 {
		return false
	}

	for _, needle := range needles {
		found := false
		for _, hay := range haystack {
			if hay == needle {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
