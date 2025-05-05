package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestVM_Snapshot(t *testing.T) {
	ctx := context.Background()
	tc := Setup(t)

	vmName := tc.GenerateResourceName("vm-snapshot")
	t.Cleanup(func() { tc.CleanupVM(t, vmName) })

	taskIDVM, err := CreateTestVM(t, ctx, tc, vmName)
	require.NoError(t, err)
	require.NotEmpty(t, taskIDVM)

	taskVM, err := tc.Client.Task().Wait(ctx, string(taskIDVM))
	require.NoError(t, err)
	require.Equal(t, payloads.Success, taskVM.Status, "VM creation task failed: %s", taskVM.Message)
	require.NotEqual(t, uuid.Nil, taskVM.Result.ID, "Task result does not contain VM ID")
	vmID := taskVM.Result.ID
	require.NotEqual(t, uuid.Nil, vmID)

	t.Logf("VM created with ID: %s", vmID)

	snapshotName := "integration-test-snapshot"
	taskID, err := tc.Client.VM().Snapshot().Create(ctx, vmID, snapshotName)
	require.NoError(t, err)
	require.NotEmpty(t, taskID)

	task, err := tc.Client.Task().Wait(ctx, string(taskID))
	require.NoError(t, err)
	require.Equal(t, payloads.Success, task.Status, "Snapshot creation task failed: %s", task.Message)
	require.NotEqual(t, uuid.Nil, task.Result.ID, "Task result does not contain Snapshot ID")
	snapshotID := task.Result.ID

	snapshot, err := tc.Client.VM().Snapshot().GetByID(ctx, snapshotID)
	require.NoError(t, err)
	require.NotNil(t, snapshot)
	require.Equal(t, snapshotName, snapshot.NameLabel)

	allSnapshots, err := tc.Client.VM().Snapshot().List(ctx, map[string]any{"limit": 10})
	require.NoError(t, err)
	found := false
	for _, s := range allSnapshots {
		if s.ID == snapshotID {
			found = true
			break
		}
	}
	require.True(t, found, "Created snapshot not found in list")

	if os.Getenv("XOA_ALLOW_SNAPSHOT_REVERT") == "true" {
		err = tc.Client.VM().Snapshot().Revert(ctx, vmID, snapshotID)
		assert.NoError(t, err)

		time.Sleep(5 * time.Second)
	}

	err = tc.Client.VM().Snapshot().Delete(ctx, snapshotID)
	assert.NoError(t, err)

	// List (verify snapshot is deleted)
	// Note: Snapshot List does not filter by VM ID in the current interface
	allSnapshots, err = tc.Client.VM().Snapshot().List(ctx, map[string]any{})
	require.NoError(t, err)
	found = false
	for _, s := range allSnapshots {
		if s.ID == snapshotID {
			found = true
			break
		}
	}
	require.False(t, found, "Snapshot should have been deleted")

	if !tc.SkipTeardown {
		err = tc.Client.VM().Delete(ctx, vmID)
		assert.NoError(t, err)
	}
}

// Helper function to create a VM for snapshot tests
func CreateTestVM(t *testing.T, ctx context.Context, tc *TestClient, name string) (payloads.TaskID, error) {
	poolID := uuid.FromStringOrNil(tc.PoolID)
	templateID := uuid.FromStringOrNil(tc.TemplateID)
	require.NotEqual(t, uuid.Nil, poolID, "Failed to parse XOA_POOL_ID")
	require.NotEqual(t, uuid.Nil, templateID, "Failed to parse XOA_TEMPLATE_ID")

	return tc.Client.VM().Create(ctx, &payloads.VM{
		NameLabel:       name,
		NameDescription: "VM for snapshot integration test",
		Template:        templateID,
		PoolID:          poolID,
		CPUs:            payloads.CPUs{Number: 1},
		Memory:          payloads.Memory{Static: []int64{1073741824, 1073741824}},
	})
}
