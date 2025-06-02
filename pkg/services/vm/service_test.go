package vm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	mock_library "github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library/mock"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

func setupTestServer(t *testing.T) (*httptest.Server, library.VM) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var vmID uuid.UUID
		pathParts := strings.Split(r.URL.Path, "/")
		if len(pathParts) >= 3 && pathParts[1] == "rest" && pathParts[2] == "v0" {
			if len(pathParts) >= 5 && pathParts[3] == "vms" {
				idStr := pathParts[4]
				if id, err := uuid.FromString(idStr); err == nil {
					vmID = id
				}
			}
		}

		switch {
		case r.URL.Path == "/rest/v0/vms" && r.Method == http.MethodGet:
			query := r.URL.Query()
			fields := query.Get("fields")
			filter := query.Get("filter")

			id1 := uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000001"))
			id2 := uuid.Must(uuid.FromString("00000000-0000-0000-0000-000000000002"))

			if fields != "" {
				// With fields parameter: return VM objects directly
				vms := []*payloads.VM{
					{
						ID:         id1,
						UUID:       id1.String(),
						NameLabel:  "VM 1",
						PowerState: payloads.PowerStateRunning,
					},
					{
						ID:         id2,
						UUID:       id2.String(),
						NameLabel:  "VM 2",
						PowerState: payloads.PowerStateHalted,
					},
				}

				// Apply filter if present
				if filter != "" {
					filtered := []*payloads.VM{}
					for _, vm := range vms {
						if strings.Contains(filter, "power_state:Running") && vm.PowerState == payloads.PowerStateRunning {
							filtered = append(filtered, vm)
						} else if strings.Contains(filter, "power_state:Halted") && vm.PowerState == payloads.PowerStateHalted {
							filtered = append(filtered, vm)
						} else if !strings.Contains(filter, "power_state:") {
							filtered = append(filtered, vm)
						}
					}
					vms = filtered
				}

				err := json.NewEncoder(w).Encode(vms)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				// Without fields parameter: return VM paths
				paths := []string{
					fmt.Sprintf("/rest/v0/vms/%s", id1),
					fmt.Sprintf("/rest/v0/vms/%s", id2),
				}
				err := json.NewEncoder(w).Encode(paths)
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}

		case r.URL.Path == "/rest/v0/vms" && r.Method == http.MethodPost:
			var vm payloads.VM
			_ = json.NewDecoder(r.Body).Decode(&vm)
			vm.ID = uuid.Must(uuid.NewV4())
			vm.PowerState = payloads.PowerStateHalted
			_ = json.NewEncoder(w).Encode(vm)

		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/") && len(pathParts) == 5 && r.Method == http.MethodGet:
			vmName := "VM 1"
			powerState := payloads.PowerStateRunning
			if vmID.String() == "00000000-0000-0000-0000-000000000002" {
				vmName = "VM 2"
				powerState = payloads.PowerStateHalted
			}

			err := json.NewEncoder(w).Encode(payloads.VM{
				ID:         vmID,
				UUID:       vmID.String(),
				NameLabel:  vmName,
				PowerState: powerState,
			})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/") && len(pathParts) == 5 && r.Method == http.MethodPost:
			var vm payloads.VM
			_ = json.NewDecoder(r.Body).Decode(&vm)
			vm.ID = vmID
			_ = json.NewEncoder(w).Encode(vm)

		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/") && len(pathParts) == 5 && r.Method == http.MethodDelete:
			err := json.NewEncoder(w).Encode(map[string]bool{"success": true})
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		case strings.HasPrefix(r.URL.Path, "/rest/v0/vms/") && strings.Contains(r.URL.Path, "/actions/"):
			parts := strings.Split(r.URL.Path, "/")
			if len(parts) != 7 {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "invalid action path format: %s", r.URL.Path)
				return
			}
			action := parts[6]
			vmIDStr := parts[4]

			_, err := uuid.FromString(vmIDStr)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "invalid VM UUID in action path: %s", vmIDStr)
				return
			}

			var requestBody map[string]string
			_ = json.NewDecoder(r.Body).Decode(&requestBody)
			if reqIDStr, ok := requestBody["id"]; !ok || reqIDStr != vmIDStr {
				w.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(w, "VM ID in path (%s) does not match ID in body (%s)", vmIDStr, reqIDStr)
				return
			}

			switch action {
			case "start", "clean_shutdown", "hard_shutdown", "clean_reboot", "hard_reboot", "snapshot":
				err := json.NewEncoder(w).Encode(map[string]bool{"success": true})
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			default:
				w.WriteHeader(http.StatusNotFound)
			}

		case strings.HasPrefix(r.URL.Path, "/rest/v0/pools/") && strings.HasSuffix(r.URL.Path, "/actions/create_vm"):
			var createParams map[string]any
			_ = json.NewDecoder(r.Body).Decode(&createParams)

			pathParts := strings.Split(r.URL.Path, "/")
			poolID := pathParts[4]

			vm := payloads.VM{
				ID:              uuid.Must(uuid.NewV4()),
				NameLabel:       createParams["name_label"].(string),
				NameDescription: createParams["name_description"].(string),
				PowerState:      payloads.PowerStateHalted,
				PoolID:          uuid.Must(uuid.FromString(poolID)),
			}

			_ = json.NewEncoder(w).Encode(vm)

		default:
			fmt.Printf("Unhandled path: %s\n", r.URL.Path)
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

	var ctrl = gomock.NewController(t)

	snapshotService := mock_library.NewMockSnapshot(ctrl)

	return server, New(restClient, snapshotService, log)
}

