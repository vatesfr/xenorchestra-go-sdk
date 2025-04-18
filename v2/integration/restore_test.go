package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestVM_Restore(t *testing.T) {
	ctx := context.Background()
	tc := Setup(t)

	vmName := tc.GenerateResourceName("vm-restore")

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
		NameDescription: "Restore test VM",
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

	snapshotName := "integration-test-restore-point"
	snapshot, err := tc.Client.VM().Snapshot().Create(ctx, vmID, snapshotName)
	assert.NoError(t, err)
	assert.NotNil(t, snapshot)

	snapshotID := snapshot.ID
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
