package payloads

import (
	"time"

	"github.com/gofrs/uuid"
)

// BackupJobType defines the type of backup job
type BackupJobType string

const (
	BackupJobTypeDelta    BackupJobType = "delta"
	BackupJobTypeFull     BackupJobType = "full"
	BackupJobTypeMetadata BackupJobType = "metadata"
)

type BackupJob struct {
	ID       uuid.UUID     `json:"id,omitempty"`
	Name     string        `json:"name"`
	Mode     BackupJobType `json:"mode"`
	Schedule string        `json:"schedule"`
	Enabled  bool          `json:"enabled"`
	// VMs can be one of:
	// - A string for a single VM ID
	// - A []string for multiple VM IDs
	// - A map[string]struct{} for backward compatibility
	// - Raw API response data when retrieving jobs
	VMs      any            `json:"vms,omitempty"`
	Settings BackupSettings `json:"settings,omitempty"`
	// Type represents the job type (vm, metadata, mirror)
	// This is not part of the API response but is set by the service
	Type string `json:"-"`
}

// VMSelection converts the VMs field to the proper API format
// - String VM ID becomes {"id": "vm-id"}
// - []string VM IDs becomes {"id": {"__or": ["vm-id1", "vm-id2"]}}
// - map[string]struct{} gets converted to one of the above formats
func (j *BackupJob) VMSelection() interface{} {
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
				return map[string]interface{}{
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

type BackupSettings struct {
	Retention          int      `json:"retention,omitempty"`
	RemoteEnabled      bool     `json:"remoteEnabled,omitempty"`
	RemoteRetention    int      `json:"remote_retention,omitempty"`
	ReportRecipients   []string `json:"report_recipients,omitempty"`
	ReportWhenFailOnly bool     `json:"report_when_fail_only,omitempty"`
	OfflineBackup      bool     `json:"offline_backup,omitempty"`
	CheckpointSnapshot bool     `json:"checkpoint_snapshot,omitempty"`
	CompressionEnabled bool     `json:"compression_enabled,omitempty"`
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

type BackupLog struct {
	ID       uuid.UUID       `json:"id"`
	Name     string          `json:"name"`
	Status   BackupLogStatus `json:"status"`
	Error    string          `json:"error,omitempty"`
	Duration int             `json:"duration"`
	Size     int             `json:"size"`
}

type VMBackup struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	JobID      string    `json:"job_id"`
	BackupTime time.Time `json:"backup_time"`
	Size       int64     `json:"size"`
	Type       string    `json:"type"`
	CanRestore bool      `json:"can_restore"`
}
