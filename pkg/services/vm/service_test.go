package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/docker/go-units"
	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	mock "github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library/mock"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

const (
	mockVMID1       = "00000000-0000-0000-0000-000000000001"
	mockVMID2       = "00000000-0000-0000-0000-000000000002"
	mockCreatedVMID = "10000000-0000-0000-0000-000000000001"
	mockPoolID      = "201b228b-2f91-4138-969c-49cae8780448"
)

var mockVMs = func() []payloads.VM {
	return []payloads.VM{
		{
			ID:         uuid.Must(uuid.FromString(mockVMID1)),
			NameLabel:  "VM 1",
			PowerState: payloads.PowerStateRunning,
		},
		{
			ID:         uuid.Must(uuid.FromString(mockVMID2)),
			NameLabel:  "VM 2",
			PowerState: payloads.PowerStateHalted,
		},
	}
}

func lookupMockVM(id string) (payloads.VM, bool) {
	for _, vm := range mockVMs() {
		if vm.ID.String() == id {
			return vm, true
		}
	}
	if id == mockCreatedVMID {
		return payloads.VM{
			ID:         uuid.Must(uuid.FromString(mockCreatedVMID)),
			NameLabel:  "New VM",
			PowerState: payloads.PowerStateHalted,
		}, true
	}
	return payloads.VM{}, false
}

func setupTestServerWithHandler(t *testing.T, handler http.HandlerFunc) (*httptest.Server, library.VM, *mock.MockPool) {
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
	mockPool := mock.NewMockPool(ctrl)
	return server, New(restClient, mockTask, mockPool, log).(*Service), mockPool
}

func setupTestServer(t *testing.T) (*httptest.Server, library.VM, *mock.MockPool) {
	mux := http.NewServeMux()

	writeJSON := func(w http.ResponseWriter, r *http.Request, payload any) {
		w.Header().Set("Content-Type", "application/json")
		t.Logf("Request made to mock server: %s %s", r.Method, r.URL.Path)
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}

	mux.HandleFunc("GET /rest/v0/vms", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, r, mockVMs())
	})

	mux.HandleFunc("GET /rest/v0/vms/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vmID, err := uuid.FromString(r.PathValue("id"))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		vm, ok := lookupMockVM(vmID.String())
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, r, vm)
	})

	mux.HandleFunc("DELETE /rest/v0/vms/{id}", func(w http.ResponseWriter, r *http.Request) {
		vmID := r.PathValue("id")
		if vmID != mockVMID1 && vmID != mockVMID2 && vmID != mockCreatedVMID {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, r, map[string]bool{"success": true})
	})

	mux.HandleFunc("POST /rest/v0/vms/{id}/actions/{action}", func(w http.ResponseWriter, r *http.Request) {
		if _, err := uuid.FromString(r.PathValue("id")); err != nil {
			http.NotFound(w, r)
			return
		}

		action := r.PathValue("action")
		t.Logf("Action requested: %s", action)
		switch action {
		case "start", "clean_shutdown", "hard_shutdown", "clean_reboot", "hard_reboot", "snapshot", "restart",
			"suspend", "resume", "pause", "unpause":
			writeJSON(w, r, payloads.TaskIDResponse{TaskID: "task-123"})
		default:
			http.NotFound(w, r)
		}
	})

	mux.HandleFunc("GET /rest/v0/vms/{id}/vdis", func(w http.ResponseWriter, r *http.Request) {
		vmID := r.PathValue("id")
		if vmID != mockVMID1 && vmID != mockVMID2 && vmID != mockCreatedVMID {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, r, []payloads.VDI{
			{
				ID:        uuid.Must(uuid.FromString("30000000-0000-0000-0000-000000000001")),
				NameLabel: "VDI 1",
				VDIType:   payloads.VDITypeUser,
				Size:      20 * units.GB,
			},
			{
				ID:        uuid.Must(uuid.FromString("30000000-0000-0000-0000-000000000002")),
				NameLabel: "VDI 2",
				VDIType:   payloads.VDITypeUser,
				Size:      2 * units.GB,
			},
		})
	})

	server := httptest.NewServer(mux)

	restClient := &client.Client{
		HttpClient: http.DefaultClient,
		BaseURL:    &url.URL{Scheme: "http", Host: server.URL[7:], Path: "/rest/v0"},
		AuthToken:  "test-token",
	}

	log, err := logger.New(false, []string{"stdout"}, []string{"stderr"})
	if err != nil {
		panic(err)
	}

	// Create mock controller and mocks
	ctrl := gomock.NewController(t)
	mockTask := mock.NewMockTask(ctrl)
	mockPool := mock.NewMockPool(ctrl)

	return server, New(restClient, mockTask, mockPool, log), mockPool
}

