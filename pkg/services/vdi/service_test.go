package vdi

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
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

var mockVDIs = func() []*payloads.VDI {
	return []*payloads.VDI{
		{
			UUID:            uuid.Must(uuid.FromString(testVDIID1)),
			Type:            "VDI",
			NameLabel:       "VDI 1",
			NameDescription: "Test VDI 1",
			VDIType:         "user",
		},
		{
			UUID:            uuid.Must(uuid.FromString(testVDIID2)),
			Type:            "VDI",
			NameLabel:       "VDI 2",
			NameDescription: "Test VDI 2",
			VDIType:         "user",
			Size:            8589934592,
			Usage:           17152,
		},
	}
}

const (
	testVDIID1        = "c77f9955-c1d2-4b39-aa1c-73cdb2dacb7e"
	testVDIID2        = "d88fa066-d2e3-5c4a-bc2d-84deb3eadcbf"
	testVDIIDNotFound = "e99fb177-e3f4-6d5b-cd3e-95efc4fbedc0"
)

func setupTestServerWithHandler(t *testing.T, handler http.HandlerFunc) (*Service, *httptest.Server) {
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

	return New(restClient, log).(*Service), server
}

func setupTestServer(t *testing.T) (*httptest.Server, *Service) {
	mux := http.NewServeMux()

	// GET /rest/v0/vdis - List all VDIs
	// No limit, no filter handling for simplicity, just return based on ID
	// limit and filter handling is tested in TestGetAll with a custom handler
	mux.HandleFunc("GET /rest/v0/vdis", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vdis := mockVDIs()
		err := json.NewEncoder(w).Encode(vdis)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// GET /rest/v0/vdis/{id} - Get specific VDI
	mux.HandleFunc("GET /rest/v0/vdis/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		idStr := r.PathValue("id")
		vdiID, err := uuid.FromString(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var vdi *payloads.VDI
		switch vdiID.String() {
		case testVDIID1:
			vdi = mockVDIs()[0]
		case testVDIID2:
			vdi = mockVDIs()[1]
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}
		err = json.NewEncoder(w).Encode(vdi)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	server := httptest.NewServer(mux)

	restClient := &client.Client{
		HttpClient: http.DefaultClient,
		BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
		AuthToken:  "test-token",
	}

	log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	return server, New(restClient, log).(*Service)
}

// This is a basic test to ensure the service can be instantiated.
func TestNew(t *testing.T) {
	cfg := &config.Config{
		Url:   "http://localhost",
		Token: "test-token",
	}
	c, err := client.New(cfg)
	assert.NoError(t, err)

	log, _ := logger.New(true, nil, nil)
	svc := New(c, log)

	assert.NotNil(t, svc)
}

func TestVDIService_Get_ConnectionError(t *testing.T) {
	// This test mainly checks that the code compiles and imports are correct
	cfg := &config.Config{
		Url:   "http://localhost",
		Token: "test-token",
	}
	c, err := client.New(cfg)
	assert.NoError(t, err)

	log, _ := logger.New(true, nil, nil)
	svc := New(c, log)

	_, err = svc.Get(t.Context(), uuid.Nil)
	// Since we don't have a real server, we expect an error or it to try to connect
	// In this unit test setup without mocks, we mainly ensure type safety
	assert.Error(t, err)
}

func TestGet(t *testing.T) {
	server, svc := setupTestServer(t)
	defer server.Close()

	t.Run("get existing VDI by ID", func(t *testing.T) {

		vdiID := uuid.Must(uuid.FromString(testVDIID1))

		result, err := svc.Get(t.Context(), vdiID)

		assert.NoError(t, err)
		require.NotNil(t, result)
		vdi := mockVDIs()[0]
		assert.Equal(t, vdiID, result.UUID)
		assert.Equal(t, vdi.NameLabel, result.NameLabel)
		assert.Equal(t, vdi.VDIType, result.VDIType)
		assert.Equal(t, vdi.Size, result.Size)
	})

	t.Run("get unexisting VDI by ID", func(t *testing.T) {

		vdiID := uuid.Must(uuid.FromString(testVDIIDNotFound))
		result, err := svc.Get(t.Context(), vdiID)
		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("get VDI by nil ID", func(t *testing.T) {
		result, err := svc.Get(t.Context(), uuid.Nil)
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestGetAll(t *testing.T) {
	t.Run("passes limit and filter parameters", func(t *testing.T) {
		limit := 42
		filter := "filter-to-check"
		called := false
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			assert.Equal(t, http.MethodGet, r.Method)
			values := r.URL.Query()
			assert.Equal(t, fmt.Sprintf("%d", limit), values.Get("limit"))
			assert.Equal(t, filter, values.Get("filter"))
			w.Header().Set("Content-Type", "application/json")
			err := json.NewEncoder(w).Encode([]*payloads.VDI{})
			assert.NoError(t, err)
		})
		service, server := setupTestServerWithHandler(t, handler)
		defer server.Close()
		vdis, err := service.GetAll(context.Background(), limit, filter)
		assert.NoError(t, err)
		assert.NotNil(t, vdis)
		assert.True(t, called)
	})

	t.Run("returns error on http error", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not found", http.StatusNotFound)
		})
		service, server := setupTestServerWithHandler(t, handler)
		defer server.Close()
		vdis, err := service.GetAll(context.Background(), 0, "")
		assert.Error(t, err)
		assert.Nil(t, vdis)
	})

	t.Run("returns error on invalid json", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte("not a json"))
			assert.NoError(t, err)
		})
		service, server := setupTestServerWithHandler(t, handler)
		defer server.Close()
		vdis, err := service.GetAll(context.Background(), 0, "")
		assert.Error(t, err)
		assert.Nil(t, vdis)
	})

	t.Run("successfully retrieves all VDIs", func(t *testing.T) {
		server, service := setupTestServer(t)
		defer server.Close()

		vdis, err := service.GetAll(context.Background(), 0, "")
		assert.NoError(t, err)
		assert.NotNil(t, vdis)
		assert.Len(t, vdis, 2)
		assert.Equal(t, "VDI 1", vdis[0].NameLabel)
		assert.Equal(t, "VDI 2", vdis[1].NameLabel)
	})
}
