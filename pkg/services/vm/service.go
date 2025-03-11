package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

func New(client *client.Client, log *logger.Logger) library.VM {
	return &Service{
		client: client,
		log:    log,
	}
}

func (s *Service) formatPath(path string) string {
	return fmt.Sprintf("vms/%s", path)
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

	fmt.Printf("Found %d VM references, fetching details for first %d\n", len(vmURLs), maxVMs)

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

func (s *Service) Create(ctx context.Context, vm *payloads.VM) (*payloads.VM, error) {
	if vm.PoolID == uuid.Nil {
		return nil, fmt.Errorf("pool ID is required for VM creation")
	}

	createParams := map[string]any{
		"template":         vm.Template.String(),
		"name_label":       vm.NameLabel,
		"name_description": vm.NameDescription,
		"boot":             false,
	}

	if len(vm.VIFs) > 0 {
		vifs := make([]map[string]any, 0, len(vm.VIFs))
		for _, vifID := range vm.VIFs {
			vifs = append(vifs, map[string]any{
				"network": vifID,
			})
		}
		createParams["vifs"] = vifs
	}

	endpoint := fmt.Sprintf("pools/%s/actions/create_vm", vm.PoolID)

	s.log.Debug("Creating VM with params",
		zap.String("endpoint", endpoint),
		zap.Any("params", createParams))

	var rawResponse string
	err := client.TypedPost(ctx, s.client, endpoint, createParams, &rawResponse)

	if err != nil {
		if strings.Contains(err.Error(), "invalid character '/' looking for beginning of value") {
			errParts := strings.Split(err.Error(), "body: ")
			if len(errParts) > 1 {
				taskPath := strings.TrimSpace(errParts[len(errParts)-1])
				s.log.Info("VM creation started as async task", zap.String("taskPath", taskPath))

				return s.waitForVMCreationTask(ctx, taskPath)
			}
		}

		s.log.Error("failed to create VM",
			zap.String("poolID", vm.PoolID.String()),
			zap.String("nameLabel", vm.NameLabel),
			zap.Error(err))
		return nil, err
	}

	var result payloads.VM
	if err := json.Unmarshal([]byte(rawResponse), &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// NOTE: This method should be placed in the task service at least for the polling logic, vm is still
// part of it for the creation part, however we need to follow clean pattern for all services (SOC).
func (s *Service) waitForVMCreationTask(ctx context.Context, taskPath string) (*payloads.VM, error) {
	taskID := strings.TrimPrefix(taskPath, "/rest/v0/tasks/")

	s.log.Info("Waiting for VM creation task to complete", zap.String("taskID", taskID))

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.After(5 * time.Minute)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, fmt.Errorf("timed out waiting for VM creation task to complete")
		case <-ticker.C:
			var task struct {
				Status string          `json:"status"`
				Result json.RawMessage `json:"result"`
			}

			err := client.TypedGet(ctx, s.client, fmt.Sprintf("tasks/%s", taskID), core.EmptyParams, &task)
			if err != nil {
				s.log.Error("failed to get task status", zap.String("taskID", taskID), zap.Error(err))
				continue
			}

			s.log.Debug("Task status", zap.String("taskID", taskID), zap.String("status", task.Status))

			switch task.Status {
			case "success":
				var vmID string

				if err := json.Unmarshal(task.Result, &vmID); err == nil {
					s.log.Info("VM creation task completed successfully with string ID",
						zap.String("taskID", taskID),
						zap.String("vmID", vmID))

					uuid, err := uuid.FromString(vmID)
					if err != nil {
						return nil, fmt.Errorf("invalid VM ID returned by task: %s", vmID)
					}
					return s.GetByID(ctx, uuid)
				}

				var resultObj struct {
					ID string `json:"id"`
				}
				if err := json.Unmarshal(task.Result, &resultObj); err == nil && resultObj.ID != "" {
					s.log.Info("VM creation task completed successfully with object containing ID",
						zap.String("taskID", taskID),
						zap.String("vmID", resultObj.ID))

					uuid, err := uuid.FromString(resultObj.ID)
					if err != nil {
						return nil, fmt.Errorf("invalid VM ID returned by task: %s", resultObj.ID)
					}
					return s.GetByID(ctx, uuid)
				}

				s.log.Warn("Couldn't extract VM ID from task result",
					zap.String("taskID", taskID),
					zap.String("rawResult", string(task.Result)))

				time.Sleep(5 * time.Second)

				vms, err := s.List(ctx)
				if err != nil {
					return nil, fmt.Errorf("task completed but failed to list VMs: %w", err)
				}

				for _, v := range vms {
					if v.NameLabel == "AM-TEST-XO-SDK-TEST" {
						return v, nil
					}
				}

				return nil, fmt.Errorf("task completed successfully but could not determine created VM ID")

			case "failure":
				var errorObj map[string]any
				if err := json.Unmarshal(task.Result, &errorObj); err == nil {
					if msg, ok := errorObj["message"].(string); ok {
						return nil, fmt.Errorf("VM creation task failed: %s", msg)
					}
				}
				return nil, fmt.Errorf("VM creation task failed")

			default:
				continue
			}
		}
	}
}

func (s *Service) Update(ctx context.Context, vm *payloads.VM) (*payloads.VM, error) {
	var result payloads.VM
	path := core.NewPathBuilder().Resource("vms").ID(vm.ID).Build()
	err := client.TypedPost(ctx, s.client, path, vm, &result)
	if err != nil {
		s.log.Error("failed to update VM", zap.Error(err))
		return nil, err
	}
	return &result, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	var result struct {
		Success bool `json:"success"`
	}
	path := core.NewPathBuilder().Resource("vms").ID(id).Build()
	err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("failed to delete VM", zap.String("vmID", id.String()), zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) Start(ctx context.Context, id uuid.UUID) error {
	var result struct {
		Success bool `json:"success"`
	}

	payload := map[string]any{
		"id": id.String(),
	}

	path := core.NewPathBuilder().
		Resource("vms").
		Wildcard().
		ActionsGroup().
		Action("start").
		Build()

	err := client.TypedPost(ctx, s.client, path, payload, &result)
	if err != nil {
		s.log.Error("failed to start VM", zap.String("vmID", id.String()), zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) CleanShutdown(ctx context.Context, id uuid.UUID) error {
	var result struct {
		Success bool `json:"success"`
	}

	payload := map[string]any{
		"id": id.String(),
	}

	path := core.NewPathBuilder().
		Resource("vms").
		Wildcard().
		ActionsGroup().
		Action("clean_shutdown").
		Build()

	err := client.TypedPost(ctx, s.client, path, payload, &result)
	if err != nil {
		s.log.Error("failed to clean shutdown VM", zap.String("vmID", id.String()), zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) HardShutdown(ctx context.Context, id uuid.UUID) error {
	var result struct {
		Success bool `json:"success"`
	}

	payload := map[string]any{
		"id": id.String(),
	}

	path := core.NewPathBuilder().
		Resource("vms").
		Wildcard().
		ActionsGroup().
		Action("hard_shutdown").
		Build()

	err := client.TypedPost(ctx, s.client, path, payload, &result)
	if err != nil {
		s.log.Error("failed to hard shutdown VM", zap.String("vmID", id.String()), zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) CleanReboot(ctx context.Context, id uuid.UUID) error {
	var result struct {
		Success bool `json:"success"`
	}

	payload := map[string]any{
		"id": id.String(),
	}

	path := core.NewPathBuilder().
		Resource("vms").
		Wildcard().
		ActionsGroup().
		Action("clean_reboot").
		Build()

	err := client.TypedPost(ctx, s.client, path, payload, &result)
	if err != nil {
		s.log.Error("failed to clean reboot VM", zap.String("vmID", id.String()), zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) HardReboot(ctx context.Context, id uuid.UUID) error {
	var result struct {
		Success bool `json:"success"`
	}

	payload := map[string]any{
		"id": id.String(),
	}

	path := core.NewPathBuilder().
		Resource("vms").
		Wildcard().
		ActionsGroup().
		Action("hard_reboot").
		Build()

	err := client.TypedPost(ctx, s.client, path, payload, &result)
	if err != nil {
		s.log.Error("failed to hard reboot VM", zap.String("vmID", id.String()), zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) Snapshot(ctx context.Context, id uuid.UUID, name string) error {
	var result struct {
		Success bool `json:"success"`
	}

	payload := map[string]any{
		"id":   id.String(),
		"name": name,
	}

	path := core.NewPathBuilder().
		Resource("vms").
		Wildcard().
		ActionsGroup().
		Action("snapshot").
		Build()

	err := client.TypedPost(ctx, s.client, path, payload, &result)
	if err != nil {
		s.log.Error("failed to snapshot VM", zap.String("vmID", id.String()), zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) Restart(ctx context.Context, id uuid.UUID) error {
	var result struct {
		Success bool `json:"success"`
	}

	path := core.NewPathBuilder().
		Resource("vms").
		ID(id).
		Action("restart").
		Build()

	err := client.TypedPost(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("failed to restart VM", zap.String("vmID", id.String()), zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) Suspend(ctx context.Context, id uuid.UUID) error {
	var result struct {
		Success bool `json:"success"`
	}

	path := core.NewPathBuilder().
		Resource("vms").
		ID(id).
		Action("suspend").
		Build()

	err := client.TypedPost(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("failed to suspend VM", zap.String("vmID", id.String()), zap.Error(err))
		return err
	}
	return nil
}

func (s *Service) Resume(ctx context.Context, id uuid.UUID) error {
	var result struct {
		Success bool `json:"success"`
	}

	path := core.NewPathBuilder().
		Resource("vms").
		ID(id).
		Action("resume").
		Build()

	err := client.TypedPost(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("failed to resume VM", zap.String("vmID", id.String()), zap.Error(err))
		return err
	}
	return nil
}
