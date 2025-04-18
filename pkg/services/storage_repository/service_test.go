package storage_repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

func setupStorageRepositoryTestServer(t *testing.T) (*httptest.Server, *Service) {
	storageRepos := []*payloads.StorageRepository{
		{
			ID:              uuid.Must(uuid.NewV4()),
			UUID:            "storage-repo-1",
			NameLabel:       "Storage Repo 1",
			NameDescription: "Test Storage Repository 1",
			PoolID:          uuid.Must(uuid.NewV4()),
			SRType:          "lvm",
			PhysicalUsage:   1024 * 1024 * 1024,
			Size:            10 * 1024 * 1024 * 1024,
			Usage:           5 * 1024 * 1024 * 1024,
			Tags:            []string{"tag1", "tag2"},
		},
		{
			ID:              uuid.Must(uuid.NewV4()),
			UUID:            "storage-repo-2",
			NameLabel:       "Storage Repo 2",
			NameDescription: "Test Storage Repository 2",
			PoolID:          uuid.Must(uuid.NewV4()),
			SRType:          "nfs",
			PhysicalUsage:   2 * 1024 * 1024 * 1024,
			Size:            20 * 1024 * 1024 * 1024,
			Usage:           10 * 1024 * 1024 * 1024,
			Tags:            []string{"tag2", "tag3"},
		},
	}

	poolID := uuid.Must(uuid.NewV4())
	storageRepos = append(storageRepos, &payloads.StorageRepository{
		ID:              uuid.Must(uuid.NewV4()),
		UUID:            "storage-repo-3",
		NameLabel:       "Storage Repo 3",
		NameDescription: "Test Storage Repository 3",
		PoolID:          poolID,
		SRType:          "local",
		PhysicalUsage:   500 * 1024 * 1024,      // 500 MB
		Size:            5 * 1024 * 1024 * 1024, // 5 GB
		Usage:           1 * 1024 * 1024 * 1024, // 1 GB
		Tags:            []string{"tag1", "tag4"},
	})

	errorID := uuid.Must(uuid.NewV4())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasPrefix(r.URL.Path, "/srs") {
			pathParts := strings.Split(r.URL.Path, "/")

			if r.URL.Path == "/srs" && r.Method == http.MethodGet {
				urls := make([]string, 0)
				poolIDFilter := r.URL.Query().Get("$poolId")
				nameFilter := r.URL.Query().Get("name_label")
				typeFilter := r.URL.Query().Get("SR_type")

				for _, sr := range storageRepos {
					if poolIDFilter != "" {
						poolUUID, err := uuid.FromString(poolIDFilter)
						if err != nil || sr.PoolID != poolUUID {
							continue
						}
					}
					if nameFilter != "" && sr.NameLabel != nameFilter {
						continue
					}
					if typeFilter != "" && sr.SRType != typeFilter {
						continue
					}

					urls = append(urls, fmt.Sprintf("/srs/%s", sr.ID))
				}

				json.NewEncoder(w).Encode(urls)
				return
			}

			if len(pathParts) == 3 && r.Method == http.MethodGet {
				idStr := pathParts[2]
				id, err := uuid.FromString(idStr)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				if id == errorID {
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "Storage repository not found",
					})
					return
				}

				var foundRepo *payloads.StorageRepository
				for _, sr := range storageRepos {
					if sr.ID == id {
						foundRepo = sr
						break
					}
				}

				if foundRepo == nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				json.NewEncoder(w).Encode(foundRepo)
				return
			}

			if strings.HasSuffix(r.URL.Path, "/tags") && r.Method == http.MethodPost {
				idStr := pathParts[2]
				id, err := uuid.FromString(idStr)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				if id == errorID {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": false,
						"error":   "Failed to add tag",
					})
					return
				}

				var foundRepo *payloads.StorageRepository
				for _, sr := range storageRepos {
					if sr.ID == id {
						foundRepo = sr
						break
					}
				}

				if foundRepo == nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				var payload map[string]string
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				tag := payload["tag"]
				if tag == "" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				for _, t := range foundRepo.Tags {
					if t == tag {
						break
					}
				}

				foundRepo.Tags = append(foundRepo.Tags, tag)

				json.NewEncoder(w).Encode(map[string]bool{
					"success": true,
				})
				return
			}

			if len(pathParts) == 5 && pathParts[3] == "tags" && r.Method == http.MethodDelete {
				idStr := pathParts[2]
				id, err := uuid.FromString(idStr)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}

				tag := pathParts[4]

				if id == errorID {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]any{
						"success": false,
						"error":   "Failed to remove tag",
					})
					return
				}

				var foundRepo *payloads.StorageRepository
				for _, sr := range storageRepos {
					if sr.ID == id {
						foundRepo = sr
						break
					}
				}

				if foundRepo == nil {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				var newTags []string
				for _, t := range foundRepo.Tags {
					if t != tag {
						newTags = append(newTags, t)
					}
				}
				foundRepo.Tags = newTags

				json.NewEncoder(w).Encode(map[string]bool{
					"success": true,
				})
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
	}))

	baseURL, _ := url.Parse(server.URL)
	restClient := &client.Client{
		HttpClient: http.DefaultClient,
		BaseURL:    baseURL,
		AuthToken:  "test-token",
	}

	log, _ := logger.New(false)
	service := &Service{
		client: restClient,
		log:    log,
	}

	return server, service
}

