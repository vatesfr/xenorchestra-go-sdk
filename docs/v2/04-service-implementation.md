# Service Implementation Guide

This guide explains how to implement a new service in the Xen Orchestra Go SDK v2. We'll use the existing VM service as a reference.

## Overview of Service Implementation

Adding a new service to the SDK involves these steps:

1. Define the service interface in the library package
2. Create the payload structs in the payloads package
3. Implement the service in its own package
4. Register the service in the main client

## Step 1: Define the Service Interface

First, create an interface in the `library` package that defines the operations for your service.

Example for a hypothetical Snapshot service:

```go
// pkg/services/library/snapshot.go
package library

import (
    "context"
    
    "github.com/gofrs/uuid"
    "github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
)

//go:generate mockgen --build_flags=--mod=mod --destination mock/snapshot.go . Snapshot

type Snapshot interface {
    GetByID(ctx context.Context, id uuid.UUID) (*payloads.Snapshot, error)
    // Add other methods as needed
}
```

Add this interface to the Library interface:

```go
// pkg/services/library/library.go
package library

type Library interface {
    VM() VM
    Snapshot() Snapshot  // Add this line
    // Other services...
}
```

## Step 2: Create Payload Structs

Define the data structures needed for your service in the payloads package:

```go
// pkg/payloads/snapshot.go
package payloads

import (
    "github.com/gofrs/uuid"
    "time"
)

type Snapshot struct {
    ID           uuid.UUID `json:"id,omitempty"`
    NameLabel    string    `json:"name_label"`
    // Other fields as needed
}
```

## Step 3: Implement the Service

Create a new package for your service implementation:

```go
// pkg/services/snapshot/service.go
package snapshot

import (
    "context"
    "fmt"
    
    "github.com/gofrs/uuid"
    "github.com/vatesfr/xenorchestra-go-sdk/internal/common/logger"
    "github.com/vatesfr/xenorchestra-go-sdk/pkg/payloads"
    "github.com/vatesfr/xenorchestra-go-sdk/pkg/services/library"
    "github.com/vatesfr/xenorchestra-go-sdk/v2/client"
)

type Service struct {
    client *client.Client
    log    *logger.Logger
}

func New(client *client.Client, log *logger.Logger) library.Snapshot {
    return &Service{client: client, log: log}
}

func (s *Service) formatPath(path string) string {
    return fmt.Sprintf("/rest/v0/%s", path)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*payloads.Snapshot, error) {
    var result payloads.Snapshot
    err := client.TypedGet(ctx, s.client, s.formatPath(fmt.Sprintf("snapshots/%s", id)), struct{}{}, &result)
    if err != nil {
        return nil, err
    }
    return &result, nil
}
```

## Step 4: Write Tests for the Service

Check vm/service_test.go for reference:

```go
// pkg/services/snapshot/service_test.go
```

## Step 5: Register the Service in the Main Client

Update the `XOClient` struct in `v2/xo.go` to include your new service:

```go
// v2/xo.go
package v2

import (
    // imports...
    "github.com/vatesfr/xenorchestra-go-sdk/pkg/services/snapshot"
)

type XOClient struct {
    vmService       library.VM
    // other services...
    snapshotService library.Snapshot  // Add this line
}

func New(config *config.Config) (library.Library, error) {
    client, err := client.New(config)
    if err != nil {
        return nil, err
    }

    log, err := logger.New(config.Development, config.LogOutputPaths, config.LogErrorOutputPaths)
    if err != nil {
        return nil, err
    }

    return &XOClient{
        vmService:       vm.New(client, log),
        // other services...
        snapshotService: snapshot.New(client, log),  // Add this line
    }, nil
}

func (c *XOClient) VM() library.VM {
    return c.vmService
}
// other services...
func (c *XOClient) Snapshot() library.Snapshot {  // Add this method
    return c.snapshotService
}
```
## Step 6: Document Your Service

Add documentation for your service including:

1. Overview of what the service does
2. Examples of how to use it
3. Any special considerations or limitations

## Example Usage

After implementation, your service can be used like this:

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/gofrs/uuid"
    "github.com/vatesfr/xenorchestra-go-sdk/pkg/config"
    v2 "github.com/vatesfr/xenorchestra-go-sdk/v2"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
    defer cancel()
    
    cfg, err := config.New()
    if err != nil {
        panic(err)
    }
    
    client, err := v2.New(cfg)
    if err != nil {
        panic(err)
    }
    
    // Use your new service
    vmID := uuid.Must(uuid.FromString("87654321-4321-4321-4321-210987654321"))
    snapshots, err := client.Snapshot().List(ctx, vmID)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Found %d snapshots\n", len(snapshots))
    for _, snapshot := range snapshots {
        fmt.Printf("- %s (Created: %s)\n", snapshot.NameLabel, snapshot.CreationDate)
    }
}
```

## Best Practices

- Follow existing patterns for consistency
- Use proper error handling and logging
- Implement comprehensive tests
- Document public APIs
- Handle asynchronous operations properly
- Provide context support for all methods
- Use type-safe UUID handling throughout
