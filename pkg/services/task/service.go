package task

import (
	"context"
	"errors"
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

func New(client *client.Client, log *logger.Logger) library.Task {
	return &Service{client: client, log: log}
}

// cleanDuplicateV0Path removes the redundant "/rest/v0" from paths.
// This is needed because VM creation returns a path with "/rest/v0" prefix,
// but our client already includes "/v0/rest" in the base URL.
func (s *Service) cleanDuplicateV0Path(path string) string {
	if !strings.HasPrefix(path, "/") {
		return path
	}
	return strings.TrimPrefix(path, "/rest/v0/tasks/")
}

func (s *Service) Get(ctx context.Context, path string) (*payloads.Task, error) {
	taskID := s.cleanDuplicateV0Path(path)

	s.log.Debug("Getting task", zap.String("taskID", taskID), zap.String("originalPath", path))

	taskPath := core.NewPathBuilder().Resource("tasks").IDString(taskID).Build()

	var result payloads.Task
	err := client.TypedGet(ctx, s.client, taskPath, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to get task", zap.String("taskID", taskID), zap.Error(err))
		return nil, err
	}

	// TODO: remove noisy logs after development.
	logFields := []zap.Field{
		zap.String("taskID", taskID),
		zap.String("status", string(result.Status)),
	}

	if result.Status == payloads.Success {
		if result.Result.ID != uuid.Nil {
			logFields = append(logFields, zap.String("resultID", result.Result.ID.String()))
		}
		if result.Result.StringID != "" {
			logFields = append(logFields, zap.String("resultStringID", result.Result.StringID))
		}
	}

	s.log.Debug("Task retrieved successfully", logFields...)

	return &result, nil
}

func (s *Service) Abort(ctx context.Context, id string) error {
	path := core.NewPathBuilder().Resource("tasks").IDString(id).Action("abort").Build()

	var result struct {
		Success bool `json:"success"`
	}

	err := client.TypedPost(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to abort task", zap.String("taskID", id), zap.Error(err))
		return err
	}

	if !result.Success {
		return errors.New("failed to abort task, the API returned a non-success response")
	}
	return nil
}

func (s *Service) Wait(ctx context.Context, id string) (*payloads.Task, error) {
	const defaultTimeout = 5 * time.Minute
	return s.WaitWithTimeout(ctx, id, defaultTimeout)
}

func (s *Service) WaitWithTimeout(ctx context.Context, id string, timeout time.Duration) (*payloads.Task, error) {
	taskID := s.cleanDuplicateV0Path(id)
	s.log.Debug("Waiting for task completion", zap.String("taskID", taskID), zap.Duration("timeout", timeout))

	deadline := time.Now().Add(timeout)
	pollInterval := 2 * time.Second

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		task, err := s.Get(ctx, taskID)
		if err != nil {
			s.log.Error("Error checking task status", zap.String("taskID", taskID), zap.Error(err))
			time.Sleep(pollInterval)
			continue
		}

		if task.Status == payloads.Success || task.Status == payloads.Failure {
			logFields := []zap.Field{
				zap.String("taskID", taskID),
				zap.String("status", string(task.Status)),
			}

			if task.Status == payloads.Success {
				if task.Result.ID != uuid.Nil {
					logFields = append(logFields, zap.String("resultID", task.Result.ID.String()))
				}
				if task.Result.StringID != "" {
					logFields = append(logFields, zap.String("resultStringID", task.Result.StringID))
				}
			}

			s.log.Debug("Task completed", logFields...)
			return task, nil
		}

		s.log.Debug("Task in progress",
			zap.String("taskID", taskID),
			zap.String("status", string(task.Status)))

		time.Sleep(pollInterval)
	}

	return nil, fmt.Errorf("task %s did not complete within the timeout of %v", taskID, timeout)
}

func IsTaskURL(s string) bool {
	return strings.HasPrefix(s, "/rest/v0/tasks/")
}

func ExtractTaskID(taskURL string) string {
	return strings.TrimPrefix(taskURL, "/rest/v0/tasks/")
}

func (s *Service) HandleTaskResponse(
	ctx context.Context,
	response string,
	waitForCompletion bool,
) (*payloads.Task, bool, error) {
	if !IsTaskURL(response) {
		return nil, false, nil
	}

	taskID := ExtractTaskID(response)
	s.log.Debug("Got task URL", zap.String("taskID", taskID))

	if !waitForCompletion {
		task, err := s.Get(ctx, taskID)
		return task, true, err
	}

	task, err := s.Wait(ctx, taskID)
	return task, true, err
}
