package hub_recipe

import (
	"context"
	"encoding/json"

	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

type Service struct {
	client       *client.Client
	legacyClient *v1.Client
	jsonrpcSvc   library.JSONRPC
	log          *logger.Logger
}

func New(
	client *client.Client,
	legacyClient *v1.Client,
	jsonrpcSvc library.JSONRPC,
	log *logger.Logger,
) library.HubRecipe {
	return &Service{
		client:       client,
		legacyClient: legacyClient,
		jsonrpcSvc:   jsonrpcSvc,
		log:          log,
	}
}

func (s *Service) CreateK8sCluster(ctx context.Context, cluster *payloads.K8sClusterOptions) (string, error) {
	var tag string

	var params map[string]interface{}
	tmp, _ := json.Marshal(cluster)
	json.Unmarshal(tmp, &params)

	err := s.jsonrpcSvc.Call("xoa.recipe.createKubernetesCluster", params, &tag)

	if err != nil {
		return "", err
	}

	return tag, nil
}
