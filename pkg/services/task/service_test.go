package task

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

func setupTestServer(t *testing.T) (*httptest.Server, library.Task) {
	// Compile regex pattern once for efficiency
	taskURLRegex := regexp.MustCompile(`(?m)^/rest/v0/tasks/([^/]+)(?:/([^/]+))?$`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Use regex to extract taskID and optional action
		matches := taskURLRegex.FindStringSubmatch(r.URL.Path)
		if len(matches) == 0 {
			fmt.Printf("Unhandled path: %s %s\n", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		taskID := matches[1]
		action := ""
		if len(matches) > 2 {
			action = matches[2]
		}

		switch {
		case action == "" && r.Method == http.MethodGet:
			// Handle different task scenarios based on task ID
			var task payloads.Task
			switch taskID {
			case "success-task-123":
				task = payloads.Task{
					ID:     "success-task-123",
					Name:   "Test Success Task",
					Status: payloads.Success,
					Properties: payloads.Properties{
						Name: "vm.create",
					},
					Started:   payloads.APITime(time.Now().Add(-5 * time.Minute)),
					UpdatedAt: payloads.APITime(time.Now().Add(-1 * time.Minute)),
					EndedAt:   payloads.APITime(time.Now().Add(-1 * time.Minute)),
					Result: payloads.TaskResult{
						ID:       uuid.Must(uuid.FromString("361f2903-2c09-486e-9eff-91debeeee304")),
						StringID: "361f2903-2c09-486e-9eff-91debeeee304",
					},
				}
			case "failure-task-456":
				task = payloads.Task{
					ID:      "failure-task-456",
					Status:  payloads.Failure,
					Message: "VM not found",
				}
			case "running-task-789":
				task = payloads.Task{
					ID:     "running-task-789",
					Status: payloads.Running,
				}
			default:
				w.WriteHeader(http.StatusNotFound)
				return
			}

			err := json.NewEncoder(w).Encode(task)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		case action == "abort" && r.Method == http.MethodPost:
			switch taskID {
			case "abortable-task":
				err := json.NewEncoder(w).Encode(map[string]bool{"success": true})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			case "non-abortable-task":
				err := json.NewEncoder(w).Encode(map[string]bool{"success": false})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			default:
				w.WriteHeader(http.StatusNotFound)
			}

		default:
			fmt.Printf("Unhandled path: %s %s\n", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
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
	server, service := setupTestServer(t)
	defer server.Close()

	t.Run("successful task", func(t *testing.T) {
		task, err := service.Get(context.Background(), "success-task-123")

		assert.NoError(t, err)
		assert.Equal(t, "success-task-123", task.ID)
		assert.Equal(t, "Test Success Task", task.Name)
		assert.Equal(t, payloads.Success, task.Status)
		assert.Equal(t, "vm.create", task.Properties.Name)
		assert.Equal(t, uuid.Must(uuid.FromString("361f2903-2c09-486e-9eff-91debeeee304")), task.Result.ID)
	})

	t.Run("task with full path", func(t *testing.T) {
		task, err := service.Get(context.Background(), "/rest/v0/tasks/success-task-123")

		assert.NoError(t, err)
		assert.Equal(t, "success-task-123", task.ID)
		assert.Equal(t, "Test Success Task", task.Name)
		assert.Equal(t, payloads.Success, task.Status)
	})

	t.Run("invalid task ID format", func(t *testing.T) {
		_, err := service.Get(context.Background(), "")
		assert.Error(t, err)
	})

	t.Run("non-existent task", func(t *testing.T) {
		_, err := service.Get(context.Background(), "non-existent-task")

		assert.Error(t, err)
	})
}

func TestAbort(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	t.Run("successful abort", func(t *testing.T) {
		err := service.Abort(context.Background(), "abortable-task")

		assert.NoError(t, err)
	})

	t.Run("failed abort", func(t *testing.T) {
		err := service.Abort(context.Background(), "non-abortable-task")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to abort task")
	})

	t.Run("non-existent task", func(t *testing.T) {
		err := service.Abort(context.Background(), "non-existent-task")

		assert.Error(t, err)
	})
}

func TestWait(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	t.Run("wait for completed task", func(t *testing.T) {
		task, err := service.Wait(context.Background(), "success-task-123")

		assert.NoError(t, err)
		assert.Equal(t, "success-task-123", task.ID)
		assert.Equal(t, payloads.Success, task.Status)
	})

	t.Run("wait for failed task", func(t *testing.T) {
		task, err := service.Wait(context.Background(), "failure-task-456")

		assert.NoError(t, err)
		assert.Equal(t, "failure-task-456", task.ID)
		assert.Equal(t, payloads.Failure, task.Status)
		assert.Equal(t, "VM not found", task.Message)
	})

	t.Run("wait with context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := service.Wait(ctx, "running-task-789")

		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("wait with timeout", func(t *testing.T) {
		// Use WaitWithTimeout directly for a very short timeout
		taskService := service.(*Service)
		_, err := taskService.WaitWithTimeout(context.Background(), "running-task-789", 100*time.Millisecond)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})

	t.Run("context timeout during wait", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := service.Wait(ctx, "running-task-789")
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("wait for non-existent-task task", func(t *testing.T) {
		// Should be in timeout
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, err := service.Wait(ctx, "non-existent-task")

		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

func TestHandleTaskResponse(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	t.Run("handle task URL without waiting", func(t *testing.T) {
		task, isTask, err := service.HandleTaskResponse(context.Background(), "/rest/v0/tasks/success-task-123", false)

		assert.NoError(t, err)
		assert.True(t, isTask)
		assert.Equal(t, "success-task-123", task.ID)
		assert.Equal(t, payloads.Success, task.Status)
	})

	t.Run("handle task URL with waiting", func(t *testing.T) {
		task, isTask, err := service.HandleTaskResponse(context.Background(), "/rest/v0/tasks/success-task-123", true)

		assert.NoError(t, err)
		assert.True(t, isTask)
		assert.Equal(t, "success-task-123", task.ID)
		assert.Equal(t, payloads.Success, task.Status)
	})

	t.Run("handle non-task response", func(t *testing.T) {
		task, isTask, err := service.HandleTaskResponse(context.Background(), "some-other-response", false)

		assert.NoError(t, err)
		assert.False(t, isTask)
		assert.Nil(t, task)
	})

	t.Run("handle empty response", func(t *testing.T) {
		task, isTask, err := service.HandleTaskResponse(context.Background(), "", false)

		assert.NoError(t, err)
		assert.False(t, isTask)
		assert.Nil(t, task)
	})
}

func TestIsTaskURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid task URL", "/rest/v0/tasks/task-123", true},
		{"valid task URL with longer ID", "/rest/v0/tasks/very-long-task-id-12345", true},
		{"invalid URL - wrong prefix", "/api/v1/tasks/task-123", false},
		{"invalid URL - no task ID", "/rest/v0/tasks/", false},
		{"invalid URL - completely different", "/rest/v0/vms/vm-123", false},
		{"empty string", "", false},
		{"just task ID", "task-123", false},
		{"wrong version", "/rest/v1/tasks/task-123", false},
		{"missing leading slash", "rest/v0/tasks/task-123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTaskURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractTaskID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"valid task URL", "/rest/v0/tasks/task-123", "task-123"},
		{"valid task URL with UUID", "/rest/v0/tasks/550e8400-e29b-41d4-a716-446655440000",
			"550e8400-e29b-41d4-a716-446655440000"},
		{"URL without prefix", "task-123", "task-123"},
		{"empty string", "", ""},
		{"just prefix", "/rest/v0/tasks/", ""},
		{"nested path", "/rest/v0/tasks/123/456", "123/456"},
		{"special char", "/rest/v0/tasks/task-with-special-chars_123", "task-with-special-chars_123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractTaskID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanDuplicateV0Path(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	taskService := service.(*Service)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"full task path", "/rest/v0/tasks/task-123", "task-123"},
		{"task ID only", "task-123", "task-123"},
		{"path without leading slash", "rest/v0/tasks/task-123", "rest/v0/tasks/task-123"},
		{"empty string", "", ""},
		{"just task prefix", "/rest/v0/tasks/", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := taskService.cleanDuplicateV0Path(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}
