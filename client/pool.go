package client

import (
	"fmt"
	"log/slog"
	"os"
)

type Pool struct {
	Id          string  `json:"id"`
	NameLabel   string  `json:"name_label"`
	Description string  `json:"name_description"`
	Cpus        CpuInfo `json:"cpus"`
	DefaultSR   string  `json:"default_SR"`
	Master      string  `json:"master"`
}

type CpuInfo struct {
	Cores   int64 `json:"cores"`
	Sockets int64 `json:"sockets"`
}

func (p Pool) Compare(obj interface{}) bool {
	otherPool := obj.(Pool)

	if otherPool.Id == p.Id {
		return true
	}

	if otherPool.NameLabel != p.NameLabel {
		return false
	}
	return true
}

func (c *Client) GetPoolByName(name string) (pools []Pool, err error) {
	obj, err := c.FindFromGetAllObjects(Pool{NameLabel: name})
	if err != nil {
		return
	}
	pools = obj.([]Pool)

	return pools, nil
}

func (c *Client) GetPools(pool Pool) (pools []Pool, err error) {
	obj, err := c.FindFromGetAllObjects(pool)
	if err != nil {
		return
	}
	pools = obj.([]Pool)

	return pools, nil
}

func FindPoolForTests(pool *Pool) {
	poolName, found := os.LookupEnv("XOA_POOL")

	if !found {
		slog.Error("The XOA_POOL environment variable must be set")
		os.Exit(-1)
	}
	c, err := NewClient(GetConfigFromEnv())
	if err != nil {
		slog.Error("failed to create client", "error", err)
		os.Exit(-1)
	}

	pools, err := c.GetPoolByName(poolName)

	if err != nil {
		slog.Error("failed to find a pool", "pool", poolName, "error", err)
		os.Exit(-1)
	}

	if len(pools) != 1 {
		slog.Error(fmt.Sprintf("Found %d pools with name_label %s."+
			"Please use a label that is unique so tests are reproducible.\n", len(pools), poolName))
		os.Exit(-1)
	}

	*pool = pools[0]
}
