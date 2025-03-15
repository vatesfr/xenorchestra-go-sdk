# Be aware that the XO API migration over REST API is not yet complete.
# The v2 SDK does not yet support all the features of the v1 SDK.
# We are working to add support for all the features as soon as possible.
# In the meantime, you can use the v1 SDK to interact with Xen Orchestra.
# Another note is that the v2 can use the backup which isn't available in the v1.
# You can start the migration by using both version 1 and 2 in parallel. 

# Migration Guide: v1 to v2

This guide helps you migrate from the Xen Orchestra Go SDK v1 to v2. The v2 SDK introduces several breaking changes but provides a more consistent and easier-to-use API.

## Key Differences

| Feature          | v1                              | v2                                  |
|------------------|--------------------------------|-------------------------------------|
| API Protocol     | JSON-RPC                       | REST API                            |
| ID Type          | String                         | UUID                                |
| Client Interface | Flat structure                 | Method chaining                     |
| Error Handling   | Custom error types             | Standard Go errors with context     |
| Context Support  | Limited                        | All operations support context      |
| Typing           | Runtime type checking          | Compile-time type checking          |
| Initialization   | `GetConfigFromEnv()` function  | `config.New()` function            |

## Client Initialization

### v1:
```go
config := client.GetConfigFromEnv()
xoClient, err := client.NewClient(config)
if err != nil {
    // Handle error
}
```

### v2:
```go
cfg, err := config.New()
if err != nil {
    // Handle error
}

client, err := v2.New(cfg)
if err != nil {
    // Handle error
}
```

## VM Operations Comparison

### Getting a VM by ID

#### v1:
```go
vm, err := xoClient.GetVm(client.Vm{Id: "12345678-1234-1234-1234-123456789012"})
if err != nil {
    // Handle error
}
```

#### v2:
```go
vmID, err := uuid.FromString("12345678-1234-1234-1234-123456789012")
if err != nil {
    // Handle error
}

vm, err := client.VM().GetByID(ctx, vmID)
if err != nil {
    // Handle error
}
```

### Listing VMs

#### v1:
```go
vms, err := xoClient.GetVms(client.Vm{})
if err != nil {
    // Handle error
}
```

#### v2:
```go
vms, err := client.VM().List(ctx)
if err != nil {
    // Handle error
}
```

### Creating a VM

#### v1:
```go
vm, err := xoClient.CreateVm(client.Vm{
    NameLabel:       "test-vm",
    NameDescription: "Test VM",
    Template:        "template-id",
}, 5*time.Minute)
if err != nil {
    // Handle error
}
```

#### v2:
```go
templateID, _ := uuid.FromString("template-id")
newVM, err := client.VM().Create(ctx, &payloads.VM{
    NameLabel:       "test-vm",
    NameDescription: "Test VM",
    Template:        templateID,
})
if err != nil {
    // Handle error
}
```

### VM Lifecycle Operations

#### v1:
```go
err := xoClient.StartVm("vm-id")
if err != nil {
    // Handle error
}

err = xoClient.HaltVm("vm-id")
if err != nil {
    // Handle error
}
```

#### v2:
```go
vmID, _ := uuid.FromString("vm-id")

err := client.VM().Start(ctx, vmID)
if err != nil {
    // Handle error
}

err = client.VM().CleanShutdown(ctx, vmID)
if err != nil {
    // Handle error
}
```

## Working with UUIDs

The v2 SDK uses the `gofrs/uuid` package for type-safe UUID handling, and XO uses the version 4 UUIDs:

```go
import "github.com/gofrs/uuid"

// Creating a UUID from string
id, err := uuid.FromString("12345678-1234-1234-1234-123456789012")
if err != nil {
    // Handle error
}

// Using a constant UUID (must be valid)
id := uuid.Must(uuid.FromString("12345678-1234-1234-1234-123456789012"))

// Generating a new UUID
id, err := uuid.NewV4()
if err != nil {
    // Handle error
}
```

## Context Support

All v2 operations accept a context, allowing timeout and cancellation:

```go
// Create a context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Use context in API calls
vm, err := client.VM().GetByID(ctx, vmID)
```

## Complete Example

Here's a complete example showing how to migrate a simple use case:

### v1:
```go
package main

import (
    "fmt"

    "github.com/vatesfr/xenorchestra-go-sdk/client"
)

func main() {
    config := client.GetConfigFromEnv()
    xoClient, err := client.NewClient(config)
    if err != nil {
        panic(err)
    }

    vms, err := xoClient.GetVms(client.Vm{})
    if err != nil {
        panic(err)
    }

    for _, vm := range vms {
        fmt.Printf("VM: %s (ID: %s)\n", vm.NameLabel, vm.Id)
    }
}
```

### v2:
```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
    v2 "github.com/vatesfr/xenorchestra-go-sdk/v2"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    cfg, err := config.New()
    if err != nil {
        panic(err)
    }

    client, err := v2.New(cfg)
    if err != nil {
        panic(err)
    }

    vms, err := client.VM().List(ctx)
    if err != nil {
        panic(err)
    }

    for _, vm := range vms {
        fmt.Printf("VM: %s (ID: %s)\n", vm.NameLabel, vm.ID)
    }
}
```