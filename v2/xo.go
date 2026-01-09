/*
See the documentation in the /docs/v2 or README.md file for more information
about the v2 design choices, how to add new services, etc.
*/
package v2

import (
	"sync"

	"github.com/subosito/gotenv"
	v1 "github.com/vatesfr/xenorchestra-go-sdk/client"
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/jsonrpc"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/pool"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/task"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/vm"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
	"go.uber.org/zap"
)

type XOClient struct {
	vmService   library.VM
	taskService library.Task
	poolService library.Pool
	// We can provide access to the v1 client directly, allowing users to:
	// 1. Access v1 functionality without initializing a separate client
	// 2. Use v2 features while maintaining backward compatibility
	// 3. Gradually migrate from v1 to v2 without managing multiple clients
	// The v1 client is created lazily on first access to V1Client() or when
	// JSON-RPC calls are made. This allows v2 client creation without requiring
	// an active XOA connection at initialization time.
	// This also avoid to open a websocket connection if the user only uses REST API.
	v1InitOnce sync.Once // Mutex to initialize v1 client once
	v1InitErr  error
	v1Config   v1.Config
	v1Client   v1.XOClient
	// Internal JSON-RPC service, we won't expose it to the user.
	// The purpose of this service is to provide a common interface for the
	// JSON-RPC calls, and to handle the errors and logging. When the REST
	// API will be fully released, this service will be removed. FYI, this
	// is only for methods that are not part of the v1 client.
	jsonrpcSvc library.JSONRPC
	log        *logger.Logger
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

	v1URL := config.Url
	if v1URL != "" {
		if v1URL[:4] == "http" {
			if v1URL[:5] == "https" {
				v1URL = "wss" + v1URL[5:]
			} else {
				v1URL = "ws" + v1URL[4:]
			}
		}
	}

	v1Config := v1.Config{
		Url:                v1URL,
		Username:           config.Username,
		Password:           config.Password,
		Token:              config.Token,
		InsecureSkipVerify: config.InsecureSkipVerify,
	}

	log, err := logger.New(config.Development)
	if err != nil {
		return nil, err
	}

	taskService := task.New(client, log)

	xoClient := &XOClient{
		vmService:   vm.New(client, taskService, log),
		taskService: taskService,
		poolService: pool.New(client, taskService, log),
		v1Config:    v1Config,
		log:         log,
	}

	// Create a lazy JSONRPC service that will trigger v1Client creation on first call
	xoClient.jsonrpcSvc = jsonrpc.NewLazy(xoClient.initV1Client, log)

	return xoClient, nil
}

// initV1Client initializes the v1 client lazily and thread-safely.
// It is called at most once due to sync.Once.
func (c *XOClient) initV1Client() (*v1.Client, error) {
	c.v1InitOnce.Do(func() {
		c.v1Client, c.v1InitErr = v1.NewClient(c.v1Config)
	})
	if c.v1InitErr != nil {
		if c.v1InitErr != nil {
			c.log.Error("Failed to initialize v1 client", zap.Error(c.v1InitErr))
		}
		return nil, c.v1InitErr
	}
	if c.v1Client == nil {
		return nil, nil
	}
	return c.v1Client.(*v1.Client), nil
}

func (c *XOClient) VM() library.VM {
	return c.vmService
}

func (c *XOClient) Task() library.Task {
	return c.taskService
}

func (c *XOClient) Pool() library.Pool {
	return c.poolService
}

func (c *XOClient) V1Client() v1.XOClient {
	_, _ = c.initV1Client()
	return c.v1Client
}
