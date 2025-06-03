package payloads

import (
	"time"

	"github.com/gofrs/uuid"
)

// RestAPIJobQuery represents the different query paths available for the REST API
// when working with backup jobs. These are used to specify which type of data
// or operation you want to perform through the REST endpoints.
type RestAPIJobQuery string

const (
	RestAPIJobQueryVM       RestAPIJobQuery = "vm"       // Query for VM-related backup operations
	RestAPIJobQueryMetadata RestAPIJobQuery = "metadata" // Query for metadata backup operations
	RestAPIJobQueryMirror   RestAPIJobQuery = "mirror"   // Query for mirror backup operations
)

// BackupJobType defines the specific type of backup operation to be performed.
// Each type has different characteristics in terms of data transfer and storage efficiency.
type BackupJobType string

const (
	BackupJobTypeDelta    BackupJobType = "delta"    // Incremental backup containing only changes since last backup
	BackupJobTypeFull     BackupJobType = "full"     // Complete backup of all VM data
	BackupJobTypeMetadata BackupJobType = "metadata" // Backup of VM metadata only (configuration, etc.)
	BackupJobTypeMirror   BackupJobType = "mirror"   // Mirror/replication backup for disaster recovery
)

// BackupJobMode represents the operational mode of the backup job.
// This determines the overall behavior and purpose of the backup operation.
type BackupJobMode string

const (
	BackupJobModeBackup   BackupJobMode = "backup"   // Standard backup operation
	BackupJobModeMirror   BackupJobMode = "mirror"   // Mirror/replication mode
	BackupJobModeMetadata BackupJobMode = "metadata" // Metadata-only backup mode
)

// BackupJob represents the structure for creating and updating backup jobs.
// This struct is used as the request payload when communicating with the XenOrchestra API
// to define backup job configurations including VMs to backup, schedules, and settings.
type BackupJob struct {
	ID           uuid.UUID      `json:"id,omitempty"`           // Unique identifier for the backup job
	Name         string         `json:"name"`                   // Human-readable name for the backup job
	Mode         BackupJobType  `json:"mode"`                   // Type of backup operation (delta, full, etc.)
	VMs          any            `json:"vms,omitempty"`          // VM selection criteria (can be string, []string, or map)
	Type         BackupJobMode  `json:"type"`                   // Operational mode of the backup job
	Schedule     uuid.UUID      `json:"schedule"`               // Reference to the schedule that triggers this job
	Enabled      bool           `json:"enabled"`                // Whether the backup job is active
	Settings     BackupSettings `json:"settings,omitempty"`     // Detailed backup configuration settings
	Pools        any            `json:"pools,omitempty"`        // Pool selection criteria for the backup
	XOMetadata   bool           `json:"xoMetadata,omitempty"`   // Whether to include XenOrchestra metadata
	SourceRemote *string        `json:"sourceRemote,omitempty"` // Source remote for mirror/replication jobs
	Filter       map[string]any `json:"filter,omitempty"`       // Additional filtering criteria for VM selection
	Remotes      any            `json:"remotes,omitempty"`      // Remote storage targets (can be string, []string, or map)
	Compression  *string        `json:"compression,omitempty"`  // Compression algorithm to use (e.g., "zstd")
}

// BackupJobResponse represents the structure returned by the XenOrchestra REST API
// when querying backup jobs. This is the response payload format and may differ
// slightly from the request format, particularly in how settings are structured.
type BackupJobResponse struct {
	ID           uuid.UUID      `json:"id,omitempty"`           // Unique identifier for the backup job
	Name         string         `json:"name"`                   // Human-readable name for the backup job
	Mode         BackupJobType  `json:"mode"`                   // Type of backup operation (delta, full, etc.)
	VMs          any            `json:"vms,omitempty"`          // VM selection criteria as returned by API
	Type         BackupJobMode  `json:"type"`                   // Operational mode of the backup job
	Schedule     uuid.UUID      `json:"schedule"`               // Reference to the schedule that triggers this job
	Enabled      bool           `json:"enabled"`                // Whether the backup job is active
	Settings     map[string]any `json:"settings,omitempty"`     // Raw settings format as returned by REST API
	Pools        any            `json:"pools,omitempty"`        // Pool selection criteria for the backup
	XOMetadata   bool           `json:"xoMetadata,omitempty"`   // Whether to include XenOrchestra metadata
	SourceRemote *string        `json:"sourceRemote,omitempty"` // Source remote for mirror/replication jobs
	Filter       map[string]any `json:"filter,omitempty"`       // Additional filtering criteria for VM selection
	Remotes      any            `json:"remotes,omitempty"`      // Remote storage targets as returned by API
	Compression  *string        `json:"compression,omitempty"`  // Compression algorithm being used
}

