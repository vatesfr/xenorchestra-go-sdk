# Architecture Guide

The v2 SDK uses a clean, interface-based architecture that makes it easy to use, test, and extend. This document explains the key architectural components.

## Component Overview


The SDK consists of these main components:
```
v2/
├── client/ # HTTP client implementation
├── xo.go # Main client entry point
pkg/
├── config/ # Configuration handling
├── payloads/ # API data structures
├── services/ # Service implementations
├── library/ # Service interfaces
├── vm/ # VM service implementation
└── ... # Other service implementations
internal/
└── common/ # Shared utilities
├── core/ # Core types and constants
└── logger/ # Logging implementation
```


## Key Design Patterns

### Interface-Based Design

The SDK uses interfaces to define service contracts in the `library` package. These interfaces define the operations available for each resource type.

Example from `library/vm.go`:

```go
type VM interface {
    GetByID(ctx context.Context, id uuid.UUID) (*payloads.VM, error)
    List(ctx context.Context) ([]*payloads.VM, error)
    Create(ctx context.Context, vm *payloads.VM) (*payloads.VM, error)
    // ... other methods
}
```

This approach allows for:
- Clear API documentation
- Easy mocking for tests
- Swappable implementations

### Service Registry Pattern

The `XOClient` struct acts as a registry for service implementations, providing a coherent API for accessing different services:

```go
// In v2/xo.go
type XOClient struct {
    vmService library.VM
    // other services will be added here
}

func (c *XOClient) VM() library.VM {
    return c.vmService
}
```

This allows for method chaining like `client.VM().Create(...)`.

### Generic HTTP Methods

The client package uses Go generics to provide type-safe HTTP methods:

```go
// In client/client.go
func TypedGet[P any, R any](ctx context.Context, c *Client, endpoint string, params P, result *R) error {
    // implementation
}
```

This makes service implementations cleaner and more type-safe.

### Payload Structs

API data structures are defined in the `payloads` package, separate from service logic:

```go
// In pkg/payloads/vm.go
type VM struct {
    ID              uuid.UUID         `json:"id,omitempty"`
    NameLabel       string            `json:"name_label"`
    NameDescription string            `json:"name_description"`
    // ... other fields
}
```

## Client Initialization Flow

1. Configuration is loaded from environment variables or provided directly
2. The HTTP client is created with the appropriate authentication
3. Service implementations are initialized
4. The main client registers all services
5. The client is ready to use

## Async Operations

This implementation is subject to change in a future version of the XO REST API, likely adopting long polling or Server-Sent Events (SSE) for improved asynchronous communication.

Some API operations (like VM creation) are asynchronous. The SDK handles this by:

1. Making the initial request that returns a task URL
2. Polling the task status until completion
3. Retrieving the final result when the task succeeds

This is encapsulated in service methods for a clean API.

## Error Handling

Errors are propagated from the HTTP client to the service methods and finally to the caller. Detailed error information is available to help diagnose issues.

## Logging Implementation

The SDK uses a structured logger based on zap for efficient and informative logging. Log levels are automatically adjusted based on whether the SDK is in development or production mode.
The SDK implements a structured logger based on the [zap](https://github.com/uber-go/zap) logging library. The logger is configured in the `internal/common/logger` package:

```go
// In internal/common/logger/logger.go
type Logger struct {
	*zap.Logger
}
```

See https://github.com/uber-go/zap for more information on the logging library used.
