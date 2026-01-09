package task

import (
	"context"
	"errors"
	"fmt"
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
	if taskID == "" {
		return nil, fmt.Errorf("invalid taskID: %s", path)
	}

	s.log.Debug("Getting task", zap.String("taskID", taskID), zap.String("originalPath", path))

	taskPath := core.NewPathBuilder().Resource("tasks").IDString(taskID).Build()

	var result payloads.Task
	err := client.TypedGet(ctx, s.client, taskPath, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to get task", zap.String("taskID", taskID), zap.Error(err))
		return nil, err
	}

	s.log.Debug("Task retrieved successfully", zap.String("status", (string)(result.Status)))

	return &result, nil
}

func (s *Service) GetAll(ctx context.Context, limit int, filter string) ([]*payloads.Task, error) {
	path := core.NewPathBuilder().Resource("tasks").Build()
	params := make(map[string]any)
	if limit > 0 {
		params["limit"] = limit
	}
	// Get all fields to retrieve complete task objects
	params["fields"] = "*"

	if filter != "" {
		params["filter"] = filter
	}

	var results []*payloads.Task
	err := client.TypedGet(ctx, s.client, path, params, &results)
	if err != nil {
		s.log.Error("Failed to get all tasks", zap.Error(err))
		return nil, err
	}

	return results, nil
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

func (s *Service) WaitWithTimeout(ctx context.Context, id string, timeout time.Duration) (*payloads.Task, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return s.Wait(ctx, id)
}

func (s *Service) Wait(ctx context.Context, id string) (*payloads.Task, error) {
	taskID := s.cleanDuplicateV0Path(id)
	s.log.Debug("Waiting for task completion", zap.String("taskID", taskID))

	pollInterval := 2 * time.Second

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			task, err := s.Get(ctx, taskID)
			if err != nil {
				s.log.Error("Error checking task status", zap.String("taskID", taskID), zap.Error(err))
				time.Sleep(pollInterval)
				continue
			}

			if task.Status == payloads.Success || task.Status == payloads.Failure {
				s.log.Debug("Task completed",
					zap.String("taskID", taskID),
					zap.String("status", string(task.Status)))
				return task, nil
			}

			s.log.Debug("Task in progress",
				zap.String("taskID", taskID),
				zap.String("status", string(task.Status)))

			time.Sleep(pollInterval)
		}
	}
}

func (s *Service) HandleTaskResponse(
	ctx context.Context,
	response payloads.TaskIDResponse,
	waitForCompletion bool,
) (*payloads.Task, error) {
	if response.TaskID == "" {
		return nil, fmt.Errorf("No TaskID found in the payload: %v", response)
	}

	taskID := response.TaskID
	s.log.Debug("Got task URL", zap.String("taskID", taskID))

	if !waitForCompletion {
		task, err := s.Get(ctx, taskID)
		return task, err
	}

	task, err := s.Wait(ctx, taskID)
	return task, err
}
