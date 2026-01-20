package integration

import (
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestVmCreation(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := testPrefix + "creation-test-" + uuid.Must(uuid.NewV4()).String()
	params := &payloads.CreateVMParams{
		NameLabel: vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
	}

	vmID, err := client.VM().Create(ctx, intTests.testPool.ID, params)
	require.NoErrorf(t, err, "error while creating VM %s in pool %s: %v", vmName, intTests.testPool.ID, err)
	require.NotEqual(t, uuid.Nil, vmID, "created VM ID should not be nil")
}

func TestVmDeletion(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := testPrefix + "deletion-test-" + uuid.Must(uuid.NewV4()).String()
	params := &payloads.CreateVMParams{
		NameLabel: vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
	}

	vm, err := client.VM().Create(ctx, intTests.testPool.ID, params)
	require.NoErrorf(t, err, "error while creating VM %s in pool %s: %v", vmName, intTests.testPool.ID, err)
	require.NotNil(t, vm, "created VM should not be nil")

	err = client.VM().Delete(ctx, vm.ID)
	require.NoErrorf(t, err, "error while deleting VM %s: %v", vm.ID, err)

	// Verify the VM is gone
	_, err = client.VM().GetByID(ctx, vm.ID)
	require.Error(t, err, "expected error when getting deleted VM")
}

func TestVmStart(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := testPrefix + "start-test-" + uuid.Must(uuid.NewV4()).String()
	params := &payloads.CreateVMParams{
		NameLabel: vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
	}

	vm, err := client.VM().Create(ctx, intTests.testPool.ID, params)
	require.NoError(t, err)
	require.NotNil(t, vm)
	assert.Equal(t, payloads.PowerStateHalted, vm.PowerState, "newly created VM should be halted")

	// Since Start might be asynchronous/background task in XO, we might need to poll
	// but for now let's see if it's already Running or wait a bit
	taskID, err := client.VM().Start(ctx, vm.ID, nil)
	require.NoError(t, err)
	task := waitForTask(t, ctx, client, taskID)
	require.Equal(t, payloads.Success, task.Status, "start task should succeed")

	v, err := client.VM().GetByID(ctx, vm.ID)
	require.NoError(t, err)
	assert.Equal(t, payloads.PowerStateRunning, v.PowerState, "VM should be running after start")
}

func TestVmHardShutdown(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := testPrefix + "hard-shutdown-test-" + uuid.Must(uuid.NewV4()).String()
	params := &payloads.CreateVMParams{
		NameLabel: vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		Boot:      ptr(true),
	}

	vm, err := client.VM().Create(ctx, intTests.testPool.ID, params)
	require.NoError(t, err)

	// Hard Shutdown
	taskID, err := client.VM().HardShutdown(ctx, vm.ID)
	require.NoError(t, err)
	task := waitForTask(t, ctx, client, taskID)
	require.Equal(t, payloads.Success, task.Status, "hard shutdown task should succeed")

	v, err := client.VM().GetByID(ctx, vm.ID)
	require.NoError(t, err)
	assert.Equal(t, payloads.PowerStateHalted, v.PowerState, "VM should be halted after hard shutdown")
}

func TestVmCleanShutdown(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := testPrefix + "clean-shutdown-test-" + uuid.Must(uuid.NewV4()).String()
	params := &payloads.CreateVMParams{
		NameLabel: vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		Boot:      ptr(true),
	}

	vm, err := client.VM().Create(ctx, intTests.testPool.ID, params)
	require.NoError(t, err)

	// Wait for agent to be ready
	waitForVMReady(t, ctx, client, vm.ID)

	// Clean Shutdown
	taskID, err := client.VM().CleanShutdown(ctx, vm.ID)
	require.NoError(t, err)
	task := waitForTask(t, ctx, client, taskID)
	require.Equal(t, payloads.Success, task.Status, "clean shutdown task should succeed")

	v, err := client.VM().GetByID(ctx, vm.ID)
	require.NoError(t, err)
	assert.Equal(t, payloads.PowerStateHalted, v.PowerState, "VM should be halted after clean shutdown")
}

func TestVmCleanReboot(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := testPrefix + "clean-reboot-test-" + uuid.Must(uuid.NewV4()).String()
	params := &payloads.CreateVMParams{
		NameLabel: vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		Boot:      ptr(true),
	}

	vm, err := client.VM().Create(ctx, intTests.testPool.ID, params)
	require.NoError(t, err)

	// Wait for agent to be ready
	waitForVMReady(t, ctx, client, vm.ID)

	// Clean Reboot
	taskID, err := client.VM().CleanReboot(ctx, vm.ID)
	require.NoError(t, err)
	task := waitForTask(t, ctx, client, taskID)
	require.Equal(t, payloads.Success, task.Status, "clean reboot task should succeed")

	v, err := client.VM().GetByID(ctx, vm.ID)
	require.NoError(t, err)
	assert.Equal(t, payloads.PowerStateRunning, v.PowerState, "VM should be running after clean reboot")
}

func TestVmHardReboot(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := testPrefix + "hard-reboot-test-" + uuid.Must(uuid.NewV4()).String()
	params := &payloads.CreateVMParams{
		NameLabel: vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		Boot:      ptr(true),
	}

	vm, err := client.VM().Create(ctx, intTests.testPool.ID, params)
	require.NoError(t, err)

	// Hard Reboot
	taskID, err := client.VM().HardReboot(ctx, vm.ID)
	require.NoError(t, err)
	task := waitForTask(t, ctx, client, taskID)
	require.Equal(t, payloads.Success, task.Status, "hard reboot task should succeed")

	v, err := client.VM().GetByID(ctx, vm.ID)
	require.NoError(t, err)
	assert.Equal(t, payloads.PowerStateRunning, v.PowerState, "VM should be running after hard reboot")
}

func TestVmPauseUnpause(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := testPrefix + "pause-test-" + uuid.Must(uuid.NewV4()).String()
	params := &payloads.CreateVMParams{
		NameLabel: vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		Boot:      ptr(true),
	}

	vm, err := client.VM().Create(ctx, intTests.testPool.ID, params)
	require.NoError(t, err)

	// Pause
	taskID, err := client.VM().Pause(ctx, vm.ID)
	require.NoError(t, err)
	task := waitForTask(t, ctx, client, taskID)
	require.Equal(t, payloads.Success, task.Status, "pause task should succeed")

	v, err := client.VM().GetByID(ctx, vm.ID)
	require.NoError(t, err)
	assert.Equal(t, payloads.PowerStatePaused, v.PowerState, "VM should be paused after pause")

	// Unpause
	taskID, err = client.VM().Unpause(ctx, vm.ID)
	require.NoError(t, err)
	task = waitForTask(t, ctx, client, taskID)
	require.Equal(t, payloads.Success, task.Status, "unpause task should succeed")

	v, err = client.VM().GetByID(ctx, vm.ID)
	require.NoError(t, err)
	assert.Equal(t, payloads.PowerStateRunning, v.PowerState, "VM should be running after unpause")
}

func TestVmSuspendResume(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := testPrefix + "suspend-test-" + uuid.Must(uuid.NewV4()).String()
	params := &payloads.CreateVMParams{
		NameLabel: vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		Boot:      ptr(true),
	}

	vm, err := client.VM().Create(ctx, intTests.testPool.ID, params)
	require.NoError(t, err)

	waitForVMReady(t, ctx, client, vm.ID)

	// Suspend
	taskID, err := client.VM().Suspend(ctx, vm.ID)
	require.NoError(t, err)
	task := waitForTask(t, ctx, client, taskID)
	require.Equal(t, payloads.Success, task.Status, "suspend task should succeed")

	v, err := client.VM().GetByID(ctx, vm.ID)
	require.NoError(t, err)
	assert.Equal(t, payloads.PowerStateSuspended, v.PowerState, "VM should be suspended after suspend")

	// Resume
	taskID, err = client.VM().Resume(ctx, vm.ID)
	require.NoError(t, err)
	task = waitForTask(t, ctx, client, taskID)
	require.Equal(t, payloads.Success, task.Status, "resume task should succeed")

	v, err = client.VM().GetByID(ctx, vm.ID)
	require.NoError(t, err)
	assert.Equal(t, payloads.PowerStateRunning, v.PowerState, "VM should be running after resume")
}

func TestVmSnapshot(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := testPrefix + "snapshot-test-" + uuid.Must(uuid.NewV4()).String()
	params := &payloads.CreateVMParams{
		NameLabel: vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		Boot:      ptr(true),
	}

	vm, err := client.VM().Create(ctx, intTests.testPool.ID, params)
	require.NoError(t, err)

	// Snapshot
	snapshotName := vmName + "-snapshot"
	taskID, err := client.VM().Snapshot(ctx, vm.ID, snapshotName)
	require.NoError(t, err)
	task := waitForTask(t, ctx, client, taskID)
	require.Equal(t, payloads.Success, task.Status, "snapshot task should succeed")

	// Verify snapshot exists
	// TODO: check the snapshot details more thoroughly
	vm, err = client.VM().GetByID(ctx, vm.ID)
	require.NoError(t, err)
	assert.Len(t, vm.Snapshots, 1, "vm should have one snapshot")
}

func TestVMs(t *testing.T) {
	ctx, client, testPrefix := SetupTestContext(t)

	// Setup that runs before all VM subtests
	createVMsForTest(t, ctx, client.Pool(), 2, testPrefix+"test-vms-A-")
	createVMsForTest(t, ctx, client.Pool(), 3, testPrefix+"test-vms-B-")

	t.Run("ListWithLimit", testVMListWithLimit)
	t.Run("TestVMListWithNoLimit", testVMListWithNoLimit)
	t.Run("TestVMListWithFilter", testVMListWithFilter)
}

func testVMListWithLimit(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)
	t.Parallel()
	testCases := []struct {
		name string
		n    int
	}{
		{"n=1", 1},
		{"n=3", 3},
		{"n=5", 5},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Test with limit parameter
			vms, err := client.VM().GetAll(ctx, tc.n, "")
			require.NoError(t, err)
			require.NotNil(t, vms)
			assert.LessOrEqual(t, len(vms), tc.n)
		})
	}
}

func testVMListWithNoLimit(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)
	t.Parallel()
	vms, err := client.VM().GetAll(ctx, 0, "")
	require.NoError(t, err)
	require.NotNil(t, vms)
	// Adjust expectation - we now create only 5 VMs, but there might be other VMs in the system
	assert.Greater(t, len(vms), 5, "expected more than 5 VMs in total")
}

func testVMListWithFilter(t *testing.T) {
	ctx, client, testPrefix := SetupTestContext(t)
	t.Parallel()

	// Test prefix is based on test name, extract main part as we are in a subtest
	mainTestPrefix := strings.Split(testPrefix, "/")

	// Filter by the specific prefix just created
	vms, err := client.VM().GetAll(ctx, 0, "name_label:"+mainTestPrefix[0]+"-test-vms-A-")
	require.NoError(t, err)
	require.NotNil(t, vms)
	assert.Len(t, vms, 2, "expected 2 VMs with the specified name label")
}