func TestGetByID(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	id := uuid.Must(uuid.NewV4())
	vm, err := service.GetByID(context.Background(), id)

	assert.NoError(t, err)
	assert.Equal(t, id, vm.ID)
	assert.Equal(t, "VM 1", vm.NameLabel)
	assert.Equal(t, payloads.PowerStateRunning, vm.PowerState)
}

func TestList(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	t.Run("List without query options (paths approach)", func(t *testing.T) {
		vms, err := service.List(context.Background(), nil)

		assert.NoError(t, err)
		assert.Len(t, vms, 2)
		assert.Equal(t, "VM 1", vms[0].NameLabel)
		assert.Equal(t, "VM 2", vms[1].NameLabel)
		assert.Equal(t, payloads.PowerStateRunning, vms[0].PowerState)
		assert.Equal(t, payloads.PowerStateHalted, vms[1].PowerState)
	})

	t.Run("List with limit only (paths approach)", func(t *testing.T) {
		query := &payloads.VMQueryOptions{
			Limit: 10,
		}
		vms, err := service.List(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, vms, 2)
		assert.Equal(t, "VM 1", vms[0].NameLabel)
		assert.Equal(t, "VM 2", vms[1].NameLabel)
	})

	t.Run("List with fields (objects approach)", func(t *testing.T) {
		query := &payloads.VMQueryOptions{
			Fields: []string{payloads.VMFieldNameLabel, payloads.VMFieldPowerState, payloads.VMFieldUUID},
		}
		vms, err := service.List(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, vms, 2)
		assert.Equal(t, "VM 1", vms[0].NameLabel)
		assert.Equal(t, "VM 2", vms[1].NameLabel)
		assert.Equal(t, payloads.PowerStateRunning, vms[0].PowerState)
		assert.Equal(t, payloads.PowerStateHalted, vms[1].PowerState)
	})

	t.Run("List with fields and filter (objects approach)", func(t *testing.T) {
		query := &payloads.VMQueryOptions{
			Fields: []string{payloads.VMFieldNameLabel, payloads.VMFieldPowerState},
			Filter: "power_state:Running",
		}
		vms, err := service.List(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, vms, 1)
		assert.Equal(t, "VM 1", vms[0].NameLabel)
		assert.Equal(t, payloads.PowerStateRunning, vms[0].PowerState)
	})

	t.Run("List with utility function", func(t *testing.T) {
		query := &payloads.VMQueryOptions{
			Fields: []string{payloads.VMFieldNameLabel, payloads.VMFieldPowerState},
			Filter: "power_state:Running",
		}

		vms, err := service.List(context.Background(), query)

		assert.NoError(t, err)
		assert.Len(t, vms, 1)
		assert.Equal(t, "VM 1", vms[0].NameLabel)
		assert.Equal(t, payloads.PowerStateRunning, vms[0].PowerState)
	})
}

func TestCreate(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	newVM := &payloads.VM{
		NameLabel:       "New VM from Test",
		NameDescription: "Test VM Creation",
		Template:        uuid.Must(uuid.NewV4()),
		PoolID:          uuid.Must(uuid.NewV4()),
		CPUs: payloads.CPUs{
			Number: 1,
		},
		Memory: payloads.Memory{
			Static: []int64{1073741824, 1073741824},
		},
	}

	taskID, err := service.Create(context.Background(), newVM)

	assert.NoError(t, err)
	assert.NotEmpty(t, taskID, "Create should return a task ID")
}

func TestUpdate(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	id := uuid.Must(uuid.NewV4())
	updateVM := &payloads.VM{
		ID:              id,
		NameLabel:       "Updated VM",
		NameDescription: "Updated description",
	}

	vm, err := service.Update(context.Background(), updateVM)

	assert.NoError(t, err)
	assert.Equal(t, id, vm.ID)
	assert.Equal(t, "Updated VM", vm.NameLabel)
	assert.Equal(t, "Updated description", vm.NameDescription)
}

func TestDelete(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	id := uuid.Must(uuid.NewV4())
	err := service.Delete(context.Background(), id)

	assert.NoError(t, err)
}

func TestPowerOperations(t *testing.T) {
	server, service := setupTestServer(t)
	defer server.Close()

	id := uuid.Must(uuid.NewV4())

	err := service.Start(context.Background(), id)
	assert.NoError(t, err)

	err = service.CleanShutdown(context.Background(), id)
	assert.NoError(t, err)
}
