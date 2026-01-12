package integration

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

var vmsIDs []string

func createVMsForTest(t *testing.T, ctx context.Context, count int, name string) []string {
	vmIDs := make([]string, 0, count)
	for i := 0; i < count; i++ {
		vmName := integrationTestPrefix + name + uuid.Must(uuid.NewV4()).String()
		params := payloads.CreateVMParams{
			NameLabel: vmName,
			Template:  uuid.FromStringOrNil(testTemplate.Id),
		}

		vmID, err := testClient.Pool().CreateVM(ctx, testPool.ID, params)
		require.NoErrorf(t, err, "error while creating VM %s in pool %s: %v", vmName, testPool.ID, err)
		require.NotEqual(t, uuid.Nil, vmID, "created VM ID should not be nil")
		vmIDs = append(vmIDs, vmID.String())
	}

	return vmIDs
}

func TestVmCreation(t *testing.T) {
	ctx, cleanup := SetupTestContext(t)
	defer cleanup()

	vmName := integrationTestPrefix + "creation-test-" + uuid.Must(uuid.NewV4()).String()
	params := &payloads.CreateVMParams{
		NameLabel: vmName,
		Template:  uuid.FromStringOrNil(testTemplate.Id),
	}

	vmID, err := testClient.VM().Create(ctx, testPool.ID, params)
	require.NoErrorf(t, err, "error while creating VM %s in pool %s: %v", vmName, testPool.ID, err)
	require.NotEqual(t, uuid.Nil, vmID, "created VM ID should not be nil")
}

func TestVMs(t *testing.T) {
	ctx, cleanup := SetupTestContext(t)
	defer cleanup()

	// Setup that runs before all VM subtests
	vmsIDs = createVMsForTest(t, ctx, 2, "test-vms-A-")
	vmsIDs = createVMsForTest(t, ctx, 3, "test-vms-B-")

	// TODO: run subtests in parallel but this require refactoring the setup/teardown
	//  to avoid cleanup removing VMs created by other tests
	t.Run("ListWithLimit", TestVMListWithLimit)
	t.Run("TestVMListWithNoLimit", TestVMListWithNoLimit)
	t.Run("TestVMListWithFilter", TestVMListWithFilter)

	// Teardown - cleanup created VMs
	_ = cleanupVMsWithPrefix(integrationTestPrefix + "test-vms-")
}
func TestVMListWithLimit(t *testing.T) {
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
			vms, err := testClient.VM().GetAll(t.Context(), tc.n, "")
			require.NoError(t, err)
			require.NotNil(t, vms)
			assert.LessOrEqual(t, len(vms), tc.n)
		})
	}
}

func TestVMListWithNoLimit(t *testing.T) {
	vms, err := testClient.VM().GetAll(t.Context(), 0, "")
	require.NoError(t, err)
	require.NotNil(t, vms)
	// Adjust expectation - we now create only 5 VMs, but there might be other VMs in the system
	assert.Greater(t, len(vms), 5, "expected more than 5 VMs in total")
}

func TestVMListWithFilter(t *testing.T) {
	vms, err := testClient.VM().GetAll(t.Context(), 0, "name_label:"+integrationTestPrefix+"test-vms-A-")
	require.NoError(t, err)
	require.NotNil(t, vms)
	assert.Len(t, vms, 2, "expected 2 VMs with the specified name label")
}
