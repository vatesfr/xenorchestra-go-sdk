# Xen Orchestra Go SDK v2

## Overview

The Xen Orchestra Go SDK v2 provides a modern, type-safe interface to interact with the Xen Orchestra API. This version moves from the JSON-RPC API used in v1 to a REST API implementation, offering significant improvements in usability, maintainability, and type safety.

## Key Features

- **REST API Support**: Uses Xen Orchestra's modern REST API
- **Type Safety**: Leverages Go generics for type-safe API interactions
- **Method Chaining**: Improved API ergonomics with `client.VM().Create()` pattern
- **Context Support**: All operations support context for better timeout and cancellation handling
- **UUID Support**: Native UUID type support instead of string IDs
- **Structured Logging**: Built-in logging with configurable verbosity levels
- **Interface-Based Design**: Clean interfaces for better testability and mocking
- **Go Mocking**: Mocking is supported for all interfaces by adding the `//go:generate mockgen` tag in the interface file

## Installation

```bash
go get github.com/vatesfr/xenorchestra-go-sdk/v2
```

## Quick Start

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
    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()

    // Initialize configuration
    cfg, err := config.New()
    if err != nil {
        panic(err)
    }

    // Create client
    client, err := v2.New(cfg)
    if err != nil {
        panic(err)
    }

    // List all VMs (GetAll with no limit and no filter)
    vms, err := client.VM().GetAll(ctx, 0, "")
    if err != nil {
        panic(err)
    }

    // Display VMs
    fmt.Printf("Found %d VMs\n", len(vms))
    for i, vm := range vms {
        fmt.Printf("%d. %s (ID: %s, Power: %s)\n", i+1, vm.NameLabel, vm.ID, vm.PowerState)
    }
}
```

## Environment Variables

The SDK uses the following environment variables for configuration:

- `XOA_URL`: URL of the Xen Orchestra server
- `XOA_USER`: Username for authentication
- `XOA_PASSWORD`: Password for authentication
- `XOA_TOKEN`: Authentication token (recommended: can be used instead of user/password)
- `XOA_INSECURE`: Set to "true" to skip TLS certificate verification
- `XOA_DEVELOPMENT`: Set to "true" to enable development mode with additional logging
- `XOA_RETRY_MODE`: Retry strategy ("none" or "backoff")
- `XOA_RETRY_MAX_TIME`: Maximum time to wait between retries (default: 5 minutes)
- `XOA_CLIENT_TIMEOUT`: HTTP client timeout (default: 30 seconds)


### Custom Logging Sinks

The SDK uses `uber-go/zap` for logging. You can register custom sinks to redirect logs to your own infrastructure.

Example of registering a custom sink:

```go
import (
    "fmt"
    "net/url"
    "go.uber.org/zap"
)

type MySink struct {
    // your custom sink fields
}

func (s *MySink) Write(p []byte) (n int, err error) {
    fmt.Printf("Custom Log: %s", p)
    return len(p), nil
}

func (s *MySink) Sync() error { return nil }
func (s *MySink) Close() error { return nil }

func init() {
    _ = zap.RegisterSink("mysink", func(u *url.URL) (zap.Sink, error) {
        return &MySink{}, nil
    })
}

func main() {
    cfg, _ := config.New()
    cfg.LogOutputPaths = []string{"stdout", "mysink://"}
    
    client, _ := v2.New(cfg)
    // ...
}
```

This is particularly useful for integration with test runners, as seen in the SDK's own integration tests.

## Error Handling

When the Xen Orchestra REST API answers with a non-2xx status, every v2
service returns an `*apierror.Error` (package
`github.com/vatesfr/xenorchestra-go-sdk/pkg/apierror`). It carries the status
code, the request method and URL, the raw body and, when the body is a Xen
Orchestra JSON error, its `error` message and `data`.

Inspect it with the helpers instead of parsing the error text:

```go
vm, err := client.VM().GetByID(ctx, vmID)
switch {
case err == nil:
    // use vm
case apierror.IsNoSuchObject(err, payloads.ResourceTypeVM, vmID.String()):
    // Xen Orchestra itself answered 404 for this VM:
    // {"error":"no such VM <id>","data":{"id":"<id>","type":"VM"}}
case apierror.IsNotFound(err):
    // A 404 that does not name this VM: a reverse proxy, a wrong URL...
default:
    if apiErr, ok := apierror.As(err); ok {
        log.Printf("XO answered %d: %s", apiErr.StatusCode, apiErr.Message)
    }
}
```

`IsNoSuchObject` is the strict check: only Xen Orchestra can echo the
requested id in `data.id`, so a 404 from another server is never mistaken for
a missing object. `err.Error()` keeps the historical
`API error: <status> - <body>` text.

When wrapping SDK errors in your own code, use `%w` so `errors.As` keeps
working.

## Next Steps

- [Architecture Guide](02-architecture.md) - Learn about the design patterns used in the SDK
- [Migration Guide](03-migration-guide.md) - Migrate from v1 to v2
- [Service Implementation Guide](04-service-implementation.md) - Learn how to add new services