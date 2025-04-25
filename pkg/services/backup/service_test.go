package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	mock_library "github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library/mock"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func setupBackupTestServer(t *testing.T) (*httptest.Server, library.Backup) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasPrefix(r.URL.Path, "/rest/v0/") {
			switch {
			case r.URL.Path == "/rest/v0/backup/jobs/vm" && r.Method == http.MethodGet:
				var jobURLs []string

				job1ID := uuid.Must(uuid.NewV4())
				job2ID := uuid.Must(uuid.NewV4())

				jobURLs = append(jobURLs,
					fmt.Sprintf("/rest/v0/backup/jobs/vm/%s", job1ID),
					fmt.Sprintf("/rest/v0/backup/jobs/vm/%s", job2ID),
				)

				if err := json.NewEncoder(w).Encode(jobURLs); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			case strings.HasPrefix(r.URL.Path, "/rest/v0/backup/jobs/vm/") && r.Method == http.MethodGet:
				parts := strings.Split(r.URL.Path, "/")
				jobID := parts[len(parts)-1]

				id, err := uuid.FromString(jobID)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				if jobID == "nonexistent-id" {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				job := payloads.BackupJob{
					ID:       id,
					Name:     "test-backup-job",
					Mode:     "full",
					Schedule: "0 0 * * *",
					Enabled:  true,
					VMs:      []string{uuid.Must(uuid.NewV4()).String()},
					Settings: payloads.BackupSettings{
						Retention:          7,
						CompressionEnabled: true,
						ReportWhenFailOnly: false,
					},
				}

				if err := json.NewEncoder(w).Encode(job); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			case strings.HasPrefix(r.URL.Path, "/rest/v0/backup/jobs/vm/") && r.Method == http.MethodPut:
				var job payloads.BackupJob
				err := json.NewDecoder(r.Body).Decode(&job)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				if err := json.NewEncoder(w).Encode(job); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			default:
				w.WriteHeader(http.StatusNotFound)
			}
			return
		}

		// Handle JSON-RPC requests
		if r.URL.Path == "/" && r.Method == http.MethodPost {
			var request struct {
				Method string          `json:"method"`
				Params json.RawMessage `json:"params"`
				ID     int             `json:"id"`
			}

			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			switch request.Method {
			case "backupNg.createJob":
				newJobID := uuid.Must(uuid.NewV4())
				response := map[string]interface{}{
					"result": newJobID.String(),
					"id":     request.ID,
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			case "backupNg.updateJob":
				response := map[string]interface{}{
					"result": true,
					"id":     request.ID,
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			case "backupNg.deleteJob":
				response := map[string]interface{}{
					"result": true,
					"id":     request.ID,
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			case "backupNg.runJob":
				taskID := uuid.Must(uuid.NewV4())
				taskURL := fmt.Sprintf("/rest/v0/tasks/%s", taskID)

				response := map[string]interface{}{
					"result": taskURL,
					"id":     request.ID,
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}

			default:
				response := map[string]interface{}{
					"error": map[string]interface{}{
						"code":    404,
						"message": fmt.Sprintf("Method not found: %s", request.Method),
					},
					"id": request.ID,
				}
				if err := json.NewEncoder(w).Encode(response); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
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

	legacyClient := &v1.Client{
		// We only need this to satisfy the interface requirement
		// but it won't be used in the tests we're performing
	}

	ctrl := gomock.NewController(t)
	mockTaskService := mock_library.NewMockTask(ctrl)
	mockJSONRPC := mock_library.NewMockJSONRPC(ctrl)

	mockJSONRPC.EXPECT().
		Call("backupNg.createJob", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(method string, params map[string]any, result any, logContext ...zap.Field) error {
			*(result.(*string)) = uuid.Must(uuid.NewV4()).String()
			return nil
		}).AnyTimes()

	mockJSONRPC.EXPECT().
		Call("backupNg.updateJob", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(method string, params map[string]any, result any, logContext ...zap.Field) error {
			*(result.(*bool)) = true
			return nil
		}).AnyTimes()

	mockJSONRPC.EXPECT().
		Call("backupNg.deleteJob", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(method string, params map[string]any, result any, logContext ...zap.Field) error {
			*(result.(*bool)) = true
			return nil
		}).AnyTimes()

	mockJSONRPC.EXPECT().
		Call("backupNg.runJob", gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(method string, params map[string]any, result any, logContext ...zap.Field) error {
			*(result.(*string)) = fmt.Sprintf("/rest/v0/tasks/%s", uuid.Must(uuid.NewV4()))
			return nil
		}).AnyTimes()

	mockJSONRPC.EXPECT().
		ValidateResult(true, "backup job deletion", gomock.Any()).
		Return(nil).AnyTimes()

	log, _ := logger.New(false)

	backupService := New(restClient, legacyClient, mockTaskService, mockJSONRPC, log)

	return server, backupService
}

func TestListJobs(t *testing.T) {
	server, service := setupBackupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	jobs, err := service.ListJobs(ctx, 0)

	assert.NoError(t, err)
	assert.NotNil(t, jobs)
	assert.Len(t, jobs, 2)
}

func TestGetJob(t *testing.T) {
	server, service := setupBackupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("existing job", func(t *testing.T) {
		jobID := uuid.Must(uuid.NewV4())
		job, err := service.GetJob(ctx, jobID.String())

		assert.NoError(t, err)
		assert.NotNil(t, job)
		assert.Equal(t, jobID, job.ID)
		assert.Equal(t, "test-backup-job", job.Name)
	})

	t.Run("nonexistent job", func(t *testing.T) {
		job, err := service.GetJob(ctx, "nonexistent-id")

		assert.Error(t, err)
		assert.Nil(t, job)
	})
}

func TestCreateJob(t *testing.T) {
	server, service := setupBackupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	jobName := "new-backup-job"
	newJob := &payloads.BackupJob{
		Name:     jobName,
		Mode:     payloads.BackupJobTypeFull,
		Schedule: "0 0 * * *",
		Enabled:  true,
		VMs:      []string{uuid.Must(uuid.NewV4()).String()},
		Settings: payloads.BackupSettings{
			Retention:          14,
			CompressionEnabled: true,
			ReportWhenFailOnly: false,
		},
	}

	createdJob, err := service.CreateJob(ctx, newJob)

	assert.NoError(t, err)
	assert.NotNil(t, createdJob)
	assert.NotEqual(t, uuid.Nil, createdJob.ID)
	assert.Equal(t, jobName, createdJob.Name)
}

func TestUpdateJob(t *testing.T) {
	server, service := setupBackupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	jobID := uuid.Must(uuid.NewV4())
	updateJob := &payloads.BackupJob{
		ID:       jobID,
		Name:     "updated-backup-job",
		Mode:     payloads.BackupJobTypeFull,
		Schedule: "0 0 * * *",
		Enabled:  true,
		Type:     "vm",
		VMs:      []string{uuid.Must(uuid.NewV4()).String()},
		Settings: payloads.BackupSettings{
			Retention:          14,
			CompressionEnabled: true,
			ReportWhenFailOnly: false,
		},
	}

	updatedJob, err := service.UpdateJob(ctx, updateJob)

	assert.NoError(t, err)
	assert.NotNil(t, updatedJob)
	assert.Equal(t, jobID, updatedJob.ID)
	assert.Equal(t, "updated-backup-job", updatedJob.Name)
}

func TestDeleteJob(t *testing.T) {
	server, service := setupBackupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	jobID := uuid.Must(uuid.NewV4())
	err := service.DeleteJob(ctx, jobID)

	assert.NoError(t, err)
}

func TestRunJob(t *testing.T) {
	server, service := setupBackupTestServer(t)
	defer server.Close()

	ctx := context.Background()

	jobID := uuid.Must(uuid.NewV4())
	taskID, err := service.RunJob(ctx, jobID)

	assert.NoError(t, err)
	assert.NotEmpty(t, taskID)
}
