package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestVM_Snapshot(t *testing.T) {
	ctx := context.Background()
	tc := Setup(t)

	vmName := tc.GenerateResourceName("vm-snapshot")

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
		NameDescription: "Snapshot test VM",
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

	snapshotName := "integration-test-snapshot"
	snapshot, err := tc.Client.VM().Snapshot().Create(ctx, vmID, snapshotName)
	assert.NoError(t, err)
	assert.NotNil(t, snapshot)
	assert.Equal(t, snapshotName, snapshot.NameLabel)
	assert.Equal(t, vmID, snapshot.VmID)

	snapshotID := snapshot.ID
	t.Logf("Snapshot created with ID: %s", snapshotID)

	retrievedSnapshot, err := tc.Client.VM().Snapshot().GetByID(ctx, snapshotID)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedSnapshot)
	assert.Equal(t, snapshotName, retrievedSnapshot.NameLabel)

	snapshots, err := tc.Client.VM().Snapshot().ListByVM(ctx, vmID, 0)
	assert.NoError(t, err)

	var found bool
	for _, s := range snapshots {
		if s.ID == snapshotID {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected to find the created snapshot in the list")

	if os.Getenv("XOA_ALLOW_SNAPSHOT_REVERT") == "true" {
		err = tc.Client.VM().Snapshot().Revert(ctx, vmID, snapshotID)
		assert.NoError(t, err)

		time.Sleep(5 * time.Second)
	}

	err = tc.Client.VM().Snapshot().Delete(ctx, snapshotID)
	assert.NoError(t, err)

	snapshots, err = tc.Client.VM().Snapshot().ListByVM(ctx, vmID, 0)
	assert.NoError(t, err)

	found = false
	for _, s := range snapshots {
		if s.ID == snapshotID {
			found = true
			break
		}
	}
	assert.False(t, found, "Expected the snapshot to be deleted")

	if !tc.SkipTeardown {
		err = tc.Client.VM().Delete(ctx, vmID)
		assert.NoError(t, err)
	}
}
