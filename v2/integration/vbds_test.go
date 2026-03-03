package integration

import (
	"testing"
	"time"

	"github.com/docker/go-units"
	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestVBDGet(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	// Create a VM — it comes with at least one system-disk VBD.
	vm, err := client.VM().Create(ctx, intTests.testPool.ID, &payloads.CreateVMParams{
		NameLabel: testPrefix + "vbd-get-vm",
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
	})
	require.NoError(t, err, "creating VM should succeed")
	require.NotEmpty(t, vm.VBDs, "VM should have at least one VBD")

	vbdID := vm.VBDs[0]
	require.NotEqual(t, uuid.Nil, vbdID, "VM's first VBD ID should be valid")

	t.Run("GetByValidID", func(t *testing.T) {
		t.Parallel()
		vbd, err := client.VBD().Get(ctx, vbdID)
		require.NoError(t, err, "fetching VBD by valid ID should succeed")
		require.NotNil(t, vbd)
		assert.Equal(t, vbdID, vbd.UUID, "VBD UUID should match requested ID")
		assert.Equal(t, vm.ID, vbd.VM, "VBD should belong to the created VM")
	})

	t.Run("GetByInvalidID", func(t *testing.T) {
		t.Parallel()
		_, err := client.VBD().Get(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
		require.Error(t, err, "expected error when fetching VBD with non-existent ID")
	})
}

func TestVBDGetAll(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	// Create a VM with two extra data disks so we have a known set of VBDs to filter on.
	vm, err := client.VM().Create(ctx, intTests.testPool.ID, &payloads.CreateVMParams{
		NameLabel: testPrefix + "vbd-getall-vm",
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		VDIs: []payloads.VDIParams{
			{
				NameLabel: ptr(testPrefix + "vbd-getall-vdi-1"),
				Size:      ptr(int64(512 * units.MB)),
			},
			{
				NameLabel: ptr(testPrefix + "vbd-getall-vdi-2"),
				Size:      ptr(int64(512 * units.MB)),
			},
		},
	})
	require.NoError(t, err, "creating VM should succeed")
	require.GreaterOrEqual(t, len(vm.VBDs), 2, "VM should have at least 2 VBDs after creation with 2 VDIs")

	t.Run("NoLimit", func(t *testing.T) {
		t.Parallel()
		vbds, err := client.VBD().GetAll(ctx, 0, "")
		require.NoError(t, err)
		require.NotNil(t, vbds)
		assert.GreaterOrEqual(t, len(vbds), len(vm.VBDs), "GetAll should return at least as many VBDs as the created VM has")
	})

	t.Run("WithLimit", func(t *testing.T) {
		t.Parallel()
		vbds, err := client.VBD().GetAll(ctx, 1, "")
		require.NoError(t, err)
		require.NotNil(t, vbds)
		assert.Len(t, vbds, 1, "GetAll with limit=1 should return exactly one VBD")
	})

	t.Run("WithVMFilter", func(t *testing.T) {
		t.Parallel()
		vbds, err := client.VBD().GetAll(ctx, 0, "VM:"+vm.ID.String())
		require.NoError(t, err)
		require.NotNil(t, vbds)
		assert.Len(t, vbds, len(vm.VBDs), "filter by VM ID should return exactly the VM's VBDs")
		for _, vbd := range vbds {
			assert.Equal(t, vm.ID, vbd.VM, "every returned VBD should belong to the filtered VM")
		}
	})
}

func TestVBDCreate(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vm, err := client.VM().Create(ctx, intTests.testPool.ID, &payloads.CreateVMParams{
		NameLabel: testPrefix + "vbd-create-vm",
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
	})
	require.NoError(t, err, "creating VM should succeed")

	vdiID := createVDIForTest(t, ctx, client.V1Client(), testPrefix+"vbd-create-vdi", 512*units.MB)

	vbdID, err := client.VBD().Create(ctx, &payloads.CreateVBDParams{
		VM:  vm.ID,
		VDI: vdiID,
	})
	require.NoError(t, err, "creating VBD should succeed")
	require.NotEqual(t, uuid.Nil, vbdID, "created VBD ID should not be nil")

	// XO REST API has eventual consistency: wait for the VBD to become visible.
	require.Eventually(t, func() bool {
		_, err := client.VBD().Get(ctx, vbdID)
		return err == nil
	}, 30*time.Second, 1*time.Second, "VBD should become visible after creation")

	// Verify the VBD exists and has the expected fields.
	vbd, err := client.VBD().Get(ctx, vbdID)
	require.NoError(t, err, "fetching newly created VBD should succeed")
	assert.Equal(t, vm.ID, vbd.VM, "VBD should be attached to the created VM")
	require.NotNil(t, vbd.VDI, "VBD should reference a VDI")
	assert.Equal(t, vdiID, *vbd.VDI, "VBD VDI should match the created VDI")

	vm, err = client.VM().GetByID(ctx, vm.ID)
	require.NoError(t, err, "fetching VM after VBD creation should succeed")
	assert.Contains(t, vm.VBDs, vbdID, "VM's VBD list should include the newly created VBD")
}

func TestVBDDelete(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	vm, err := client.VM().Create(ctx, intTests.testPool.ID, &payloads.CreateVMParams{
		NameLabel: testPrefix + "vbd-delete-vm",
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
	})
	require.NoError(t, err, "creating VM should succeed")

	vdiID := createVDIForTest(t, ctx, client.V1Client(), testPrefix+"vbd-delete-vdi", 512*units.MB)
	vbdID := createVBDForTest(t, ctx, client, vm.ID, vdiID)

	err = client.VBD().Delete(ctx, vbdID)
	require.NoError(t, err, "deleting VBD should succeed")

	// XO REST API has eventual consistency: the VBD may still be visible briefly after deletion.
	require.Eventually(t, func() bool {
		_, err := client.VBD().Get(ctx, vbdID)
		return err != nil
	}, 30*time.Second, 1*time.Second, "VBD should not be visible after deletion")

	vdis, err := client.VM().GetVDIs(ctx, vm.ID, 0, "")
	require.NoError(t, err, "fetching VDIs after VBD deletion should succeed")
	assert.NotContains(t, vdis, vbdID, "VM's VDI list should not include the deleted VBD")
}

func TestVBDConnectDisconnect(t *testing.T) {
	t.Parallel()
	ctx, client, testPrefix := SetupTestContext(t)

	// Create and start a VM.
	vm, err := client.VM().Create(ctx, intTests.testPool.ID, &payloads.CreateVMParams{
		NameLabel: testPrefix + "vbd-connect-vm",
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		Boot:      ptr(true),
	})
	require.NoError(t, err, "creating VM should succeed")
	require.Equal(t, vm.PowerState, payloads.PowerStateRunning, "VM should be running after creation with Boot=true")

	// VBD hotplug requires PV drivers. Wait for the VM to report an IP, confirming guest tools are active.
	waitForVMReady(t, ctx, client, vm.ID)

	// Create VDI and VBD while VM is running.
	// XO auto-attaches VBDs created on running VMs when PV drivers are active.
	vdiID := createVDIForTest(t, ctx, client.V1Client(), testPrefix+"vbd-connect-vdi", 512*units.MB)
	vbdID := createVBDForTest(t, ctx, client, vm.ID, vdiID)

	// Wait for XO auto-attach to complete.
	require.Eventually(t, func() bool {
		v, err := client.VBD().Get(ctx, vbdID)
		return err == nil && v.Attached
	}, 1*time.Minute, 2*time.Second, "VBD should be auto-attached after creation on running VM")

	// Disconnect first so we have a clean state to test Connect from.
	disconnectTaskID, err := client.VBD().Disconnect(ctx, vbdID)
	require.NoError(t, err, "setup Disconnect should succeed")
	disconnectTask := waitForTask(t, ctx, client, disconnectTaskID)
	require.Equalf(t, payloads.Success, disconnectTask.Status,
		"setup disconnect task should succeed: %v", disconnectTask.Result.Message)

	require.Eventually(t, func() bool {
		v, err := client.VBD().Get(ctx, vbdID)
		return err == nil && !v.Attached
	}, 1*time.Minute, 2*time.Second, "VBD should be detached after Disconnect")

	// Connect the VBD.
	connectTaskID, err := client.VBD().Connect(ctx, vbdID)
	require.NoError(t, err, "VBD Connect should succeed")
	connectTask := waitForTask(t, ctx, client, connectTaskID)
	require.Equalf(t, payloads.Success, connectTask.Status,
		"VBD connect task should succeed: %v", connectTask.Result.Message)

	// Verify the VBD is now attached.
	require.Eventually(t, func() bool {
		v, err := client.VBD().Get(ctx, vbdID)
		return err == nil && v.Attached
	}, 1*time.Minute, 2*time.Second, "VBD should be attached after Connect")
}
