package integration

import (
	"os"
	"strconv"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestGetPoolByID(t *testing.T) {
	ctx, cleanup := SetupTestContext(t)
	defer cleanup()

	pool, err := testClient.Pool().Get(ctx, testPool.ID)
	if err != nil {
		t.Fatalf("error while fetching pool %s: %v", testPool.ID, err)
	}
	assert.Equal(t, testPool.ID, pool.ID, "pool ID should match")
}

func TestGetPoolByInvalidID(t *testing.T) {
	ctx, cleanup := SetupTestContext(t)
	defer cleanup()

	_, err := testClient.Pool().Get(ctx, uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"))
	if err == nil {
		t.Fatal("expected error when fetching pool with invalid ID, got nil")
	}
	assert.Contains(t, err.Error(), "404 Not Found", "error message should indicate not found")
}

func TestCreateVM(t *testing.T) {
	ctx, cleanup := SetupTestContext(t)
	defer cleanup()

	vmName := "test-vm"
	params := payloads.CreateVMParams{
		NameLabel: integrationTestPrefix + vmName,
		Template:  uuid.FromStringOrNil(testTemplate.Id),
	}

	vmID, err := testClient.Pool().CreateVM(ctx, testPool.ID, params)
	if err != nil {
		t.Fatalf("error while creating VM in pool %s: %v", testPool.ID, err)
	}
	assert.NotEqual(t, uuid.Nil, vmID, "created VM ID should not be nil")

	// Cleanup
	err = testClient.VM().Delete(ctx, vmID)
	if err != nil {
		t.Errorf("error while deleting VM %s: %v", vmID, err)
	}
}

func TestCreateVMInvalidTemplate(t *testing.T) {
	ctx, cleanup := SetupTestContext(t)
	defer cleanup()

	vmName := "test-vm-invalid-template"
	params := payloads.CreateVMParams{
		NameLabel: integrationTestPrefix + vmName,
		Template:  uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426655440000"), // Invalid template ID
	}

	_, err := testClient.Pool().CreateVM(ctx, testPool.ID, params)
	if err == nil {
		t.Fatal("expected error when creating VM with invalid template ID, got nil")
	}
	assert.Contains(t, err.Error(), "vm creation failed: no such object", "error message should indicate bad request")
}

func TestCreateNetwork(t *testing.T) {
	ctx, cleanup := SetupTestContext(t)
	defer cleanup()

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
		Name:        integrationTestPrefix + networkName,
		Pif:         uuid.FromStringOrNil(testNetwork.PIFs[0]), // Use the first PIF, only one PIF is expected
		MTU:         &[]uint{1450}[0],
		Description: "Test network created by xo-sdk-go",
		Vlan:        vlan,
	}
	networkID, err := testClient.Pool().CreateNetwork(ctx, testPool.ID, params)
	if err != nil {
		t.Fatalf("error while creating network in pool %s: %v", testPool.ID, err)
	}
	assert.NotEqual(t, uuid.Nil, networkID, "created network ID should not be nil")

	// Get network using v1 client to verify creation
	// TODO use v2 Network service when available
	createdNetwork, err := v1TestClient.GetNetwork(v1.Network{
		Id: networkID.String(),
	})
	if err != nil {
		t.Fatalf("error fetching created network %s: %v", networkID, err)
	}
	assert.Equal(t, params.Name, createdNetwork.NameLabel, "created network name should match")
	assert.Equal(t, testPool.ID.String(), createdNetwork.PoolId, "created network PoolID should match")

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
	err = v1TestClient.DeleteNetwork(networkID.String())
	if err != nil {
		t.Fatal("error deleting network:", err)
		return
	}
}
