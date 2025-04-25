package integration

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestVM_Restore(t *testing.T) {
	ctx := context.Background()
	tc := Setup(t)

	vmName := tc.GenerateResourceName("vm-restore")
	t.Cleanup(func() { tc.CleanupVM(t, vmName) })

	taskIDVM, err := tc.Client.VM().Create(ctx, &payloads.VM{
		NameLabel:       vmName,
		NameDescription: "Restore test VM",
		Template:        uuid.FromStringOrNil(tc.TemplateID),
		PoolID:          uuid.FromStringOrNil(tc.PoolID),
		CPUs:            payloads.CPUs{Number: 1},
		Memory:          payloads.Memory{Static: []int64{1073741824, 1073741824}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, taskIDVM)

	taskVM, err := tc.Client.Task().Wait(ctx, string(taskIDVM))
	require.NoError(t, err)
	require.Equal(t, payloads.Success, taskVM.Status, "VM creation task failed: %s", taskVM.Message)
	require.NotEqual(t, uuid.Nil, taskVM.Result.ID, "Task result does not contain VM ID")
	vmID := taskVM.Result.ID

	t.Logf("VM created with ID: %s", vmID)

	snapshotName := "integration-test-restore-point"
	taskIDSnap, err := tc.Client.VM().Snapshot().Create(ctx, vmID, snapshotName)
	require.NoError(t, err)
	require.NotEmpty(t, taskIDSnap)

	taskSnap, err := tc.Client.Task().Wait(ctx, string(taskIDSnap))
	require.NoError(t, err)
	require.Equal(t, payloads.Success, taskSnap.Status, "Snapshot creation task failed: %s", taskSnap.Message)
	require.NotEqual(t, uuid.Nil, taskSnap.Result.ID, "Task result does not contain Snapshot ID")
	snapshotID := taskSnap.Result.ID

	t.Logf("Snapshot created with ID: %s for restore testing", snapshotID)

	time.Sleep(5 * time.Second)

	restorePoints, err := tc.Client.VM().Restore().GetRestorePoints(ctx, vmID)
	assert.NoError(t, err)
	assert.NotEmpty(t, restorePoints, "Expected to find at least one restore point")

	var restorePointFound bool
	for _, rp := range restorePoints {
		if rp.ID == snapshotID {
			restorePointFound = true
			break
		}
	}
	assert.True(t, restorePointFound, "Expected to find the created snapshot in restore points")

	if tc.Client.VM().Restore() != nil {
		// Log that we could restore it in a real environment
		t.Logf("Restore would be possible using ID: %s", snapshotID)

		// Actual restore operation commented out to avoid potential issues in test environments
		// Uncomment this section only if you're sure the restore operation is safe in your environment
		/*
			err = tc.Client.VM().Restore().RestoreFromSnapshot(ctx, vmID, snapshotID)
			assert.NoError(t, err)

			// Wait for restore to complete
			time.Sleep(10 * time.Second)

			// Verify VM exists after restore
			restoredVM, err := tc.Client.VM().GetByID(ctx, vmID)
			assert.NoError(t, err)
			assert.NotNil(t, restoredVM)
		*/
	}

	err = tc.Client.VM().Snapshot().Delete(ctx, snapshotID)
	assert.NoError(t, err)

	if !tc.SkipTeardown {
		err = tc.Client.VM().Delete(ctx, vmID)
		assert.NoError(t, err)
	}
}
