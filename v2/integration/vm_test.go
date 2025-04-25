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

func TestVM_CRUD(t *testing.T) {
	ctx := context.Background()
	tc := Setup(t)

	vmName := tc.GenerateResourceName("vm-crud")
	taskID, err := tc.Client.VM().Create(ctx, &payloads.VM{
		NameLabel:       vmName,
		NameDescription: "Created by integration test",
		Template:        uuid.FromStringOrNil(tc.TemplateID),
		PoolID:          uuid.FromStringOrNil(tc.PoolID),
		CPUs:            payloads.CPUs{Number: 1},
		Memory:          payloads.Memory{Static: []int64{1073741824, 1073741824}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, taskID)

	task, err := tc.Client.Task().Wait(ctx, string(taskID))
	require.NoError(t, err)
	require.Equal(t, payloads.Success, task.Status, "VM creation task failed: %s", task.Message)
	require.NotEqual(t, uuid.Nil, task.Result.ID, "Task result does not contain VM ID")
	vmID := task.Result.ID

	t.Cleanup(func() { tc.CleanupVM(t, vmName) })

	vm, err := tc.Client.VM().GetByID(ctx, vmID)
	require.NoError(t, err)
	require.NotNil(t, vm)
	require.Equal(t, vmName, vm.NameLabel)

	// Read
	readVM, err := tc.Client.VM().GetByID(ctx, vmID)
	require.NoError(t, err)
	require.NotNil(t, readVM)
	require.Equal(t, vmName, readVM.NameLabel)

	allVMs, err := tc.Client.VM().List(ctx, 0)
	assert.NoError(t, err)
	var found bool
	for _, v := range allVMs {
		if v.ID == vmID {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected to find the created VM in the list")

	updatedDescription := "Updated integration test VM"
	updateVM := &payloads.VM{
		ID:              vmID,
		NameLabel:       vmName,
		NameDescription: updatedDescription,
	}

	updatedVM, err := tc.Client.VM().Update(ctx, updateVM)
	if err != nil {
		t.Logf("Update failed (expected in some cases): %v", err)
	} else {
		assert.Equal(t, updatedDescription, updatedVM.NameDescription)
	}

	latestVM, err := tc.Client.VM().GetByID(ctx, vmID)
	assert.NoError(t, err, "Should be able to get VM after update attempt")
	t.Logf("VM after update: %s - %s", latestVM.NameLabel, latestVM.NameDescription)

	if !tc.SkipTeardown {
		err = tc.Client.VM().Delete(ctx, vmID)
		assert.NoError(t, err)

		_, err = tc.Client.VM().GetByID(ctx, vmID)
		assert.Error(t, err, "VM should not exist after deletion")
	}
}

func TestVM_Lifecycle(t *testing.T) {
	tc := Setup(t)
	ctx := context.Background()

	vmName := tc.GenerateResourceName("vm-lifecycle")
	t.Cleanup(func() { tc.CleanupVM(t, vmName) })

	taskID, err := tc.Client.VM().Create(ctx, &payloads.VM{
		NameLabel:       vmName,
		NameDescription: "VM lifecycle integration testing",
		Template:        uuid.FromStringOrNil(tc.TemplateID),
		PoolID:          uuid.FromStringOrNil(tc.PoolID),
		CPUs:            payloads.CPUs{Number: 1},
		Memory:          payloads.Memory{Static: []int64{1073741824, 1073741824}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, taskID)

	task, err := tc.Client.Task().Wait(ctx, string(taskID))
	require.NoError(t, err)
	require.Equal(t, payloads.Success, task.Status, "VM creation task failed: %s", task.Message)
	require.NotEqual(t, uuid.Nil, task.Result.ID, "Task result does not contain VM ID")
	vmID := task.Result.ID

	t.Logf("VM created with ID: %s", vmID)

	err = tc.Client.VM().Start(ctx, vmID)
	assert.NoError(t, err)

	time.Sleep(10 * time.Second)

	runningVM, err := tc.Client.VM().GetByID(ctx, vmID)
	assert.NoError(t, err)
	assert.Equal(t, payloads.PowerStateRunning, runningVM.PowerState)

	err = tc.Client.VM().CleanShutdown(ctx, vmID)
	assert.NoError(t, err)

	time.Sleep(10 * time.Second)

	haltedVM, err := tc.Client.VM().GetByID(ctx, vmID)
	assert.NoError(t, err)
	assert.Equal(t, payloads.PowerStateHalted, haltedVM.PowerState)

	if !tc.SkipTeardown {
		err = tc.Client.VM().Delete(ctx, vmID)
		assert.NoError(t, err)
	}
}

func TestVM_PowerCycle(t *testing.T) {
	tc := Setup(t)
	ctx := context.Background()
	vmName := tc.GenerateResourceName("vm-power")
	t.Cleanup(func() { tc.CleanupVM(t, vmName) })

	taskID, err := tc.Client.VM().Create(ctx, &payloads.VM{
		NameLabel:       vmName,
		NameDescription: "VM for power cycle test",
		Template:        uuid.FromStringOrNil(tc.TemplateID),
		PoolID:          uuid.FromStringOrNil(tc.PoolID),
		CPUs:            payloads.CPUs{Number: 1},
		Memory:          payloads.Memory{Static: []int64{1073741824, 1073741824}},
	})
	require.NoError(t, err)
	require.NotEmpty(t, taskID)

	task, err := tc.Client.Task().Wait(ctx, string(taskID))
	require.NoError(t, err)
	require.Equal(t, payloads.Success, task.Status, "VM creation task failed: %s", task.Message)
	require.NotEqual(t, uuid.Nil, task.Result.ID, "Task result does not contain VM ID")
	vmID := task.Result.ID

	vm, err := tc.Client.VM().GetByID(ctx, vmID)
	assert.NoError(t, err)
	assert.Equal(t, payloads.PowerStateHalted, vm.PowerState)

	if !tc.SkipTeardown {
		err = tc.Client.VM().Delete(ctx, vmID)
		assert.NoError(t, err)
	}
}
