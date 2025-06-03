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

func (s *Service) ListJobs(
	ctx context.Context,
	limit int,
	query payloads.RestAPIJobQuery) ([]*payloads.BackupJobResponse, error) {
	var allJobs []*payloads.BackupJobResponse

	params := make(map[string]any)
	if limit <= 0 {
		params["limit"] = core.DefaultTaskListLimit
	} else {
		params["limit"] = limit
	}

	typePath := core.NewPathBuilder().
		Resource("backup").
		Resource("jobs").
		Resource(string(query)).Build()

	var jobPaths []string
	err := client.TypedGet(ctx, s.client, typePath, params, &jobPaths)
	if err != nil {
		s.log.Warn("Failed to get backup job paths for type",
			zap.String("type", string(query)),
			zap.Error(err))
		return nil, err
	}

	for _, jobPath := range jobPaths {
		pathParts := strings.Split(jobPath, "/")
		if len(pathParts) < 7 {
			s.log.Warn("Invalid backup job path format, skipping",
				zap.String("jobPath", jobPath))
			continue
		}

		jobID := pathParts[len(pathParts)-1]
		job, err := s.GetJob(ctx, jobID, query)
		if err != nil {
			s.log.Warn("Failed to get backup job details, skipping",
				zap.String("jobPath", jobPath),
				zap.String("jobID", jobID),
				zap.Error(err))
			continue
		}

		allJobs = append(allJobs, job)
	}

	return allJobs, nil
}

func (s *Service) GetJob(
	ctx context.Context,
	id string,
	query payloads.RestAPIJobQuery) (*payloads.BackupJobResponse, error) {
	var result payloads.BackupJobResponse
	path := core.NewPathBuilder().
		Resource("backup").
		Resource("jobs").
		Resource(string(query)).
		IDString(id).
		Build()

	// First, get basic job info from REST API
	err := client.TypedGet(ctx, s.client, path, core.EmptyParams, &result)
	if err != nil {
		s.log.Error("Failed to get backup job from REST API", zap.String("id", id), zap.Error(err))
		return nil, fmt.Errorf("backup job not found with id: %s", id)
	}

	// Then, get complete settings from JSONRPC API to supplement missing fields
	params := map[string]any{
		"id": id,
	}

	var apiMethod string
	switch query {
	case payloads.RestAPIJobQueryMetadata:
		apiMethod = "metadataBackup.getJob"
	case payloads.RestAPIJobQueryMirror:
		apiMethod = "mirrorBackup.getJob"
	default:
		apiMethod = "backupNg.getJob"
	}

	var jsonrpcResult map[string]any
	if err := s.jsonrpcSvc.Call(apiMethod, params, &jsonrpcResult); err != nil {
		s.log.Warn("Failed to get complete settings from JSONRPC, using REST data only",
			zap.String("id", id), zap.Error(err))
	} else {
		if jsonrpcSettings, exists := jsonrpcResult["settings"]; exists {
			if settingsMap, ok := jsonrpcSettings.(map[string]any); ok {
				result.Settings = settingsMap

				// Extract schedule ID from settings keys
				// Schedule keys have exportRetention, remote keys have deleteFirst only
				for key := range settingsMap {
					if key != "" { // Skip the default "" key
						if keySettings, ok := settingsMap[key].(map[string]any); ok {
							// If this key has exportRetention, it's a schedule ID
							if _, hasExportRetention := keySettings["exportRetention"]; hasExportRetention {
								if scheduleUUID, err := uuid.FromString(key); err == nil {
									result.Schedule = scheduleUUID
									break
								}
							}
						}
					}
				}
			}
		}

		if compression, exists := jsonrpcResult["compression"]; exists {
			if compressionStr, ok := compression.(string); ok {
				result.Compression = &compressionStr
			}
		}
	}

	result.Type = payloads.BackupJobModeBackup
	return &result, nil
}

