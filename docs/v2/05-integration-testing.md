# Xen Orchestra v2 SDK Integration Tests

This document explains how to set up and run integration tests for the Xen Orchestra v2 SDK. These tests validate that the SDK works correctly against a real Xen Orchestra instance.

## Overview

The integration tests connect to a real Xen Orchestra instance running in a lab environment. These tests can be run:

1. Locally on your development machine
2. Automatically in GitHub Actions CI/CD pipeline

For secure access to lab environments, we use WireGuard VPN (see [WireGuard Setup](#wireguard-setup) section below).

## Requirements

- A running Xen Orchestra instance
- Go 1.20 or higher
- Environment variables set for authentication and resource identification
- WireGuard (for connecting to lab environments)

## Environment Variables

The following environment variables are required for the integration tests:

```
XOA_URL                      # WebSocket URL to Xen Orchestra (e.g., wss://xoa.example.com)
XOA_USER                     # Username for authentication
XOA_PASSWORD                 # Password for authentication (or use XOA_TOKEN instead)
XOA_TOKEN                    # Authentication token (alternative to username/password)
XOA_INTEGRATION_TESTS=true   # Flag to enable integration tests
XOA_POOL_ID                  # UUID of a pool to use for tests
XOA_TEMPLATE_ID              # UUID of a VM template to use for tests
XOA_NETWORK_ID               # UUID of a network to use for tests
XOA_STORAGE_ID               # UUID of a storage repository (optional, for backup tests)
XOA_SKIP_TEARDOWN            # If set to "true", resources created during tests will not be deleted (useful for debugging)
```

## Running Tests

You can run the integration tests using the following commands:

```bash
# Run all tests
make run-integration-tests

# Or manually:
go test -v ./v2/integration

# Run specific tests
go test -v ./v2/integration -run TestVM_CRUD

# Skip teardown of created resources
go test -v ./v2/integration -skip-teardown
```

The test output will be saved to `v2/integration/integration_tests.log` when using the Makefile target.

## Test Structure

The integration tests have the following structure:

1. `setup.go` - Contains common setup code for the tests
2. `vm_test.go` - Tests for VM operations (CRUD, lifecycle)
3. `snapshot_test.go` - Tests for snapshot operations
4. `restore_test.go` - Tests for restore operations
5. `backup_test.go` - Tests for backup operations

## Adding New Tests

When adding new tests, follow these guidelines:

1. Create a new file named `{service}_test.go`
2. Use the `Setup()` function to create a test client
3. Generate unique resource names using `tc.GenerateResourceName()`
4. Clean up any leftover resources from previous test runs
5. If your test creates resources, clean them up at the end of the test if `tc.SkipTeardown` is false

## Troubleshooting

### VPN Connection Issues

If you can't connect to the Xen Orchestra server:

1. Check that the WireGuard connection is active:
   ```
   sudo wg show
   ```

2. Verify your WireGuard configuration matches the server's configuration

3. Test basic connectivity:
   ```
   ping <xoa-server-ip>
   ```

### Integration Test Failures

If the integration tests fail:

1. Check the test logs for specific error messages

2. Verify that all required environment variables are set correctly

3. Ensure the Xen Orchestra instance is running and accessible

4. Try running a single test to isolate the issue:
   ```
   go test -v ./v2/integration -run TestAuthentication
   ```

## Best Practices

1. **Never commit private keys** to the repository

2. Keep the WireGuard VPN connected only when necessary

3. Use the `XOA_SKIP_TEARDOWN=true` environment variable during development to inspect resources

4. Clean up any leftover resources manually if tests fail unexpectedly 