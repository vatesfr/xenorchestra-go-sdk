/*
See the documentation in the /docs/v2 or README.md file for more information
about the v2 design choices, how to add new services, etc.
*/
package v2

import (
	"github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
	"github.com/vatesfr/xenorchestra-go-sdk/pkg/services/vm"
	"github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

type XOClient struct {
	vmService library.VM
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

	return &XOClient{
		vmService: vm.New(client, log),
	}, nil
}

func (c *XOClient) VM() library.VM {
	return c.vmService
}
