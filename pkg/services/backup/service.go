package backup

import (
	"context"
	"fmt"
	"strings"

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
	taskService  library.Task
	jsonrpcSvc   library.JSONRPC
	log          *logger.Logger
}

func New(
	client *client.Client,
	legacyClient *v1.Client,
	taskService library.Task,
	jsonrpcSvc library.JSONRPC,
	log *logger.Logger,
) library.Backup {
	return &Service{
		client:       client,
		legacyClient: legacyClient,
		taskService:  taskService,
		jsonrpcSvc:   jsonrpcSvc,
		log:          log,
	}
}

func (s *Service) ListJobs(ctx context.Context, limit int) ([]*payloads.BackupJob, error) {
	var allJobs []*payloads.BackupJob
	jobTypes := []string{"vm", "metadata", "mirror"}

	params := make(map[string]any)
	if limit <= 0 {
		params["limit"] = core.DefaultTaskListLimit
	} else {
		params["limit"] = limit
	}

	for _, jobType := range jobTypes {
		typePath := core.NewPathBuilder().Resource("backup").Resource("jobs").Resource(jobType).Build()

		var jobPaths []string
		err := client.TypedGet(ctx, s.client, typePath, params, &jobPaths)
		if err != nil {
			s.log.Warn("Failed to get backup job paths for type",
				zap.String("type", jobType),
				zap.Error(err))
			continue
		}

		s.log.Debug("Retrieved backup job paths",
			zap.String("type", jobType),
			zap.Int("count", len(jobPaths)))

		for _, jobPath := range jobPaths {
			pathParts := strings.Split(jobPath, "/")
			if len(pathParts) < 7 {
				s.log.Warn("Invalid backup job path format, skipping",
					zap.String("jobPath", jobPath))
				continue
			}

			jobID := pathParts[len(pathParts)-1]
			job, err := s.GetJob(ctx, jobID)
			if err != nil {
				s.log.Warn("Failed to get backup job details, skipping",
					zap.String("jobPath", jobPath),
					zap.String("jobID", jobID),
					zap.Error(err))
				continue
			}

			job.Type = jobType
			allJobs = append(allJobs, job)
		}
	}

	s.log.Debug("Retrieved full backup job objects", zap.Int("count", len(allJobs)))
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

	params := map[string]any{
		"name": job.Name,
		"mode": job.Mode,
		"vms":  job.VMSelection(),
		"schedules": map[string]any{
			"temp-schedule-1": map[string]any{
				"cron":    job.Schedule,
				"enabled": job.Enabled,
			},
		},
	}

	settingsMap := map[string]any{
		"": map[string]any{},
	}

	innerSettings := settingsMap[""].(map[string]any)

	if job.Settings.Retention > 0 {
		innerSettings["retention"] = job.Settings.Retention
	}

	if job.Settings.RemoteEnabled {
		innerSettings["remoteEnabled"] = job.Settings.RemoteEnabled
		if job.Settings.RemoteRetention > 0 {
			innerSettings["remoteRetention"] = job.Settings.RemoteRetention
		}
	}

	if job.Settings.ReportWhenFailOnly {
		innerSettings["reportWhen"] = "failure"
	} else {
		innerSettings["reportWhen"] = "always"
	}

	if job.Settings.CompressionEnabled {
		innerSettings["compression"] = "native"
	}

	if len(innerSettings) > 0 {
		params["settings"] = settingsMap
	}

	logContext := []zap.Field{
		zap.String("type", job.Type),
		zap.String("name", job.Name),
	}

	var jobIDResponse string
	if err := s.jsonrpcSvc.Call("backupNg.createJob", params, &jobIDResponse, logContext...); err != nil {
		return nil, err
	}

	jobID, err := uuid.FromString(jobIDResponse)
	if err != nil {
		s.log.Error("Failed to parse job ID from response",
			append(logContext,
				zap.String("response", jobIDResponse),
				zap.Error(err))...)
		return nil, fmt.Errorf("failed to parse job ID: %w", err)
	}

	fullJob, err := s.GetJob(ctx, jobID.String())
	if err != nil {
		s.log.Error("Failed to get job details after creation",
			append(logContext,
				zap.String("jobID", jobID.String()),
				zap.Error(err))...)
		return &payloads.BackupJob{
			ID:       jobID,
			Name:     job.Name,
			Mode:     job.Mode,
			Type:     job.Type,
			Enabled:  job.Enabled,
			Schedule: job.Schedule,
			VMs:      job.VMs,
			Settings: job.Settings,
		}, nil
	}

	return fullJob, nil
}

