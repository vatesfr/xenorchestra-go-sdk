package library

import (
	"context"

	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

type HubRecipe interface {
	CreateK8sCluster(ctx context.Context, cluster *payloads.K8sClusterOptions) (string, error)
}
