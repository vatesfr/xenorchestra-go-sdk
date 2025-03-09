/*
TODO: REMOVE THIS COMMENT. The XOClient implements the Library interface.
It means that each time we add in the library package a new service, we
need to add it in the XOClient. Like adding backup as an example.

We will have to create a new file in library called backup.go and add
the service to the Library interface and add a method backup as shown
bellow. Again see v2 provided example. The method chaining is also
great IMO. Instead of calling client with method. Instead we can do
client.VM.Create() or client.VM.GetByID() etc.
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
