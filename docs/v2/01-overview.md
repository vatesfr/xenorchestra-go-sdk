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

    // List all VMs
    vms, err := client.VM().List(ctx)
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
- `XOA_INSECURE`: Set to "true" to skip TLS certificate verification
- `XOA_DEVELOPMENT`: Set to "true" to enable development mode with additional logging
- `XOA_RETRY_MODE`: Retry strategy ("none" or "backoff")
- `XOA_RETRY_MAX_TIME`: Maximum time to wait between retries (default: 5 minutes)

## Next Steps

- [Architecture Guide](02-architecture.md) - Learn about the design patterns used in the SDK
- [Migration Guide](03-migration-guide.md) - Migrate from v1 to v2
- [Service Implementation Guide](04-service-implementation.md) - Learn how to add new services