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
	path := core.NewPathBuilder().
		Resource("backup").
		Resource("logs").
		Build()

	params := map[string]any{
		"vm":     vmID.String(),
		"status": "success",
	}

	var logs []*payloads.BackupLog
	err := client.TypedGet(ctx, s.client, path, params, &logs)
	if err != nil {
		s.log.Error("Failed to get backup logs for VM",
			zap.String("vmID", vmID.String()),
			zap.Error(err))
		return nil, err
	}

	s.log.Debug("Retrieved backup logs for VM",
		zap.String("vmID", vmID.String()),
		zap.Int("count", len(logs)))

	result := make([]*payloads.RestorePoint, 0, len(logs))
	for _, log := range logs {
		restorePoint := &payloads.RestorePoint{
			ID:         log.ID,
			Name:       log.Name,
			BackupTime: time.Now().Add(-time.Duration(log.Duration) * time.Second),
			Type:       "backup",
		}
		result = append(result, restorePoint)
	}

	return result, nil
}

func (s *Service) RestoreVM(ctx context.Context, backupID uuid.UUID, options *payloads.RestoreOptions) error {
	params := map[string]any{
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
	params := map[string]any{
		"id": options.BackupID.String(),
		"sr": options.SrID.String(),
	}

	if options.NamePattern != "" {
		params["name_pattern"] = options.NamePattern
	}
	if options.StartOnBoot {
		params["startOnBoot"] = options.StartOnBoot
	}
	if len(options.NetworkConfig) > 0 {
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