func (job *BackupJob) ToJSONRPCPayload() map[string]any {
	apiMap := make(map[string]any)

	if job.Name != "" {
		apiMap["name"] = job.Name
	}
	if job.Mode != "" {
		apiMap["mode"] = string(job.Mode)
	}
	if job.Type != "" {
		apiMap["type"] = string(job.Type)
	}

	if job.Compression != nil && *job.Compression != "" {
		apiMap["compression"] = *job.Compression
	}
	if job.ID != uuid.Nil {
		apiMap["id"] = job.ID.String()
	}

	if job.VMs != nil {
		apiMap["vms"] = job.VMSelection()
	}

	if job.Remotes != nil {
		apiMap["remotes"] = job.RemoteSelection()
	}

	apiMap["srs"] = map[string]any{
		"id": map[string]any{
			"__or": []string{},
		},
	}

	settingsMap := make(map[string]any)

	defaultSettings := make(map[string]any)

	if job.Settings.Retention != nil {
		defaultSettings["retention"] = *job.Settings.Retention
	}
	if job.Settings.ReportWhen != nil {
		defaultSettings["reportWhen"] = string(*job.Settings.ReportWhen)
	}
	if len(job.Settings.ReportRecipients) > 0 {
		defaultSettings["reportRecipients"] = job.Settings.ReportRecipients
	}
	if job.Settings.OfflineBackup != nil {
		defaultSettings["offlineBackup"] = *job.Settings.OfflineBackup
	}
	if job.Settings.OfflineSnapshot != nil {
		defaultSettings["offlineSnapshot"] = *job.Settings.OfflineSnapshot
	}
	if job.Settings.CheckpointSnapshot != nil {
		defaultSettings["checkpointSnapshot"] = *job.Settings.CheckpointSnapshot
	}
	if job.Settings.RemoteEnabled != nil {
		defaultSettings["remoteEnabled"] = *job.Settings.RemoteEnabled
	}
	if job.Settings.Timezone != nil && *job.Settings.Timezone != "" {
		defaultSettings["timezone"] = *job.Settings.Timezone
	}
	if job.Settings.DeleteFirst != nil {
		defaultSettings["deleteFirst"] = *job.Settings.DeleteFirst
	}
	if job.Settings.MergeBackupsSynchronously != nil {
		defaultSettings["mergeBackupsSynchronously"] = *job.Settings.MergeBackupsSynchronously
	}
	if job.Settings.MaxExportRate != nil {
		defaultSettings["maxExportRate"] = *job.Settings.MaxExportRate
	}
	if job.Settings.NRetriesVmBackupFailures != nil {
		defaultSettings["nRetriesVmBackupFailures"] = *job.Settings.NRetriesVmBackupFailures
	}
	if job.Settings.Timeout != nil {
		defaultSettings["timeout"] = *job.Settings.Timeout
	}
	if job.Settings.BackupReportTpl != nil && *job.Settings.BackupReportTpl != "" {
		defaultSettings["backupReportTpl"] = *job.Settings.BackupReportTpl
	}
	if len(job.Settings.LongTermRetention) > 0 {
		defaultSettings["longTermRetention"] = job.Settings.LongTermRetention
	}
	if job.Settings.CompressionEnabled != nil {
		defaultSettings["compressionEnabled"] = *job.Settings.CompressionEnabled
	}
	if job.Settings.RemoteRetention != nil {
		defaultSettings["remoteRetention"] = *job.Settings.RemoteRetention
	}
	if job.Settings.CopyRetention != nil {
		defaultSettings["copyRetention"] = *job.Settings.CopyRetention
	}
	if job.Settings.CbtDestroySnapshotData != nil {
		defaultSettings["cbtDestroySnapshotData"] = *job.Settings.CbtDestroySnapshotData
	}
	if job.Settings.Concurrency != nil {
		defaultSettings["concurrency"] = *job.Settings.Concurrency
	}
	if job.Settings.NbdConcurrency != nil {
		defaultSettings["nbdConcurrency"] = *job.Settings.NbdConcurrency
	}
	if job.Settings.PreferNbd != nil {
		defaultSettings["preferNbd"] = *job.Settings.PreferNbd
	}
	if job.Settings.RetentionPoolMetadata != nil {
		defaultSettings["retentionPoolMetadata"] = *job.Settings.RetentionPoolMetadata
	}
	if job.Settings.RetentionXOMetadata != nil {
		defaultSettings["retentionXoMetadata"] = *job.Settings.RetentionXOMetadata
	}

	settingsMap[""] = defaultSettings

	// Schedule-specific settings (only exportRetention)
	if job.Schedule != uuid.Nil && job.Settings.ExportRetention != nil {
		scheduleSettings := map[string]any{
			"exportRetention": *job.Settings.ExportRetention,
		}
		settingsMap[job.Schedule.String()] = scheduleSettings
	}

	// Same here as schedule settings
	if job.Remotes != nil && job.Settings.DeleteFirst != nil {
		if remoteSelection := job.RemoteSelection(); remoteSelection != nil {
			if remoteMap, ok := remoteSelection.(map[string]any); ok {
				if id, exists := remoteMap["id"]; exists {
					switch v := id.(type) {
					case string:
						settingsMap[v] = map[string]any{
							"deleteFirst": *job.Settings.DeleteFirst,
						}
					case map[string]any:
						if orList, exists := v["__or"]; exists {
							if orSlice, ok := orList.([]string); ok {
								for _, rID := range orSlice {
									settingsMap[rID] = map[string]any{
										"deleteFirst": *job.Settings.DeleteFirst,
									}
								}
							}
						}
					}
				}
			}
		}
	}

	apiMap["settings"] = settingsMap

	return apiMap
}

