package client

import (
	"errors"
	"fmt"
	"net"
)

type VIF struct {
	Id         string `json:"id"`
	Attached   bool   `json:"attached"`
	Network    string `json:"$network"`
	Device     string `json:"device"`
	MacAddress string `json:"MAC"`
	VmId       string `json:"$VM"`
}

// Compare reports whether other matches the VIF used as a query.
//
// When the query has an ID, only the ID is compared. Otherwise every
// non-empty criterion (VM, device, MAC address) must match. MAC addresses
// are not unique (cloned or replicated VMs keep them), so a lookup by MAC
// address should be scoped to a VM to identify a single VIF.
func (v VIF) Compare(obj interface{}) bool {
	other := obj.(VIF)
	if v.Id != "" {
		return v.Id == other.Id
	}

	if v.VmId == "" && v.Device == "" && v.MacAddress == "" {
		return false
	}

	if v.VmId != "" && v.VmId != other.VmId {
		return false
	}

	if v.Device != "" && v.Device != other.Device {
		return false
	}

	if v.MacAddress != "" && !isSameMacAddress(v.MacAddress, other.MacAddress) {
		return false
	}

	return true
}

func isSameMacAddress(a, b string) bool {
	macA, errA := net.ParseMAC(a)
	macB, errB := net.ParseMAC(b)
	if errA != nil || errB != nil {
		return a == b
	}
	return macA.String() == macB.String()
}

func (c *Client) GetVIFs(vm *Vm) ([]VIF, error) {
	obj, err := c.FindFromGetAllObjects(VIF{VmId: vm.Id})

	if _, ok := err.(NotFound); ok {
		return []VIF{}, nil
	}

	if err != nil {
		return nil, err
	}
	vifs, ok := obj.([]VIF)
	if !ok {
		return []VIF{}, errors.New("failed to coerce response into VIF slice")
	}

	return vifs, nil
}

func (c *Client) GetVIF(vifReq *VIF) (*VIF, error) {

	obj, err := c.FindFromGetAllObjects(VIF{
		Id:         vifReq.Id,
		VmId:       vifReq.VmId,
		Device:     vifReq.Device,
		MacAddress: vifReq.MacAddress,
	})

	if err != nil {
		return nil, err
	}

	vifs := obj.([]VIF)

	if len(vifs) > 1 {
		return nil, fmt.Errorf("received %d VIFs but was expecting a single VIF to be returned", len(vifs))
	}
	return &vifs[0], nil
}

func (c *Client) CreateVIF(vm *Vm, vif *VIF) (*VIF, error) {

	var id string
	params := map[string]interface{}{
		"network": vif.Network,
		"vm":      vm.Id,
	}
	if vif.MacAddress != "" {
		params["mac"] = vif.MacAddress
	}
	err := c.Call("vm.createInterface", params, &id)

	if err != nil {
		return nil, err
	}

	return c.GetVIF(&VIF{Id: id})
}

func (c *Client) ConnectVIF(vifReq *VIF) (err error) {
	vif, err := c.GetVIF(vifReq)

	if err != nil {
		return
	}
	var success bool
	err = c.Call("vif.connect", map[string]interface{}{
		"id": vif.Id,
	}, &success)
	return
}

func (c *Client) DisconnectVIF(vifReq *VIF) (err error) {
	vif, err := c.GetVIF(vifReq)

	if err != nil {
		return
	}

	var success bool
	err = c.Call("vif.disconnect", map[string]interface{}{
		"id": vif.Id,
	}, &success)
	return
}

func (c *Client) DeleteVIF(vifReq *VIF) (err error) {
	var vif *VIF

	// This is a request that is looking the VIF
	// up by macaddress and needs to lookup the ID first.
	if vifReq.Id == "" {
		vif, err = c.GetVIF(vifReq)

		if err != nil {
			return err
		}
	} else {
		vif = vifReq
	}

	err = c.DisconnectVIF(vif)

	if err != nil {
		return err
	}

	params := map[string]interface{}{
		"id": vif.Id,
	}
	var result bool
	err = c.Call("vif.delete", params, &result)
	c.logger.Debug("Calling vif.delete received err", "error", err)

	if err != nil {
		return err
	}

	return nil
}
