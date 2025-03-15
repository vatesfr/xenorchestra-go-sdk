package payloads

import (
	"time"

	"github.com/gofrs/uuid"
)

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
	// NOTE: For development purposes, will be replaced with the right type.
	VMs      any            `json:"vms"`
	Settings BackupSettings `json:"settings,omitempty"`
	// Type represents the job type (vm, metadata, mirror)
	// This is not part of the API response but is set by the service
	Type string `json:"-"`
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
