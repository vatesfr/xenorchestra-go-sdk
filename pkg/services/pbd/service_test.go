package pbd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	mock "github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library/mock"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

const (
	testHostID        = "a1b2c3d4-0000-0000-0000-000000000010"
	testSRID          = "a1b2c3d4-0000-0000-0000-000000000020"
	testPBDID1        = "b22c3d4e-2345-6789-bcde-222233334444"
	testPBDID2        = "c33d4e5f-3456-789a-cdef-333344445555"
	testPBDIDNotFound = "d44e5f60-4567-89ab-def0-444455556666"
)

var mockPBDs = func() []*payloads.PBD {
	return []*payloads.PBD{
		{
			UUID:         uuid.Must(uuid.FromString(testPBDID1)),
			Type:         payloads.ResourceTypePBD,
			Attached:     true,
			Host:         uuid.Must(uuid.FromString(testHostID)),
			SR:           uuid.Must(uuid.FromString(testSRID)),
			DeviceConfig: map[string]string{"device": "/dev/sda"},
			OtherConfig:  map[string]string{},
		},
		{
			UUID:         uuid.Must(uuid.FromString(testPBDID2)),
			Type:         payloads.ResourceTypePBD,
			Attached:     false,
			Host:         uuid.Must(uuid.FromString(testHostID)),
			SR:           uuid.Must(uuid.FromString(testSRID)),
			DeviceConfig: map[string]string{"server": "nfs-host", "serverpath": "/export"},
			OtherConfig:  map[string]string{},
		},
	}
}

func setupTestServerWithHandler(t *testing.T, handler http.HandlerFunc) (*Service, *httptest.Server, *mock.MockTask) {
	t.Helper()
	server := httptest.NewServer(handler)

	log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	baseURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("Failed to parse server URL: %v", err)
	}

	restClient := &client.Client{
		HttpClient: server.Client(),
		BaseURL:    baseURL,
		AuthToken:  "test-token",
	}

	ctrl := gomock.NewController(t)
	mockTask := mock.NewMockTask(ctrl)

	return New(restClient, mockTask, log).(*Service), server, mockTask
}

