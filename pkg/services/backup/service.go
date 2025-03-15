package backup

import (
	"context"
	"fmt"
	"strings"
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

func New(client *client.Client, log *logger.Logger, taskService library.Task) library.Backup {
	return &Service{
		client:      client,
		log:         log,
		taskService: taskService,
	}
}

func (s *Service) ListJobs(ctx context.Context) ([]*payloads.BackupJob, error) {
	var allJobs []*payloads.BackupJob

	jobTypes := []string{"vm", "metadata", "mirror"}

	for _, jobType := range jobTypes {
		s.log.Debug("Fetching jobs for type", zap.String("type", jobType))

		var jobURLs []string
		typePath := core.NewPathBuilder().Resource("backup").Resource("jobs").Resource(jobType).Build()

		err := client.TypedGet(ctx, s.client, typePath, core.EmptyParams, &jobURLs)
		if err != nil {
			s.log.Warn("Failed to get backup job URLs for type",
				zap.String("type", jobType),
				zap.Error(err))
			continue
		}

		s.log.Debug("Retrieved job URLs",
			zap.String("type", jobType),
			zap.Int("count", len(jobURLs)))
		for _, urlPath := range jobURLs {
			parts := strings.Split(urlPath, "/")
			if len(parts) < 1 {
				s.log.Warn("Invalid job URL format", zap.String("url", urlPath))
				continue
			}

			idStr := parts[len(parts)-1]

			// Get the individual job
			var job payloads.BackupJob
			jobPath := core.NewPathBuilder().
				Resource("backup").
				Resource("jobs").
				Resource(jobType).
				IDString(idStr).
				Build()

			err := client.TypedGet(ctx, s.client, jobPath, core.EmptyParams, &job)
			if err != nil {
				s.log.Warn("Failed to get backup job by ID",
					zap.String("type", jobType),
					zap.String("id", idStr),
					zap.Error(err))
				continue
			}

			job.Type = jobType
			allJobs = append(allJobs, &job)
		}
	}

	return allJobs, nil
}

func (s *Service) GetJob(ctx context.Context, id string) (*payloads.BackupJob, error) {
	jobTypes := []string{"vm", "metadata", "mirror"}

	for _, jobType := range jobTypes {
		var result payloads.BackupJob
		path := core.NewPathBuilder().
			Resource("backup").
			Resource("jobs").
			Resource(jobType).
			IDString(id).
			Build()

		err := client.TypedGet(ctx, s.client, path, core.EmptyParams, &result)
		if err == nil {
			result.Type = jobType
			return &result, nil
		}
	}

	s.log.Error("Failed to get backup job", zap.String("id", id))
	return nil, fmt.Errorf("backup job not found with id: %s", id)
}

func (s *Service) CreateJob(ctx context.Context, job *payloads.BackupJob) (*payloads.BackupJob, error) {
	if job.Type == "" {
		job.Type = "vm"
	}

	var result payloads.BackupJob
	path := core.NewPathBuilder().
		Resource("backup").
		Resource("jobs").
		Resource(job.Type).
		Build()

	err := client.TypedPost(ctx, s.client, path, job, &result)
	if err != nil {
		s.log.Error("Failed to create backup job",
			zap.String("type", job.Type),
			zap.Error(err))
		return nil, err
	}

	result.Type = job.Type
	return &result, nil
}

func (s *Service) UpdateJob(ctx context.Context, job *payloads.BackupJob) (*payloads.BackupJob, error) {
	if job.Type == "" {
		job.Type = "vm"
	}

	var result payloads.BackupJob
	path := core.NewPathBuilder().
		Resource("backup").
		Resource("jobs").
		Resource(job.Type).
		ID(job.ID).
		Build()

	err := client.TypedPut(ctx, s.client, path, job, &result)
	if err != nil {
		s.log.Error("Failed to update backup job",
			zap.String("id", job.ID.String()),
			zap.String("type", job.Type),
			zap.Error(err))
		return nil, err
	}

	result.Type = job.Type
	return &result, nil
}

func (s *Service) DeleteJob(ctx context.Context, id uuid.UUID) error {
	jobTypes := []string{"vm", "metadata", "mirror"}

	for _, jobType := range jobTypes {
		path := core.NewPathBuilder().
			Resource("backup").
			Resource("jobs").
			Resource(jobType).
			ID(id).
			Build()

		var result struct {
			Success bool `json:"success"`
		}

		err := client.TypedDelete(ctx, s.client, path, core.EmptyParams, &result)
		if err == nil {
			return nil
		}
	}

	s.log.Error("Failed to delete backup job", zap.String("id", id.String()))
	return fmt.Errorf("failed to delete backup job with id: %s", id.String())
}

func (s *Service) RunJob(ctx context.Context, id uuid.UUID) (string, error) {
	jobTypes := []string{"vm", "metadata", "mirror"}

	for _, jobType := range jobTypes {
		path := core.NewPathBuilder().
			Resource("backup").
			Resource("jobs").
			Resource(jobType).
			ID(id).
			Action("run").
			Build()

		var response string
		err := client.TypedPost(ctx, s.client, path, core.EmptyParams, &response)
		if err == nil {
			if task.IsTaskURL(response) {
				taskID := task.ExtractTaskID(response)
				s.log.Debug("Backup job run started",
					zap.String("jobID", id.String()),
					zap.String("taskID", taskID))
				return taskID, nil
			}
			return response, nil
		}
	}

	s.log.Error("Failed to run backup job", zap.String("id", id.String()))
	return "", fmt.Errorf("failed to run backup job with id: %s", id.String())
}

func (s *Service) ListLogs(ctx context.Context, id uuid.UUID) ([]*payloads.BackupLog, error) {
	var result []*payloads.BackupLog
	path := core.NewPathBuilder().
		Resource("backup").
		Resource("logs").
		Build()

	options := map[string]any{
		"job_id": id.String(),
	}

	err := client.TypedGet(ctx, s.client, path, options, &result)
	if err != nil {
		s.log.Error("Failed to list backup logs", zap.Error(err))
		return nil, err
	}
	return result, nil
}

func (s *Service) ListVMBackups(ctx context.Context, vmID uuid.UUID) ([]*payloads.VMBackup, error) {
	var logs []*payloads.BackupLog
	path := core.NewPathBuilder().
		Resource("backup").
		Resource("logs").
		Build()

	// TODO: replace with a limit parameter
	options := map[string]any{
		"limit": 100,
	}

	err := client.TypedGet(ctx, s.client, path, options, &logs)
	if err != nil {
		s.log.Error("Failed to list backup logs", zap.Error(err))
		return nil, err
	}

	// Filter logs for this VM and convert to VMBackup objects
	var backups []*payloads.VMBackup
	for _, log := range logs {
		if log.Status == payloads.BackupLogStatusSuccess {
			backup := &payloads.VMBackup{
				ID:         log.ID,
				Name:       log.Name,
				Size:       int64(log.Size),
				BackupTime: time.Now(),
				JobID:      "",
				Type:       "vm",
				CanRestore: true,
			}
			backups = append(backups, backup)
		}
	}

	s.log.Debug("Filtered VM backups from logs",
		zap.String("vmID", vmID.String()),
		zap.Int("totalLogs", len(logs)),
		zap.Int("matchingBackups", len(backups)))

	return backups, nil
}
