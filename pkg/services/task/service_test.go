package task

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/core"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

var mockTasks = map[string]*payloads.Task{
	"task-pending": {
		ID:      "task-pending",
		Name:    "test-task-pending",
		Status:  payloads.Pending,
		Started: payloads.APITime(time.Now().Add(-1 * time.Second)),
	},
	"task-success": {
		ID:      "task-success",
		Name:    "test-task-success",
		Status:  payloads.Success,
		Started: payloads.APITime(time.Now().Add(-5 * time.Second)),
		EndedAt: payloads.APITime(time.Now()),
		Result:  payloads.TaskResult{ID: uuid.Must(uuid.NewV4())},
		Message: "Task completed successfully",
	},
	"task-failure": {
		ID:      "task-failure",
		Name:    "test-task-failure",
		Status:  payloads.Failure,
		Started: payloads.APITime(time.Now().Add(-5 * time.Second)),
		EndedAt: payloads.APITime(time.Now()),
		Message: "Task failed",
		Stack:   "Error details",
	},
	"task-progress-start": {
		ID:      "task-progress-start",
		Name:    "test-task-progress",
		Status:  payloads.Pending,
		Started: payloads.APITime(time.Now().Add(-10 * time.Second)),
	},
}

// Simulate task progression for Wait test
var progressCounter = 0

func setupTaskTestServer(t *testing.T) (*httptest.Server, library.Task) {
	progressCounter = 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.URL.Path == "/rest/v0/tasks" && r.Method == http.MethodGet:
			limitStr := r.URL.Query().Get("limit")
			limit := core.DefaultTaskListLimit
			if limitStr != "" {
				if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
					limit = parsedLimit
				}
			}

			taskPaths := make([]string, 0, len(mockTasks))
			count := 0
			for id := range mockTasks {
				if count >= limit {
					break
				}
				taskPaths = append(taskPaths, fmt.Sprintf("/rest/v0/tasks/%s", id))
				count++
			}
			if err := json.NewEncoder(w).Encode(taskPaths); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return

		case strings.HasPrefix(r.URL.Path, "/rest/v0/tasks/") &&
			!strings.HasSuffix(r.URL.Path, "/abort") && r.Method == http.MethodGet:
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) != 5 {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, `{"error": "Invalid task path format: %s"}`, r.URL.Path)
				return
			}
			taskID := parts[4]

			if taskID == "task-progress-start" {
				progressCounter++
				task := mockTasks[taskID]
				taskCopy := *task
				switch progressCounter {
				case 1:
					taskCopy.Status = payloads.Pending
				case 2, 3:
					taskCopy.Status = payloads.Running
				default:
					taskCopy.Status = payloads.Success
					taskCopy.EndedAt = payloads.APITime(time.Now())
					taskCopy.Message = "Task completed successfully after polling"
				}
				if err := json.NewEncoder(w).Encode(taskCopy); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
				}
				return
			}

			task, exists := mockTasks[taskID]
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, `{"error": "Task not found: %s"}`, taskID)
				return
			}

			if err := json.NewEncoder(w).Encode(task); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return

		case strings.HasSuffix(r.URL.Path, "/abort") && r.Method == http.MethodPost:
			result := map[string]bool{"success": true}
			if err := json.NewEncoder(w).Encode(result); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return

		default:
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, `{"error": "Unhandled path: %s %s"}`, r.Method, r.URL.Path)
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
	server, service := setupTaskTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("successful task", func(t *testing.T) {
		task, err := service.Get(ctx, "task-success")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "task-success", string(task.ID))
		assert.Equal(t, payloads.Success, task.Status)
		assert.NotEqual(t, uuid.Nil, task.Result.ID)
	})

	t.Run("pending task", func(t *testing.T) {
		task, err := service.Get(ctx, "task-pending")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "task-pending", string(task.ID))
		assert.Equal(t, payloads.Pending, task.Status)
	})

	t.Run("failed task", func(t *testing.T) {
		task, err := service.Get(ctx, "task-failure")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "task-failure", string(task.ID))
		assert.Equal(t, payloads.Failure, task.Status)
		assert.Equal(t, "Task failed", task.Message)
	})

	t.Run("non-existent task", func(t *testing.T) {
		task, err := service.Get(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Nil(t, task)
	})

	t.Run("with duplicate path", func(t *testing.T) {
		task, err := service.Get(ctx, "/rest/v0/tasks/task-success")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "task-success", string(task.ID))
	})
}

func TestAbort(t *testing.T) {
	server, service := setupTaskTestServer(t)
	defer server.Close()

	ctx := context.Background()

	err := service.Abort(ctx, "task-pending")
	assert.NoError(t, err)
}

func TestWait(t *testing.T) {
	server, service := setupTaskTestServer(t)
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
		task, err := service.Wait(ctx, "task-progress-start")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, payloads.Success, task.Status)
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
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
	server, service := setupTaskTestServer(t)
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

func TestList(t *testing.T) {
	server, service := setupTaskTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("list tasks with default limit", func(t *testing.T) {
		tasks, err := service.List(ctx, map[string]any{})
		assert.NoError(t, err)
		assert.NotNil(t, tasks)

		expectedCount := min(len(mockTasks), core.DefaultTaskListLimit)
		assert.Len(t, tasks, expectedCount)
	})

	t.Run("list tasks with custom limit", func(t *testing.T) {
		tasks, err := service.List(ctx, map[string]any{"limit": 2})
		assert.NoError(t, err)
		assert.NotNil(t, tasks)
		assert.Len(t, tasks, 2)
	})
}
