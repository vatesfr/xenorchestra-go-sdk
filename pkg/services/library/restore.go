package library

import (
	"context"

	"github.com/gofrs/uuid"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

// I am not if the restore interface shouldn't be added to the VM interface.
// It's related to the VM however it's a different concept as well since from
// the rest perspective it's on his own path... TODO: Check with the DevOps team.
//
//go:generate go run go.uber.org/mock/mockgen -source=$GOFILE -destination=mock/restore.go -package=mock_library Restore
type Restore interface {
	GetRestorePoints(ctx context.Context, vmID uuid.UUID) ([]*payloads.RestorePoint, error)
	RestoreVM(ctx context.Context, backupID uuid.UUID, options *payloads.RestoreOptions) error
	ImportVM(ctx context.Context, options *payloads.ImportOptions) (*payloads.Task, error)

	ListRestoreLogs(ctx context.Context, limit int) ([]*payloads.RestoreLog, error)
	GetRestoreLog(ctx context.Context, id string) (*payloads.RestoreLog, error)
}
