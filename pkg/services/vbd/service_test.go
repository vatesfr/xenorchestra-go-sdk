package vbd

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
	testVMID          = "a1b2c3d4-0000-0000-0000-000000000001"
	testVDIID         = "a1b2c3d4-0000-0000-0000-000000000002"
	testVBDID         = "a1b2c3d4-0000-0000-0000-000000000003"
	testVBDID1        = "b22c3d4e-2345-6789-bcde-222233334444"
	testVBDID2        = "c33d4e5f-3456-789a-cdef-333344445555"
	testVBDIDNotFound = "d44e5f60-4567-89ab-def0-444455556666"
)

var mockVBDs = func() []*payloads.VBD {
	device1 := "xvda"
	return []*payloads.VBD{
		{
			UUID:      uuid.Must(uuid.FromString(testVBDID1)),
			Type:      "VBD",
			Attached:  true,
			Bootable:  true,
			Device:    &device1,
			IsCDDrive: false,
			Position:  "0",
			ReadOnly:  false,
			VM:        uuid.Must(uuid.FromString(testVMID)),
		},
		{
			UUID:      uuid.Must(uuid.FromString(testVBDID2)),
			Type:      "VBD",
			Attached:  false,
			Bootable:  false,
			Device:    nil,
			IsCDDrive: false,
			Position:  "1",
			ReadOnly:  true,
			VM:        uuid.Must(uuid.FromString(testVMID)),
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

	// GET /rest/v0/vbds - List all VBDs
	// limit and filter handling is tested in TestGetAll with a custom handler
	mux.HandleFunc("GET /rest/v0/vbds", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(mockVBDs()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// DELETE /rest/v0/vbds/{id} - Delete a VBD
	mux.HandleFunc("DELETE /rest/v0/vbds/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		switch idStr {
		case testVBDID1, testVBDID2:
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	// GET /rest/v0/vbds/{id} - Get specific VBD
	mux.HandleFunc("GET /rest/v0/vbds/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		idStr := r.PathValue("id")

		vbdID, err := uuid.FromString(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var vbd *payloads.VBD
		switch vbdID.String() {
		case testVBDID1:
			vbd = mockVBDs()[0]
		case testVBDID2:
			vbd = mockVBDs()[1]
		default:
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(vbd); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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

func TestCreate(t *testing.T) {
	vmID := uuid.Must(uuid.FromString(testVMID))
	vdiID := uuid.Must(uuid.FromString(testVDIID))
	createdID := uuid.Must(uuid.FromString(testVBDID))

	// successHandler returns createdID and captures the request body.
	makeSuccessHandler := func(capturedBody *map[string]json.RawMessage) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if err := json.NewDecoder(r.Body).Decode(capturedBody); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(struct {
				ID string `json:"id"`
			}{ID: createdID.String()}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		}
	}

	t.Run("returns error when params is nil", func(t *testing.T) {
		svc, server, _ := setupTestServerWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		})
		defer server.Close()

		id, err := svc.Create(t.Context(), nil)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "params cannot be nil")
		assert.Equal(t, uuid.Nil, id)
	})

	t.Run("returns error when VM ID is nil", func(t *testing.T) {
		svc, server, _ := setupTestServerWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		})
		defer server.Close()

		id, err := svc.Create(t.Context(), &payloads.CreateVBDParams{
			VM:  uuid.Nil,
			VDI: vdiID,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "VM ID cannot be empty")
		assert.Equal(t, uuid.Nil, id)
	})

	t.Run("returns error when VDI ID is nil", func(t *testing.T) {
		svc, server, _ := setupTestServerWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
			t.Error("handler should not be called")
		})
		defer server.Close()

		id, err := svc.Create(t.Context(), &payloads.CreateVBDParams{
			VM:  vmID,
			VDI: uuid.Nil,
		})

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "VDI ID cannot be empty")
		assert.Equal(t, uuid.Nil, id)
	})

	t.Run("returns error on http error", func(t *testing.T) {
		svc, server, _ := setupTestServerWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})
		defer server.Close()

		id, err := svc.Create(t.Context(), &payloads.CreateVBDParams{VM: vmID, VDI: vdiID})

		assert.Error(t, err)
		assert.Equal(t, uuid.Nil, id)
	})

	t.Run("only VM and VDI sent when no optional fields set", func(t *testing.T) {
		var capturedBody map[string]json.RawMessage
		svc, server, _ := setupTestServerWithHandler(t, makeSuccessHandler(&capturedBody))
		defer server.Close()

		id, err := svc.Create(t.Context(), &payloads.CreateVBDParams{VM: vmID, VDI: vdiID})

		require.NoError(t, err)
		assert.Equal(t, createdID, id)

		// Only "VM" and "VDI" must be present — zero-value optional fields are omitted.
		assert.Len(t, capturedBody, 2, "expected exactly 2 fields in request body, got: %v", capturedBody)
		var gotVM, gotVDI uuid.UUID
		require.NoError(t, json.Unmarshal(capturedBody["VM"], &gotVM))
		require.NoError(t, json.Unmarshal(capturedBody["VDI"], &gotVDI))
		assert.Equal(t, vmID, gotVM)
		assert.Equal(t, vdiID, gotVDI)
	})

	t.Run("optional fields forwarded to server", func(t *testing.T) {
		var capturedBody map[string]json.RawMessage
		svc, server, _ := setupTestServerWithHandler(t, makeSuccessHandler(&capturedBody))
		defer server.Close()

		unpluggable := true
		id, err := svc.Create(t.Context(), &payloads.CreateVBDParams{
			VM:          vmID,
			VDI:         vdiID,
			Type:        payloads.VBDTypeDisk,
			Mode:        payloads.VBDModeRW,
			Bootable:    true,
			Userdevice:  "0",
			Unpluggable: &unpluggable,
		})

		require.NoError(t, err)
		assert.Equal(t, createdID, id)

		var gotType string
		require.NoError(t, json.Unmarshal(capturedBody["type"], &gotType))
		assert.Equal(t, string(payloads.VBDTypeDisk), gotType)

		var gotMode string
		require.NoError(t, json.Unmarshal(capturedBody["mode"], &gotMode))
		assert.Equal(t, string(payloads.VBDModeRW), gotMode)

		var gotBootable bool
		require.NoError(t, json.Unmarshal(capturedBody["bootable"], &gotBootable))
		assert.True(t, gotBootable)

		var gotUserdevice string
		require.NoError(t, json.Unmarshal(capturedBody["userdevice"], &gotUserdevice))
		assert.Equal(t, "0", gotUserdevice)

		var gotUnpluggable bool
		require.NoError(t, json.Unmarshal(capturedBody["unpluggable"], &gotUnpluggable))
		assert.True(t, gotUnpluggable)
	})
}

