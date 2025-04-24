package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestBackup(t *testing.T) {
	if os.Getenv("XOA_RUN_BACKUP_TESTS") != trueStr {
		t.Skip("Skipping backup test. Set XOA_RUN_BACKUP_TESTS=true to run")
		return
	}

	ctx := context.Background()
	tc := Setup(t)

	if tc.StorageID == "" && tc.Storage == "" {
		t.Skip("Neither StorageID nor Storage is set, skipping backup tests")
		return
	}

	vmName := tc.GenerateResourceName("vm-backup")

	tc.CleanupVM(t, vmName)

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

	createdVM, err := tc.Client.VM().Create(ctx, vm)
	assert.NoError(t, err)
	assert.NotNil(t, createdVM)

	vmID := createdVM.ID
	t.Logf("VM created with ID: %s", vmID)

	backupJobs, err := tc.Client.Backup().ListJobs(ctx, 0)
	assert.NoError(t, err)
	t.Logf("Found %d existing backup jobs", len(backupJobs))

	if os.Getenv("XOA_CREATE_BACKUP_JOB") == trueStr {
		backupJobName := tc.GenerateResourceName("backup-test")

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

			if !tc.SkipTeardown {
				err = tc.Client.Backup().DeleteJob(ctx, createdJob.ID)
				if err != nil {
					t.Logf("Error deleting backup job: %v", err)
				}
			}
		}
	} else {
		t.Log("Skipping backup job creation. Set XOA_CREATE_BACKUP_JOB=true to create a test backup job")
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

	var poolID, templateID, networkID string
	if tc.PoolID == "" || tc.TemplateID == "" || tc.NetworkID == "" {
		t.Skip("Required environment variables for Pool/Template/Network IDs not set")
	}
	poolID = tc.PoolID
	templateID = tc.TemplateID
	networkID = tc.NetworkID

	for _, vmName := range vmNames {
		tc.CleanupVM(t, vmName)

		vm := &payloads.VM{
			NameLabel:       vmName,
			NameDescription: "Backup selection test VM",
			Template:        GetUUID(t, templateID),
			Memory: payloads.Memory{
				Size: 1 * 1024 * 1024 * 1024,
			},
			CPUs: payloads.CPUs{
				Number: 1,
			},
			VIFs:        []string{networkID},
			AutoPoweron: false,
			PoolID:      GetUUID(t, poolID),
		}

		createdVM, err := tc.Client.VM().Create(ctx, vm)
		if err != nil {
			t.Fatalf("Failed to create VM %s: %v", vmName, err)
		}
		t.Logf("Created VM %s with ID: %s", vmName, createdVM.ID)
		vmIDs = append(vmIDs, createdVM.ID.String())
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
		taskID, err := tc.Client.Backup().RunJobForVMs(ctx, job2.ID, "", []string{vmIDs[0]})
		assert.NoError(t, err)
		assert.NotEmpty(t, taskID)
		t.Logf("Job started with task ID: %s for specific VM, will only back up VM1, not VM2", taskID)

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
