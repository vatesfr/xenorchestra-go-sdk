package integration

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestTask_Integration(t *testing.T) {
	ctx := context.Background()
	tc := Setup(t)

	t.Run("task service access", func(t *testing.T) {
		taskService := tc.Client.Task()
		assert.NotNil(t, taskService, "Task service should be accessible")
	})

	t.Run("snapshot create wait for task", func(t *testing.T) {
		vmName := tc.GenerateResourceName("vm-task-test")
		tc.CleanupVM(t, vmName)

		var poolID, templateID string
		if tc.PoolID != "" {
			poolID = tc.PoolID
		} else {
			t.Skip("Pool ID resolution not implemented, please set XOA_POOL_ID")
		}

		if tc.TemplateID != "" {
			templateID = tc.TemplateID
		} else {
			t.Skip("Template ID resolution not implemented, please set XOA_TEMPLATE_ID")
		}

		vmID := createTestVMAndWait(t, ctx, tc, vmName, poolID, templateID)

		defer cleanupVM(t, ctx, tc, vmID)

		snapshotName := "integration-task-test-snapshot"
		taskID, err := tc.Client.VM().Snapshot().Create(ctx, vmID, snapshotName)
		assert.NoError(t, err)
		assert.NotNil(t, taskID)

		task, err := tc.Client.Task().Wait(ctx, string(taskID))
		assert.NoError(t, err)
		if task != nil && task.Status == payloads.Success && task.Result.ID != uuid.Nil {
			snapshotID := task.Result.ID
			t.Logf("Snapshot task succeeded, snapshot ID: %s", snapshotID)
			if !tc.SkipTeardown {
				err = tc.Client.VM().Snapshot().Delete(ctx, snapshotID)
				assert.NoError(t, err, "Failed to cleanup snapshot %s", snapshotID)
			}
		} else {
			t.Logf("Snapshot task did not succeed or did not return ID. Status: %s, Error: %v", task.Status, err)
		}
	})
}

func createTestVMAndWait(
	t *testing.T,
	ctx context.Context,
	tc *TestClient,
	name,
	poolIDStr,
	templateIDStr string) uuid.UUID {
	t.Helper()

	poolID := uuid.FromStringOrNil(poolIDStr)
	templateID := uuid.FromStringOrNil(templateIDStr)
	require.NotEqual(t, uuid.Nil, poolID, "Failed to parse XOA_POOL_ID for task test VM")
	require.NotEqual(t, uuid.Nil, templateID, "Failed to parse XOA_TEMPLATE_ID for task test VM")

	vm := &payloads.VM{
		NameLabel:       name,
		NameDescription: "Task test VM",
		Template:        templateID,
		Memory: payloads.Memory{
			Size: 1 * 1024 * 1024 * 1024, // 1 GB
		},
		CPUs: payloads.CPUs{
			Number: 1,
		},
		AutoPoweron: false,
		PoolID:      poolID,
	}

	taskID, err := tc.Client.VM().Create(ctx, vm)
	assert.NoError(t, err)
	assert.NotNil(t, taskID)

	task, err := tc.Client.Task().Wait(ctx, string(taskID))
	require.NoError(t, err)
	require.Equal(t, payloads.Success, task.Status, "VM creation task failed: %s", task.Message)
	require.NotEqual(t, uuid.Nil, task.Result.ID, "Task result does not contain VM ID")
	vmID := task.Result.ID

	t.Logf("VM created with ID: %s", vmID)
	return vmID
}

func cleanupVM(t *testing.T, ctx context.Context, tc *TestClient, vmID uuid.UUID) {
	t.Helper()

	if tc.SkipTeardown {
		t.Logf("Skipping cleanup of VM %s", vmID)
		return
	}

	err := tc.Client.VM().Delete(ctx, vmID)
	if err != nil {
		t.Logf("Failed to delete VM %s: %v", vmID, err)
	} else {
		t.Logf("Successfully deleted VM %s", vmID)
	}
}
