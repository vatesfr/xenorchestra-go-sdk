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
	// Part of the VM interface with their own interfaces
	snapshotService library.Snapshot

	client *client.Client
	log    *logger.Logger
}

func New(
	client *client.Client,
	snapshot library.Snapshot,
	log *logger.Logger,
) library.VM {
	return &Service{
		client:          client,
		snapshotService: snapshot,
		log:             log,
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

func (s *Service) List(ctx context.Context, options map[string]any) ([]*payloads.VM, error) {
	path := core.NewPathBuilder().Resource("vms").Build()

	// The API returns VM URL paths, not VM objects directly
	var vmPaths []string
	err := client.TypedGet(ctx, s.client, path, options, &vmPaths)
	if err != nil {
		s.log.Error("failed to list VM paths", zap.Error(err))
		return nil, err
	}

	s.log.Debug("Retrieved VM paths", zap.Int("count", len(vmPaths)))

	// Fetch each VM by ID
	var vms []*payloads.VM
	for _, vmPath := range vmPaths {
		// Extract the VM ID from the path (/rest/v0/vms/uuid)
		idStr := strings.TrimPrefix(vmPath, "/rest/v0/vms/")
		vmID, err := uuid.FromString(idStr)
		if err != nil {
			s.log.Warn("Invalid VM path format, skipping",
				zap.String("vmPath", vmPath),
				zap.Error(err))
			continue
		}

		vm, err := s.GetByID(ctx, vmID)
		if err != nil {
			s.log.Warn("Failed to get VM details, skipping",
				zap.String("vmPath", vmPath),
				zap.String("vmID", vmID.String()),
				zap.Error(err))
			continue
		}

		vms = append(vms, vm)
	}

	s.log.Debug("Retrieved full VM objects", zap.Int("count", len(vms)))
	return vms, nil
}

func (s *Service) Create(ctx context.Context, vm *payloads.VM) (payloads.TaskID, error) {
	if vm.PoolID == uuid.Nil {
		return "", fmt.Errorf("pool ID is required for VM creation")
	}

	createParams := map[string]any{
		"template":         vm.Template.String(),
		"name_label":       vm.NameLabel,
		"name_description": vm.NameDescription,
		"boot":             false,
	}

	if len(vm.VIFs) > 0 {
		vifs := make([]map[string]any, 0, len(vm.VIFs))
		for _, vifRef := range vm.VIFs {
			vifs = append(vifs, map[string]any{
				"network": vifRef,
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

	var response string
	err := client.TypedPost(ctx, s.client, path, createParams, &response)
	if err != nil {
		s.log.Error("failed to initiate VM creation", zap.Error(err), zap.Any("params", createParams))
		return "", fmt.Errorf("failed to create VM: %w", err)
	}

	return core.ExtractTaskID(response), nil
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
	path := core.NewPathBuilder().Resource("vms").ID(id).Build()

	// Try getting the result as a string first to handle "OK" responses
	// This is for debugging purposes
	var stringResult string
	err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &stringResult)

	if err == nil {
		// Check for plain text "OK" response
		if strings.TrimSpace(stringResult) == "OK" {
			s.log.Debug("Successfully deleted VM with string response",
				zap.String("vmID", id.String()),
				zap.String("response", stringResult))
			return nil
		}

		s.log.Debug("VM delete response",
			zap.String("vmID", id.String()),
			zap.String("response", stringResult))
	} else if strings.Contains(err.Error(), "invalid character 'O' looking for beginning of value") {
		// This error happens when the API returns "OK" but TypedDelete tries to parse it as JSON
		s.log.Debug("Received 'OK' response for VM deletion", zap.String("vmID", id.String()))
		return nil
	}

	// If the string approach didn't work, try with a structured response
	if err != nil {
		var result struct {
			Success bool `json:"success"`
		}
		err = client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result)
		if err != nil {
			s.log.Error("failed to delete VM", zap.String("vmID", id.String()), zap.Error(err))
			return err
		}

		if !result.Success {
			s.log.Warn("VM delete operation reported non-success", zap.String("vmID", id.String()))
			return fmt.Errorf("vm delete operation returned success=false")
		}
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
		ID(id).
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
		ID(id).
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
		ID(id).
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
		ID(id).
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
		ID(id).
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

func (s *Service) Snapshot() library.Snapshot {
	return s.snapshotService
}
