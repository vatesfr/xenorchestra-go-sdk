package snapshot

import (
	"context"
	"errors"
	"strings"

	"github.com/gofrs/uuid"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/core"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/zap"
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
) library.Snapshot {
	return &Service{
		client:       client,
		legacyClient: legacyClient,
		jsonrpcSvc:   jsonrpcSvc,
		log:          log,
	}
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*payloads.Snapshot, error) {
	var result payloads.Snapshot
	path := core.NewPathBuilder().Resource("vm-snapshots").ID(id).Build()

	err := client.TypedGet(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("failed to get snapshot by id", zap.Error(err))
		return nil, err
	}
	return &result, nil
}

func (s *Service) List(ctx context.Context, options map[string]any) ([]*payloads.Snapshot, error) {
	path := core.NewPathBuilder().Resource("vm-snapshots").Build()
	params := make(map[string]any)
	for k, v := range options {
		params[k] = v
	}
	if _, ok := options["limit"]; !ok {
		params["limit"] = core.DefaultTaskListLimit
	}

	var snapshotPaths []string
	err := client.TypedGet(ctx, s.client, path, params, &snapshotPaths)
	if err != nil {
		s.log.Error("Failed to list snapshot paths", zap.Error(err), zap.Any("options", options))
		return nil, err
	}
	s.log.Debug("Retrieved snapshot paths", zap.Int("count", len(snapshotPaths)))

	var snapshots []*payloads.Snapshot
	for _, snapshotPath := range snapshotPaths {
		idStr := strings.TrimPrefix(snapshotPath, "/rest/v0/vm-snapshots/")
		if idStr == snapshotPath {
			idStr = strings.TrimPrefix(snapshotPath, "/vm-snapshots/")
		}
		snapshotID, err := uuid.FromString(idStr)
		if err != nil {
			s.log.Warn("Invalid snapshot path format, skipping",
				zap.String("snapshotPath", snapshotPath),
				zap.Error(err))
			continue
		}

		snapshot, err := s.GetByID(ctx, snapshotID)
		if err != nil {
			s.log.Warn("Failed to get snapshot details, skipping",
				zap.String("snapshotPath", snapshotPath),
				zap.String("snapshotID", snapshotID.String()),
				zap.Error(err))
			continue
		}
		snapshots = append(snapshots, snapshot)
	}

	s.log.Debug("Retrieved full snapshot objects", zap.Int("count", len(snapshots)))
	return snapshots, nil
}

func (s *Service) Create(ctx context.Context, vmID uuid.UUID, name string) (payloads.TaskID, error) {
	payload := map[string]any{
		"name_label": name,
		"id":         vmID.String(),
	}

	path := core.NewPathBuilder().
		Resource("vms").
		ID(vmID).
		ActionsGroup().
		Action("snapshot").
		Build()

	var response string
	err := client.TypedPost(ctx, s.client, path, payload, &response)
	if err != nil {
		s.log.Error("failed to create snapshot", zap.Error(err))
		return "", err
	}

	return core.ExtractTaskID(response), nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	path := core.NewPathBuilder().Resource("vm-snapshots").ID(id).Build()

	s.log.Debug("Deleting VM snapshot",
		zap.String("snapshotID", id.String()),
		zap.String("path", path))

	var stringResult string
	err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &stringResult)

	if err == nil {
		if strings.TrimSpace(stringResult) == "OK" {
			s.log.Debug("Successfully deleted snapshot with string response",
				zap.String("snapshotID", id.String()),
				zap.String("response", stringResult))
			return nil
		}

		s.log.Debug("Received string response but not OK",
			zap.String("snapshotID", id.String()),
			zap.String("response", stringResult))
	} else if strings.Contains(err.Error(), "invalid character 'O' looking for beginning of value") {
		s.log.Debug("Received 'OK' response for deletion", zap.String("snapshotID", id.String()))
		return nil
	}

	if err != nil {
		var result struct {
			Success bool `json:"success"`
		}
		err = client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result)
		if err != nil {
			s.log.Error("failed to delete snapshot", zap.Error(err))
			return err
		}

		if !result.Success {
			s.log.Error("failed to delete snapshot", zap.String("message", "failed to delete snapshot"))
			return errors.New("failed to delete snapshot")
		}
	}

	return nil
}

func (s *Service) Revert(ctx context.Context, vmID uuid.UUID, snapshotID uuid.UUID) error {
	params := map[string]any{
		"id":       vmID.String(),
		"snapshot": snapshotID.String(),
	}

	logContext := []zap.Field{
		zap.String("vmID", vmID.String()),
		zap.String("snapshotID", snapshotID.String()),
	}

	var result bool
	if err := s.jsonrpcSvc.Call("vm.revert", params, &result, logContext...); err != nil {
		return err
	}

	return s.jsonrpcSvc.ValidateResult(result, "snapshot revert", logContext...)
}
