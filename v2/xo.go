/*
See the documentation in the /docs/v2 or README.md file for more information
about the v2 design choices, how to add new services, etc.
*/
package v2

import (
	"github.com/subosito/gotenv"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/backup"
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
	// As I shared in the library package, I am not sure if
	// this interface shouldn't be in the VM service.
	backupService library.Backup

	// Storage repository service
	storageRepositoryService library.StorageRepository

	// They have been added to the VM service because they are related to the VM.
	// However shall we let the user to have access to the restore and snapshot services ?
	// restoreService library.Restore
	// snapshotService library.Snapshot
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
	backupService := backup.New(client, log, taskService)

	return &XOClient{
		vmService:                vm.New(client, taskService, restoreService, snapshotService, log),
		taskService:              taskService,
		backupService:            backupService,
		storageRepositoryService: storageRepositoryService,
	}, nil
}

func (c *XOClient) VM() library.VM {
	return c.vmService
}

func (c *XOClient) Task() library.Task {
	return c.taskService
}

/*
func (c *XOClient) Restore() library.Restore {
	return c.restoreService
}
*/

func (c *XOClient) Backup() library.Backup {
	return c.backupService
}

func (c *XOClient) StorageRepository() library.StorageRepository {
	return c.storageRepositoryService
}
