package sr

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
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	mock "github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library/mock"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/mock/gomock"
)

const (
	testSRID1        = "a1b2c3d4-1111-2222-3333-000000000001"
	testSRID2        = "a1b2c3d4-1111-2222-3333-000000000002"
	testSRIDNotFound = "d44e5f60-4567-89ab-def0-444455556666"
	testPoolID       = "b2c3d4e5-0000-0000-0000-000000000001"
	testFakeTaskID   = "task-abc"
	testTokenValue   = "test-token"
)

var mockSRs = func() []*payloads.StorageRepository {
	return []*payloads.StorageRepository{
		{
			ID:                uuid.Must(uuid.FromString(testSRID1)),
			UUID:              uuid.Must(uuid.FromString(testSRID1)),
			Type:              payloads.ResourceTypeSR,
			Pool:              uuid.Must(uuid.FromString(testPoolID)),
			NameLabel:         "Local storage",
			NameDescription:   "Local storage on host",
			SRType:            "lvm",
			Shared:            false,
			Size:              1073741824,
			Usage:             536870912,
			PhysicalUsage:     536870912,
			InMaintenanceMode: false,
			ContentType:       "user",
			Tags:              []string{},
		},
		{
			ID:                uuid.Must(uuid.FromString(testSRID2)),
			UUID:              uuid.Must(uuid.FromString(testSRID2)),
			Type:              payloads.ResourceTypeSR,
			Pool:              uuid.Must(uuid.FromString(testPoolID)),
			NameLabel:         "NFS share",
			NameDescription:   "Shared NFS storage",
			SRType:            "nfs",
			Shared:            true,
			Size:              2147483648,
			Usage:             1073741824,
			PhysicalUsage:     1073741824,
			InMaintenanceMode: false,
			ContentType:       "user",
			Tags:              []string{},
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
		AuthToken:  testTokenValue,
	}

	ctrl := gomock.NewController(t)
	mockTask := mock.NewMockTask(ctrl)

	return New(restClient, mockTask, log).(*Service), server, mockTask
}

func setupTestServer(t *testing.T) (*httptest.Server, *Service, *mock.MockTask) {
	t.Helper()
	mux := http.NewServeMux()

	// GET /rest/v0/srs - List all SRs
	mux.HandleFunc("GET /rest/v0/srs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockSRs()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// GET /rest/v0/srs/{id} - Get specific SR
	mux.HandleFunc("GET /rest/v0/srs/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		idStr := r.PathValue("id")

		srID, err := uuid.FromString(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var sr *payloads.StorageRepository
		switch srID.String() {
		case testSRID1:
			sr = mockSRs()[0]
		case testSRID2:
			sr = mockSRs()[1]
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(sr); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// POST /rest/v0/srs/{id}/actions/{action} - ReclaimSpace/Scan
	mux.HandleFunc("POST /rest/v0/srs/{id}/actions/{action}", func(w http.ResponseWriter, r *http.Request) {
		if _, err := uuid.FromString(r.PathValue("id")); err != nil {
			http.NotFound(w, r)
			return
		}
		switch r.PathValue("action") {
		case "reclaim_space", "scan":
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(payloads.TaskIDResponse{TaskID: testFakeTaskID}); err != nil {
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
		AuthToken:  testTokenValue,
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
		Token: testTokenValue,
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

	t.Run("get existing SR by ID", func(t *testing.T) {
		srID := uuid.Must(uuid.FromString(testSRID1))

		result, err := svc.Get(t.Context(), srID)

		assert.NoError(t, err)
		require.NotNil(t, result)
		sr := mockSRs()[0]
		assert.Equal(t, srID, result.UUID)
		assert.Equal(t, sr.NameLabel, result.NameLabel)
		assert.Equal(t, sr.SRType, result.SRType)
		assert.Equal(t, sr.Shared, result.Shared)
	})

	t.Run("get non-existent SR by ID", func(t *testing.T) {
		srID := uuid.Must(uuid.FromString(testSRIDNotFound))

		result, err := svc.Get(t.Context(), srID)

		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestGetAll(t *testing.T) {
	t.Run("passes limit and filter parameters", func(t *testing.T) {
		limit := 42
		filter := "shared?"
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			assert.Equal(t, http.MethodGet, r.Method)
			values := r.URL.Query()
			assert.Equal(t, fmt.Sprintf("%d", limit), values.Get("limit"))
			assert.Equal(t, filter, values.Get("filter"))
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]*payloads.StorageRepository{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
		service, server, _ := setupTestServerWithHandler(t, handler)
		defer server.Close()

		srs, err := service.GetAll(context.Background(), limit, filter)

		assert.NoError(t, err)
		assert.NotNil(t, srs)
		assert.True(t, called)
	})

	t.Run("does not send limit param when zero", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			values := r.URL.Query()
			assert.Empty(t, values.Get("limit"))
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]*payloads.StorageRepository{}); err != nil {
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

		srs, err := service.GetAll(context.Background(), 0, "")

		assert.Error(t, err)
		assert.Nil(t, srs)
	})

	t.Run("successfully retrieves all SRs", func(t *testing.T) {
		server, service, _ := setupTestServer(t)
		defer server.Close()

		srs, err := service.GetAll(context.Background(), 0, "")

		assert.NoError(t, err)
		require.NotNil(t, srs)
		assert.Len(t, srs, 2)
		assert.Equal(t, uuid.Must(uuid.FromString(testSRID1)), srs[0].UUID)
		assert.Equal(t, uuid.Must(uuid.FromString(testSRID2)), srs[1].UUID)
	})
}

func TestReclaimSpace(t *testing.T) {
	srID := uuid.Must(uuid.FromString(testSRID1))

	t.Run("successfully reclaims space on an SR", func(t *testing.T) {
		server, svc, mockTask := setupTestServer(t)
		defer server.Close()

		mockTask.EXPECT().
			HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: testFakeTaskID}, false).
			Return(&payloads.Task{ID: testFakeTaskID}, nil)

		taskID, err := svc.ReclaimSpace(t.Context(), srID)

		assert.NoError(t, err)
		assert.Equal(t, testFakeTaskID, taskID)
	})

	t.Run("returns error on http error", func(t *testing.T) {
		svc, server, _ := setupTestServerWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})
		defer server.Close()

		taskID, err := svc.ReclaimSpace(t.Context(), srID)

		assert.Error(t, err)
		assert.Empty(t, taskID)
	})

	t.Run("returns error when task handling fails", func(t *testing.T) {
		server, svc, mockTask := setupTestServer(t)
		defer server.Close()

		mockTask.EXPECT().
			HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: testFakeTaskID}, false).
			Return(nil, fmt.Errorf("task failed"))

		taskID, err := svc.ReclaimSpace(t.Context(), srID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SR reclaim_space failed")
		assert.Empty(t, taskID)
	})
}

func TestScan(t *testing.T) {
	srID := uuid.Must(uuid.FromString(testSRID1))

	t.Run("successfully scans an SR", func(t *testing.T) {
		server, svc, mockTask := setupTestServer(t)
		defer server.Close()

		mockTask.EXPECT().
			HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: testFakeTaskID}, false).
			Return(&payloads.Task{ID: testFakeTaskID}, nil)

		taskID, err := svc.Scan(t.Context(), srID)

		assert.NoError(t, err)
		assert.Equal(t, testFakeTaskID, taskID)
	})

	t.Run("returns error on http error", func(t *testing.T) {
		svc, server, _ := setupTestServerWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})
		defer server.Close()

		taskID, err := svc.Scan(t.Context(), srID)

		assert.Error(t, err)
		assert.Empty(t, taskID)
	})

	t.Run("returns error when task handling fails", func(t *testing.T) {
		server, svc, mockTask := setupTestServer(t)
		defer server.Close()

		mockTask.EXPECT().
			HandleTaskResponse(gomock.Any(), payloads.TaskIDResponse{TaskID: testFakeTaskID}, false).
			Return(nil, fmt.Errorf("task failed"))

		taskID, err := svc.Scan(t.Context(), srID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "SR scan failed")
		assert.Empty(t, taskID)
	})
}
