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

func TestJSONRPC_Integration(t *testing.T) {
	ctx := context.Background()
	tc := Setup(t)

	t.Run("backup job operations", func(t *testing.T) {
		if os.Getenv("XOA_RUN_BACKUP_TESTS") != trueStr {
			t.Skip("Skipping backup job operations test. Set XOA_RUN_BACKUP_TESTS=" + trueStr + " to run")
			return
		}

		backupJobName := tc.GenerateResourceName("backup-test")

		vmName := tc.GenerateResourceName("vm-backup")
		tc.CleanupVM(t, vmName)

		vm := createTestVMForJsonRpc(t, ctx, tc, vmName)
		defer func() {
			if !tc.SkipTeardown {
				err := tc.Client.VM().Delete(ctx, vm.ID)
				if err != nil {
					t.Logf("Failed to delete VM %s: %v", vm.ID, err)
				}
			}
		}()

		existingJobs, err := tc.Client.Backup().ListJobs(ctx, 0)
		assert.NoError(t, err)
		t.Logf("Found %d existing backup jobs", len(existingJobs))

		if os.Getenv("XOA_CREATE_BACKUP_JOB") != trueStr {
			t.Log("Skipping backup job creation. Set XOA_CREATE_BACKUP_JOB=" + trueStr + " to create a test backup job")
			return
		}

		backupJob := &payloads.BackupJob{
			Name:     backupJobName,
			Mode:     "full",
			Schedule: "0 0 * * *", // Midnight every day'
			Enabled:  false,
			VMs:      vm.ID.String(),
			Settings: payloads.BackupSettings{
				Retention:          7,
				CompressionEnabled: true,
				ReportWhenFailOnly: false,
			},
		}

		createdJob, err := tc.Client.Backup().CreateJob(ctx, backupJob)
		if err != nil {
			t.Logf("Creating backup job failed (may be expected): %v", err)
			return
		}

		assert.NotNil(t, createdJob)
		assert.Equal(t, backupJobName, createdJob.Name)
		t.Logf("Created backup job with ID: %s", createdJob.ID)

		retrievedJob, err := tc.Client.Backup().GetJob(ctx, createdJob.ID.String())
		assert.NoError(t, err)
		assert.Equal(t, createdJob.ID, retrievedJob.ID)

		updatedJobName := backupJobName + "-updated"
		retrievedJob.Name = updatedJobName

		updatedJob, err := tc.Client.Backup().UpdateJob(ctx, retrievedJob)
		if err != nil {
			t.Logf("Updating backup job failed (may be expected): %v", err)
		} else {
			assert.Equal(t, updatedJobName, updatedJob.Name)
		}

		if !tc.SkipTeardown {
			err = tc.Client.Backup().DeleteJob(ctx, createdJob.ID)
			if err != nil {
				t.Logf("Deleting backup job failed: %v", err)
			}
		}
	})

	t.Run("snapshot operations", func(t *testing.T) {
		vmName := tc.GenerateResourceName("vm-snapshot-jsonrpc")
		tc.CleanupVM(t, vmName)

		vm := createTestVMForJsonRpc(t, ctx, tc, vmName)
		defer func() {
			if !tc.SkipTeardown {
				err := tc.Client.VM().Delete(ctx, vm.ID)
				if err != nil {
					t.Logf("Failed to delete VM %s: %v", vm.ID, err)
				}
			}
		}()

		snapshotName := "jsonrpc-test-snapshot"
		taskIDSnap, err := tc.Client.VM().Snapshot().Create(ctx, vm.ID, snapshotName)
		require.NoError(t, err)
		require.NotEmpty(t, taskIDSnap)

		taskSnap, err := tc.Client.Task().Wait(ctx, string(taskIDSnap))
		require.NoError(t, err)
		require.Equal(t, payloads.Success, taskSnap.Status, "Snapshot creation task failed: %s", taskSnap.Message)
		require.NotEqual(t, uuid.Nil, taskSnap.Result.ID, "Task result does not contain Snapshot ID")
		snapshotID := taskSnap.Result.ID
		t.Logf("Created snapshot with ID: %s", snapshotID)

		allSnapshots, err := tc.Client.VM().Snapshot().List(ctx, 0)
		require.NoError(t, err)
		found := false
		for _, s := range allSnapshots {
			if s.ID == snapshotID {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find the created snapshot in the list")

		if os.Getenv("XOA_ALLOW_SNAPSHOT_REVERT") == trueStr {
			err = tc.Client.VM().Snapshot().Revert(ctx, vm.ID, snapshotID)
			assert.NoError(t, err)
			t.Log("Successfully reverted snapshot")
			time.Sleep(5 * time.Second)
		} else {
			t.Log("Skipping snapshot revert test - requires XOA_ALLOW_SNAPSHOT_REVERT=" + trueStr)
		}

		if !tc.SkipTeardown {
			err = tc.Client.VM().Snapshot().Delete(ctx, snapshotID)
			assert.NoError(t, err)
			t.Log("Successfully deleted snapshot")
		}
	})
}

func createTestVMForJsonRpc(
	t *testing.T,
	ctx context.Context,
	tc *TestClient,
	name string,
) *payloads.VM {
	t.Helper()

	poolID := uuid.FromStringOrNil(tc.PoolID)
	templateID := uuid.FromStringOrNil(tc.TemplateID)
	require.NotEqual(t, uuid.Nil, poolID, "Failed to parse XOA_POOL_ID for JSON-RPC test VM")
	require.NotEqual(t, uuid.Nil, templateID, "Failed to parse XOA_TEMPLATE_ID for JSON-RPC test VM")

	vm := &payloads.VM{
		NameLabel:       name,
		NameDescription: "JSON-RPC test VM",
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
	require.NoError(t, err)
	require.NotEmpty(t, taskID)

	task, err := tc.Client.Task().Wait(ctx, string(taskID))
	require.NoError(t, err)
	require.Equal(t, payloads.Success, task.Status, "VM creation task failed: %s", task.Message)
	require.NotEqual(t, uuid.Nil, task.Result.ID, "Task result does not contain VM ID")
	vmID := task.Result.ID

	createdVM, err := tc.Client.VM().GetByID(ctx, vmID)
	require.NoError(t, err)
	require.NotNil(t, createdVM)
	t.Logf("VM created with ID: %s", vmID)
	return createdVM
}
