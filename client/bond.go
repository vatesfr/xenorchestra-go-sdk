package client

import (
	"fmt"
)

type Bond struct {
	Master string `json:"master"`
	Mode   string `json:"mode"`
	Id     string `json:"id"`
	Uuid   string `json:"uuid"`
	PoolId string `json:"$poolId"`
}

func (b Bond) Compare(obj interface{}) bool {
	other, ok := obj.(Bond)
	if !ok {
		return false
	}
	if b.Id != "" && b.Id == other.Id {
		return true
	}
	if b.PoolId != "" && b.PoolId == other.PoolId {
		return false
	}
	if b.Master != "" && b.Master != other.Master {
		return false
	}
	if b.Mode != "" && b.Mode != other.Mode {
		return false
	}
	if b.Uuid != "" && b.Uuid != other.Uuid {
		return false
	}
	return true
}

func (c *Client) GetBond(bondReq Bond) (*Bond, error) {
	obj, err := c.FindFromGetAllObjects(bondReq)
	if err != nil {
		return nil, err
	}
	bonds := obj.([]Bond)
	if len(bonds) != 1 {
		return nil, fmt.Errorf("expected to find a single Bond from request %+v, instead found %d", bondReq.Id, len(bonds))
	}
	return &bonds[0], nil
}

func (c *Client) GetBonds(bondReq Bond) ([]Bond, error) {
	obj, err := c.FindFromGetAllObjects(bondReq)
	if err != nil {
		return nil, err
	}
	bonds := obj.([]Bond)
	return bonds, nil
}
