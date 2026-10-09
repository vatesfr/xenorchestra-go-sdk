package tagger

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/core"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/zap"
)

type Tagger struct {
	client       *client.Client
	log          *logger.Logger
	resourceType payloads.ResourceType
}

func New(client *client.Client, log *logger.Logger, resourceType payloads.ResourceType) *Tagger {
	return &Tagger{
		client:       client,
		log:          log,
		resourceType: resourceType,
	}
}

func (t *Tagger) Add(ctx context.Context, id uuid.UUID, tag string) error {
	return t.AddS(ctx, id.String(), tag)
}

func (t *Tagger) AddS(ctx context.Context, id string, tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}
	if id == "" {
		return fmt.Errorf("resource id cannot be empty")
	}

	path := core.NewPathBuilder().Resource(t.resourceType.Path()).IDString(id).Resource("tags").IDString(tag).Build()

	var result struct{}

	if err := client.TypedPut(ctx, t.client, path, core.EmptyParams, &result); err != nil {
		t.log.Error(fmt.Sprintf("Failed to add tag to %s", t.resourceType),
			zap.String("id", id), zap.String("tag", tag), zap.Error(err))
		return err
	}

	return nil
}

func (t *Tagger) Remove(ctx context.Context, id uuid.UUID, tag string) error {
	return t.RemoveS(ctx, id.String(), tag)
}

func (t *Tagger) RemoveS(ctx context.Context, id string, tag string) error {
	if tag == "" {
		return fmt.Errorf("tag cannot be empty")
	}
	if id == "" {
		return fmt.Errorf("resource id cannot be empty")
	}

	path := core.NewPathBuilder().Resource(t.resourceType.Path()).IDString(id).Resource("tags").IDString(tag).Build()

	var result struct{}

	if err := client.TypedDelete(ctx, t.client, path, core.EmptyParams, &result); err != nil {
		t.log.Error(fmt.Sprintf("Failed to remove tag from %s", t.resourceType),
			zap.String("id", id), zap.String("tag", tag), zap.Error(err))
		return err
	}

	return nil
}
