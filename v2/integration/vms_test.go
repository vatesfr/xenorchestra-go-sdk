package integration

import (
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestVmCreation(t *testing.T) {
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
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := testPrefix + "start-test-" + uuid.Must(uuid.NewV4()).String()
	params := &payloads.CreateVMParams{
		NameLabel: vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
	}

	vm, err := client.VM().Create(ctx, intTests.testPool.ID, params)
	require.NoError(t, err)
	require.NotNil(t, vm)
	assert.Equal(t, payloads.PowerStateHalted, vm.PowerState)

	// Since Start might be asynchronous/background task in XO, we might need to poll
	// but for now let's see if it's already Running or wait a bit
	taskID, err := client.VM().Start(ctx, vm.ID, nil)
	require.NoError(t, err)
	assert.Eventually(t, func() bool {
		task, err := client.Task().Get(ctx, taskID)
		if err != nil {
			return false
		}
		return task.Status == payloads.Success
	}, 60*time.Second, 5*time.Second, "Task should be successful")

	v, err := client.VM().GetByID(ctx, vm.ID)
	require.NoError(t, err)
	assert.Equal(t, payloads.PowerStateRunning, v.PowerState)
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