func (s *Service) CreateJob(ctx context.Context, job *payloads.BackupJob) (*payloads.BackupJobResponse, error) {
	params := job.ToJSONRPCPayload()

	logContext := []zap.Field{
		zap.String("type", string(job.Type)),
		zap.String("name", job.Name),
	}

	var apiMethod string
	switch job.Type {
	case payloads.BackupJobModeMetadata:
		apiMethod = "metadataBackup.createJob"
	case payloads.BackupJobModeMirror:
		apiMethod = "mirrorBackup.createJob"
	default:
		apiMethod = "backupNg.createJob"
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

	var query payloads.RestAPIJobQuery
	switch job.Type {
	case payloads.BackupJobModeMetadata:
		query = payloads.RestAPIJobQueryMetadata
	case payloads.BackupJobModeMirror:
		query = payloads.RestAPIJobQueryMirror
	default:
		query = payloads.RestAPIJobQueryVM
	}

	fullJob, getErr := s.GetJob(ctx, jobID.String(), query)
	if getErr != nil {
		s.log.Error("Failed to get backup job", zap.String("id", jobID.String()), zap.Error(getErr))
		return &payloads.BackupJobResponse{
			ID:   jobID,
			Name: job.Name,
			Type: job.Type,
			Mode: job.Mode,
		}, nil
	}

	return fullJob, nil
}

func (s *Service) UpdateJob(ctx context.Context, job *payloads.BackupJob) (*payloads.BackupJobResponse, error) {
	params := job.ToJSONRPCPayload()

	logContext := []zap.Field{
		zap.String("jobID", job.ID.String()),
	}

	switch job.Type {
	case payloads.BackupJobModeMetadata:
		if err := s.jsonrpcSvc.Call("metadataBackup.editJob", params, nil, logContext...); err != nil {
			return nil, fmt.Errorf("API call to metadataBackup.editJob failed: %w", err)
		}
	default:
		if err := s.jsonrpcSvc.Call("backupNg.editJob", params, nil, logContext...); err != nil {
			return nil, fmt.Errorf("API call to backupNg.editJob failed: %w", err)
		}
	}

	var query payloads.RestAPIJobQuery
	switch job.Type {
	case payloads.BackupJobModeMetadata:
		query = payloads.RestAPIJobQueryMetadata
	default:
		query = payloads.RestAPIJobQueryVM
	}

	return s.GetJob(ctx, job.ID.String(), query)
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

	job, err := s.GetJob(ctx, id.String(), payloads.RestAPIJobQueryVM)
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

	var response string
	switch job.Type {
	case payloads.BackupJobModeMetadata:
		apiMethod := "metadataBackup.runJob"
		if err := s.jsonrpcSvc.Call(apiMethod, params, &response, logContext...); err != nil {
			return "", err
		}
	default:
		apiMethod := "backupNg.runJob"
		if err := s.jsonrpcSvc.Call(apiMethod, params, &response, logContext...); err != nil {
			return "", err
		}
	}

	if task.IsTaskURL(response) {
		taskID := task.ExtractTaskID(response)
		return taskID, nil
	}

	return response, nil
}

func (s *Service) RunJobForVMs(
	ctx context.Context,
	id uuid.UUID,
	vmIDs []string,
	settingsOverride *payloads.BackupSettings,
) (string, error) {
	if len(vmIDs) == 0 {
		return "", fmt.Errorf("no VM IDs specified for RunJobForVMs")
	}

	job, err := s.GetJob(ctx, id.String(), payloads.RestAPIJobQueryVM)
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

	jobTypeStr := string(job.Type)

	logContext := []zap.Field{
		zap.String("jobID", id.String()),
		zap.String("type", jobTypeStr),
		zap.Int("vmCount", len(vmIDs)),
	}

	var response string
	switch job.Type {
	case payloads.BackupJobModeMetadata:
		apiMethod := "metadataBackup.runJob"
		if err := s.jsonrpcSvc.Call(apiMethod, params, &response, logContext...); err != nil {
			return "", fmt.Errorf("API call to %s for job ID %s failed: %w", apiMethod, id.String(), err)
		}
	default:
		apiMethod := "backupNg.runJob"
		if err := s.jsonrpcSvc.Call(apiMethod, params, &response, logContext...); err != nil {
			return "", fmt.Errorf("API call to %s for job ID %s failed: %w", apiMethod, id.String(), err)
		}
	}

	if task.IsTaskURL(response) {
		taskID := task.ExtractTaskID(response)
		return taskID, nil
	}

	return response, nil
}