func TestGetByID(t *testing.T) {
	server, service := setupStorageRepositoryTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("get existing storage repository", func(t *testing.T) {
		repos, err := service.List(ctx, nil, 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, repos)

		repo, err := service.GetByID(ctx, repos[0].ID)
		assert.NoError(t, err)
		assert.NotNil(t, repo)
		assert.Equal(t, repos[0].ID, repo.ID)
		assert.Equal(t, repos[0].NameLabel, repo.NameLabel)
	})

	t.Run("get non-existent storage repository", func(t *testing.T) {
		errorID := uuid.Must(uuid.NewV4())
		repo, err := service.GetByID(ctx, errorID)
		assert.Error(t, err)
		assert.Nil(t, repo)
	})
}

func TestList(t *testing.T) {
	server, service := setupStorageRepositoryTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("list all repositories", func(t *testing.T) {
		repos, err := service.List(ctx, nil, 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, repos)
		assert.Len(t, repos, 3)
	})

	t.Run("list with limit", func(t *testing.T) {
		repos, err := service.List(ctx, nil, 2)
		assert.NoError(t, err)
		assert.NotEmpty(t, repos)
		assert.Len(t, repos, 2)
	})

	t.Run("list with filters", func(t *testing.T) {
		filter := &payloads.StorageRepositoryFilter{
			SRType: "lvm",
		}
		repos, err := service.List(ctx, filter, 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, repos)
		for _, repo := range repos {
			assert.Equal(t, "lvm", repo.SRType)
		}
	})

	t.Run("list with tag filter", func(t *testing.T) {
		filter := &payloads.StorageRepositoryFilter{
			Tags: []string{"tag1"},
		}
		repos, err := service.List(ctx, filter, 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, repos)

		for _, repo := range repos {
			hasTag := false
			for _, tag := range repo.Tags {
				if tag == "tag1" {
					hasTag = true
					break
				}
			}
			assert.True(t, hasTag, "Repository should have tag1")
		}
	})
}

func TestListByPool(t *testing.T) {
	server, service := setupStorageRepositoryTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("list by valid pool ID", func(t *testing.T) {
		repos, err := service.List(ctx, nil, 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, repos)
		assert.GreaterOrEqual(t, len(repos), 3)

		poolID := repos[2].PoolID
		poolRepos, err := service.ListByPool(ctx, poolID, 0)
		assert.NoError(t, err)
		assert.NotEmpty(t, poolRepos)

		for _, repo := range poolRepos {
			assert.Equal(t, poolID, repo.PoolID)
		}
	})

	t.Run("list by non-existent pool ID", func(t *testing.T) {
		nonExistentPoolID := uuid.Must(uuid.NewV4())
		repos, err := service.ListByPool(ctx, nonExistentPoolID, 0)
		assert.NoError(t, err)
		assert.Empty(t, repos)
	})
}

func TestAddTag(t *testing.T) {
	server, service := setupStorageRepositoryTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("add tag to repository", func(t *testing.T) {
		repos, err := service.List(ctx, nil, 1)
		assert.NoError(t, err)
		assert.NotEmpty(t, repos)

		repo := repos[0]
		newTag := "new-test-tag"

		err = service.AddTag(ctx, repo.ID, newTag)
		assert.NoError(t, err)

		updatedRepo, err := service.GetByID(ctx, repo.ID)
		assert.NoError(t, err)

		hasTag := false
		for _, tag := range updatedRepo.Tags {
			if tag == newTag {
				hasTag = true
				break
			}
		}
		assert.True(t, hasTag, "Tag should have been added to repository")
	})

	t.Run("add tag error", func(t *testing.T) {
		errorID := uuid.Must(uuid.NewV4())
		err := service.AddTag(ctx, errorID, "error-tag")
		assert.Error(t, err)
	})
}

func TestRemoveTag(t *testing.T) {
	server, service := setupStorageRepositoryTestServer(t)
	defer server.Close()

	ctx := context.Background()

	t.Run("remove tag from repository", func(t *testing.T) {
		repos, err := service.List(ctx, nil, 1)
		assert.NoError(t, err)
		assert.NotEmpty(t, repos)

		repo := repos[0]
		assert.NotEmpty(t, repo.Tags)
		tagToRemove := repo.Tags[0]

		err = service.RemoveTag(ctx, repo.ID, tagToRemove)
		assert.NoError(t, err)

		updatedRepo, err := service.GetByID(ctx, repo.ID)
		assert.NoError(t, err)

		for _, tag := range updatedRepo.Tags {
			assert.NotEqual(t, tagToRemove, tag, "Tag should have been removed from repository")
		}
	})

	t.Run("remove tag error", func(t *testing.T) {
		errorID := uuid.Must(uuid.NewV4())
		err := service.RemoveTag(ctx, errorID, "error-tag")
		assert.Error(t, err)
	})
}

func TestContainsAllTags(t *testing.T) {
	t.Run("empty needles", func(t *testing.T) {
		haystack := []string{"tag1", "tag2"}
		needles := []string{}
		result := containsAllTags(haystack, needles)
		assert.True(t, result, "Should return true when needles is empty")
	})

	t.Run("empty haystack", func(t *testing.T) {
		haystack := []string{}
		needles := []string{"tag1"}
		result := containsAllTags(haystack, needles)
		assert.False(t, result, "Should return false when haystack is empty")
	})

	t.Run("contains all tags", func(t *testing.T) {
		haystack := []string{"tag1", "tag2", "tag3"}
		needles := []string{"tag1", "tag3"}
		result := containsAllTags(haystack, needles)
		assert.True(t, result, "Should return true when haystack contains all needles")
	})

	t.Run("missing some tags", func(t *testing.T) {
		haystack := []string{"tag1", "tag2"}
		needles := []string{"tag1", "tag3"}
		result := containsAllTags(haystack, needles)
		assert.False(t, result, "Should return false when haystack doesn't contain all needles")
	})
}
