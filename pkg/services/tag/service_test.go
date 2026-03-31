package tag

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

const (
	testResourceID = "a1b2c3d4-1111-2222-3333-000000000001"
)

func newTestLogger(t *testing.T) *logger.Logger {
	t.Helper()
	log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return log
}

// setupTestServer creates an httptest.Server with modular handlers for Add and Remove tag endpoints.
func setupTestServer(t *testing.T, resourceType payloads.ResourceType) (*httptest.Server, library.TagService) {
	t.Helper()

	mux := http.NewServeMux()
	resourcePath := resourceType.Path()

	// PUT /rest/v0/{resource}/{id}/tags/{tag} - Add tag
	mux.HandleFunc(fmt.Sprintf("PUT /rest/v0/%s/{id}/tags/{tag}", resourcePath),
		func(w http.ResponseWriter, r *http.Request) {
			if _, err := uuid.FromString(r.PathValue("id")); err != nil {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

	// DELETE /rest/v0/{resource}/{id}/tags/{tag} - Remove tag
	mux.HandleFunc(fmt.Sprintf("DELETE /rest/v0/%s/{id}/tags/{tag}", resourcePath),
		func(w http.ResponseWriter, r *http.Request) {
			if _, err := uuid.FromString(r.PathValue("id")); err != nil {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	restClient := &client.Client{
		HttpClient: server.Client(),
		BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
		AuthToken:  "test-token",
	}

	return server, New(restClient, newTestLogger(t), resourceType)
}

// setupTestServerWithHandler creates an httptest.Server with a custom handler for focused unit tests.
func setupTestServerWithHandler(
	t *testing.T, handler http.HandlerFunc, resourceType payloads.ResourceType) (*httptest.Server, library.TagService) {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Failed to parse server URL: %v", err)
	}

	restClient := &client.Client{
		HttpClient: server.Client(),
		BaseURL:    baseURL,
		AuthToken:  "test-token",
	}

	return server, New(restClient, newTestLogger(t), resourceType)
}

func TestNew(t *testing.T) {
	log := newTestLogger(t)
	restClient := &client.Client{
		BaseURL:   &url.URL{Scheme: "http", Host: "localhost"},
		AuthToken: "test-token",
	}

	svc := New(restClient, log, payloads.ResourceTypeSR)

	assert.NotNil(t, svc)
}

func TestAdd(t *testing.T) {
	t.Run("returns error for empty tag", func(t *testing.T) {
		_, svc := setupTestServer(t, payloads.ResourceTypeSR)
		id := uuid.Must(uuid.FromString(testResourceID))

		err := svc.Add(context.Background(), id, "")

		assert.Error(t, err)
	})

	t.Run("successfully adds a tag", func(t *testing.T) {
		_, svc := setupTestServer(t, payloads.ResourceTypeSR)
		id := uuid.Must(uuid.FromString(testResourceID))

		err := svc.Add(context.Background(), id, "my-tag")

		assert.NoError(t, err)
	})

	t.Run("uses PUT method with correct resource path", func(t *testing.T) {
		var requestMethod, requestPath string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestMethod = r.Method
			requestPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		})
		_, svc := setupTestServerWithHandler(t, handler, payloads.ResourceTypeSR)
		id := uuid.Must(uuid.FromString(testResourceID))

		err := svc.Add(context.Background(), id, "my-tag")

		assert.NoError(t, err)
		assert.Equal(t, http.MethodPut, requestMethod)
		assert.Equal(t, fmt.Sprintf("/srs/%s/tags/my-tag", testResourceID), requestPath)
	})

	t.Run("uses resource type path for VM", func(t *testing.T) {
		var requestPath string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		})
		_, svc := setupTestServerWithHandler(t, handler, payloads.ResourceTypeVM)
		id := uuid.Must(uuid.FromString(testResourceID))

		err := svc.Add(context.Background(), id, "my-tag")

		assert.NoError(t, err)
		assert.Equal(t, fmt.Sprintf("/vms/%s/tags/my-tag", testResourceID), requestPath)
	})

	t.Run("returns error on HTTP failure", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})
		_, svc := setupTestServerWithHandler(t, handler, payloads.ResourceTypeSR)
		id := uuid.Must(uuid.FromString(testResourceID))

		err := svc.Add(context.Background(), id, "my-tag")

		assert.Error(t, err)
	})
}

func TestRemove(t *testing.T) {
	t.Run("returns error for empty tag", func(t *testing.T) {
		_, svc := setupTestServer(t, payloads.ResourceTypeSR)
		id := uuid.Must(uuid.FromString(testResourceID))

		err := svc.Remove(context.Background(), id, "")

		assert.Error(t, err)
	})

	t.Run("successfully removes a tag", func(t *testing.T) {
		_, svc := setupTestServer(t, payloads.ResourceTypeSR)
		id := uuid.Must(uuid.FromString(testResourceID))

		err := svc.Remove(context.Background(), id, "my-tag")

		assert.NoError(t, err)
	})

	t.Run("uses DELETE method with correct resource path", func(t *testing.T) {
		var requestMethod, requestPath string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestMethod = r.Method
			requestPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		})
		_, svc := setupTestServerWithHandler(t, handler, payloads.ResourceTypeSR)
		id := uuid.Must(uuid.FromString(testResourceID))

		err := svc.Remove(context.Background(), id, "my-tag")

		assert.NoError(t, err)
		assert.Equal(t, http.MethodDelete, requestMethod)
		assert.Equal(t, fmt.Sprintf("/srs/%s/tags/my-tag", testResourceID), requestPath)
	})

	t.Run("returns error on HTTP failure", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})
		_, svc := setupTestServerWithHandler(t, handler, payloads.ResourceTypeSR)
		id := uuid.Must(uuid.FromString(testResourceID))

		err := svc.Remove(context.Background(), id, "my-tag")

		assert.Error(t, err)
	})
}
