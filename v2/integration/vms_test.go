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

func TestVMs(t *testing.T) {
	ctx, client, testPrefix := SetupTestContext(t)

	// Setup that runs before all VM subtests
	createVMsForTest(t, ctx, client.Pool(), 2, testPrefix+"test-vms-A-")
	createVMsForTest(t, ctx, client.Pool(), 3, testPrefix+"test-vms-B-")

	// TODO: run subtests in parallel but this require refactoring the setup/teardown
	//  to avoid cleanup removing VMs created by other tests
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
