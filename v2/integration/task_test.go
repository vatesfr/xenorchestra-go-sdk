package integration

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
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

		vm := createTestVM(t, ctx, tc, vmName, poolID, templateID, networkID)
		defer cleanupVM(t, ctx, tc, vm.ID)

		snapshotName := "integration-task-test-snapshot"
		snapshot, err := tc.Client.VM().Snapshot().Create(ctx, vm.ID, snapshotName)
		assert.NoError(t, err)
		assert.NotNil(t, snapshot)
		assert.Equal(t, snapshotName, snapshot.NameLabel)

		err = tc.Client.VM().Snapshot().Delete(ctx, snapshot.ID)
		assert.NoError(t, err)
	})
}

func createTestVM(t *testing.T, ctx context.Context, tc *TestClient, name, poolID, templateID, networkID string) *payloads.VM {
	t.Helper()

	vm := &payloads.VM{
		NameLabel:       name,
		NameDescription: "Task test VM",
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

// Helper function to clean up a VM
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
