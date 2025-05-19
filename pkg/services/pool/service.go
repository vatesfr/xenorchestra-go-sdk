package pool

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

func New(
	client *client.Client,
	log *logger.Logger,
) library.Pool {
	return &Service{
		client: client,
		log:    log,
	}
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (*payloads.Pool, error) {
	path := core.NewPathBuilder().Resource("pools").ID(id).Build()
	var result payloads.Pool
	if err := client.TypedGet(ctx, s.client, path, core.EmptyParams, &result); err != nil {
		s.log.Error("Failed to get pool by ID", zap.String("poolID", id.String()), zap.Error(err))
		return nil, err
	}
	return &result, nil
}

func (s *Service) GetAll(ctx context.Context, limit int) ([]*payloads.Pool, error) {
	path := core.NewPathBuilder().Resource("pools").Build()
	params := make(map[string]any)
	if limit > 0 {
		params["limit"] = limit
	}

	var result []*payloads.Pool
	if err := client.TypedGet(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to get all pools", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (s *Service) CreateVM(ctx context.Context, poolID string, params payloads.CreateVMParams) (string, error) {
	path := core.NewPathBuilder().Resource("pools").IDString(poolID).Resource("vms").Build()

	var vmID string
	if err := client.TypedPost(ctx, s.client, path, params, &vmID); err != nil {
		s.log.Error("Failed to create VM on pool", zap.String("poolID", poolID), zap.Error(err), zap.Any("params", params))
		return "", fmt.Errorf("failed to create VM on pool %s: %w", poolID, err)
	}
	return vmID, nil
}

func (s *Service) performPoolAction(ctx context.Context, action string) (string, error) {
	path := core.NewPathBuilder().Resource("pools").Build()
	params := map[string]any{"action": action}
	var result string
	if err := client.TypedPost(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to perform pool action", zap.String("action", action), zap.Error(err))
		return "", fmt.Errorf("failed to perform pool action '%s': %w", action, err)
	}
	return result, nil
}

func (s *Service) EmergencyShutdown(ctx context.Context) (string, error) {
	return s.performPoolAction(ctx, "emergency_shutdown")
}

func (s *Service) RollingReboot(ctx context.Context) (string, error) {
	return s.performPoolAction(ctx, "rolling_reboot")
}

func (s *Service) RollingUpdate(ctx context.Context) (string, error) {
	return s.performPoolAction(ctx, "rolling_update")
}
