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

## Next Steps

- [Architecture Guide](02-architecture.md) - Learn about the design patterns used in the SDK
- [Migration Guide](03-migration-guide.md) - Migrate from v1 to v2
- [Service Implementation Guide](04-service-implementation.md) - Learn how to add new services