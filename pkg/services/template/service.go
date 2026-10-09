// Package template implements the VM template service (REST resource
// "vm-templates"). Templates are first-class objects in Xen Orchestra: unlike
// VMs they are not part of the "vms" collection and are addressed by a string
// id: for default templates it is the composite "<poolUuid>-<templateUuid>"
// value, for non-default templates it is the bare template UUID.
package template

import (
	"context"

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
	client     *client.Client
	log        *logger.Logger
	tagService *tagger.Tagger
}

// New returns a library.Template implementation backed by the REST API.
func New(client *client.Client, log *logger.Logger) library.Template {
	return &Service{
		client:     client,
		log:        log,
		tagService: tagger.New(client, log, payloads.ResourceTypeVMTemplate),
	}
}

// Get retrieves a VM template by its id (a string: the composite
// "<poolUuid>-<templateUuid>" value for default templates, the bare template
// UUID for non-default templates).
func (s *Service) Get(ctx context.Context, id string) (*payloads.Template, error) {
	var result payloads.Template
	path := core.NewPathBuilder().Resource(payloads.ResourceTypeVMTemplate.Path()).IDString(id).Build()
	err := client.TypedGet(
		ctx,
		s.client,
		path,
		core.EmptyParams,
		&result,
	)
	if err != nil {
		s.log.Error("Failed to get template by ID", zap.String("templateID", id), zap.Error(err))
		return nil, err
	}
	return &result, nil
}

// GetAll retrieves VM templates with configurable limit and filtering.
func (s *Service) GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Template, error) {
	path := core.NewPathBuilder().Resource(payloads.ResourceTypeVMTemplate.Path()).Build()
	params := make(map[string]any)
	if limit > 0 {
		params["limit"] = limit
	}
	// Get all fields to retrieve complete template objects
	params["fields"] = "*"

	if filter != "" {
		params["filter"] = filter
	}

	var result []*payloads.Template
	if err := client.TypedGet(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to get all templates", zap.Error(err))
		return nil, err
	}
	return result, nil
}

// AddTag adds a tag to a template.
func (s *Service) AddTag(ctx context.Context, id string, tag string) error {
	return s.tagService.AddS(ctx, id, tag)
}

// RemoveTag removes a tag from a template.
func (s *Service) RemoveTag(ctx context.Context, id string, tag string) error {
	return s.tagService.RemoveS(ctx, id, tag)
}

// GetTasks retrieves the tasks associated with a template, with optional
// limit and filtering.
func (s *Service) GetTasks(ctx context.Context, id string, limit int, filter string) ([]*payloads.Task, error) {
	return tasker.GetTasksS(ctx, s.client, s.log, payloads.ResourceTypeVMTemplate, id, limit, filter)
}
