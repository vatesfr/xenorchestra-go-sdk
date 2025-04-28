/*
See the documentation in the /docs/v2 or README.md file for more information
about the v2 design choices, how to add new services, etc.
*/
package v2

import (
	"github.com/subosito/gotenv"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/backup"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/hub_recipe"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/jsonrpc"
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

	// Hub recipe service
	hubRecipeService library.HubRecipe

	// They have been added to the VM service because they are related to the VM.
	// However shall we let the user to have access to the restore and snapshot services ?
	// restoreService library.Restore
	// snapshotService library.Snapshot

	// We can provide access to the v1 client directly, allowing users to:
	// 1. Access v1 functionality without initializing a separate client
	// 2. Use v2 features while maintaining backward compatibility
	// 3. Gradually migrate from v1 to v2 without managing multiple clients
	v1Client v1.XOClient
	// Internal JSON-RPC service, we won't expose it to the user.
	// The purpose of this service is to provide a common interface for the
	// JSON-RPC calls, and to handle the errors and logging. When the REST
	// API will be fully released, this service will be removed. FYI, this
	// is only for methods that are not part of the v1 client.
	jsonrpcSvc library.JSONRPC
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

	v1Config := v1.Config{
		Url:                config.Url,
		Username:           config.Username,
		Password:           config.Password,
		Token:              config.Token,
		InsecureSkipVerify: config.InsecureSkipVerify,
	}

	v1Client, err := v1.NewClient(v1Config)
	if err != nil {
		return nil, err
	}
	legacyClient := v1Client.(*v1.Client)

	log, err := logger.New(config.Development)
	if err != nil {
		return nil, err
	}

	taskService := task.New(client, log)
	jsonrpcSvc := jsonrpc.New(legacyClient, log)
	restoreService := restore.New(client, legacyClient, taskService, jsonrpcSvc, log)
	snapshotService := snapshot.New(client, legacyClient, jsonrpcSvc, log)
	storageRepositoryService := storage_repository.New(client, log)
	backupService := backup.New(client, legacyClient, taskService, jsonrpcSvc, log)
	hubRecipeService := hub_recipe.New(client, legacyClient, jsonrpcSvc, log)

	return &XOClient{
		vmService:                vm.New(client, restoreService, snapshotService, log),
		taskService:              taskService,
		backupService:            backupService,
		storageRepositoryService: storageRepositoryService,
		hubRecipeService:         hubRecipeService,
		v1Client:                 v1Client,
		jsonrpcSvc:               jsonrpc.New(legacyClient, log),
	}, nil
}

func (c *XOClient) VM() library.VM {
	return c.vmService
}

func (c *XOClient) Task() library.Task {
	return c.taskService
}

/*
See comments on the XOClient struct.
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

func (c *XOClient) HubRecipe() library.HubRecipe {
	return c.hubRecipeService
}

func (c *XOClient) V1Client() v1.XOClient {
	return c.v1Client
}
