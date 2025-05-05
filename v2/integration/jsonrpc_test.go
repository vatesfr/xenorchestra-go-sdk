package integration

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestJSONRPC_Integration(t *testing.T) {
	ctx := context.Background()
	tc := Setup(t)

	t.Run("create VM and poll task", func(t *testing.T) {
		vmName := tc.GenerateResourceName("vm-jsonrpc")
		t.Cleanup(func() { tc.CleanupVM(t, vmName) })

		poolID := uuid.FromStringOrNil(tc.PoolID)
		templateID := uuid.FromStringOrNil(tc.TemplateID)
		require.NotEqual(t, uuid.Nil, poolID, "Failed to parse XOA_POOL_ID")
		require.NotEqual(t, uuid.Nil, templateID, "Failed to parse XOA_TEMPLATE_ID")

		vm := &payloads.VM{
			NameLabel:       vmName,
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
		require.Equal(t, vmName, createdVM.NameLabel)
		t.Logf("VM created with ID: %s", vmID)

		if !tc.SkipTeardown {
			err := tc.Client.VM().Delete(ctx, vmID)
			if err != nil {
				t.Logf("Failed to delete VM %s: %v", vmID, err)
			}
		}
	})
}
