package task

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

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

func (s *Service) Get(ctx context.Context, path string) (*payloads.Task, error) {
	taskID := core.CleanDuplicateV0Path(path)
	taskPath := core.NewPathBuilder().Resource("tasks").IDString(taskID).Build()

	var result payloads.Task
	err := client.TypedGet(ctx, s.client, taskPath, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to get task", zap.String("taskID", taskID), zap.Error(err))
		return nil, err
	}

	return &result, nil
}

func (s *Service) List(ctx context.Context, options map[string]any) ([]*payloads.Task, error) {
	path := core.NewPathBuilder().Resource("tasks").Build()

	params := make(map[string]any)

	maps.Copy(params, options)

	if _, ok := options["limit"]; !ok {
		params["limit"] = core.DefaultTaskListLimit
	}

	var taskPaths []string
	err := client.TypedGet(ctx, s.client, path, params, &taskPaths)
	if err != nil {
		s.log.Error("Failed to list task paths", zap.Error(err))
		return nil, err
	}

	s.log.Debug("Retrieved task paths", zap.Int("count", len(taskPaths)))

	var tasks []*payloads.Task
	for _, taskPath := range taskPaths {
		taskID := core.CleanDuplicateV0Path(taskPath)

		task, err := s.Get(ctx, taskID)
		if err != nil {
			s.log.Warn("Failed to get task details, skipping",
				zap.String("taskPath", taskPath),
				zap.String("taskID", taskID),
				zap.Error(err))
			continue
		}

		tasks = append(tasks, task)
	}

	s.log.Debug("Retrieved full task objects", zap.Int("count", len(tasks)))
	return tasks, nil
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
	taskID := core.CleanDuplicateV0Path(id)

	deadline := time.Now().Add(timeout)
	pollInterval := 7 * time.Second

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		s.log.Debug("Polling task status", zap.String("taskID", taskID))
		task, err := s.Get(ctx, taskID)
		if err != nil {
			s.log.Warn("Error checking task status, will retry",
				zap.String("taskID", taskID),
				zap.Error(err))
			time.Sleep(pollInterval)
			continue
		}

		if task.Status == payloads.Success || task.Status == payloads.Failure {
			return task, nil
		}

		time.Sleep(pollInterval)
	}

	return nil, fmt.Errorf("task %s did not complete within the timeout of %v", taskID, timeout)
}

func IsTaskURL(s string) bool {
	isTask := strings.HasPrefix(s, "/rest/v0/tasks/")
	return isTask
}

func ExtractTaskID(taskURL string) string {
	return strings.TrimPrefix(taskURL, "/rest/v0/tasks/")
}

func (s *Service) HandleTaskResponse(
	ctx context.Context,
	response string,
) (*payloads.Task, error) {
	s.log.Info("Handling potential task response", zap.String("response", response))

	if !IsTaskURL(response) {
		s.log.Info("Response is not a task URL", zap.String("response", response))
		return nil, nil
	}

	taskID := ExtractTaskID(response)
	s.log.Info("Response is a task URL", zap.String("taskID", taskID))

	task, err := s.Wait(ctx, taskID)
	if err != nil {
		s.log.Error("Failed waiting for task", zap.String("taskID", taskID), zap.Error(err))
		return nil, err
	}

	return task, nil
}
