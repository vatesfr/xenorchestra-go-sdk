package client

import (
	"testing"
)

func TestGetVIFs(t *testing.T) {

	c, err := NewClient(GetConfigFromEnv())

	if err != nil {
		t.Fatalf("failed to create client with error: %v", err)
	}

	vifs, err := c.GetVIFs(&accVm)
	if err != nil {
		t.Fatalf("failed to get VIFs with id: %s, with error: %v", accVm.Id, err)
	}

	for _, vif := range vifs {
		if vif.Device == "" {
			t.Errorf("expecting `Device` field to be set on VIF")
		}

		if vif.MacAddress == "" {
			t.Errorf("expecting `MacAddress` field to be set on VIF")
		}

		if vif.Network == "" {
			t.Errorf("expecting `Network` field to be set on VIF")
		}

		if vif.VmId != accVm.Id {
			t.Errorf("VIF's VmId `%s` should have matched: %v", vif.VmId, accVm)
		}

		if len(vif.Device) == 0 {
			t.Errorf("expecting `Device` field to be set on VIF instead received: %s", vif.Device)
		}

		// 		if !vif.Attached {
		// 			t.Errorf("expecting `Attached` field to be true on VIF instead received: %t", vif.Attached)
		// 		}
	}
}

func TestGetVIF(t *testing.T) {

	c, err := NewClient(GetConfigFromEnv())

	if err != nil {
		t.Fatalf("failed to create client with error: %v", err)
	}

	vifs, err := c.GetVIFs(&accVm)
	if err != nil {
		t.Fatalf("failed to get VIF with error: %v", err)
	}

	expectedVIF := vifs[0]

	vif, err := c.GetVIF(&VIF{
		MacAddress: expectedVIF.MacAddress,
	})

	if err != nil {
		t.Fatalf("failed to get VIF with error: %v", err)
	}

	if vif.MacAddress != expectedVIF.MacAddress {
		t.Errorf("expected VIF: %v does not match the VIF we received %v", expectedVIF, vif)
	}
}

func TestCreateVIF_DeleteVIF(t *testing.T) {
	c, err := NewClient(GetConfigFromEnv())

	if err != nil {
		t.Fatalf("failed to create client with error: %v", err)
	}

	vm, err := c.GetVm(Vm{Id: accVm.Id})

	if err != nil {
		t.Fatalf("failed to get VM with error: %v", err)
	}

	vif, err := c.CreateVIF(vm, &VIF{Network: accDefaultNetwork.Id})

	if err != nil {
		t.Fatalf("failed to create VIF with error: %v", err)
	}

	err = c.DeleteVIF(vif)

	if err != nil {
		t.Errorf("failed to delete the VIF with error: %v", err)
	}
}

func TestVIFCompare(t *testing.T) {
	vif := VIF{
		Id:         "vif-id",
		VmId:       "vm-id",
		Device:     "1",
		MacAddress: "02:00:00:00:00:01",
	}

	cases := []struct {
		name     string
		query    VIF
		expected bool
	}{
		{
			name:     "matches on ID",
			query:    VIF{Id: "vif-id"},
			expected: true,
		},
		{
			name:     "ID takes precedence over a matching MAC address",
			query:    VIF{Id: "other-id", MacAddress: "02:00:00:00:00:01"},
			expected: false,
		},
		{
			name:     "matches on VM",
			query:    VIF{VmId: "vm-id"},
			expected: true,
		},
		{
			name:     "matches on MAC address in another format",
			query:    VIF{MacAddress: "02-00-00-00-00-01"},
			expected: true,
		},
		{
			name:     "matches on VM and device",
			query:    VIF{VmId: "vm-id", Device: "1"},
			expected: true,
		},
		{
			name:     "same MAC address on another VM",
			query:    VIF{VmId: "other-vm", MacAddress: "02:00:00:00:00:01"},
			expected: false,
		},
		{
			name:     "same MAC address on another device",
			query:    VIF{VmId: "vm-id", Device: "0", MacAddress: "02:00:00:00:00:01"},
			expected: false,
		},
		{
			name:     "same VM with another MAC address",
			query:    VIF{VmId: "vm-id", MacAddress: "02:00:00:00:00:02"},
			expected: false,
		},
		{
			name:     "empty query",
			query:    VIF{},
			expected: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.query.Compare(vif); got != c.expected {
				t.Errorf("expected %+v.Compare(%+v) to be %t, got %t", c.query, vif, c.expected, got)
			}
		})
	}
}