func TestGetByID(t *testing.T) {
	server, service, _ := setupTestServer(t)
	defer server.Close()

	id := uuid.FromStringOrNil(mockVMID1)
	vm, err := service.GetByID(context.Background(), id)

	assert.NoError(t, err)
	assert.Equal(t, id, vm.ID)
	assert.Equal(t, "VM 1", vm.NameLabel)
	assert.Equal(t, payloads.PowerStateRunning, vm.PowerState)
}

func TestGetAll(t *testing.T) {
	server, service, _ := setupTestServer(t)
	defer server.Close()

	vms, err := service.GetAll(context.Background(), 0, "")

	assert.NoError(t, err)
	assert.Len(t, vms, 2)
	assert.Equal(t, "VM 1", vms[0].NameLabel)
	assert.Equal(t, "VM 2", vms[1].NameLabel)
}

func TestCreate(t *testing.T) {
	server, service, mockPool := setupTestServer(t)
	defer server.Close()

	// Set up mock expectations for Pool.CreateVM
	poolID := uuid.FromStringOrNil(mockPoolID)
	vmID := uuid.Must(uuid.FromString(mockCreatedVMID))

	createParams := payloads.CreateVMParams{
		NameLabel:       "New VM",
		NameDescription: "Test VM",
	}

	mockPool.EXPECT().CreateVM(gomock.Any(), poolID, createParams).Return(vmID, nil)

	vm, err := service.Create(context.Background(), poolID, &createParams)

	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, vm.ID)
	assert.Equal(t, "New VM", vm.NameLabel)
	assert.Equal(t, payloads.PowerStateHalted, vm.PowerState)
}

// TODO: Re-enable when Update is implemented
// func TestUpdate(t *testing.T) {
// 	server, service, _ := setupTestServer(t)
// 	defer server.Close()

// 	id := uuid.Must(uuid.NewV4())
// 	updateVM := &payloads.VM{
// 		ID:              id,
// 		NameLabel:       "Updated VM",
// 		NameDescription: "Updated description",
// 	}

// 	vm, err := service.Update(context.Background(), updateVM)

// 	assert.NoError(t, err)
// 	assert.Equal(t, id, vm.ID)
// 	assert.Equal(t, "Updated VM", vm.NameLabel)
// 	assert.Equal(t, "Updated description", vm.NameDescription)
// }

func TestDelete(t *testing.T) {
	server, service, _ := setupTestServer(t)
	defer server.Close()

	t.Run("delete existing VM", func(t *testing.T) {
		id := uuid.Must(uuid.FromString(mockVMID1))
		err := service.Delete(t.Context(), id)

		assert.NoError(t, err)
	})

	t.Run("delete unknown VM", func(t *testing.T) {
		id := uuid.Must(uuid.NewV6())
		err := service.Delete(t.Context(), id)
		assert.Error(t, err, "deleting unknown VM should return an error")
	})
}

func TestPowerOperations(t *testing.T) {
	server, service, _ := setupTestServer(t)
	defer server.Close()

	// Get access to the mockTask from service
	s := service.(*Service)
	mockTask := s.taskService.(*mock.MockTask)

	id := uuid.Must(uuid.NewV4())

	// Expect calls to HandleTaskResponse
	mockTask.EXPECT().HandleTaskResponse(gomock.Any(), gomock.Any(), false).
		Return(&payloads.Task{ID: "task-123"}, nil).AnyTimes()

	taskID, err := service.Start(context.Background(), id, nil)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskID)

	taskID, err = service.CleanShutdown(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskID)

	taskID, err = service.Suspend(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskID)

	taskID, err = service.Resume(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskID)

	taskID, err = service.Pause(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskID)

	taskID, err = service.Unpause(context.Background(), id)
	assert.NoError(t, err)
	assert.Equal(t, "task-123", taskID)
}

func TestGetVDIs(t *testing.T) {
	server, service, _ := setupTestServer(t)
	defer server.Close()

	t.Run("returns VDIs for a VM", func(t *testing.T) {
		vmID := uuid.FromStringOrNil(mockVMID1)
		vdis, err := service.GetVDIs(context.Background(), vmID, 0, "")

		assert.NoError(t, err)
		assert.Len(t, vdis, 2)
		assert.Equal(t, "VDI 1", vdis[0].NameLabel)
		assert.Equal(t, "VDI 2", vdis[1].NameLabel)
	})

	t.Run("error when VM doesn't exist", func(t *testing.T) {
		vmID := uuid.Must(uuid.NewV6())
		vdis, err := service.GetVDIs(context.Background(), vmID, 0, "")

		assert.Error(t, err)
		assert.Nil(t, vdis)
	})

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
		server, service, _ := setupTestServerWithHandler(t, handler)
		defer server.Close()
		vdis, err := service.GetVDIs(context.Background(), uuid.Must(uuid.FromString(mockVMID1)), limit, filter)
		assert.NoError(t, err)
		assert.NotNil(t, vdis)
		assert.True(t, called)
	})

}
