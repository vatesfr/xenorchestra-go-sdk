package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate mockgen --build_flags=--mod=mod --destination mock/backup.go . Backup
type Backup interface {
	ListJobs(ctx context.Context) ([]*payloads.BackupJob, error)
	GetJob(ctx context.Context, id string) (*payloads.BackupJob, error)
	CreateJob(ctx context.Context, job *payloads.BackupJob) (*payloads.BackupJob, error)
	UpdateJob(ctx context.Context, job *payloads.BackupJob) (*payloads.BackupJob, error)
	DeleteJob(ctx context.Context, id uuid.UUID) error
	RunJob(ctx context.Context, id uuid.UUID) (string, error)
	ListLogs(ctx context.Context, id uuid.UUID) ([]*payloads.BackupLog, error)
	ListVMBackups(ctx context.Context, vmID uuid.UUID) ([]*payloads.VMBackup, error)
}
