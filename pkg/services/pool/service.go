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
	// Needed by the actions
	taskService library.Task
}

func New(
	client *client.Client,
	task library.Task,
	log *logger.Logger,
) library.Pool {
	return &Service{
		client:      client,
		taskService: task,
		log:         log,
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
	// Get all fields to retrieve complete pool objects
	params["fields"] = "*"

	var result []*payloads.Pool
	if err := client.TypedGet(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to get all pools", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (s *Service) CreateVM(ctx context.Context, poolID uuid.UUID, params payloads.CreateVMParams) (uuid.UUID, error) {
	path := core.NewPathBuilder().Resource("pools").IDString(poolID.String()).ActionsGroup().Action("create_vm").Build()

	var response string
	err := client.TypedPost(ctx, s.client, path, params, &response)
	if err != nil {
		s.log.Error("failed to create VM on pool",
			zap.String("poolID", poolID.String()),
			zap.Any("params", params),
			zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to create VM on pool %s: %w", poolID, err)
	}

	s.log.Debug("Received response from create_vm", zap.String("response", response))

	// Use the task service to handle the response
	taskResult, isTask, err := s.taskService.HandleTaskResponse(ctx, response, true)
	if err != nil {
		s.log.Error("Task handling failed", zap.Error(err))
		return uuid.Nil, fmt.Errorf("VM creation task failed: %w", err)
	}
	if isTask {
		if taskResult.Status != payloads.Success {
			s.log.Error("Task failed",
				zap.String("status", string(taskResult.Status)),
				zap.String("message", taskResult.Result.Message),
				zap.String("stack", taskResult.Result.Stack))
			return uuid.Nil, fmt.Errorf("VM creation failed: %s", taskResult.Result.Message)
		}

		// If task successful
		vmID := taskResult.Result.ID
		if vmID == uuid.Nil {
			s.log.Debug("Task result has no VM ID", zap.Any("taskResult.Result", taskResult.Result))
			return uuid.Nil, fmt.Errorf("failed to retrieve VM ID from task result: %s", taskResult.Result.Message)
		}

		return vmID, nil
	}
	return uuid.Nil, fmt.Errorf("unexpected response from API call: %s", response)
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

func (s *Service) RollingReboot(ctx context.Context, poolID uuid.UUID) error {
	path := core.NewPathBuilder().Resource("pools").IDString(poolID.String()).ActionsGroup().Action("rolling_reboot").Build()

	params := map[string]any{}
	var response string

	err := client.TypedPost(ctx, s.client, path, params, &response)
	if err != nil {
		s.log.Error("failed to rolling reboot the pool",
			zap.String("poolID", poolID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to rolling reboot the pool %s: %w", poolID, err)
	}

	// Use the task service to handle the response
	taskResult, isTask, err := s.taskService.HandleTaskResponse(ctx, response, true)
	if err != nil {
		s.log.Error("Task handling failed", zap.Error(err))
		return fmt.Errorf("Pool rolling reboot failed: %w", err)
	}

	if isTask && taskResult.Status == payloads.Success {
		return nil
	} else {
		s.log.Error("Task failed",
			zap.String("status", string(taskResult.Status)),
			zap.String("message", taskResult.Result.Message),
			zap.String("stack", taskResult.Result.Stack))
		return fmt.Errorf("Pool rolling reboot failed: %s", taskResult.Result.Message)
	}
}

func (s *Service) RollingUpdate(ctx context.Context, poolID uuid.UUID) error {
	path := core.NewPathBuilder().Resource("pools").IDString(poolID.String()).ActionsGroup().Action("rolling_update").Build()

	params := map[string]any{}
	var response string

	err := client.TypedPost(ctx, s.client, path, params, &response)
	if err != nil {
		s.log.Error("failed to rolling update the pool",
			zap.String("poolID", poolID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to rolling update the pool %s: %w", poolID, err)
	}

	// Use the task service to handle the response
	taskResult, isTask, err := s.taskService.HandleTaskResponse(ctx, response, true)
	if err != nil {
		s.log.Error("Task handling failed", zap.Error(err))
		return fmt.Errorf("Pool rolling update failed: %w", err)
	}

	if isTask && taskResult.Status == payloads.Success {
		return nil
	} else {
		s.log.Error("Task failed",
			zap.String("status", string(taskResult.Status)),
			zap.String("message", taskResult.Result.Message),
			zap.String("stack", taskResult.Result.Stack))
		return fmt.Errorf("Pool rolling update failed: %s", taskResult.Result.Message)
	}
}
