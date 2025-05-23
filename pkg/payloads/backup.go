package payloads

import (
	"time"

	"github.com/gofrs/uuid"
)

// This is used to query the different path for the rest api
type RestAPIJobQuery string

const (
	RestAPIJobQueryVM       RestAPIJobQuery = "vm"
	RestAPIJobQueryMetadata RestAPIJobQuery = "metadata"
	RestAPIJobQueryMirror   RestAPIJobQuery = "mirror"
)

// BackupJobType defines the type of backup job
type BackupJobType string

const (
	BackupJobTypeDelta    BackupJobType = "delta"
	BackupJobTypeFull     BackupJobType = "full"
	BackupJobTypeMetadata BackupJobType = "metadata"
	BackupJobTypeMirror   BackupJobType = "mirror"
)

type BackupJobMode string

const (
	BackupJobModeBackup   BackupJobMode = "backup"
	BackupJobModeMirror   BackupJobMode = "mirror"
	BackupJobModeMetadata BackupJobMode = "metadata"
)

// BackupJob represents the structure for creating/updating backup jobs (request payload ONLY)
type BackupJob struct {
	ID           uuid.UUID      `json:"id,omitempty"`
	Name         string         `json:"name"`
	Mode         BackupJobType  `json:"mode"`
	VMs          any            `json:"vms,omitempty"`
	Type         BackupJobMode  `json:"type"`
	Schedule     uuid.UUID      `json:"schedule"`
	Enabled      bool           `json:"enabled"`
	Settings     BackupSettings `json:"settings,omitempty"`
	Pools        any            `json:"pools,omitempty"`
	XOMetadata   bool           `json:"xoMetadata,omitempty"`
	SourceRemote *string        `json:"sourceRemote,omitempty"`
	Filter       map[string]any `json:"filter,omitempty"`
	Remotes      any            `json:"remotes,omitempty"`
	Compression  *string        `json:"compression,omitempty"`
}

// BackupJobResponse represents the structure returned by the REST API (response payload ONLY)
type BackupJobResponse struct {
	ID           uuid.UUID      `json:"id,omitempty"`
	Name         string         `json:"name"`
	Mode         BackupJobType  `json:"mode"`
	VMs          any            `json:"vms,omitempty"`
	Type         BackupJobMode  `json:"type"`
	Schedule     uuid.UUID      `json:"schedule"`
	Enabled      bool           `json:"enabled"`
	Settings     map[string]any `json:"settings,omitempty"` // Raw REST API format
	Pools        any            `json:"pools,omitempty"`
	XOMetadata   bool           `json:"xoMetadata,omitempty"`
	SourceRemote *string        `json:"sourceRemote,omitempty"`
	Filter       map[string]any `json:"filter,omitempty"`
	Remotes      any            `json:"remotes,omitempty"`
	Compression  *string        `json:"compression,omitempty"`
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

type LongTermRetentionDurationKey string

const (
	Weekly  LongTermRetentionDurationKey = "weekly"
	Monthly LongTermRetentionDurationKey = "monthly"
	Yearly  LongTermRetentionDurationKey = "yearly"
)

type LongTermRetentionDuration struct {
	Retention int            `json:"retention"`
	Settings  map[string]any `json:"settings"`
}

// LongTermRetentionObject is the object for the long term retention settings
type LongTermRetentionObject map[LongTermRetentionDurationKey]LongTermRetentionDuration

type ReportWhen string

const (
	ReportWhenFailOnly ReportWhen = "failure"
	ReportWhenAlways   ReportWhen = "always"
	ReportWhenError    ReportWhen = "error"
)

type Compression string

const (
	Zstd Compression = "zstd"
)

type BackupSettings struct {
	Retention                 *int                    `json:"retention,omitempty"`
	ReportWhen                *ReportWhen             `json:"reportWhen,omitempty"`
	ReportRecipients          []string                `json:"reportRecipients,omitempty"`
	OfflineBackup             *bool                   `json:"offlineBackup,omitempty"`
	OfflineSnapshot           *bool                   `json:"offlineSnapshot,omitempty"`
	CheckpointSnapshot        *bool                   `json:"checkpointSnapshot,omitempty"`
	CompressionEnabled        *bool                   `json:"compressionEnabled,omitempty"`
	RemoteEnabled             *bool                   `json:"remoteEnabled,omitempty"`
	RemoteRetention           *int                    `json:"remote_retention,omitempty"`
	Timezone                  *string                 `json:"timezone,omitempty"`
	CopyRetention             *int                    `json:"copyRetention,omitempty"`
	ExportRetention           *int                    `json:"exportRetention,omitempty"`
	DeleteFirst               *bool                   `json:"deleteFirst,omitempty"`
	MergeBackupsSynchronously *bool                   `json:"mergeBackupsSynchronously,omitempty"`
	CbtDestroySnapshotData    *bool                   `json:"cbtDestroySnapshotData,omitempty"`
	Concurrency               *int                    `json:"concurrency,omitempty"`
	LongTermRetention         LongTermRetentionObject `json:"longTermRetention,omitempty"`
	MaxExportRate             *int                    `json:"maxExportRate,omitempty"`
	NRetriesVmBackupFailures  *int                    `json:"nRetriesVmBackupFailures,omitempty"`
	NbdConcurrency            *int                    `json:"nbdConcurrency,omitempty"`
	PreferNbd                 *bool                   `json:"preferNbd,omitempty"`
	RetentionPoolMetadata     *int                    `json:"retentionPoolMetadata,omitempty"`
	RetentionXOMetadata       *int                    `json:"retentionXoMetadata,omitempty"`
	Timeout                   *int                    `json:"timeout,omitempty"`
	BackupReportTpl           *string                 `json:"backupReportTpl,omitempty"`
}

type BackupLogOptions struct {
	Limit  int       `json:"limit,omitempty"`
	Start  time.Time `json:"start,omitempty"`
	End    time.Time `json:"end,omitempty"`
	JobID  string    `json:"job_id,omitempty"`
	Status string    `json:"status,omitempty"`
}

type BackupLogStatus string

const (
	BackupLogStatusPending BackupLogStatus = "pending"
	BackupLogStatusRunning BackupLogStatus = "running"
	BackupLogStatusSuccess BackupLogStatus = "success"
)