// VMSelection converts the VMs field to the proper API format
// - String VM ID becomes {"id": "vm-id"}
// - []string VM IDs becomes {"id": {"__or": ["vm-id1", "vm-id2"]}}
// - map[string]struct{} gets converted to one of the above formats
func (j *BackupJob) VMSelection() any {
	switch v := j.VMs.(type) {
	case string:
		// Single VM ID as string
		return map[string]any{
			"id": v,
		}
	case []string:
		// Multiple VM IDs as string slice
		if len(v) == 1 {
			return map[string]any{
				"id": v[0],
			}
		}
		return map[string]any{
			"id": map[string]any{
				"__or": v,
			},
		}
	case map[string]struct{}:
		// Backward compatibility with map[string]struct{}
		if len(v) == 0 {
			return nil
		}
		if len(v) == 1 {
			for vmID := range v {
				return map[string]any{
					"id": vmID,
				}
			}
		}
		vmIDs := make([]string, 0, len(v))
		for vmID := range v {
			vmIDs = append(vmIDs, vmID)
		}
		return map[string]any{
			"id": map[string]any{
				"__or": vmIDs,
			},
		}
	default:
		// Return as-is for API responses or other formats
		return j.VMs
	}
}

// RemoteSelection converts the Remotes field to the proper API format
// - String Remote ID becomes {"id": "remote-id"}
// - []string Remote IDs becomes {"id": {"__or": ["remote-id1", "remote-id2"]}}
// - map[string]struct{} gets converted to one of the above formats
func (j *BackupJob) RemoteSelection() any {
	switch v := j.Remotes.(type) {
	case string:
		// Single Remote ID as string
		return map[string]any{
			"id": v,
		}
	case []string:
		// Multiple Remote IDs as string slice
		if len(v) == 0 {
			return nil
		}
		if len(v) == 1 {
			return map[string]any{
				"id": v[0],
			}
		}
		return map[string]any{
			"id": map[string]any{
				"__or": v,
			},
		}
	case map[string]struct{}:
		// Backward compatibility with map[string]struct{}
		if len(v) == 0 {
			return nil
		}
		if len(v) == 1 {
			for remoteID := range v {
				return map[string]any{
					"id": remoteID,
				}
			}
		}
		remoteIDs := make([]string, 0, len(v))
		for remoteID := range v {
			remoteIDs = append(remoteIDs, remoteID)
		}
		return map[string]any{
			"id": map[string]any{
				"__or": remoteIDs,
			},
		}
	default:
		// If it's already in the map[string]any{"id": ...} format or nil, return as-is.
		// This handles the case where the API might return it in the correct format already,
		// or if it was deliberately set to nil or the specific map structure.
		return j.Remotes
	}
}

// LongTermRetentionDurationKey defines the available time periods for long-term retention policies.
// These keys are used to configure how long backups should be kept for different retention cycles.
type LongTermRetentionDurationKey string

const (
	Daily   LongTermRetentionDurationKey = "daily"   // Daily retention cycle
	Weekly  LongTermRetentionDurationKey = "weekly"  // Weekly retention cycle
	Monthly LongTermRetentionDurationKey = "monthly" // Monthly retention cycle
	Yearly  LongTermRetentionDurationKey = "yearly"  // Yearly retention cycle
)

// LongTermRetentionDuration specifies the retention period and associated settings
// for a particular time cycle (weekly, monthly, yearly). This allows for complex
// retention policies where different types of backups are kept for different durations.
type LongTermRetentionDuration struct {
	Retention int            `json:"retention"` // Number of backups to retain for this cycle
	Settings  map[string]any `json:"settings"`  // Additional settings specific to this retention cycle
}

// LongTermRetentionObject is a map that defines the complete long-term retention policy
// by associating each retention duration key (weekly, monthly, yearly) with its
// corresponding retention configuration.
type LongTermRetentionObject map[LongTermRetentionDurationKey]LongTermRetentionDuration

