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
