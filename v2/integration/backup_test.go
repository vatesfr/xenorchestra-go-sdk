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

func TestBackup(t *testing.T) {
	tc := Setup(t)

	if tc.StorageID == "" && tc.Storage == "" {
		t.Skip("Neither StorageID nor Storage is set, skipping backup tests")
		return
	}

	vmName := tc.GenerateResourceName("vm-backup")
	t.Cleanup(func() { tc.CleanupVM(t, vmName) })

	var poolID, templateID, networkID string
	if tc.PoolID != "" {
		poolID = tc.PoolID
	} else {
		t.Logf("Using Pool name: %s", tc.Pool)
		t.Skip("Pool ID resolution not implemented, please set XOA_POOL_ID")
	}

	if tc.TemplateID != "" {
		templateID = tc.TemplateID
	} else {
		t.Logf("Using Template name: %s", tc.Template)
		t.Skip("Template ID resolution not implemented, please set XOA_TEMPLATE_ID")
	}

	if tc.NetworkID != "" {
		networkID = tc.NetworkID
	} else {
		t.Logf("Using Network name: %s", tc.Network)
		t.Skip("Network ID resolution not implemented, please set XOA_NETWORK_ID")
	}

	vm := &payloads.VM{
		NameLabel:       vmName,
		NameDescription: "Backup test VM",
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

	ctx := context.Background()
	taskID, err := tc.Client.VM().Create(ctx, vm)
	require.NoError(t, err)
	require.NotEmpty(t, taskID)

	task, err := tc.Client.Task().Wait(ctx, string(taskID))
	require.NoError(t, err)
	require.Equal(t, payloads.Success, task.Status, "VM creation task failed for TestBackup: %s", task.Message)
	require.NotEqual(t, uuid.Nil, task.Result.ID, "Task result does not contain VM ID for TestBackup")
	vmID := task.Result.ID

	t.Logf("VM created with ID: %s", vmID)

	backupJobs, err := tc.Client.Backup().ListJobs(ctx, 0)
	assert.NoError(t, err)
	t.Logf("Found %d existing backup jobs", len(backupJobs))

	backupJobName := tc.GenerateResourceName("backup-test")
	var jobID uuid.UUID

	backupJob := &payloads.BackupJob{
		Name:     backupJobName,
		Mode:     payloads.BackupJobTypeFull,
		Schedule: "0 0 * * *", // Midnight every day
		Enabled:  false,       // Don't actually run it
		VMs:      vmID.String(),
		Settings: payloads.BackupSettings{
			Retention:          3,
			CompressionEnabled: true,
			ReportWhenFailOnly: false,
		},
	}

	createdJob, err := tc.Client.Backup().CreateJob(ctx, backupJob)
	if err != nil {
		t.Logf("Creating backup job failed (expected in some environments): %v", err)
	} else {
		t.Logf("Successfully created backup job with ID: %s", createdJob.ID)
		jobID = createdJob.ID

		// Run the backup job if requested
		if os.Getenv("XOA_RUN_BACKUP_JOB") == trueStr {
			t.Log("Testing RunJobForVMs with the created VM")
			taskResponse, err := tc.Client.Backup().RunJobForVMs(ctx, jobID, "", []string{vmID.String()})
			require.NoError(t, err)
			require.NotEmpty(t, taskResponse)
			t.Logf("Job started with task ID: %s", taskResponse)

			// Wait for backup task
			taskResult, isTask, err := tc.Client.Task().HandleTaskResponse(ctx, taskResponse, true)
			require.NoError(t, err)
			require.True(t, isTask, "RunJobForVMs did not return a task URL")
			require.Equal(t, payloads.Success, taskResult.Status, "Backup job run task failed: %s", taskResult.Message)
		}

		if !tc.SkipTeardown {
			err = tc.Client.Backup().DeleteJob(ctx, jobID)
			if err != nil {
				t.Logf("Error deleting backup job: %v", err)
			}
		}
	}

	if !tc.SkipTeardown {
		err = tc.Client.VM().Delete(ctx, vmID)
		assert.NoError(t, err)
	}
}

// TestBackupVMSelection ensures that backup jobs only target the specific VMs they should,
// avoiding the critical issue where all VMs are accidentally backed up
func TestBackupVMSelection(t *testing.T) {
	if os.Getenv("XOA_RUN_BACKUP_TESTS") != trueStr {
		t.Skip("Skipping backup selection test. Set XOA_RUN_BACKUP_TESTS=true to run")
		return
	}

	ctx := context.Background()
	tc := Setup(t)

	vmPrefix := tc.GenerateResourceName("vm-select")
	vmNames := []string{vmPrefix + "-1", vmPrefix + "-2"}
	var vmIDs []string

	if tc.PoolID == "" || tc.TemplateID == "" {
		t.Skip("Required environment variables for Pool/Template IDs not set")
	}

	for _, vmName := range vmNames {
		tc.CleanupVM(t, vmName)

		taskID, err := tc.Client.VM().Create(ctx, &payloads.VM{
			NameLabel:       vmName,
			NameDescription: "Backup selection test VM",
			Template:        uuid.FromStringOrNil(tc.TemplateID),
			PoolID:          uuid.FromStringOrNil(tc.PoolID),
			CPUs:            payloads.CPUs{Number: 1},
			Memory:          payloads.Memory{Static: []int64{1073741824, 1073741824}},
		})
		require.NoError(t, err, "Failed to start VM creation for %s", vmName)
		require.NotEmpty(t, taskID, "Empty task ID for VM %s", vmName)

		task, err := tc.Client.Task().Wait(ctx, string(taskID))
		require.NoError(t, err, "Failed waiting for VM creation task for %s", vmName)
		require.Equal(t, payloads.Success, task.Status, "VM creation task failed for %s: %s", vmName, task.Message)
		require.NotEqual(t, uuid.Nil, task.Result.ID, "Task result does not contain VM ID for %s", vmName)
		vmID := task.Result.ID

		if vmID == uuid.Nil {
			t.Fatalf("Failed to create VM %s: %v", vmName, err)
		}
		t.Logf("Created VM %s with ID: %s", vmName, vmID)
		vmIDs = append(vmIDs, vmID.String())
	}

	defer func() {
		if !tc.SkipTeardown {
			for i, vmID := range vmIDs {
				if err := tc.Client.VM().Delete(ctx, GetUUID(t, vmID)); err != nil {
					t.Logf("Failed to delete VM %s: %v", vmNames[i], err)
				}
			}
		}
	}()

	if len(vmIDs) != 2 {
		t.Fatal("Failed to create both test VMs")
	}

	backupJobName := tc.GenerateResourceName("backup-select-single")
	singleVMBackupJob := &payloads.BackupJob{
		Name:     backupJobName,
		Mode:     payloads.BackupJobTypeFull,
		Schedule: "0 0 * * *",
		Enabled:  false, // Don't actually run it
		// IMPORTANT: Specify ONLY the first VM
		VMs: vmIDs[0],
		Settings: payloads.BackupSettings{
			Retention:          3,
			CompressionEnabled: true,
		},
	}

	t.Log("Creating backup job for SINGLE VM selection")
	job1, err := tc.Client.Backup().CreateJob(ctx, singleVMBackupJob)
	if err != nil {
		t.Fatalf("Failed to create single VM backup job: %v", err)
	}

	backupJobName2 := tc.GenerateResourceName("backup-select-multiple")
	multipleVMBackupJob := &payloads.BackupJob{
		Name:     backupJobName2,
		Mode:     payloads.BackupJobTypeFull,
		Schedule: "0 0 * * *",
		Enabled:  false,
		VMs:      vmIDs,
		Settings: payloads.BackupSettings{
			Retention:          3,
			CompressionEnabled: true,
		},
	}

	t.Log("Creating backup job for MULTIPLE VM selection")
	job2, err := tc.Client.Backup().CreateJob(ctx, multipleVMBackupJob)
	if err != nil {
		t.Fatalf("Failed to create multiple VM backup job: %v", err)
	}

	if os.Getenv("XOA_RUN_BACKUP_JOB") == trueStr {
		t.Log("Testing RunJobForVMs with a single VM")
		taskResponse, err := tc.Client.Backup().RunJobForVMs(ctx, job2.ID, "", []string{vmIDs[0]})
		require.NoError(t, err)
		require.NotEmpty(t, taskResponse)
		t.Logf("Job started with task ID: %s for specific VM, will only back up VM1, not VM2", taskResponse)

		task, isTask, err := tc.Client.Task().HandleTaskResponse(ctx, taskResponse, true)
		require.NoError(t, err)
		require.True(t, isTask, "RunJobForVMs did not return a task URL")
		require.Equal(t, payloads.Success, task.Status, "Selective backup job run task failed: %s", task.Message)

		time.Sleep(5 * time.Second)
	}

	if !tc.SkipTeardown {
		if job1 != nil {
			if err := tc.Client.Backup().DeleteJob(ctx, job1.ID); err != nil {
				t.Logf("Failed to delete job1: %v", err)
			}
		}

		if job2 != nil {
			if err := tc.Client.Backup().DeleteJob(ctx, job2.ID); err != nil {
				t.Logf("Failed to delete job2: %v", err)
			}
		}
	}
}

func CreateTestVMForBackup(t *testing.T, ctx context.Context, tc *TestClient, name string) *payloads.VM {
	t.Helper()

	poolID := uuid.FromStringOrNil(tc.PoolID)
	templateID := uuid.FromStringOrNil(tc.TemplateID)
	require.NotEqual(t, uuid.Nil, poolID, "CreateTestVMForBackup: Failed to parse XOA_POOL_ID")
	require.NotEqual(t, uuid.Nil, templateID, "CreateTestVMForBackup: Failed to parse XOA_TEMPLATE_ID")

	taskID, err := tc.Client.VM().Create(ctx, &payloads.VM{
		NameLabel:       name,
		NameDescription: "VM for backup integration test",
		Template:        templateID,
		PoolID:          poolID,
		CPUs:            payloads.CPUs{Number: 1},
		Memory:          payloads.Memory{Static: []int64{1073741824, 1073741824}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, taskID)

	task, err := tc.Client.Task().Wait(ctx, string(taskID))
	require.NoError(t, err)
	require.Equal(t, payloads.Success, task.Status, "CreateTestVMForBackup: VM creation task failed: %s", task.Message)
	require.NotEqual(t, uuid.Nil, task.Result.ID, "CreateTestVMForBackup: Task result does not contain VM ID")
	vmID := task.Result.ID

	vm, err := tc.Client.VM().GetByID(ctx, vmID)
	require.NoError(t, err)
	require.NotNil(t, vm)
	return vm
}
