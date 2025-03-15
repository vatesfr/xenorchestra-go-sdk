package payloads

import (
	"time"

	"github.com/gofrs/uuid"
)

type RestorePoint struct {
	ID         uuid.UUID `json:"id"`
	Name       string    `json:"name"`
	BackupTime time.Time `json:"backup_time"`
	JobID      string    `json:"job_id"`
	Type       string    `json:"type"`
	Size       int64     `json:"size"`
}

type RestoreOptions struct {
	StartAfterRestore bool      `json:"start_after_restore,omitempty"`
	PoolID            uuid.UUID `json:"pool_id,omitempty"`
	SrID              uuid.UUID `json:"sr_id,omitempty"`
	NewNamePattern    string    `json:"new_name_pattern,omitempty"`
}

type ImportOptions struct {
	BackupID      uuid.UUID         `json:"backup_id,omitempty"`
	SrID          uuid.UUID         `json:"sr_id"`
	NamePattern   string            `json:"name_pattern,omitempty"`
	StartOnBoot   bool              `json:"start_on_boot,omitempty"`
	NetworkConfig map[string]string `json:"network_config,omitempty"`
}

type RestoreLog struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	VMName    string    `json:"vm_name,omitempty"`
	BackupID  string    `json:"backup_id,omitempty"`
	SrID      string    `json:"sr_id,omitempty"`
	Error     string    `json:"error,omitempty"`
}
