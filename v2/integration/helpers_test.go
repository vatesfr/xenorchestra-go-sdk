package integration

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
)

// createVMsForTest helps create multiple VMs for listing or batch tests
func createVMsForTest(t *testing.T, ctx context.Context, pool library.Pool, count int, name string) []string {
	vmIDs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		vmName := name + uuid.Must(uuid.NewV4()).String()
		params := payloads.CreateVMParams{
			NameLabel: vmName,
			Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		}

		vmID, err := pool.CreateVM(ctx, intTests.testPool.ID, params)
		require.NoErrorf(t, err, "error while creating VM %s in pool %s: %v", vmName, intTests.testPool.ID, err)
		require.NotEqual(t, uuid.Nil, vmID, "created VM ID should not be nil")
		vmIDs = append(vmIDs, vmID.String())
	}

	return vmIDs
}

func ptr[T any](v T) *T {
	return &v
}

func waitForTask(t *testing.T, ctx context.Context, client library.Library, taskID string) *payloads.Task {
	t.Helper()
	var task *payloads.Task
	assert.Eventually(t, func() bool {
		var err error
		task, err = client.Task().Get(ctx, taskID)
		if err != nil {
			return false
		}
		return task.Status != payloads.Pending
	}, 2*time.Minute, 5*time.Second, "Task %s should not stay pending", taskID)
	return task
}

func waitForVMReady(t *testing.T, ctx context.Context, client library.Library, vmID uuid.UUID) {
	t.Helper()
	assert.Eventually(t, func() bool {
		vm, err := client.VM().GetByID(ctx, vmID)
		if err != nil {
			return false
		}
		return vm.PowerState == payloads.PowerStateRunning && vm.MainIpAddress != ""
	}, 2*time.Minute, 10*time.Second, "VM %s should be running and reported an IP", vmID)
}

// createVDIForTest creates a VDI with the specified name and size using the v1 client and returns its ID
// TODO: replace with v2 client once VDI creation is supported in v2
func createVDIForTest(t *testing.T, ctx context.Context, client v1.XOClient, name string, size int64) uuid.UUID {
	t.Helper()

	var id string

	if client, ok := intTests.v1Client.(*v1.Client); ok {
		err := client.Call("disk.create", map[string]any{
			"name": name,
			"size": size,
			"sr":   intTests.testSR.Id,
		}, &id)
		require.NoError(t, err, "error while creating VDI %s in SR %s: %v", name, intTests.testSR.Id, err)
	}
	return uuid.FromStringOrNil(id)
}