func TestGet(t *testing.T) {
	server, svc, _ := setupTestServer(t)
	defer server.Close()

	t.Run("get existing VBD by ID", func(t *testing.T) {
		vbdID := uuid.Must(uuid.FromString(testVBDID1))

		result, err := svc.Get(t.Context(), vbdID)

		assert.NoError(t, err)
		require.NotNil(t, result)
		vbd := mockVBDs()[0]
		assert.Equal(t, vbdID, result.UUID)
		assert.Equal(t, vbd.Attached, result.Attached)
		assert.Equal(t, vbd.Bootable, result.Bootable)
		assert.Equal(t, vbd.Position, result.Position)
		assert.Equal(t, vbd.ReadOnly, result.ReadOnly)
		require.NotNil(t, result.Device)
		assert.Equal(t, *vbd.Device, *result.Device)
	})

	t.Run("get non-existent VBD by ID", func(t *testing.T) {
		vbdID := uuid.Must(uuid.FromString(testVBDIDNotFound))

		result, err := svc.Get(t.Context(), vbdID)

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
			if err := json.NewEncoder(w).Encode([]*payloads.VBD{}); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
		service, server, _ := setupTestServerWithHandler(t, handler)
		defer server.Close()

		vbds, err := service.GetAll(context.Background(), limit, filter)

		assert.NoError(t, err)
		assert.NotNil(t, vbds)
		assert.True(t, called)
	})

	t.Run("does not send limit param when zero", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			values := r.URL.Query()
			assert.Empty(t, values.Get("limit"))
			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode([]*payloads.VBD{}); err != nil {
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

		vbds, err := service.GetAll(context.Background(), 0, "")

		assert.Error(t, err)
		assert.Nil(t, vbds)
	})

	t.Run("successfully retrieves all VBDs", func(t *testing.T) {
		server, service, _ := setupTestServer(t)
		defer server.Close()

		vbds, err := service.GetAll(context.Background(), 0, "")

		assert.NoError(t, err)
		require.NotNil(t, vbds)
		assert.Len(t, vbds, 2)
		assert.Equal(t, uuid.Must(uuid.FromString(testVBDID1)), vbds[0].UUID)
		assert.Equal(t, uuid.Must(uuid.FromString(testVBDID2)), vbds[1].UUID)
	})
}

func TestDelete(t *testing.T) {
	t.Run("successfully deletes an existing VBD", func(t *testing.T) {
		server, svc, _ := setupTestServer(t)
		defer server.Close()

		err := svc.Delete(t.Context(), uuid.Must(uuid.FromString(testVBDID1)))

		assert.NoError(t, err)
	})

	t.Run("returns error on http error", func(t *testing.T) {
		svc, server, _ := setupTestServerWithHandler(t, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		})
		defer server.Close()

		err := svc.Delete(t.Context(), uuid.Must(uuid.FromString(testVBDID1)))

		assert.Error(t, err)
	})
}