func setupTestServer(t *testing.T) (*httptest.Server, *Service, *mock.MockTask) {
	t.Helper()
	mux := http.NewServeMux()

	// GET /rest/v0/pbds - List all PBDs
	mux.HandleFunc("GET /rest/v0/pbds", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockPBDs()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// GET /rest/v0/pbds/{id} - Get specific PBD
	mux.HandleFunc("GET /rest/v0/pbds/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		idStr := r.PathValue("id")

		pbdID, err := uuid.FromString(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var pbd *payloads.PBD
		switch pbdID.String() {
		case testPBDID1:
			pbd = mockPBDs()[0]
		case testPBDID2:
			pbd = mockPBDs()[1]
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(pbd); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// POST /rest/v0/pbds/{id}/actions/{action} - Plug/Unplug a PBD
	mux.HandleFunc("POST /rest/v0/pbds/{id}/actions/{action}", func(w http.ResponseWriter, r *http.Request) {
		if _, err := uuid.FromString(r.PathValue("id")); err != nil {
			http.NotFound(w, r)
			return
		}
		switch r.PathValue("action") {
		case "plug", "unplug":
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(payloads.TaskIDResponse{TaskID: "task-abc"}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		default:
			http.NotFound(w, r)
		}
	})

	server := httptest.NewServer(mux)

	restClient := &client.Client{
		HttpClient: server.Client(),
		BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
		AuthToken:  "test-token",
	}

	log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	ctrl := gomock.NewController(t)
	mockTask := mock.NewMockTask(ctrl)
	return server, New(restClient, mockTask, log).(*Service), mockTask
}

func TestNew(t *testing.T) {
	cfg := &config.Config{
		Url:   "http://localhost",
		Token: "test-token",
	}
	c, err := client.New(cfg)
	assert.NoError(t, err)

	log, _ := logger.New(true, nil, nil)
	ctrl := gomock.NewController(t)
	mockTask := mock.NewMockTask(ctrl)
	svc := New(c, mockTask, log)

	assert.NotNil(t, svc)
}

func TestGet(t *testing.T) {
	server, svc, _ := setupTestServer(t)
	defer server.Close()

	t.Run("get existing PBD by ID", func(t *testing.T) {
		pbdID := uuid.Must(uuid.FromString(testPBDID1))

		result, err := svc.Get(t.Context(), pbdID)

		assert.NoError(t, err)
		require.NotNil(t, result)
		pbd := mockPBDs()[0]
		assert.Equal(t, pbdID, result.UUID)
		assert.Equal(t, pbd.Attached, result.Attached)
		assert.Equal(t, pbd.Host, result.Host)
		assert.Equal(t, pbd.SR, result.SR)
		assert.Equal(t, pbd.DeviceConfig, result.DeviceConfig)
	})

	t.Run("get non-existent PBD by ID", func(t *testing.T) {
		pbdID := uuid.Must(uuid.FromString(testPBDIDNotFound))

		result, err := svc.Get(t.Context(), pbdID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestGetAll(t *testing.T) {
	t.Run("passes limit and filter parameters", func(t *testing.T) {
		limit := 42
		filter := "attached?"
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			assert.Equal(t, http.MethodGet, r.Method)
			values := r.URL.Query()
			assert.Equal(t, fmt.Sprintf("%d", limit), values.Get("limit"))
			assert.Equal(t, filter, values.Get("filter"))
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]*payloads.PBD{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
		service, server, _ := setupTestServerWithHandler(t, handler)
		defer server.Close()

		pbds, err := service.GetAll(context.Background(), limit, filter)

		assert.NoError(t, err)
		assert.NotNil(t, pbds)
		assert.True(t, called)
	})

	t.Run("does not send limit param when zero", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			values := r.URL.Query()
			assert.Empty(t, values.Get("limit"))
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]*payloads.PBD{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
		service, server, _ := setupTestServerWithHandler(t, handler)
		defer server.Close()

		_, err := service.GetAll(context.Background(), 0, "")
		assert.NoError(t, err)
	})

	t.Run("returns error on http error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})
		service, server, _ := setupTestServerWithHandler(t, handler)
		defer server.Close()

		pbds, err := service.GetAll(context.Background(), 0, "")

		assert.Error(t, err)
		assert.Nil(t, pbds)
	})

	t.Run("successfully retrieves all PBDs", func(t *testing.T) {
		server, service, _ := setupTestServer(t)
		defer server.Close()

		pbds, err := service.GetAll(context.Background(), 0, "")

		assert.NoError(t, err)
		require.NotNil(t, pbds)
		assert.Len(t, pbds, 2)
		assert.Equal(t, uuid.Must(uuid.FromString(testPBDID1)), pbds[0].UUID)
		assert.Equal(t, uuid.Must(uuid.FromString(testPBDID2)), pbds[1].UUID)
	})
}

func TestPlug(t *testing.T) {
	pbdID := uuid.Must(uuid.FromString(testPBDID1))

	t.Run("successfully plugs a PBD", func(t *testing.T) {
		server, svc, mockTask := setupTestServer(t)
		defer server.Close()

		mockTask.EXPECT().
			HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: "task-abc"}, false).
			Return(&payloads.Task{ID: "task-abc"}, nil)

		taskID, err := svc.Plug(t.Context(), pbdID)

		assert.NoError(t, err)
		assert.Equal(t, "task-abc", taskID)
	})

	t.Run("returns error on http error", func(t *testing.T) {
		svc, server, _ := setupTestServerWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})
		defer server.Close()

		taskID, err := svc.Plug(t.Context(), pbdID)

		assert.Error(t, err)
		assert.Empty(t, taskID)
	})

	t.Run("returns error when task handling fails", func(t *testing.T) {
		server, svc, mockTask := setupTestServer(t)
		defer server.Close()

		mockTask.EXPECT().
			HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: "task-abc"}, false).
			Return(nil, fmt.Errorf("task failed"))

		taskID, err := svc.Plug(t.Context(), pbdID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PBD plug failed")
		assert.Empty(t, taskID)
	})
}

func TestUnplug(t *testing.T) {
	pbdID := uuid.Must(uuid.FromString(testPBDID1))

	t.Run("successfully unplugs a PBD", func(t *testing.T) {
		server, svc, mockTask := setupTestServer(t)
		defer server.Close()

		mockTask.EXPECT().
			HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: "task-abc"}, false).
			Return(&payloads.Task{ID: "task-abc"}, nil)

		taskID, err := svc.Unplug(t.Context(), pbdID)

		assert.NoError(t, err)
		assert.Equal(t, "task-abc", taskID)
	})

	t.Run("returns error on http error", func(t *testing.T) {
		svc, server, _ := setupTestServerWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})
		defer server.Close()

		taskID, err := svc.Unplug(t.Context(), pbdID)

		assert.Error(t, err)
		assert.Empty(t, taskID)
	})

	t.Run("returns error when task handling fails", func(t *testing.T) {
		server, svc, mockTask := setupTestServer(t)
		defer server.Close()

		mockTask.EXPECT().
			HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: "task-abc"}, false).
			Return(nil, fmt.Errorf("task failed"))

		taskID, err := svc.Unplug(t.Context(), pbdID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "PBD unplug failed")
		assert.Empty(t, taskID)
	})
}
