package tasker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

const (
	testResourceID = "a1b2c3d4-1111-2222-3333-000000000001"
	testToken      = "test-token"
)

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return log
}

// newTestClient creates a test client connected to the given server URL.
func newTestClient(t *testing.T, serverURL string) *client.Client {
	t.Helper()
	baseURL, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("Failed to parse server URL: %v", err)
	}
	baseURL.Path = "/rest/v0"
	return &client.Client{
		HttpClient: http.DefaultClient,
		BaseURL:    baseURL,
		AuthToken:  testToken,
	}
}

// setupTestServer creates an httptest.Server with a standard handler for GetTasks endpoint.
func setupTestServer(t *testing.T, resourceType payloads.ResourceType) (*httptest.Server, *client.Client) {
	t.Helper()

	mux := http.NewServeMux()
	resourcePath := resourceType.Path()

	mux.HandleFunc(fmt.Sprintf("GET /rest/v0/%s/{id}/tasks", resourcePath),
		func(w http.ResponseWriter, r *http.Request) {
			if _, err := uuid.FromString(r.PathValue("id")); err != nil {
				http.NotFound(w, r)
				return
			}
			tasks := []*payloads.Task{
				{ID: "task1", Status: "success"},
				{ID: "task2", Status: "failure"},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tasks)
		})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return server, newTestClient(t, server.URL)
}

// setupTestServerWithHandler creates an httptest.Server with a custom handler for focused unit tests.
func setupTestServerWithHandler(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *client.Client) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	return server, newTestClient(t, server.URL)
}

func TestGetTasks(t *testing.T) {
	t.Run("successfully gets tasks", func(t *testing.T) {
		_, c := setupTestServer(t, payloads.ResourceTypeSR)
		log := newTestLogger(t)
		id := uuid.Must(uuid.FromString(testResourceID))

		tasks, err := GetTasks(context.Background(), c, log, payloads.ResourceTypeSR, id, 0, "")

		assert.NoError(t, err)
		assert.Len(t, tasks, 2)
		assert.Equal(t, "task1", tasks[0].ID)
		assert.Equal(t, "task2", tasks[1].ID)
	})

	t.Run("uses resource type path for different resources", func(t *testing.T) {
		testCases := []struct {
			resourceType payloads.ResourceType
			expectedPath string
		}{
			{payloads.ResourceTypeVDI, "/vdis"},
			{payloads.ResourceTypeVBD, "/vbds"},
			{payloads.ResourceTypeNetwork, "/networks"},
			{payloads.ResourceTypeSR, "/srs"},
		}

		for _, tc := range testCases {
			t.Run(string(tc.resourceType), func(t *testing.T) {
				var requestPath string
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requestPath = r.URL.Path
					w.Header().Set("Content-Type", "application/json")
					_ = json.NewEncoder(w).Encode([]*payloads.Task{})
				})
				_, c := setupTestServerWithHandler(t, handler)
				log := newTestLogger(t)
				id := uuid.Must(uuid.FromString(testResourceID))

				_, err := GetTasks(context.Background(), c, log, tc.resourceType, id, 0, "")

				assert.NoError(t, err)
				assert.Equal(t, fmt.Sprintf("/rest/v0%s/%s/tasks", tc.expectedPath, testResourceID), requestPath)
			})
		}
	})

	t.Run("includes limit parameter when provided", func(t *testing.T) {
		var queryParams url.Values
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			queryParams = r.URL.Query()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]*payloads.Task{})
		})
		_, c := setupTestServerWithHandler(t, handler)
		log := newTestLogger(t)
		id := uuid.Must(uuid.FromString(testResourceID))

		_, err := GetTasks(context.Background(), c, log, payloads.ResourceTypeSR, id, 10, "")

		assert.NoError(t, err)
		assert.Equal(t, "10", queryParams.Get("limit"))
		assert.Equal(t, "*", queryParams.Get("fields"))
	})

	t.Run("includes filter parameter when provided", func(t *testing.T) {
		var queryParams url.Values
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			queryParams = r.URL.Query()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]*payloads.Task{})
		})
		_, c := setupTestServerWithHandler(t, handler)
		log := newTestLogger(t)
		id := uuid.Must(uuid.FromString(testResourceID))

		_, err := GetTasks(context.Background(), c, log, payloads.ResourceTypeSR, id, 0, "status:success")

		assert.NoError(t, err)
		assert.Equal(t, "status:success", queryParams.Get("filter"))
		assert.Equal(t, "*", queryParams.Get("fields"))
	})

	t.Run("includes both limit and filter when provided", func(t *testing.T) {
		var queryParams url.Values
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			queryParams = r.URL.Query()
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]*payloads.Task{})
		})
		_, c := setupTestServerWithHandler(t, handler)
		log := newTestLogger(t)
		id := uuid.Must(uuid.FromString(testResourceID))

		_, err := GetTasks(context.Background(), c, log, payloads.ResourceTypeSR, id, 5, "status:failure")

		assert.NoError(t, err)
		assert.Equal(t, "5", queryParams.Get("limit"))
		assert.Equal(t, "status:failure", queryParams.Get("filter"))
		assert.Equal(t, "*", queryParams.Get("fields"))
	})

	t.Run("returns empty list when no tasks found", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]*payloads.Task{})
		})
		_, c := setupTestServerWithHandler(t, handler)
		log := newTestLogger(t)
		id := uuid.Must(uuid.FromString(testResourceID))

		tasks, err := GetTasks(context.Background(), c, log, payloads.ResourceTypeSR, id, 0, "")

		assert.NoError(t, err)
		assert.Empty(t, tasks)
	})

	t.Run("returns error on HTTP failure", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})
		_, c := setupTestServerWithHandler(t, handler)
		log := newTestLogger(t)
		id := uuid.Must(uuid.FromString(testResourceID))

		tasks, err := GetTasks(context.Background(), c, log, payloads.ResourceTypeSR, id, 0, "")

		assert.Error(t, err)
		assert.Nil(t, tasks)
	})

	t.Run("returns error for invalid resource ID", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
		_, c := setupTestServerWithHandler(t, handler)
		log := newTestLogger(t)
		id := uuid.Must(uuid.FromString(testResourceID))

		tasks, err := GetTasks(context.Background(), c, log, payloads.ResourceTypeSR, id, 0, "")

		assert.Error(t, err)
		assert.Nil(t, tasks)
	})
}
