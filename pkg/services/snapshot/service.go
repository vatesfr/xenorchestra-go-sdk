package snapshot

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/core"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/task"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/zap"
)

type Service struct {
	client *client.Client
	log    *logger.Logger
}

func New(client *client.Client, log *logger.Logger) library.Snapshot {
	return &Service{
		client: client,
		log:    log,
	}
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*payloads.Snapshot, error) {
	var result payloads.Snapshot
	path := core.NewPathBuilder().Resource("vm-snapshots").ID(id).Build()
	s.log.Debug("Getting VM snapshot by ID",
		zap.String("snapshotID", id.String()),
		zap.String("path", path))

	err := client.TypedGet(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("failed to get snapshot by id", zap.Error(err))
		return nil, err
	}
	return &result, nil
}

func (s *Service) ListByVM(ctx context.Context, vmID uuid.UUID) ([]*payloads.Snapshot, error) {
	var snapshotURLs []string
	path := core.NewPathBuilder().Resource("vm-snapshots").Build()

	s.log.Debug("Listing VM snapshots",
		zap.String("vmID", vmID.String()),
		zap.String("path", path))

	// Remove the vm filter parameter as it may not be supported
	// Instead, we'll fetch all snapshots and filter by VM ID
	err := client.TypedGet(ctx, s.client, path, core.EmptyParams, &snapshotURLs)
	if err != nil {
		s.log.Error("failed to list snapshots by vm", zap.Error(err))
		return nil, err
	}

	s.log.Debug("Retrieved snapshot URLs", zap.Int("count", len(snapshotURLs)))

	var snapshots []*payloads.Snapshot
	for _, urlPath := range snapshotURLs {
		parts := strings.Split(urlPath, "/")
		if len(parts) < 1 {
			s.log.Warn("Invalid snapshot URL format", zap.String("url", urlPath))
			continue
		}

		idStr := parts[len(parts)-1]
		id, err := uuid.FromString(idStr)
		if err != nil {
			s.log.Warn("Invalid snapshot ID format",
				zap.String("id", idStr),
				zap.Error(err))
			continue
		}

		snapshot, err := s.GetByID(ctx, id)
		if err != nil {
			s.log.Warn("Failed to get snapshot by ID",
				zap.String("id", id.String()),
				zap.Error(err))
			continue
		}

		if snapshot.VmID == vmID {
			snapshots = append(snapshots, snapshot)
		}
	}

	s.log.Debug("Filtered snapshots for VM",
		zap.String("vmID", vmID.String()),
		zap.Int("totalCount", len(snapshotURLs)),
		zap.Int("filteredCount", len(snapshots)))

	return snapshots, nil
}

func (s *Service) Create(ctx context.Context, vmID uuid.UUID, name string) (*payloads.Snapshot, error) {
	payload := map[string]any{
		"name_label": name,
	}

	path := core.NewPathBuilder().
		Resource("vms").
		ID(vmID).
		ActionsGroup().
		Action("snapshot").
		Build()

	s.log.Debug("Creating snapshot using REST API v0",
		zap.String("vmID", vmID.String()),
		zap.String("name", name),
		zap.String("path", path))

	var response string
	err := client.TypedPost(ctx, s.client, path, payload, &response)
	if err != nil {
		s.log.Error("failed to create snapshot", zap.Error(err))
		return nil, err
	}

	// TODO: remove noisy logs after development.
	s.log.Debug("Received response from snapshot creation", zap.String("response", response))

	taskService := task.New(s.client, s.log)

	taskResult, isTask, err := taskService.HandleTaskResponse(ctx, response, true)
	if err != nil {
		return nil, fmt.Errorf("snapshot creation task failed: %w", err)
	}

	if isTask {
		if taskResult.Status != payloads.Success {
			return nil, fmt.Errorf("snapshot creation failed: %s", taskResult.Message)
		}

		snapshotID := taskResult.Result.ID
		if snapshotID == uuid.Nil && taskResult.Result.StringID != "" {
			parsedID, err := uuid.FromString(taskResult.Result.StringID)
			if err == nil {
				snapshotID = parsedID
			}
		}

		if snapshotID != uuid.Nil {
			s.log.Debug("Task result has snapshot ID, fetching snapshot", zap.String("snapshotID", snapshotID.String()))
			return s.GetByID(ctx, snapshotID)
		}
	}

	s.log.Debug("Listing VM snapshots to find the created snapshot", zap.String("name", name))
	snapshots, err := s.ListByVM(ctx, vmID)
	if err != nil {
		s.log.Error("failed to list snapshots by vm", zap.Error(err))
		return nil, fmt.Errorf("could not determine created snapshot: %w", err)
	}

	// TODO: this is for development purpose and will be removed before the PR
	// in favour of something cleaner.
	var newest *payloads.Snapshot
	for _, snapshot := range snapshots {
		if snapshot.NameLabel == name && (newest == nil || snapshot.CreationDate.After(newest.CreationDate)) {
			newest = snapshot
		}
	}

	if newest == nil {
		s.log.Error("failed to find snapshot by name", zap.String("name", name))
		return nil, errors.New("snapshot creation completed but snapshot not found")
	}

	s.log.Debug("Found snapshot by name", zap.String("snapshotID", newest.ID.String()))
	return newest, nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	path := core.NewPathBuilder().Resource("vm-snapshots").ID(id).Build()

	s.log.Debug("Deleting VM snapshot",
		zap.String("snapshotID", id.String()),
		zap.String("path", path))

	// TODO: Modify to pass the payload instead of the method scope result.
	// This is for development purpose and will be removed before the PR.
	var result struct {
		Success bool `json:"success"`
	}
	err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("failed to delete snapshot", zap.Error(err))
		return err
	}

	if !result.Success {
		s.log.Error("failed to delete snapshot", zap.String("message", "failed to delete snapshot"))
		return errors.New("failed to delete snapshot")
	}

	return nil
}

func (s *Service) Revert(ctx context.Context, vmID uuid.UUID, snapshotID uuid.UUID) error {
	payload := map[string]any{
		"vm":       vmID.String(),
		"snapshot": snapshotID.String(),
	}

	path := core.NewPathBuilder().
		Resource("vm-snapshots").
		ActionsGroup().
		Action("revert").
		Build()

	s.log.Debug("Reverting VM to snapshot",
		zap.String("vmID", vmID.String()),
		zap.String("snapshotID", snapshotID.String()),
		zap.String("path", path))

	var result struct {
		Success bool `json:"success"`
	}
	err := client.TypedPost(ctx, s.client, path, payload, &result)
	if err != nil {
		s.log.Error("failed to revert snapshot", zap.Error(err))
		return err
	}

	if !result.Success {
		s.log.Error("failed to revert snapshot", zap.String("message", "failed to revert snapshot"))
		return errors.New("failed to revert snapshot")
	}

	return nil
}
