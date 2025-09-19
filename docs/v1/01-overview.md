# Xen Orchestra Go SDK v1

## Overview

The Xen Orchestra Go SDK v1 provides a Go interface to interact with the Xen Orchestra API using JSON-RPC over WebSocket. This version includes structured logging capabilities and retry mechanisms for robust client operations.

## Key Features

- **JSON-RPC API Support**: Uses Xen Orchestra's JSON-RPC API over WebSocket
- **Structured Logging**: Built-in logging with configurable verbosity levels using Go's `slog` package
- **Custom Logger Support**: Ability to provide your own logger instance for better integration
- **Retry Mechanisms**: Configurable retry strategies with exponential backoff
- **Environment Variable Configuration**: Easy setup using environment variables

## Installation

```bash
go get github.com/vatesfr/xenorchestra-go-sdk/client
```

## Quick Start

```go
package main

import (
	"fmt"
	"log"

	"github.com/vatesfr/xenorchestra-go-sdk/client"
)

func main() {
	// Create client using environment variables
	config := client.GetConfigFromEnv()

	c, err := client.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	// Get all VMs
	vms, err := c.GetVms(client.Vm{
		PoolId: "<pool-id>",
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d VMs\n", len(vms))
	for _, vm := range vms {
		fmt.Printf("VM: %s (ID: %s, State: %s)\n", vm.NameLabel, vm.Id, vm.PowerState)
	}
}
```

## Logging Configuration

### Default Logger

By default, the SDK creates a structured logger that outputs to stderr:

```go
config := client.GetConfigFromEnv()
c, err := client.NewClient(config) // Uses default logger
```

The default logger level is `INFO`. Set the `XOA_DEVELOPMENT` environment variable to `true` to enable `DEBUG` level logging:

```bash
export XOA_DEVELOPMENT=true
```

### Custom Logger

You can provide your own `*slog.Logger` instance for better integration with your application's logging:

```go
package main

import (
    "log/slog"
    "os"
    
    "github.com/vatesfr/xenorchestra-go-sdk/client"
)

func main() {
    // Create custom logger
    logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))

    config := client.GetConfigFromEnv()
    c, err := client.NewClientWithLogger(config, logger)
    if err != nil {
        panic(err)
    }

    // Use client...
}
```


## Environment Variables

The SDK uses the following environment variables for configuration:

| Variable | Description | Default |
|----------|-------------|---------|
| `XOA_URL` | URL of the Xen Orchestra server (required) | - |
| `XOA_USER` | Username for authentication | - |
| `XOA_PASSWORD` | Password for authentication | - |
| `XOA_TOKEN` | Authentication token (alternative to user/password) | - |
| `XOA_INSECURE` | Set to "true" to skip TLS certificate verification | false |
| `XOA_DEVELOPMENT` | Set to "true" to enable debug logging | false |
| `XOA_RETRY_MODE` | Retry strategy ("none" or "backoff") | none |
| `XOA_RETRY_MAX_TIME` | Maximum time to wait between retries | 5m |

## Retry Configuration

The SDK supports two retry modes:

### None (Default)
No retries are performed. Operations fail immediately on error.

```bash
export XOA_RETRY_MODE=none
```

### Backoff
Exponential backoff retry strategy for certain retryable errors, particularly useful when dealing with guest boot sequences and PV driver initialization.

```bash
export XOA_RETRY_MODE=backoff
export XOA_RETRY_MAX_TIME=10m
```

## Migration to v2

For new projects, consider using the v2 SDK which provides:
- REST API support instead of JSON-RPC
- Type-safe interfaces with Go generics
- Improved error handling and context support
- Better performance and maintainability

See the [Migration Guide](../v2/03-migration-guide.md) for details on upgrading from v1 to v2.