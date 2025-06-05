/*
See the documentation in the /docs/v2 or README.md file for more information
about the v2 design choices, how to add new services, etc.
*/
package v2

import (
	"github.com/subosito/gotenv"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/restore"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/snapshot"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/storage_repository"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/task"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/vm"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

type XOClient struct {
	vmService   library.VM
	taskService library.Task

	// Storage repository service
	storageRepositoryService library.StorageRepository
}

// Added to load the .env file in the root of the project,
// to make it easier to test the SDK without having to set
// the environment variables in the machine. Not needed ?
func init() {
	_ = gotenv.Load()
}

func New(config *config.Config) (library.Library, error) {
	client, err := client.New(config)
	if err != nil {
		return nil, err
	}

	log, err := logger.New(config.Development)
	if err != nil {
		return nil, err
	}

	taskService := task.New(client, log)
	restoreService := restore.New(client, log, taskService)
	snapshotService := snapshot.New(client, log)
	storageRepositoryService := storage_repository.New(client, log)

	return &XOClient{
		vmService:                vm.New(client, taskService, restoreService, snapshotService, log),
		taskService:              taskService,
		storageRepositoryService: storageRepositoryService,
	}, nil
}

func (c *XOClient) VM() library.VM {
	return c.vmService
}

func (c *XOClient) Task() library.Task {
	return c.taskService
}

func (c *XOClient) StorageRepository() library.StorageRepository {
	return c.storageRepositoryService
}
