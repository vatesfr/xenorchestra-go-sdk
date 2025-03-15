package restore

import (
	"context"
	"fmt"
	"time"

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
	client      *client.Client
	log         *logger.Logger
	taskService library.Task
}

func New(client *client.Client, log *logger.Logger, taskService library.Task) library.Restore {
	return &Service{
		client:      client,
		log:         log,
		taskService: taskService,
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
				ID:   log.ID,
				Name: log.Name,
				// Approximate backup time based on log duration
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
	path := core.NewPathBuilder().
		Resource("backup").
		Resource("restore").
		ID(backupID).
		Build()

	var response string
	err := client.TypedPost(ctx, s.client, path, options, &response)
	if err != nil {
		s.log.Error("Failed to restore VM", zap.Error(err))
		return err
	}

	if task.IsTaskURL(response) {
		taskID := task.ExtractTaskID(response)
		s.log.Debug("VM restore started",
			zap.String("backupID", backupID.String()),
			zap.String("taskID", taskID))

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
	path := core.NewPathBuilder().
		Resource("backup").
		Resource("import").
		Build()

	var response string
	err := client.TypedPost(ctx, s.client, path, options, &response)
	if err != nil {
		s.log.Error("Failed to import VM", zap.Error(err))
		return nil, err
	}

	if task.IsTaskURL(response) {
		taskID := task.ExtractTaskID(response)
		s.log.Debug("VM import started", zap.String("taskID", taskID))

		taskResult, err := s.taskService.Get(ctx, taskID)
		if err != nil {
			return nil, err
		}

		return taskResult, nil
	}

	// NOTE: For development purposes, we return a dummy task.
	// will be replaced with the right type.
	dummyTask := &payloads.Task{
		Status: payloads.Success,
	}

	return dummyTask, nil
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
