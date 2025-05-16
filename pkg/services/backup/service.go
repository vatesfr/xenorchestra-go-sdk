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
	jsonrpcSvc   library.JSONRPC
	log          *logger.Logger
}

func New(
	client *client.Client,
	legacyClient *v1.Client,
	jsonrpcSvc library.JSONRPC,
	log *logger.Logger,
) library.Backup {
	return &Service{
		client:       client,
		legacyClient: legacyClient,
		jsonrpcSvc:   jsonrpcSvc,
		log:          log,
	}
}

func (s *Service) mapSettingsToAPIMap(settings payloads.BackupSettings) map[string]any {
	apiMap := make(map[string]any)

	if settings.Retention > 0 {
		apiMap["retention"] = settings.Retention
	}
	if settings.RemoteEnabled {
		apiMap["remoteEnabled"] = settings.RemoteEnabled
		if settings.RemoteRetention > 0 {
			apiMap["remoteRetention"] = settings.RemoteRetention
		}
	}
	if settings.ReportWhenFailOnly {
		apiMap["reportWhen"] = "failure"
	} else {
		apiMap["reportWhen"] = "always"
	}
	if len(settings.ReportRecipients) > 0 {
		apiMap["reportRecipients"] = settings.ReportRecipients
	}
	if settings.CompressionEnabled {
		apiMap["compression"] = "native"
	}
	apiMap["offlineBackup"] = settings.OfflineBackup
	apiMap["checkpointSnapshot"] = settings.CheckpointSnapshot

	if settings.Timezone != nil && *settings.Timezone != "" {
		apiMap["timezone"] = *settings.Timezone
	}

	return apiMap
}

func (s *Service) ListJobs(ctx context.Context, limit int) ([]*payloads.BackupJob, error) {
	var allJobs []*payloads.BackupJob
	jobTypes := []string{"vm", "metadata", "mirror", "full"}

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

			// Convert string jobType to payloads.BackupJobType
			switch jobType {
			case "vm":
				job.Type = payloads.BackupJobTypeVM
			case "metadata":
				job.Type = payloads.BackupJobTypeMetadata
			case "mirror":
				job.Type = payloads.BackupJobTypeMirror
			default:
				job.Type = payloads.BackupJobType(jobType)
			}

			allJobs = append(allJobs, job)
		}
	}

	return allJobs, nil
}

func (s *Service) GetJob(ctx context.Context, id string) (*payloads.BackupJob, error) {
	jobTypes := []payloads.BackupJobType{
		payloads.BackupJobTypeVM,
		payloads.BackupJobTypeMetadata,
		payloads.BackupJobTypeMirror,
	}

	for _, jobType := range jobTypes {
		var result payloads.BackupJob
		path := core.NewPathBuilder().
			Resource("backup").
			Resource("jobs").
			Resource(string(jobType)).
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
		if job.Mode == payloads.BackupJobTypeDelta || job.Mode == payloads.BackupJobTypeFull {
			job.Type = payloads.BackupJobTypeVM
		} else {
			job.Type = job.Mode
		}
	}

	if job.Settings == nil {
		job.Settings = make(map[string]payloads.BackupSettings)
	}
	defaultSettings, hasDefaultSettings := job.Settings[""]
	if !hasDefaultSettings {
		defaultSettings = payloads.BackupSettings{}
	}

	apiSettings := s.mapSettingsToAPIMap(defaultSettings)

	params := map[string]any{
		"name": job.Name,
		"mode": string(job.Mode),
		"vms":  job.VMSelection(),
	}

	if len(apiSettings) > 0 {
		fullSettingsMap := map[string]any{"": apiSettings}
		for key, val := range job.Settings {
			if key != "" {
				fullSettingsMap[key] = s.mapSettingsToAPIMap(val)
			}
		}
		params["settings"] = fullSettingsMap
	}

	if job.Remotes != nil {
		params["remotes"] = job.Remotes
	}

	logContext := []zap.Field{
		zap.String("type", string(job.Type)),
		zap.String("name", job.Name),
	}

	apiMethod := "backupNg.createJob"
	if job.Type == payloads.BackupJobTypeMetadata {
		apiMethod = "metadataBackup.createJob"
	} else if job.Type == payloads.BackupJobTypeMirror {
		apiMethod = "mirrorBackup.createJob"
	}

	var jobIDResponse string
	if err := s.jsonrpcSvc.Call(apiMethod, params, &jobIDResponse, logContext...); err != nil {
		return nil, fmt.Errorf("API call to %s failed: %w", apiMethod, err)
	}

	jobID, err := uuid.FromString(jobIDResponse)
	if err != nil {
		s.log.Error("Failed to parse job ID from response",
			append(logContext, zap.String("response", jobIDResponse), zap.Error(err))...)
		return nil, fmt.Errorf("failed to parse job ID '%s': %w", jobIDResponse, err)
	}

	fullJob, getErr := s.GetJob(ctx, jobID.String())
	if getErr != nil {
		s.log.Warn("CreateJob: Successfully created job but failed to GET its full details. Returning minimal info.",
			append(logContext, zap.String("jobID", jobID.String()), zap.Error(getErr))...)
		job.ID = jobID
		return job, nil
	}

	return fullJob, nil
}

