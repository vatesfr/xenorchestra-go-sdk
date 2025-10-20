package client

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
)

type StorageRepository struct {
	Id            string   `json:"id"`
	Uuid          string   `json:"uuid"`
	NameLabel     string   `json:"name_label"`
	PoolId        string   `json:"$poolId"`
	SRType        string   `json:"SR_type"`
	Container     string   `json:"$container"`
	PhysicalUsage int      `json:"physical_usage"`
	Size          int      `json:"size"`
	Usage         int      `json:"usage"`
	Tags          []string `json:"tags,omitempty"`
}

func (s StorageRepository) Compare(obj interface{}) bool {
	otherSr := obj.(StorageRepository)

	if s.Id != "" && s.Id == otherSr.Id {
		return true
	}

	if len(s.Tags) > 0 {
		for _, tag := range s.Tags {
			if !stringInSlice(tag, otherSr.Tags) {
				return false
			}
		}
	}

	labelsMatch := s.NameLabel == otherSr.NameLabel

	if s.PoolId == "" && labelsMatch {
		return true
	} else if s.PoolId == otherSr.PoolId && labelsMatch {
		return true
	}

	return false
}

func stringInSlice(needle string, haystack []string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

func (c *Client) GetStorageRepositoryById(id string) (StorageRepository, error) {
	obj, err := c.FindFromGetAllObjects(StorageRepository{Id: id})
	var sr StorageRepository

	if err != nil {
		return sr, err
	}
	srs, ok := obj.([]StorageRepository)

	if !ok {
		return sr, errors.New("failed to coerce response into StorageRepository slice")
	}

	if len(srs) != 1 {
		return sr, fmt.Errorf("expected a single storage repository to be returned, instead received: %d in the response: %v",
			len(srs), obj)
	}

	return srs[0], nil
}

func (c *Client) GetStorageRepository(sr StorageRepository) ([]StorageRepository, error) {
	obj, err := c.FindFromGetAllObjects(sr)

	if err != nil {
		return nil, err
	}
	srs, ok := obj.([]StorageRepository)

	if !ok {
		return nil, errors.New("failed to coerce response into StorageRepository slice")
	}

	return srs, nil
}

func FindStorageRepositoryForTests(pool Pool, sr *StorageRepository, tag string) {
	c, err := NewClient(GetConfigFromEnv())
	if err != nil {
		slog.Error("failed to create client", "error", err)
		os.Exit(-1)
	}

	defaultSr, err := c.GetStorageRepositoryById(pool.DefaultSR)

	if err != nil {
		slog.Error("failed to find the default storage repository",
			"SR id", pool.DefaultSR, "error", err)
		os.Exit(-1)
	}

	*sr = defaultSr

	err = c.AddTag(defaultSr.Id, tag)

	if err != nil {
		slog.Error("failed to set tag on default storage repository", "SR id", pool.DefaultSR, "error", err)
		os.Exit(-1)
	}
}

func FindIsoStorageRepositoryForTests(pool Pool, sr *StorageRepository, tag, isoSrEnvVar string) {
	isoSrName, found := os.LookupEnv(isoSrEnvVar)
	if !found {
		slog.Error("The environment variable must be set for the tests", "name", isoSrEnvVar)
		os.Exit(-1)
	}

	c, err := NewClient(GetConfigFromEnv())
	if err != nil {
		slog.Error("failed to create client", "error", err)
		os.Exit(-1)
	}

	isoSrReq := StorageRepository{
		PoolId:    pool.Id,
		NameLabel: isoSrName,
		SRType:    "iso",
	}
	isoSrs, err := c.GetStorageRepository(isoSrReq)

	if err != nil {
		slog.Error("failed to find an iso storage repository", "error", err)
		os.Exit(-1)
	}

	if len(isoSrs) != 1 {
		slog.Error(fmt.Sprintf("expected iso srs req `%v` to only return single sr, instead found %d",
			isoSrReq, len(isoSrs)))
		os.Exit(-1)
	}

	isoSr := isoSrs[0]
	*sr = isoSr

	err = c.AddTag(isoSr.Id, tag)

	if err != nil {
		slog.Error("failed to set tag on iso storage repository", "SR id", isoSr.Id, "error", err)
		os.Exit(-1)
	}
}
