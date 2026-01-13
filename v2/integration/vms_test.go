package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
)

var vmsIDs []string

func createVMsForTest(t *testing.T, ctx context.Context, pool library.Pool, count int, name string) []string {
	vmIDs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		vmName := name + uuid.Must(uuid.NewV4()).String()
		params := payloads.CreateVMParams{
			NameLabel: vmName,
			Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		}

		vmID, err := pool.CreateVM(ctx, intTests.testPool.ID, params)
		require.NoErrorf(t, err, "error while creating VM %s in pool %s: %v", vmName, intTests.testPool.ID, err)
		require.NotEqual(t, uuid.Nil, vmID, "created VM ID should not be nil")
		vmIDs = append(vmIDs, vmID.String())
	}

	return vmIDs
}

func TestVmCreation(t *testing.T) {
	ctx, cleanup, client, testPrefix := SetupTestContext(t)
	defer cleanup()

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
	ctx, cleanup, client, testPrefix := SetupTestContext(t)
	defer cleanup()

	// Setup that runs before all VM subtests
	vmsIDs = createVMsForTest(t, ctx, client.Pool(), 2, testPrefix+"test-vms-A-")
	vmsIDs = createVMsForTest(t, ctx, client.Pool(), 3, testPrefix+"test-vms-B-")

	// TODO: run subtests in parallel but this require refactoring the setup/teardown
	//  to avoid cleanup removing VMs created by other tests
	t.Run("ListWithLimit", testVMListWithLimit)
	t.Run("TestVMListWithNoLimit", testVMListWithNoLimit)
	t.Run("TestVMListWithFilter", testVMListWithFilter)
}
func testVMListWithLimit(t *testing.T) {
	ctx, _, client, _ := SetupTestContext(t)
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
			// Test with limit parameter
			vms, err := client.VM().GetAll(ctx, tc.n, "")
			require.NoError(t, err)
			require.NotNil(t, vms)
			assert.LessOrEqual(t, len(vms), tc.n)
		})
	}
}

func testVMListWithNoLimit(t *testing.T) {
	ctx, _, client, _ := SetupTestContext(t)
	vms, err := client.VM().GetAll(ctx, 0, "")
	require.NoError(t, err)
	require.NotNil(t, vms)
	// Adjust expectation - we now create only 5 VMs, but there might be other VMs in the system
	assert.Greater(t, len(vms), 5, "expected more than 5 VMs in total")
}

func testVMListWithFilter(t *testing.T) {
	ctx, cleanup, client, testPrefix := SetupTestContext(t)
	defer cleanup()

	// Test prefix is based on test name, extract main part as we are in a subtest
	mainTestPrefix := strings.Split(testPrefix, "/")

	// Filter by the specific prefix just created
	vms, err := client.VM().GetAll(ctx, 0, "name_label:"+mainTestPrefix[0]+"-test-vms-A-")
	require.NoError(t, err)
	require.NotNil(t, vms)
	assert.Len(t, vms, 2, "expected 2 VMs with the specified name label")
}
