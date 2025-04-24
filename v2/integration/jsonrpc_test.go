package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

		var poolID, templateID, networkID string
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

		if tc.NetworkID != "" {
			networkID = tc.NetworkID
		} else {
			t.Skip("Network ID resolution not implemented, please set XOA_NETWORK_ID")
		}

		vm := createTestVMForBackup(t, ctx, tc, vmName, poolID, templateID, networkID)
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
			Schedule: "0 0 * * *", // Midnight every day
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

		var poolID, templateID, networkID string
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

		if tc.NetworkID != "" {
			networkID = tc.NetworkID
		} else {
			t.Skip("Network ID resolution not implemented, please set XOA_NETWORK_ID")
		}

		vm := createTestVMForBackup(t, ctx, tc, vmName, poolID, templateID, networkID)
		defer func() {
			if !tc.SkipTeardown {
				err := tc.Client.VM().Delete(ctx, vm.ID)
				if err != nil {
					t.Logf("Failed to delete VM %s: %v", vm.ID, err)
				}
			}
		}()

		snapshotName := "jsonrpc-test-snapshot"
		snapshot, err := tc.Client.VM().Snapshot().Create(ctx, vm.ID, snapshotName)
		assert.NoError(t, err)
		assert.NotNil(t, snapshot)
		assert.Equal(t, snapshotName, snapshot.NameLabel)
		t.Logf("Created snapshot with ID: %s", snapshot.ID)

		snapshots, err := tc.Client.VM().Snapshot().ListByVM(ctx, vm.ID, 0)
		assert.NoError(t, err)
		found := false
		for _, s := range snapshots {
			if s.ID == snapshot.ID {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find the created snapshot in the list")

		// TODO: check if this is correct.
		if tc.Client.VM().Snapshot() != nil {
			if testing.Short() || os.Getenv("XOA_ALLOW_SNAPSHOT_REVERT") != trueStr {
				t.Log("Skipping snapshot revert test - requires XOA_ALLOW_SNAPSHOT_REVERT=" + trueStr)
			} else {
				err = tc.Client.VM().Snapshot().Revert(ctx, vm.ID, snapshot.ID)
				assert.NoError(t, err)
				t.Log("Successfully reverted snapshot")
				time.Sleep(5 * time.Second)
			}
		}

		if !tc.SkipTeardown {
			err = tc.Client.VM().Snapshot().Delete(ctx, snapshot.ID)
			assert.NoError(t, err)
			t.Log("Successfully deleted snapshot")
		}
	})
}

func createTestVMForBackup(
	t *testing.T,
	ctx context.Context,
	tc *TestClient,
	name, poolID, templateID, networkID string,
) *payloads.VM {
	t.Helper()

	vm := &payloads.VM{
		NameLabel:       name,
		NameDescription: "JSON-RPC test VM",
		Template:        GetUUID(t, templateID),
		Memory: payloads.Memory{
			Size: 1 * 1024 * 1024 * 1024, // 1 GB
		},
		CPUs: payloads.CPUs{
			Number: 1,
		},
		VIFs:        []string{networkID},
		AutoPoweron: false,
		PoolID:      GetUUID(t, poolID),
	}

	createdVM, err := tc.Client.VM().Create(ctx, vm)
	assert.NoError(t, err)
	assert.NotNil(t, createdVM)
	t.Logf("VM created with ID: %s", createdVM.ID)

	return createdVM
}
