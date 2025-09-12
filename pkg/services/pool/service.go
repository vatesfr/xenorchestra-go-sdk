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

func (s *Service) GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Pool, error) {
	path := core.NewPathBuilder().Resource("pools").Build()
	params := make(map[string]any)
	if limit > 0 {
		params["limit"] = limit
	}
	// Get all fields to retrieve complete pool objects
	params["fields"] = "*"

	if filter != "" {
		params["filter"] = filter
	}

	// Make the request
	var result []*payloads.Pool
	if err := client.TypedGet(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to get all pools", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (s *Service) CreateVM(ctx context.Context, poolID uuid.UUID, params payloads.CreateVMParams) (uuid.UUID, error) {
	return s.createResource(ctx, poolID, "vm", params)
}

func (s *Service) createResource(
	ctx context.Context, poolID uuid.UUID, resourceType string, params any) (uuid.UUID, error) {
	// Build the path
	action := fmt.Sprintf("create_%s", resourceType)
	path := core.NewPathBuilder().Resource("pools").IDString(poolID.String()).ActionsGroup().Action(action).Build()

	var response string
	err := client.TypedPost(ctx, s.client, path, params, &response)
	if err != nil {
		s.log.Error(fmt.Sprintf("failed to create %s on pool", resourceType),
			zap.String("poolID", poolID.String()),
			zap.Any("params", params),
			zap.Error(err))
		return uuid.Nil, fmt.Errorf("failed to create %s on pool %s: %w", resourceType, poolID, err)
	}

	s.log.Debug(fmt.Sprintf("Received response from %s", action), zap.String("response", response))

	// Use the task service to handle the response
	taskResult, isTask, err := s.taskService.HandleTaskResponse(ctx, response, true)
	if err != nil {
		s.log.Error("Task handling failed", zap.Error(err))
		return uuid.Nil, fmt.Errorf("%s creation task failed: %w", resourceType, err)
	}
	if isTask {
		if taskResult.Status != payloads.Success {
			s.log.Error("Task failed",
				zap.String("status", string(taskResult.Status)),
				zap.String("message", taskResult.Result.Message),
				zap.String("stack", taskResult.Result.Stack))
			return uuid.Nil, fmt.Errorf("%s creation failed: %s", resourceType, taskResult.Result.Message)
		}

		// If task successful
		resourceID := taskResult.Result.ID
		if resourceID == uuid.Nil {
			s.log.Debug(fmt.Sprintf("Task result has no %s ID", resourceType), zap.Any("taskResult.Result", taskResult.Result))
			return uuid.Nil, fmt.Errorf("failed to retrieve %s ID from task result: %s", resourceType, taskResult.Result.Message)
		}

		return resourceID, nil
	}
	return uuid.Nil, fmt.Errorf("unexpected response from API call: %s", response)
}

func (s *Service) performPoolAction(ctx context.Context, poolID uuid.UUID, action string) error {
	path := core.NewPathBuilder().Resource("pools").IDString(poolID.String()).ActionsGroup().Action(action).Build()

	params := map[string]any{}
	var response string

	err := client.TypedPost(ctx, s.client, path, params, &response)
	if err != nil {
		s.log.Error("failed to "+action+" the pool",
			zap.String("poolID", poolID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to %s the pool %s: %w", action, poolID, err)
	}

	// Use the task service to handle the response
	taskResult, isTask, err := s.taskService.HandleTaskResponse(ctx, response, true)
	if err != nil {
		s.log.Error("Task handling failed", zap.Error(err))
		return fmt.Errorf("pool %s failed: %w", action, err)
	}

	if isTask && taskResult.Status == payloads.Success {
		return nil
	} else {
		s.log.Error("Task failed",
			zap.String("status", string(taskResult.Status)),
			zap.String("message", taskResult.Result.Message),
			zap.String("stack", taskResult.Result.Stack))
		return fmt.Errorf("pool %s failed: %s", action, taskResult.Result.Message)
	}
}

// EmergencyShutdown performs an emergency shutdown on the specified pool.
// The call is synchronous: it will wait for the task to be completed.
func (s *Service) EmergencyShutdown(ctx context.Context, poolID uuid.UUID) error {
	return s.performPoolAction(ctx, poolID, "emergency_shutdown")
}

// RollingReboot triggers a rolling reboot on the specified pool.
// The call is synchronous: it will wait for the task to be completed.
func (s *Service) RollingReboot(ctx context.Context, poolID uuid.UUID) error {
	return s.performPoolAction(ctx, poolID, "rolling_reboot")
}

// RollingUpdate triggers a rolling update on the specified pool.
// The call is synchronous: it will wait for the task to be completed.
func (s *Service) RollingUpdate(ctx context.Context, poolID uuid.UUID) error {
	return s.performPoolAction(ctx, poolID, "rolling_update")
}

// CreateNetwork
func (s *Service) CreateNetwork(
	ctx context.Context, poolID uuid.UUID, params payloads.CreateNetworkParams) (uuid.UUID, error) {

	// Check parameters
	if params.Name == "" {
		s.log.Error("CreateNetwork failed: name cannot be empty",
			zap.String("Name", params.Name))
		return uuid.Nil, fmt.Errorf("network name cannot be empty")
	}
	if params.Vlan > 4094 {
		s.log.Error("CreateNetwork failed: vlan must be between 0 and 4094",
			zap.Uint("vlan", params.Vlan))
		return uuid.Nil, fmt.Errorf("vlan must be between 0 and 4094")
	}
	if params.Pif == uuid.Nil {
		s.log.Error("CreateNetwork failed: pifID must be set",
			zap.String("pifID", params.Pif.String()))
		return uuid.Nil, fmt.Errorf("pifID must be set")
	}

	return s.createResource(ctx, poolID, "network", params)
}
