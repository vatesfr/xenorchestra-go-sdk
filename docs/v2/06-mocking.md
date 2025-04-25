# Mocking in the Xen Orchestra SDK

This document explains how to use mocking in the Xen Orchestra v2 SDK, which is essential for writing effective unit tests. Mocks should be generated in your local environment or in CI using the makefile and/or the mockgen command tool however they should be committed to the repository.

## Overview

The SDK uses [mockgen](https://github.com/uber-go/mock), a mock generator from the Go standard library, to create mock implementations of interfaces. These mocks allow you to test individual components in isolation by simulating behavior of their dependencies.

## Setting up Mockgen

### Option 1: Using Makefile

The simplest way to generate all mocks is to use the Makefile target:

```bash
make mock
```

This command:
1. Installs mockgen if it's not already installed
2. Runs `go generate ./...` to generate all mocks in the project

### Option 2: Manual Installation and Generation

If you prefer to run commands manually:

1. Install mockgen:
   ```bash
   go install go.uber.org/mock/mockgen@v0.5.1
   ```

2. Generate mocks for all interfaces:
   ```bash
   go generate ./...
   ```

3. Or generate a specific mock:
   ```bash
   go generate ./pkg/services/library/vm.go
   ```

## How Mocking is Set Up

In each interface file, you'll find a `go:generate` comment that tells the Go toolchain how to generate a mock for that interface. For example, in `pkg/services/library/vm.go`:

```go
//go:generate mockgen --build_flags=--mod=mod --destination mock/vm.go . VM
```

This directive instructs mockgen to:
- Use the `--mod=mod` build flag to respect the module
- Generate the mock implementation in the `mock/vm.go` file
- Create a mock for the `VM` interface in the current package

## Adding Mocks for New Interfaces

When you create a new interface:

1. Add the `go:generate` comment above your interface definition:
   ```go
   //go:generate mockgen --build_flags=--mod=mod --destination mock/myinterface.go . MyInterface
   ```

2. Run `make mock` or `go generate ./...` to create the mock implementation

3. Import and use the mock in your tests

## Examples 

### See the file **pkg/services/backup/service_test.go** for an example of how to use the generated mocks in a test.