func (s *Service) UpdateJob(ctx context.Context, job *payloads.BackupJob) (*payloads.BackupJob, error) {
	if job.Type == "" {
		job.Type = "vm"
	}

	params := map[string]any{
		"id":   job.ID.String(),
		"name": job.Name,
		"mode": job.Mode,
		"vms":  job.VMSelection(),
	}

	settings := map[string]any{
		"enabled": job.Enabled,
	}
	if job.Settings.Retention > 0 {
		settings["retention"] = job.Settings.Retention
	}
	if job.Settings.RemoteEnabled {
		settings["remoteEnabled"] = job.Settings.RemoteEnabled
		if job.Settings.RemoteRetention > 0 {
			settings["remoteRetention"] = job.Settings.RemoteRetention
		}
	}
	if job.Settings.ReportWhenFailOnly {
		settings["reportWhen"] = "failure"
	} else {
		settings["reportWhen"] = "always"
	}
	if len(job.Settings.ReportRecipients) > 0 {
		settings["reportRecipients"] = job.Settings.ReportRecipients
	}
	if job.Settings.CompressionEnabled {
		settings["compression"] = "native"
	}
	settings["offlineBackup"] = job.Settings.OfflineBackup
	settings["checkpointSnapshot"] = job.Settings.CheckpointSnapshot

	if len(settings) > 0 {
		params["settings"] = map[string]any{
			"": settings,
		}
	}

	logContext := []zap.Field{
		zap.String("jobID", job.ID.String()),
		zap.String("name", job.Name),
	}

	var success bool
	if err := s.jsonrpcSvc.Call("backupNg.editJob", params, &success, logContext...); err != nil {
		if err2 := s.jsonrpcSvc.Call("backupNg.setJob", params, &success, logContext...); err2 != nil {
			s.log.Error("Failed to update backup job, tried both editJob and setJob methods",
				append(logContext,
					zap.Error(err),
					zap.Error(err2))...)
			return nil, fmt.Errorf("failed to update backup job: %w", err)
		}
	}

	if !success {
		return nil, fmt.Errorf("failed to update backup job with ID %s", job.ID)
	}

	return s.GetJob(ctx, job.ID.String())
}

func (s *Service) DeleteJob(ctx context.Context, id uuid.UUID) error {
	params := map[string]any{
		"id": id.String(),
	}

	logContext := []zap.Field{
		zap.String("jobID", id.String()),
	}

	var success bool
	if err := s.jsonrpcSvc.Call("backupNg.deleteJob", params, &success, logContext...); err != nil {
		return err
	}

	return s.jsonrpcSvc.ValidateResult(success, "backup job deletion", logContext...)
}

// RunJob runs a backup job with its default configuration.
// This runs the job for ALL VMs defined in the job.
// For selective VM backup, use RunJobForVMs instead.
//
// ⚠️ WARNING: This method will back up ALL VMs defined in the job!
// ⚠️ DO NOT use this method in integration tests - it can cause unwanted backups!
// ⚠️ ALWAYS use RunJobForVMs with explicit VM IDs instead!
func (s *Service) RunJob(ctx context.Context, id uuid.UUID) (string, error) {
	// Log a prominent warning when this method is called
	s.log.Warn("⚠️ CAUTION: Using RunJob will back up ALL VMs in the job! ⚠️",
		zap.String("jobID", id.String()),
		zap.String("recommendation", "Use RunJobForVMs with explicit VM IDs instead"))

	jobType := ""
	job, err := s.GetJob(ctx, id.String())
	if err == nil && job != nil {
		jobType = job.Type
	}

	params := map[string]any{
		"id": id.String(),
	}

	logContext := []zap.Field{
		zap.String("jobID", id.String()),
		zap.String("type", jobType),
	}

	var response string
	if err := s.jsonrpcSvc.Call("backupNg.runJob", params, &response, logContext...); err != nil {
		return "", err
	}

	if task.IsTaskURL(response) {
		taskID := task.ExtractTaskID(response)
		s.log.Debug("Backup job run started",
			append(logContext, zap.String("taskID", taskID))...)
		return taskID, nil
	}

	return response, nil
}

func (s *Service) RunJobForVMs(ctx context.Context, id uuid.UUID, schedule string, vmIDs []string) (string, error) {
	jobType := ""
	job, err := s.GetJob(ctx, id.String())
	if err == nil && job != nil {
		jobType = job.Type
	}

	if len(vmIDs) == 0 {
		return "", fmt.Errorf("no VM IDs specified - to run for all VMs, use RunJob instead")
	}

	params := map[string]any{
		"id": id.String(),
	}

	if schedule != "" {
		params["schedule"] = schedule
	}

	if len(vmIDs) == 1 {
		params["vm"] = vmIDs[0]
	} else if len(vmIDs) > 1 {
		params["vms"] = vmIDs
	}

	logContext := []zap.Field{
		zap.String("jobID", id.String()),
		zap.String("type", jobType),
		zap.Int("vmCount", len(vmIDs)),
	}

	var response string
	if err := s.jsonrpcSvc.Call("backupNg.runJob", params, &response, logContext...); err != nil {
		return "", err
	}

	if task.IsTaskURL(response) {
		taskID := task.ExtractTaskID(response)
		s.log.Debug("Selective backup job run started",
			append(logContext, zap.String("taskID", taskID))...)
		return taskID, nil
	}

	return response, nil
}
