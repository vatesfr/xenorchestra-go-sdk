package restore

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	mock_library "github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library/mock"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/zap"
)

func setupRestoreTestServer(t *testing.T) (*httptest.Server, *gomock.Controller, library.Restore) {
	ctrl := gomock.NewController(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasPrefix(r.URL.Path, "/rest/v0/") {
			switch {
			case r.URL.Path == "/rest/v0/backup/logs" && r.Method == http.MethodGet:
				logs := []*payloads.BackupLog{
					{
						ID:       uuid.Must(uuid.NewV4()),
						Name:     "backup-log-1",
						Status:   payloads.BackupLogStatusSuccess,
						Duration: 60,
						Size:     1024 * 1024 * 1024, // 1 GB
					},
					{
						ID:       uuid.Must(uuid.NewV4()),
						Name:     "backup-log-2",
						Status:   payloads.BackupLogStatusSuccess,
						Duration: 120,
						Size:     2 * 1024 * 1024 * 1024, // 2 GB
					},
					{
						ID:       uuid.Must(uuid.NewV4()),
						Name:     "backup-log-3",
						Status:   payloads.BackupLogStatusPending,
						Duration: 0,
						Size:     0,
					},
				}

				if err := json.NewEncoder(w).Encode(logs); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			case r.URL.Path == "/rest/v0/restore/logs" && r.Method == http.MethodGet:
				logs := []*payloads.RestoreLog{
					{
						ID:        uuid.Must(uuid.NewV4()).String(),
						Message:   "Restore completed successfully",
						Status:    "success",
						StartTime: time.Now().Add(-10 * time.Minute),
						EndTime:   time.Now().Add(-5 * time.Minute),
						VMName:    "test-vm-1",
						BackupID:  uuid.Must(uuid.NewV4()).String(),
						SrID:      uuid.Must(uuid.NewV4()).String(),
					},
					{
						ID:        uuid.Must(uuid.NewV4()).String(),
						Message:   "Restore completed successfully",
						Status:    "success",
						StartTime: time.Now().Add(-20 * time.Minute),
						EndTime:   time.Now().Add(-15 * time.Minute),
						VMName:    "test-vm-2",
						BackupID:  uuid.Must(uuid.NewV4()).String(),
						SrID:      uuid.Must(uuid.NewV4()).String(),
					},
				}

				var limit int
				_, err := fmt.Sscanf(r.URL.Query().Get("limit"), "%d", &limit)
				if err != nil {
					// TODO: add a default limit + warning to set it up in the docs
					limit = 0
				}

				if limit > 0 && limit < len(logs) {
					logs = logs[:limit]
				}

				if err := json.NewEncoder(w).Encode(logs); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			case strings.HasPrefix(r.URL.Path, "/rest/v0/restore/logs/") && r.Method == http.MethodGet:
				parts := strings.Split(r.URL.Path, "/")
				logID := parts[len(parts)-1]

				if logID == "nonexistent-id" {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				log := &payloads.RestoreLog{
					ID:        logID,
					Message:   "Restore completed successfully",
					Status:    "success",
					StartTime: time.Now().Add(-10 * time.Minute),
					EndTime:   time.Now().Add(-5 * time.Minute),
					VMName:    "test-vm-restore",
					BackupID:  uuid.Must(uuid.NewV4()).String(),
					SrID:      uuid.Must(uuid.NewV4()).String(),
				}

				if err := json.NewEncoder(w).Encode(log); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			default:
				w.WriteHeader(http.StatusNotFound)
			}
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	serverURL := server.URL

	restClient := &client.Client{
		HttpClient: http.DefaultClient,
		BaseURL:    &url.URL{Scheme: "http", Host: serverURL[7:], Path: "/rest/v0"},
		AuthToken:  "test-token",
	}

	legacyClient, _ := v1.NewClient(v1.Config{Url: serverURL})

	mockTaskService := mock_library.NewMockTask(ctrl)
	mockJSONRPC := mock_library.NewMockJSONRPC(ctrl)

	mockTaskService.EXPECT().
		Wait(gomock.Any(), gomock.Any()).
		Return(&payloads.Task{
			Status: payloads.Success,
		}, nil).
		AnyTimes()

	mockTaskService.EXPECT().
		Get(gomock.Any(), gomock.Any()).
		Return(&payloads.Task{
			Status: payloads.Success,
		}, nil).
		AnyTimes()

	mockJSONRPC.EXPECT().
		Call(gomock.Eq("backupNg.restoreMetadata"), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(method string, params map[string]any, result *string, fields ...zap.Field) error {
			backupID, ok := params["id"].(string)
			if ok && backupID == "error-id" {
				return fmt.Errorf("failed to restore VM")
			}

			*result = "/rest/v0/tasks/restore-task-id"
			return nil
		}).
		AnyTimes()

	mockJSONRPC.EXPECT().
		Call(gomock.Eq("backupNg.importVmBackup"), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(method string, params map[string]any, result *string, fields ...zap.Field) error {
			backupID, ok := params["id"].(string)
			if ok && backupID == "error-id" {
				return fmt.Errorf("failed to import VM backup")
			}

			*result = "/rest/v0/tasks/import-task-id"
			return nil
		}).
		AnyTimes()

	log, _ := logger.New(false)
	restoreService := New(restClient, legacyClient.(*v1.Client), mockTaskService, mockJSONRPC, log)

	return server, ctrl, restoreService
}

func TestGetRestorePoints(t *testing.T) {
	server, ctrl, service := setupRestoreTestServer(t)
	defer server.Close()
	defer ctrl.Finish()

	ctx := context.Background()

	vmID := uuid.Must(uuid.NewV4())
	restorePoints, err := service.GetRestorePoints(ctx, vmID)

	assert.NoError(t, err)
	assert.NotNil(t, restorePoints)
	assert.Len(t, restorePoints, 2, "Should only include successful backup logs as restore points")

	for _, point := range restorePoints {
		assert.NotEqual(t, uuid.Nil, point.ID)
		assert.NotEmpty(t, point.Name)
		assert.Equal(t, "backup", point.Type)
	}
}

func TestRestoreVM(t *testing.T) {
	server, ctrl, service := setupRestoreTestServer(t)
	defer server.Close()
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("successful restore", func(t *testing.T) {
		backupID := uuid.Must(uuid.NewV4())
		options := &payloads.RestoreOptions{
			StartAfterRestore: true,
			SrID:              uuid.Must(uuid.NewV4()),
			PoolID:            uuid.Must(uuid.NewV4()),
			NewNamePattern:    "restored-{name}",
		}

		err := service.RestoreVM(ctx, backupID, options)
		assert.NoError(t, err)
	})

	t.Run("restore error", func(t *testing.T) {
		errorID, _ := uuid.FromString("error-id")
		options := &payloads.RestoreOptions{
			StartAfterRestore: true,
		}

		err := service.RestoreVM(ctx, errorID, options)
		assert.Error(t, err)
	})
}

func TestImportVM(t *testing.T) {
	server, ctrl, service := setupRestoreTestServer(t)
	defer server.Close()
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("successful import", func(t *testing.T) {
		options := &payloads.ImportOptions{
			BackupID:    uuid.Must(uuid.NewV4()),
			SrID:        uuid.Must(uuid.NewV4()),
			NamePattern: "imported-{name}",
			StartOnBoot: true,
			NetworkConfig: map[string]string{
				"network1": "network-mapping1",
			},
		}

		task, err := service.ImportVM(ctx, options)
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, payloads.Success, task.Status)
	})

	t.Run("import error", func(t *testing.T) {
		errorID, _ := uuid.FromString("error-id")
		options := &payloads.ImportOptions{
			BackupID: errorID,
			SrID:     uuid.Must(uuid.NewV4()),
		}

		task, err := service.ImportVM(ctx, options)
		assert.Error(t, err)
		assert.Nil(t, task)
	})
}

func TestListRestoreLogs(t *testing.T) {
	server, ctrl, service := setupRestoreTestServer(t)
	defer server.Close()
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("list without limit", func(t *testing.T) {
		logs, err := service.ListRestoreLogs(ctx, 0)
		assert.NoError(t, err)
		assert.NotNil(t, logs)
		assert.Len(t, logs, 2)
	})

	t.Run("list with limit", func(t *testing.T) {
		logs, err := service.ListRestoreLogs(ctx, 1)
		assert.NoError(t, err)
		assert.NotNil(t, logs)
		assert.Len(t, logs, 1)
	})
}

func TestGetRestoreLog(t *testing.T) {
	server, ctrl, service := setupRestoreTestServer(t)
	defer server.Close()
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("get existing log", func(t *testing.T) {
		logID := "test-log-id"
		log, err := service.GetRestoreLog(ctx, logID)
		assert.NoError(t, err)
		assert.NotNil(t, log)
		assert.Equal(t, logID, log.ID)
		assert.Equal(t, "success", log.Status)
	})

	t.Run("get nonexistent log", func(t *testing.T) {
		log, err := service.GetRestoreLog(ctx, "nonexistent-id")
		assert.Error(t, err)
		assert.Nil(t, log)
	})
}
