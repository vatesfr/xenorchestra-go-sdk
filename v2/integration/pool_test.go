package integration

import (
	"os"
	"strconv"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestGetPoolByID(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	pool, err := client.Pool().Get(ctx, intTests.testPool.ID)
	if err != nil {
		t.Fatalf("error while fetching pool %s: %v", intTests.testPool.ID, err)
	}
	assert.Equal(t, intTests.testPool.ID, pool.ID, "pool ID should match")
}

func TestGetPoolByInvalidID(t *testing.T) {
	ctx, client, _ := SetupTestContext(t)

	_, err := client.Pool().Get(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
	if err == nil {
		t.Fatal("expected error when fetching pool with invalid ID, got nil")
	}
	assert.Contains(t, err.Error(), "404 Not Found", "error message should indicate not found")
}

func TestCreateVM(t *testing.T) {
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := "test-vm"
	params := payloads.CreateVMParams{
		NameLabel: testPrefix + vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
	}

	vmID, err := client.Pool().CreateVM(ctx, intTests.testPool.ID, params)
	if err != nil {
		t.Fatalf("error while creating VM in pool %s: %v", intTests.testPool.ID, err)
	}
	assert.NotEqual(t, uuid.Nil, vmID, "created VM ID should not be nil")

	// Cleanup
	err = client.VM().Delete(ctx, vmID)
	if err != nil {
		t.Errorf("error while deleting VM %s: %v", vmID, err)
	}
}

func TestCreateVMWithVIFDevice(t *testing.T) {
	ctx, client, testPrefix := SetupTestContext(t)

	// Use the existing test network instead of creating a new one to avoid VLAN conflicts
	// The test network is already available in intTests.testNetwork
	networkID := uuid.FromStringOrNil(intTests.testNetwork.Id)
	if networkID == uuid.Nil {
		t.Skip("No test network available, skipping VIF device test")
	}

	// Create VM with specific VIF device setting
	vmName := "test-vm-vif-device"
	deviceZero := payloads.StringifiedInt(0)

	params := payloads.CreateVMParams{
		NameLabel: testPrefix + vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		VIFs: []payloads.VIFParams{
			{
				Device:  &deviceZero,
				Network: &networkID,
			},
		},
	}

	vmID, err := client.Pool().CreateVM(ctx, intTests.testPool.ID, params)
	require.NoError(t, err, "error while creating VM with VIF device setting: %v", err)
	require.NotEqual(t, uuid.Nil, vmID, "created VM ID should not be nil")

	// Verify the VM was created and get its details
	vm, err := client.VM().GetByID(ctx, vmID)
	require.NoError(t, err, "error while fetching VM details for %s: %v", vmID, err)
	require.NotNil(t, vm, "VM details should not be nil")

	// Get the VIFs for the VM using the v1 client
	vmObj := &v1.Vm{Id: vmID.String()}
	vifs, err := intTests.v1Client.GetVIFs(vmObj)
	require.NoError(t, err, "error while getting VIFs for VM %s: %v", vmID, err)

	// Verify we have exactly one VIF (the one we specified)
	assert.Equal(t, 1, len(vifs), "VM should have exactly one VIF")

	// Verify the VIF has the correct device setting and network
	assert.Equal(t, "0", vifs[0].Device, "VIF device should be '0'")
	assert.Equal(t, networkID.String(), vifs[0].Network, "VIF should be attached to the test network")
}

// This TC is meant to verify that when a VIF is set without a device,
// it is added to the VIFs already present on the template and not replacing them.
func TestCreateVMWithVIFNoDevice(t *testing.T) {
	ctx, client, testPrefix := SetupTestContext(t)

	networkID := uuid.FromStringOrNil(intTests.testNetwork.Id)
	if networkID == uuid.Nil {
		t.Skip("No test network available, skipping VIF device test")
	}

	// Create VM with specific VIF device setting
	vmName := "test-vm-vif-device"
	params := payloads.CreateVMParams{
		NameLabel: testPrefix + vmName,
		Template:  uuid.FromStringOrNil(intTests.testTemplate.Id),
		VIFs: []payloads.VIFParams{
			{
				Network: &networkID,
			},
		},
	}

	vmID, err := client.Pool().CreateVM(ctx, intTests.testPool.ID, params)
	require.NoError(t, err, "error while creating VM with VIF device setting: %v", err)
	require.NotEqual(t, uuid.Nil, vmID, "created VM ID should not be nil")

	// Verify the VM was created and get its details
	vm, err := client.VM().GetByID(ctx, vmID)
	require.NoError(t, err, "error while fetching VM details for %s: %v", vmID, err)
	require.NotNil(t, vm, "VM details should not be nil")

	// Get the VIFs for the VM using the v1 client
	vmObj := &v1.Vm{Id: vmID.String()}
	vifs, err := intTests.v1Client.GetVIFs(vmObj)
	require.NoError(t, err, "error while getting VIFs for VM %s: %v", vmID, err)

	// Verify we have exactly one VIF (the one we specified)
	assert.Equal(t, 2, len(vifs), "VM should have exactly one VIF")

	// Verify that one of the VIFs has the network attached
	var vifWithNetwork *v1.VIF
	for i := range vifs {
		if vifs[i].Network == networkID.String() {
			vifWithNetwork = &vifs[i]
			assert.NotEqual(t, "0", vifs[i].Device, "VIF device shouldn't be '0'")
		}
	}
	assert.NotNil(t, vifWithNetwork, "One VIF should be attached to the test network")
}

func TestCreateVMInvalidTemplate(t *testing.T) {
	ctx, client, testPrefix := SetupTestContext(t)

	vmName := "test-vm-invalid-template"
	params := payloads.CreateVMParams{
		NameLabel: testPrefix + vmName,
		Template:  uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"), // Invalid template ID
	}

	_, err := client.Pool().CreateVM(ctx, intTests.testPool.ID, params)
	if err == nil {
		t.Fatal("expected error when creating VM with invalid template ID, got nil")
	}
	assert.Contains(t, err.Error(), "vm creation failed: no such object", "error message should indicate bad request")
}

func TestCreateNetwork(t *testing.T) {
	ctx, client, testPrefix := SetupTestContext(t)

	networkName := "test-network"
	// Choose VLAN from env var if provided to avoid collisions in lab
	var vlan uint = 1234
	if v := os.Getenv("XOA_TEST_VLAN"); v != "" {
		if parsed, err := strconv.ParseUint(v, 10, 0); err == nil && parsed <= 4094 {
			vlan = uint(parsed)
		} else {
			t.Logf("Ignoring invalid XOA_TEST_VLAN=%s, using default %d", v, vlan)
		}
	}
	params := payloads.CreateNetworkParams{
		Name:        testPrefix + networkName,
		Pif:         uuid.FromStringOrNil(intTests.testNetwork.PIFs[0]), // Use the first PIF, only one PIF is expected
		MTU:         &[]uint{1450}[0],
		Description: "Test network created by xo-sdk-go",
		Vlan:        vlan,
	}
	networkID, err := client.Pool().CreateNetwork(ctx, intTests.testPool.ID, params)
	if err != nil {
		t.Fatalf("error while creating network in pool %s: %v", intTests.testPool.ID, err)
	}
	assert.NotEqual(t, uuid.Nil, networkID, "created network ID should not be nil")

	// Get network using v1 client to verify creation
	// TODO use v2 Network service when available
	createdNetwork, err := intTests.v1Client.GetNetwork(v1.Network{
		Id: networkID.String(),
	})
	if err != nil {
		t.Fatalf("error fetching created network %s: %v", networkID, err)
	}
	assert.Equal(t, params.Name, createdNetwork.NameLabel, "created network name should match")
	assert.Equal(t, intTests.testPool.ID.String(), createdNetwork.PoolId, "created network PoolID should match")

	// Overflow check before uint conversion
	if createdNetwork.MTU < 0 {
		t.Errorf("Invalid MTU value: %d (negative value not allowed)", createdNetwork.MTU)
	} else {
		assert.Equal(t, *params.MTU, uint(createdNetwork.MTU), "created network MTU should match")
	}
	assert.Equal(t, params.Description, createdNetwork.NameDescription, "created network description should match")

	// Cleanup
	// For now, we use v1 client to delete the network
	t.Log("Cleaning up network:", networkID)
	err = intTests.v1Client.DeleteNetwork(networkID.String())
	if err != nil {
		t.Fatal("error deleting network:", err)
		return
	}
}
