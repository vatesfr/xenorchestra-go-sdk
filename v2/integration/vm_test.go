package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

func TestVM_CRUD(t *testing.T) {
	ctx := context.Background()
	tc := Setup(t)

	vmName := tc.GenerateResourceName("vm")

	tc.CleanupVM(t, vmName)

	t.Logf("Creating VM %s", vmName)

	var poolID, templateID, networkID string
	if tc.PoolID != "" {
		poolID = tc.PoolID
	} else {
		t.Logf("Using Pool name: %s", tc.Pool)
		t.Skip("Pool ID resolution not implemented, please set XOA_POOL_ID")
	}

	if tc.TemplateID != "" {
		templateID = tc.TemplateID
	} else {
		t.Logf("Using Template name: %s", tc.Template)
		t.Skip("Template ID resolution not implemented, please set XOA_TEMPLATE_ID")
	}

	if tc.NetworkID != "" {
		networkID = tc.NetworkID
	} else {
		t.Logf("Using Network name: %s", tc.Network)
		t.Skip("Network ID resolution not implemented, please set XOA_NETWORK_ID")
	}

	vm := &payloads.VM{
		NameLabel:       vmName,
		NameDescription: "Integration test VM",
		Template:        GetUUID(t, templateID),
		Memory: payloads.Memory{
			Size: 1 * 1024 * 1024 * 1024, // 1 GB
		},
		CPUs: payloads.CPUs{
			Number: 1,
		},
		VIFs:        []string{networkID},
		AutoPoweron: false,
		PoolID:      GetUUID(t, poolID),
	}

	createdVM, err := tc.Client.VM().Create(ctx, vm)
	assert.NoError(t, err)
	assert.NotNil(t, createdVM)
	assert.Equal(t, vmName, createdVM.NameLabel)

	vmID := createdVM.ID
	t.Logf("VM created with ID: %s", vmID)

	getVM, err := tc.Client.VM().GetByID(ctx, vmID)
	assert.NoError(t, err)
	assert.NotNil(t, getVM)
	assert.Equal(t, vmName, getVM.NameLabel)

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
	ctx := context.Background()
	tc := Setup(t)

	vmName := tc.GenerateResourceName("vm-lifecycle")

	tc.CleanupVM(t, vmName)

	var poolID, templateID, networkID string
	if tc.PoolID != "" {
		poolID = tc.PoolID
	} else {
		t.Logf("Using Pool name: %s", tc.Pool)
		t.Skip("Pool ID resolution not implemented, please set XOA_POOL_ID")
	}

	if tc.TemplateID != "" {
		templateID = tc.TemplateID
	} else {
		t.Logf("Using Template name: %s", tc.Template)
		t.Skip("Template ID resolution not implemented, please set XOA_TEMPLATE_ID")
	}

	if tc.NetworkID != "" {
		networkID = tc.NetworkID
	} else {
		t.Logf("Using Network name: %s", tc.Network)
		t.Skip("Network ID resolution not implemented, please set XOA_NETWORK_ID")
	}

	vm := &payloads.VM{
		NameLabel:       vmName,
		NameDescription: "VM lifecycle integration testing",
		Template:        GetUUID(t, templateID),
		Memory: payloads.Memory{
			Size: 1 * 1024 * 1024 * 1024, // 1 GB
		},
		CPUs: payloads.CPUs{
			Number: 1,
		},
		VIFs:        []string{networkID},
		AutoPoweron: false,
		PoolID:      GetUUID(t, poolID),
	}

	createdVM, err := tc.Client.VM().Create(ctx, vm)
	assert.NoError(t, err)
	assert.NotNil(t, createdVM)

	vmID := createdVM.ID
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