func (s *Service) UpdateJob(ctx context.Context, job *payloads.BackupJob) (*payloads.BackupJob, error) {
	if job.ID == uuid.Nil {
		return nil, fmt.Errorf("job ID is required for update")
	}

	params := map[string]any{
		"id": job.ID.String(),
	}

	if job.Name != "" {
		params["name"] = job.Name
	}

	if job.Mode != "" {
		params["mode"] = string(job.Mode)
	}

	settings := make(map[string]any)

	if job.Settings != nil {
		for key, val := range job.Settings {
			settings[key] = s.mapSettingsToAPIMap(val)
		}
	}

	params["settings"] = settings

	if job.Remotes != nil {
		params["remotes"] = job.Remotes
	}

	logContext := []zap.Field{
		zap.String("jobID", job.ID.String()),
	}

	if err := s.jsonrpcSvc.Call("backupNg.editJob", params, nil, logContext...); err != nil {
		return nil, fmt.Errorf("API call to backupNg.editJob failed: %w", err)
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
	s.log.Warn("⚠️ CAUTION: Using RunJob will back up ALL VMs in the job! ⚠️",
		zap.String("jobID", id.String()),
		zap.String("recommendation", "Use RunJobForVMs with explicit VM IDs instead"))

	job, err := s.GetJob(ctx, id.String())
	if err != nil {
		return "", fmt.Errorf("failed to get job details for RunJob: %w", err)
	}

	jobTypeStr := string(job.Type)

	params := map[string]any{
		"id": id.String(),
	}

	logContext := []zap.Field{
		zap.String("jobID", id.String()),
		zap.String("type", jobTypeStr),
	}

	apiMethod := "backupNg.runJob"

	if job.Type == payloads.BackupJobTypeMetadata {
		apiMethod = "metadataBackup.runJob"
	} else if job.Type == payloads.BackupJobTypeMirror {
		apiMethod = "mirrorBackup.runJob"
	}

	var response string
	if err := s.jsonrpcSvc.Call(apiMethod, params, &response, logContext...); err != nil {
		return "", err
	}

	if task.IsTaskURL(response) {
		taskID := task.ExtractTaskID(response)
		return taskID, nil
	}

	return response, nil
}

func (s *Service) RunJobForVMs(ctx context.Context, id uuid.UUID, vmIDs []string, settingsOverride *payloads.BackupSettings) (string, error) {
	if len(vmIDs) == 0 {
		return "", fmt.Errorf("no VM IDs specified for RunJobForVMs")
	}

	job, err := s.GetJob(ctx, id.String())
	if err != nil {
		return "", fmt.Errorf("RunJobForVMs: failed to get job details for job ID %s: %w", id.String(), err)
	}

	params := map[string]any{
		"id": id.String(),
	}

	if len(vmIDs) == 1 {
		params["vm"] = vmIDs[0]
	} else if len(vmIDs) > 1 {
		params["vms"] = vmIDs
	}

	if settingsOverride != nil {
		apiOverrideSettings := s.mapSettingsToAPIMap(*settingsOverride)
		if len(apiOverrideSettings) > 0 {
			params["settings"] = map[string]any{
				"": apiOverrideSettings,
			}
		}
	}

	jobTypeStr := string(job.Type)

	logContext := []zap.Field{
		zap.String("jobID", id.String()),
		zap.String("type", jobTypeStr),
		zap.Int("vmCount", len(vmIDs)),
	}

	var response string
	apiMethod := "backupNg.runJob"

	if job.Type == payloads.BackupJobTypeMetadata {
		apiMethod = "metadataBackup.runJob"
	} else if job.Type == payloads.BackupJobTypeMirror {
		apiMethod = "mirrorBackup.runJob"
	}

	if errCall := s.jsonrpcSvc.Call(apiMethod, params, &response, logContext...); errCall != nil {
		return "", fmt.Errorf("API call to %s for job ID %s failed: %w", apiMethod, id.String(), errCall)
	}

	if task.IsTaskURL(response) {
		taskID := task.ExtractTaskID(response)
		return taskID, nil
	}

	return response, nil
}
