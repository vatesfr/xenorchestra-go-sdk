package vm

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
	// Needed by VM for the task related but not part of the VM interface
	taskService library.Task
	poolService library.Pool

	client *client.Client
	log    *logger.Logger
}

func New(
	client *client.Client,
	task library.Task,
	pool library.Pool,
	log *logger.Logger,
) library.VM {
	return &Service{
		client:      client,
		taskService: task,
		poolService: pool,
		log:         log,
	}
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*payloads.VM, error) {
	var result payloads.VM
	path := core.NewPathBuilder().Resource("vms").ID(id).Build()
	err := client.TypedGet(
		ctx,
		s.client,
		path,
		core.EmptyParams,
		&result,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Deprecated: Use GetAll instead
func (s *Service) List(ctx context.Context) ([]*payloads.VM, error) {
	var vmURLs []string
	err := client.TypedGet(ctx, s.client, "vms", core.EmptyParams, &vmURLs)
	if err != nil {
		return nil, err
	}

	if len(vmURLs) == 0 {
		return []*payloads.VM{}, nil
	}

	// TODO: add a config to set the max number of VMs to fetch
	// We can also use the fields name_label, name_description,
	// power_state to filter the VMs we want to fetch...
	maxVMs := 10
	if len(vmURLs) < maxVMs {
		maxVMs = len(vmURLs)
	}

	s.log.Info(fmt.Sprintf("Found %d VM references, fetching details for first %d\n", len(vmURLs), maxVMs))

	result := make([]*payloads.VM, 0, maxVMs)

	for i := 0; i < maxVMs; i++ {
		urlPath := vmURLs[i]

		parts := strings.Split(urlPath, "/")
		if len(parts) < 5 {
			continue
		}

		vmID := parts[len(parts)-1]

		id, err := uuid.FromString(vmID)
		if err != nil {
			s.log.Error("invalid UUID in VM URL", zap.String("vmID", vmID))
			continue
		}

		vm, err := s.GetByID(ctx, id)
		if err != nil {
			s.log.Error("failed to fetch VM", zap.String("vmID", id.String()), zap.Error(err))
			continue
		}

		result = append(result, vm)
	}

	if len(result) == 0 {
		s.log.Error("no valid VMs found")
		return nil, fmt.Errorf("no valid VMs found")
	}

	return result, nil
}

func (s *Service) GetAll(ctx context.Context, limit int, filter string) ([]*payloads.VM, error) {
	path := core.NewPathBuilder().Resource("vms").Build()
	params := make(map[string]any)
	if limit > 0 {
		params["limit"] = limit
	}
	// Get all fields to retrieve complete pool objects
	params["fields"] = "*"

	if filter != "" {
		params["filter"] = filter
	}

	result := make([]*payloads.VM, 0, limit)
	if err := client.TypedGet(ctx, s.client, path, params, &result); err != nil {
		s.log.Error("Failed to get all pools", zap.Error(err))
		return nil, err
	}
	return result, nil
}

// VM Creation should done from the Pool service, this method is provided for convenience
func (s *Service) Create(ctx context.Context, poolID uuid.UUID, vm *payloads.CreateVMParams) (*payloads.VM, error) {
	// Delegate to Pool service for VM creation (single source of truth)
	vmID, err := s.poolService.CreateVM(ctx, poolID, *vm)
	if err != nil {
		return nil, err
	}

	// Fetch and return the created VM
	return s.GetByID(ctx, vmID)
}

func (s *Service) Update(ctx context.Context, vm *payloads.VM) (*payloads.VM, error) {
	return nil, fmt.Errorf("not yet implemented")
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	var result struct {
		Success bool `json:"success"`
	}
	path := core.NewPathBuilder().Resource("vms").ID(id).Build()
	err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil || !result.Success {
		s.log.Error("failed to delete VM", zap.String("vmID", id.String()), zap.Error(err),
			zap.Bool("success", result.Success))
		return err
	}
	return nil
}

func (s *Service) Start(ctx context.Context, id uuid.UUID, hostID *uuid.UUID) (string, error) {
	payload := map[string]any{
		"id": id.String(),
	}
	if hostID != nil {
		payload["hostId"] = hostID.String()
	}
	return s.performAction(ctx, id, "start", payload)
}

func (s *Service) CleanShutdown(ctx context.Context, id uuid.UUID) (string, error) {
	return s.performAction(ctx, id, "clean_shutdown", map[string]any{"id": id.String()})
}

func (s *Service) HardShutdown(ctx context.Context, id uuid.UUID) (string, error) {
	return s.performAction(ctx, id, "hard_shutdown", map[string]any{"id": id.String()})
}

func (s *Service) CleanReboot(ctx context.Context, id uuid.UUID) (string, error) {
	return s.performAction(ctx, id, "clean_reboot", map[string]any{"id": id.String()})
}

func (s *Service) HardReboot(ctx context.Context, id uuid.UUID) (string, error) {
	return s.performAction(ctx, id, "hard_reboot", map[string]any{"id": id.String()})
}

func (s *Service) Snapshot(ctx context.Context, id uuid.UUID, name string) (string, error) {
	return s.performAction(ctx, id, "snapshot", map[string]any{
		"id":         id.String(),
		"name_label": name,
	})
}

func (s *Service) Restart(ctx context.Context, id uuid.UUID) (string, error) {
	return s.performAction(ctx, id, "restart", nil)
}

func (s *Service) Suspend(ctx context.Context, id uuid.UUID) (string, error) {
	return s.performAction(ctx, id, "suspend", nil)
}

func (s *Service) Resume(ctx context.Context, id uuid.UUID) (string, error) {
	return s.performAction(ctx, id, "resume", nil)
}

func (s *Service) Pause(ctx context.Context, id uuid.UUID) (string, error) {
	return s.performAction(ctx, id, "pause", nil)
}

func (s *Service) Unpause(ctx context.Context, id uuid.UUID) (string, error) {
	return s.performAction(ctx, id, "unpause", nil)
}

func (s *Service) performAction(ctx context.Context, id uuid.UUID, action string, payload any) (string, error) {
	var result payloads.TaskIDResponse

	pathBuilder := core.NewPathBuilder().
		Resource("vms").
		ID(id)

	// Some actions might be directly on the resource, others in actions group
	// Based on swagger.json:
	// start, clean_shutdown, hard_shutdown, clean_reboot, hard_reboot, snapshot, pause, suspend, resume, unpause
	// ALL of them are under /vms/{id}/actions/{action}
	pathBuilder.ActionsGroup()

	path := pathBuilder.Action(action).Build()

	if payload == nil {
		payload = core.EmptyParams
	}

	err := client.TypedPost(ctx, s.client, path, payload, &result)
	if err != nil {
		s.log.Error(fmt.Sprintf("failed to %s VM", action), zap.String("vmID", id.String()), zap.Error(err))
		return "", err
	}

	taskResult, err := s.taskService.HandleTaskResponse(ctx, result, false)
	if err != nil {
		s.log.Error("Task handling failed", zap.Error(err))
		return "", fmt.Errorf("VM %s failed: %w", action, err)
	}

	if taskResult != nil {
		return taskResult.ID, nil
	}

	return "", fmt.Errorf("unexpected response from API call: %v", result)
}
