package integration

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestGetPool(t *testing.T) {
	t.Run("ByID", func(t *testing.T) {
		ctx, client, _ := SetupTestContext(t)

		pool, err := client.Pool().Get(ctx, intTests.testPool.ID)
		require.NoError(t, err, "error while fetching pool %s: %v", intTests.testPool.ID, err)
		assert.Equal(t, intTests.testPool.ID, pool.ID, "pool ID should match")
	})

	t.Run("ByInvalidID", func(t *testing.T) {
		ctx, client, _ := SetupTestContext(t)

		_, err := client.Pool().Get(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
		require.Error(t, err, "expected error when fetching pool with invalid ID, got nil")
		assert.Contains(t, err.Error(), "404 Not Found", "error message should indicate not found")
	})
}

func TestCreateVM(t *testing.T) {

	t.Run("CreateVM", func(t *testing.T) {
		t.Parallel()
		ctx, client, testPrefix := SetupTestContext(t)

		vmName := "test-vm"
		params := payloads.CreateVMParams{
			NameLabel: testPrefix + vmName,
			Template:  uuid.FromStringOrNil(intTests.testTemplateID),
		}

		vmID, err := client.Pool().CreateVM(ctx, intTests.testPool.ID, params)
		require.NoError(t, err, "error while creating VM in pool %s: %v", intTests.testPool.ID, err)
		assert.NotEqual(t, uuid.Nil, vmID, "created VM ID should not be nil")

		// Cleanup
		err = client.VM().Delete(ctx, vmID)
		if err != nil {
			t.Errorf("error while deleting VM %s: %v", vmID, err)
		}
	})

	t.Run("WithVIFDevice", func(t *testing.T) {
		t.Parallel()
		if intTests.v1Disabled {
			t.Skip("v1 client disabled, skipping VIF device test")
		}
		ctx, client, testPrefix := SetupTestContext(t)

		// Use the existing test network instead of creating a new one to avoid VLAN conflicts
		// The test network is already available in intTests.testNetwork
		networkID := intTests.testNetworkID
		if networkID == uuid.Nil {
			t.Skip("No test network available, skipping VIF device test")
		}

		// Create VM with specific VIF device setting
		vmName := "test-vm-vif-device"
		deviceZero := payloads.StringifiedInt(0)

		params := payloads.CreateVMParams{
			NameLabel: testPrefix + vmName,
			Template:  uuid.FromStringOrNil(intTests.testTemplateID),
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
	})

	// This TC is meant to verify that when a VIF is set without a device,
	// it is added to the VIFs already present on the template and not replacing them.
	t.Run("WithVIFNoDevice", func(t *testing.T) {
		t.Parallel()
		if intTests.v1Disabled {
			t.Skip("v1 client disabled, skipping VIF no-device test")
		}
		ctx, client, testPrefix := SetupTestContext(t)

		networkID := intTests.testNetworkID
		if networkID == uuid.Nil {
			t.Skip("No test network available, skipping VIF device test")
		}

		// Create VM with specific VIF device setting
		vmName := "test-vm-vif-device"
		params := payloads.CreateVMParams{
			NameLabel: testPrefix + vmName,
			Template:  uuid.FromStringOrNil(intTests.testTemplateID),
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
	})

	t.Run("InvalidTemplate", func(t *testing.T) {
		t.Parallel()
		ctx, client, testPrefix := SetupTestContext(t)

		vmName := "test-vm-invalid-template"
		params := payloads.CreateVMParams{
			NameLabel: testPrefix + vmName,
			Template:  uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"), // Invalid template ID
		}

		_, err := client.Pool().CreateVM(ctx, intTests.testPool.ID, params)
		require.Error(t, err, "expected error when creating VM with invalid template ID, got nil")
		assert.Contains(t, err.Error(),
			fmt.Sprintf("failed to create vm on pool %s: API error: 404 Not Found", intTests.testPool.ID),
			"error message should indicate bad request")
	})
}

func TestCreateNetwork(t *testing.T) {

	t.Run("with vlan", func(t *testing.T) {
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
		// Resolve PIF from the test network via v1 client
		testNetwork, err := client.Network().Get(ctx, intTests.testNetworkID)
		require.NoError(t, err, "error fetching test network %s: %v", intTests.testNetworkID, err)
		require.GreaterOrEqual(t, len(testNetwork.PIFs), 1, "test network should have at least one PIF")

		params := payloads.CreateNetworkParams{
			Name:        testPrefix + networkName,
			Pif:         testNetwork.PIFs[0],
			MTU:         &[]int{1450}[0],
			Description: "Test network created by xo-sdk-go",
			Vlan:        vlan,
		}
		networkID, err := client.Pool().CreateNetwork(ctx, intTests.testPool.ID, params)
		require.NoError(t, err, "error while creating network in pool %s: %v", intTests.testPool.ID, err)
		assert.NotEqual(t, uuid.Nil, networkID, "created network ID should not be nil")

		createdNetwork, err := client.Network().Get(ctx, networkID)
		require.NoError(t, err, "error fetching created network %s: %v", networkID, err)
		assert.Equal(t, params.Name, createdNetwork.NameLabel, "created network name should match")
		assert.Equal(t, intTests.testPool.ID, createdNetwork.Pool, "created network PoolID should match")

		assert.Equal(t, *params.MTU, createdNetwork.MTU, "created network MTU should match")
		assert.Equal(t, params.Description, createdNetwork.NameDescription, "created network description should match")

		// Cleanup
		t.Log("Cleaning up network:", networkID)
		err = client.Network().Delete(ctx, networkID)
		require.NoError(t, err, "error deleting network %s: %v", networkID, err)
	})

	t.Run("with internal network", func(t *testing.T) {
		ctx, client, testPrefix := SetupTestContext(t)

		networkName := "test-internal-network"
		mtu := 1450
		nbd := true

		params := payloads.CreateInternalNetworkParams{
			Name:        testPrefix + networkName,
			Description: "Test internal network created by xo-sdk-go",
			MTU:         &mtu,
			NBD:         &nbd,
		}

		networkID, err := client.Pool().CreateInternalNetwork(ctx, intTests.testPool.ID, params)
		require.NoError(t, err, "error while creating internal network in pool %s: %v", intTests.testPool.ID, err)
		require.NotEqual(t, uuid.Nil, networkID, "created internal network ID should not be nil")

		createdNetwork, err := client.Network().Get(ctx, networkID)
		require.NoError(t, err, "error fetching created internal network %s: %v", networkID, err)

		assert.Equal(t, params.Name, createdNetwork.NameLabel, "created internal network name should match")
		assert.Equal(t, intTests.testPool.ID, createdNetwork.Pool, "created internal network PoolID should match")
		assert.Equal(
			t, params.Description, createdNetwork.NameDescription, "created internal network description should match")
		assert.Equal(t, *params.MTU, createdNetwork.MTU, "created internal network MTU should match")

		if createdNetwork.NBD != nil {
			assert.Equal(t, *params.NBD, *createdNetwork.NBD, "created internal network NBD should match")
		}

		// Cleanup
		t.Log("Cleaning up internal network:", networkID)
		err = client.Network().Delete(ctx, networkID)
		require.NoError(t, err, "error deleting internal network %s: %v", networkID, err)
	})
}
