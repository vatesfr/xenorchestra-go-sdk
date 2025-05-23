package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -destination=mock/backup.go -package=mock_library Backup
type Backup interface {
	ListJobs(
		ctx context.Context,
		limit int,
		query payloads.RestAPIJobQuery) ([]*payloads.BackupJobResponse, error)
	GetJob(
		ctx context.Context,
		id string,
		query payloads.RestAPIJobQuery) (*payloads.BackupJobResponse, error)
	CreateJob(ctx context.Context, job *payloads.BackupJob) (*payloads.BackupJobResponse, error)
	UpdateJob(ctx context.Context, job *payloads.BackupJob) (*payloads.BackupJobResponse, error)
	DeleteJob(ctx context.Context, id uuid.UUID) error
	RunJob(ctx context.Context, id uuid.UUID) (string, error)

	RunJobForVMs(
		ctx context.Context,
		id uuid.UUID,
		vmIDs []string,
		settingsOverride *payloads.BackupSettings) (string, error)
}
