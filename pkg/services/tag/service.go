package tag

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
	client       *client.Client
	log          *logger.Logger
	resourceType payloads.ResourceType
}

func New(client *client.Client, log *logger.Logger, resourceType payloads.ResourceType) library.TagService {
	return &Service{
		client:       client,
		log:          log,
		resourceType: resourceType,
	}
}

func (s *Service) Add(ctx context.Context, id uuid.UUID, tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	path := core.NewPathBuilder().Resource(s.resourceType.Path()).ID(id).Resource("tags").IDString(tag).Build()

	var result struct{}

	if err := client.TypedPut(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error(fmt.Sprintf("Failed to add tag to %s", s.resourceType),
			zap.String("id", id.String()), zap.String("tag", tag), zap.Error(err))
		return err
	}

	return nil
}

func (s *Service) Remove(ctx context.Context, id uuid.UUID, tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}

	path := core.NewPathBuilder().Resource(s.resourceType.Path()).ID(id).Resource("tags").IDString(tag).Build()

	var result struct{}

	if err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error(fmt.Sprintf("Failed to remove tag from %s", s.resourceType),
			zap.String("id", id.String()), zap.String("tag", tag), zap.Error(err))
		return err
	}

	return nil
}
