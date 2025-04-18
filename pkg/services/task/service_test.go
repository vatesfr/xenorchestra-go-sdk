package task

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
	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

func setupTaskTestServer() (*httptest.Server, library.Task) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var taskID string
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) >= 5 && pathParts[1] == "rest" && pathParts[2] == "v0" && pathParts[3] == "tasks" {
			taskID = pathParts[4]
		}

		switch {
		case r.URL.Path == "/rest/v0/tasks/task-pending" && r.Method == http.MethodGet:
			task := payloads.Task{
				ID:      "task-pending",
				Name:    "test-task",
				Status:  payloads.Pending,
				Started: payloads.APITime(time.Now()),
			}
			json.NewEncoder(w).Encode(task)

		case r.URL.Path == "/rest/v0/tasks/task-success" && r.Method == http.MethodGet:
			resultID := uuid.Must(uuid.NewV4())
			task := payloads.Task{
				ID:      "task-success",
				Name:    "test-task-success",
				Status:  payloads.Success,
				Started: payloads.APITime(time.Now().Add(-5 * time.Second)),
				EndedAt: payloads.APITime(time.Now()),
				Result:  payloads.TaskResult{ID: resultID},
				Message: "Task completed successfully",
			}
			json.NewEncoder(w).Encode(task)

		case r.URL.Path == "/rest/v0/tasks/task-failure" && r.Method == http.MethodGet:
			task := payloads.Task{
				ID:      "task-failure",
				Name:    "test-task-failure",
				Status:  payloads.Failure,
				Started: payloads.APITime(time.Now().Add(-5 * time.Second)),
				EndedAt: payloads.APITime(time.Now()),
				Message: "Task failed",
				Stack:   "Error details",
			}
			json.NewEncoder(w).Encode(task)

		case r.URL.Path == "/rest/v0/tasks/task-string-result" && r.Method == http.MethodGet:
			task := payloads.Task{
				ID:      "task-string-result",
				Name:    "test-task-string-result",
				Status:  payloads.Success,
				Started: payloads.APITime(time.Now().Add(-5 * time.Second)),
				EndedAt: payloads.APITime(time.Now()),
				Result:  payloads.TaskResult{StringID: "resource-123"},
				Message: "Task completed successfully",
			}
			json.NewEncoder(w).Encode(task)

		case strings.HasPrefix(r.URL.Path, "/rest/v0/tasks/task-progress-") && r.Method == http.MethodGet:
			progressNum := strings.TrimPrefix(taskID, "task-progress-")
			var status payloads.Status
			if progressNum == "1" {
				status = payloads.Pending
			} else if progressNum == "2" {
				status = payloads.Running
			} else {
				status = payloads.Success
				resultID := uuid.Must(uuid.NewV4())
				task := payloads.Task{
					ID:      taskID,
					Name:    "test-task-progress",
					Status:  status,
					Started: payloads.APITime(time.Now().Add(-5 * time.Second)),
					EndedAt: payloads.APITime(time.Now()),
					Result:  payloads.TaskResult{ID: resultID},
					Message: "Task completed successfully",
				}
				json.NewEncoder(w).Encode(task)
				return
			}

			task := payloads.Task{
				ID:      taskID,
				Name:    "test-task-progress",
				Status:  status,
				Started: payloads.APITime(time.Now().Add(-5 * time.Second)),
			}
			json.NewEncoder(w).Encode(task)

		case strings.HasSuffix(r.URL.Path, "/abort") && r.Method == http.MethodPost:
			result := map[string]bool{"success": true}
			json.NewEncoder(w).Encode(result)

		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "Task not found: %s"}`, taskID)
		}
	}))

	restClient := &client.Client{
		HttpClient: http.DefaultClient,
		BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
		AuthToken:  "test-token",
	}

	log, err := logger.New(false)
	if err != nil {
		panic(err)
	}

	return server, New(restClient, log)
}

func TestGet(t *testing.T) {
	server, service := setupTaskTestServer()
	defer server.Close()

	ctx := context.Background()

	t.Run("successful task", func(t *testing.T) {
		task, err := service.Get(ctx, "task-success")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "task-success", task.ID)
		assert.Equal(t, payloads.Success, task.Status)
		assert.NotEqual(t, uuid.Nil, task.Result.ID)
	})

	t.Run("pending task", func(t *testing.T) {
		task, err := service.Get(ctx, "task-pending")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "task-pending", task.ID)
		assert.Equal(t, payloads.Pending, task.Status)
	})

	t.Run("failed task", func(t *testing.T) {
		task, err := service.Get(ctx, "task-failure")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "task-failure", task.ID)
		assert.Equal(t, payloads.Failure, task.Status)
		assert.Equal(t, "Task failed", task.Message)
	})

	t.Run("string result task", func(t *testing.T) {
		task, err := service.Get(ctx, "task-string-result")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "task-string-result", task.ID)
		assert.Equal(t, payloads.Success, task.Status)
		assert.Equal(t, "resource-123", task.Result.StringID)
	})

	t.Run("non-existent task", func(t *testing.T) {
		task, err := service.Get(ctx, "non-existent")
		assert.Error(t, err)
		assert.Nil(t, task)
	})

	t.Run("with duplicate path", func(t *testing.T) {
		task, err := service.Get(ctx, "/rest/v0/tasks/task-success")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "task-success", task.ID)
	})
}

func TestAbort(t *testing.T) {
	server, service := setupTaskTestServer()
	defer server.Close()

	ctx := context.Background()

	err := service.Abort(ctx, "task-pending")
	assert.NoError(t, err)
}

func TestWait(t *testing.T) {
	server, service := setupTaskTestServer()
	defer server.Close()

	ctx := context.Background()

	t.Run("immediate success", func(t *testing.T) {
		task, err := service.Wait(ctx, "task-success")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, payloads.Success, task.Status)
	})

	t.Run("immediate failure", func(t *testing.T) {
		task, err := service.Wait(ctx, "task-failure")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, payloads.Failure, task.Status)
	})

	t.Run("task with progress", func(t *testing.T) {
		// This simulates a task that progresses from pending to success
		task, err := service.Wait(ctx, "task-progress-3")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, payloads.Success, task.Status)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		_, err := service.Wait(ctx, "task-pending")
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestHandleTaskResponse(t *testing.T) {
	server, service := setupTaskTestServer()
	defer server.Close()

	ctx := context.Background()

	t.Run("not task url", func(t *testing.T) {
		task, isTask, err := service.HandleTaskResponse(ctx, "not-a-task-url", false)
		assert.NoError(t, err)
		assert.False(t, isTask)
		assert.Nil(t, task)
	})

	t.Run("is task url without wait", func(t *testing.T) {
		task, isTask, err := service.HandleTaskResponse(ctx, "/rest/v0/tasks/task-success", false)
		assert.NoError(t, err)
		assert.True(t, isTask)
		assert.NotNil(t, task)
		assert.Equal(t, payloads.Success, task.Status)
	})

	t.Run("is task url with wait", func(t *testing.T) {
		task, isTask, err := service.HandleTaskResponse(ctx, "/rest/v0/tasks/task-success", true)
		assert.NoError(t, err)
		assert.True(t, isTask)
		assert.NotNil(t, task)
		assert.Equal(t, payloads.Success, task.Status)
	})

	t.Run("failed task with wait", func(t *testing.T) {
		task, isTask, err := service.HandleTaskResponse(ctx, "/rest/v0/tasks/task-failure", true)
		assert.NoError(t, err)
		assert.True(t, isTask)
		assert.NotNil(t, task)
		assert.Equal(t, payloads.Failure, task.Status)
	})
}

func TestHelperFunctions(t *testing.T) {
	t.Run("IsTaskURL", func(t *testing.T) {
		assert.True(t, IsTaskURL("/rest/v0/tasks/task-id"))
		assert.False(t, IsTaskURL("/api/tasks/task-id"))
		assert.False(t, IsTaskURL("task-id"))
	})

	t.Run("ExtractTaskID", func(t *testing.T) {
		assert.Equal(t, "task-id", ExtractTaskID("/rest/v0/tasks/task-id"))
		assert.Equal(t, "task-id-with-dash", ExtractTaskID("/rest/v0/tasks/task-id-with-dash"))
	})
}