// ReportWhen defines when backup job reports should be sent to configured recipients.
// This allows administrators to control notification frequency based on job outcomes.
type ReportWhen string

const (
	ReportWhenFailOnly ReportWhen = "failure" // Send reports only when backups fail
	ReportWhenAlways   ReportWhen = "always"  // Send reports for all backup jobs (success and failure)
	ReportWhenError    ReportWhen = "error"   // Send reports only when errors occur
)

// Compression defines the available compression algorithms for backup data.
// Using compression can significantly reduce backup size and transfer time.
type Compression string

const (
	Zstd Compression = "zstd" // Zstandard compression algorithm
)

// BackupSettings contains all the detailed configuration options for backup jobs.
// These settings control various aspects of the backup process including retention,
// performance, notifications, and advanced features.
type BackupSettings struct {
	Retention                 *int                    `json:"retention,omitempty"`                 // Number of backups to retain locally
	ReportWhen                *ReportWhen             `json:"reportWhen,omitempty"`                // When to send backup reports
	ReportRecipients          []string                `json:"reportRecipients,omitempty"`          // Email addresses to receive backup reports
	OfflineBackup             *bool                   `json:"offlineBackup,omitempty"`             // Whether to shutdown VM during backup
	OfflineSnapshot           *bool                   `json:"offlineSnapshot,omitempty"`           // Whether to shutdown VM during snapshot
	CheckpointSnapshot        *bool                   `json:"checkpointSnapshot,omitempty"`        // Whether to use checkpoint snapshots
	CompressionEnabled        *bool                   `json:"compressionEnabled,omitempty"`        // Whether to enable backup compression
	RemoteEnabled             *bool                   `json:"remoteEnabled,omitempty"`             // Whether to copy backups to remote storage
	RemoteRetention           *int                    `json:"remote_retention,omitempty"`          // Number of backups to retain on remote storage
	Timezone                  *string                 `json:"timezone,omitempty"`                  // Timezone for backup scheduling
	CopyRetention             *int                    `json:"copyRetention,omitempty"`             // Retention for backup copies
	ExportRetention           *int                    `json:"exportRetention,omitempty"`           // Retention for exported backups
	DeleteFirst               *bool                   `json:"deleteFirst,omitempty"`               // Whether to delete old backups before creating new ones
	MergeBackupsSynchronously *bool                   `json:"mergeBackupsSynchronously,omitempty"` // Whether to merge delta backups synchronously
	CbtDestroySnapshotData    *bool                   `json:"cbtDestroySnapshotData,omitempty"`    // Whether to destroy snapshot data for CBT
	Concurrency               *int                    `json:"concurrency,omitempty"`               // Number of concurrent backup operations
	LongTermRetention         LongTermRetentionObject `json:"longTermRetention,omitempty"`         // Long-term retention policy configuration
	MaxExportRate             *int                    `json:"maxExportRate,omitempty"`             // Maximum export rate in bytes per second
	NRetriesVmBackupFailures  *int                    `json:"nRetriesVmBackupFailures,omitempty"`  // Number of retries for failed VM backups
	NbdConcurrency            *int                    `json:"nbdConcurrency,omitempty"`            // NBD connection concurrency level
	PreferNbd                 *bool                   `json:"preferNbd,omitempty"`                 // Whether to prefer NBD over VHD streaming
	RetentionPoolMetadata     *int                    `json:"retentionPoolMetadata,omitempty"`     // Retention period for pool metadata
	RetentionXOMetadata       *int                    `json:"retentionXoMetadata,omitempty"`       // Retention period for XenOrchestra metadata
	Timeout                   *int                    `json:"timeout,omitempty"`                   // Backup operation timeout in seconds
	BackupReportTpl           *string                 `json:"backupReportTpl,omitempty"`           // Custom template for backup reports
}

// BackupLogOptions defines the parameters for querying backup job execution logs.
// This allows filtering and pagination of backup history for monitoring and debugging purposes.
type BackupLogOptions struct {
	Limit  int       `json:"limit,omitempty"`  // Maximum number of log entries to return
	Start  time.Time `json:"start,omitempty"`  // Start date/time for log query range
	End    time.Time `json:"end,omitempty"`    // End date/time for log query range
	JobID  string    `json:"job_id,omitempty"` // Filter logs by specific backup job ID
	Status string    `json:"status,omitempty"` // Filter logs by backup execution status
}

// BackupLogStatus represents the possible execution states of a backup job.
// These statuses help track the lifecycle and outcome of backup operations.
type BackupLogStatus string

const (
	BackupLogStatusPending BackupLogStatus = "pending" // Backup job is queued but not yet started
	BackupLogStatusRunning BackupLogStatus = "running" // Backup job is currently executing
	BackupLogStatusSuccess BackupLogStatus = "success" // Backup job completed successfully
)
