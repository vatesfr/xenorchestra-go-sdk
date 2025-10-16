package vm

import (
	"context"
	"encoding/json"
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
	// Needed by VM for the task related but not part of the VM interface
	taskService library.Task

	client *client.Client
	log    *logger.Logger
}

func New(
	client *client.Client,
	task library.Task,
	log *logger.Logger,
) library.VM {
	return &Service{
		client:      client,
		taskService: task,
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

func (s *Service) List(ctx context.Context, limit int, filter string) ([]*payloads.VM, error) {
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

	path := core.NewPathBuilder().
		Resource("pools").
		ID(vm.PoolID).
		ActionsGroup().
		Action("create_vm").
		Build()

	s.log.Debug("Creating VM with params",
		zap.String("endpoint", path),
		zap.Any("params", createParams))

	var response string
	err := client.TypedPost(ctx, s.client, path, createParams, &response)
	if err != nil {
		s.log.Error("failed to create VM",
			zap.String("poolID", vm.PoolID.String()),
			zap.String("nameLabel", vm.NameLabel),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create VM: %w", err)
	}

	s.log.Debug("Received response from create_vm", zap.String("response", response))

	// Use the task service to handle the response
	taskResult, isTask, err := s.taskService.HandleTaskResponse(ctx, response, true)
	if err != nil {
		s.log.Error("Task handling failed", zap.Error(err))
		return nil, fmt.Errorf("VM creation task failed: %w", err)
	}

	if isTask {
		if taskResult.Status != payloads.Success {
			s.log.Error("Task failed",
				zap.String("status", string(taskResult.Status)),
				zap.String("message", taskResult.Result.Message))
			return nil, fmt.Errorf("VM creation failed: %s", taskResult.Result.Message)
		}

		// If task successful, get the created VM
		// Check both regular ID and StringID in the result
		// FIXME: StringID should not be used after API migration
		// TODO: Test the endpoint has been migrated to new API
		vmID := taskResult.Result.ID
		// if vmID == uuid.Nil && taskResult.Result.StringID != "" {
		// 	// Try to parse StringID as UUID
		// 	parsedID, err := uuid.FromString(taskResult.Result.StringID)
		// 	if err == nil {
		// 		vmID = parsedID
		// 	}
		// }

		if vmID != uuid.Nil {
			s.log.Debug("Task result has VM ID, fetching VM", zap.String("vmID", vmID.String()))
			return s.GetByID(ctx, vmID)
		}

		// If no valid VM ID in task result, try to find VM by name
		s.log.Debug("Task result does not have VM ID, searching by name", zap.String("nameLabel", vm.NameLabel))
	}

	// If we don't have a task URL or couldn't extract a VM ID from the task,
	// try to find VM by name
	vms, err := s.List(ctx, 0, fmt.Sprintf("name_label:%s", vm.NameLabel))
	if err != nil {
		s.log.Error("failed to list VMs", zap.Error(err))
		return nil, fmt.Errorf("could not determine created VM ID: %w", err)
	}

	for _, foundVM := range vms {
		if foundVM.NameLabel == vm.NameLabel {
			s.log.Debug("Found VM by name", zap.String("vmID", foundVM.ID.String()))
			return foundVM, nil
		}
	}

	// If we don't have a task URL, the response might be the VM directly
	var resultVM payloads.VM
	if err := json.Unmarshal([]byte(response), &resultVM); err == nil && resultVM.ID != uuid.Nil {
		s.log.Debug("Received VM directly in response", zap.String("vmID", resultVM.ID.String()))
		return &resultVM, nil
	}

	return nil, fmt.Errorf("VM creation task completed but VM not found")
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
	// TODO:FIXME: Update the method when delete endpoint is migrated to new REST API
	// PR: https://github.com/vatesfr/xen-orchestra/pull/8938is
	// var result struct {
	// 	Success bool `json:"success"`
	// }
	var result string
	path := core.NewPathBuilder().Resource("vms").ID(id).Build()
	err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil || result != "OK" {
		s.log.Error("failed to delete VM", zap.String("vmID", id.String()), zap.Error(err), zap.String("result", result))
		return err
	}
	return nil
}

func (s *Service) Start(ctx context.Context, id uuid.UUID) error {
	//TODO:FIXME: response is a task URL
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
