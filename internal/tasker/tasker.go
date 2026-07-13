package tasker

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/core"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/zap"
)

func GetTasks(
	ctx context.Context,
	c *client.Client,
	log *logger.Logger,
	resourceType payloads.ResourceType,
	id uuid.UUID,
	limit int,
	filter string,
) ([]*payloads.Task, error) {
	path := core.NewPathBuilder().Resource(resourceType.Path()).ID(id).Resource("tasks").Build()

	params := make(map[string]any)
	params["fields"] = "*"
	if limit > 0 {
		params["limit"] = limit
	}
	if filter != "" {
		params["filter"] = filter
	}

	var result []*payloads.Task

	if err := client.TypedGet(ctx, c, path, params, &result); err != nil {
		log.Error("Failed to get tasks for resource",
			zap.String("resourceType", string(resourceType)),
			zap.String("id", id.String()),
			zap.Error(err))
		return nil, err
	}

	return result, nil
}
