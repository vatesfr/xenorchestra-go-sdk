package restore

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/core"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/task"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/zap"
)

type Service struct {
	client       *client.Client
	legacyClient *v1.Client
	log          *logger.Logger
	taskService  library.Task
	jsonrpcSvc   library.JSONRPC
}

func New(
	client *client.Client,
	legacyClient *v1.Client,
	taskService library.Task,
	jsonrpcSvc library.JSONRPC,
	log *logger.Logger,
) library.Restore {
	return &Service{
		client:       client,
		legacyClient: legacyClient,
		taskService:  taskService,
		jsonrpcSvc:   jsonrpcSvc,
		log:          log,
	}
}

func (s *Service) GetRestorePoints(ctx context.Context, vmID uuid.UUID) ([]*payloads.RestorePoint, error) {
	var result []*payloads.RestorePoint

	path := core.NewPathBuilder().
		Resource("backup").
		Resource("logs").
		Build()

	// TODO: Filter logs by VM ID
	options := map[string]any{
		"limit": 200,
	}

	var logs []*payloads.BackupLog
	err := client.TypedGet(ctx, s.client, path, options, &logs)
	if err != nil {
		s.log.Error("Failed to get backup logs", zap.Error(err))
		return nil, err
	}

	s.log.Debug("Retrieved backup logs", zap.Int("count", len(logs)))

	// TODO: Filter logs by VM ID
	for _, log := range logs {
		if log.Status == payloads.BackupLogStatusSuccess {
			restorePoint := &payloads.RestorePoint{
				ID:         log.ID,
				Name:       log.Name,
				BackupTime: time.Now().Add(-time.Duration(log.Duration) * time.Second),
				Type:       "backup",
			}
			result = append(result, restorePoint)
		}
	}

	s.log.Debug("Filtered restore points for VM",
		zap.String("vmID", vmID.String()),
		zap.Int("totalLogs", len(logs)),
		zap.Int("matchingPoints", len(result)))

	return result, nil
}

func (s *Service) RestoreVM(ctx context.Context, backupID uuid.UUID, options *payloads.RestoreOptions) error {
	params := map[string]interface{}{
		"id": backupID.String(),
	}

	if options != nil {
		if options.StartAfterRestore {
			params["startOnBoot"] = options.StartAfterRestore
		}
		if options.SrID != uuid.Nil {
			params["sr"] = options.SrID.String()
		}
		if options.PoolID != uuid.Nil {
			params["targetPoolId"] = options.PoolID.String()
		}
		if options.NewNamePattern != "" {
			params["name_pattern"] = options.NewNamePattern
		}
	}

	logContext := []zap.Field{
		zap.String("backupID", backupID.String()),
	}

	var response string
	if err := s.jsonrpcSvc.Call("backupNg.restoreMetadata", params, &response, logContext...); err != nil {
		return err
	}

	if task.IsTaskURL(response) {
		taskID := task.ExtractTaskID(response)
		s.log.Debug("VM restore started via JSON-RPC",
			append(logContext, zap.String("taskID", taskID))...)

		taskResult, err := s.taskService.Wait(ctx, taskID)
		if err != nil {
			return err
		}

		if taskResult.Status != payloads.Success {
			return fmt.Errorf("restore task failed: %s", taskResult.Message)
		}
	}

	return nil
}

func (s *Service) ImportVM(ctx context.Context, options *payloads.ImportOptions) (*payloads.Task, error) {
	params := map[string]interface{}{
		"id": options.BackupID.String(),
		"sr": options.SrID.String(),
	}

	if options.NamePattern != "" {
		params["name_pattern"] = options.NamePattern
	}
	if options.StartOnBoot {
		params["startOnBoot"] = options.StartOnBoot
	}
	if options.NetworkConfig != nil && len(options.NetworkConfig) > 0 {
		params["networkMapping"] = options.NetworkConfig
	}

	logContext := []zap.Field{
		zap.String("backupID", options.BackupID.String()),
		zap.String("srID", options.SrID.String()),
	}

	var response string
	if err := s.jsonrpcSvc.Call("backupNg.importVmBackup", params, &response, logContext...); err != nil {
		return nil, err
	}

	if task.IsTaskURL(response) {
		taskID := task.ExtractTaskID(response)
		s.log.Debug("VM import started via JSON-RPC",
			append(logContext, zap.String("taskID", taskID))...)

		return s.taskService.Get(ctx, taskID)
	}

	return &payloads.Task{
		Status: payloads.Success,
	}, nil
}

func (s *Service) ListRestoreLogs(ctx context.Context, limit int) ([]*payloads.RestoreLog, error) {
	var result []*payloads.RestoreLog
	path := core.NewPathBuilder().
		Resource("restore").
		Resource("logs").
		Build()

	options := map[string]any{}
	if limit > 0 {
		options["limit"] = limit
	}

	err := client.TypedGet(ctx, s.client, path, options, &result)
	if err != nil {
		s.log.Error("Failed to list restore logs", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *Service) GetRestoreLog(ctx context.Context, id string) (*payloads.RestoreLog, error) {
	var result payloads.RestoreLog
	path := core.NewPathBuilder().
		Resource("restore").
		Resource("logs").
		IDString(id).
		Build()

	err := client.TypedGet(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to get restore log", zap.Error(err))
		return nil, err
	}

	return &result, nil
}
