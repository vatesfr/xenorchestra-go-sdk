# Xen Orchestra Go SDK v1 Documentation

This directory contains documentation for version 1 of the Xen Orchestra Go SDK.

## Documentation Index

- [**Overview**](01-overview.md) - Getting started with the SDK, features, and basic usage

## Quick Links

### For New Users
Start with the [Overview](01-overview.md) to understand the SDK's capabilities and see basic usage examples.

## Version Information

This documentation covers the JSON-RPC based v1 SDK. For new projects, consider using v2 which provides:
- REST API support
- Better type safety
- Improved performance

See the [v2 documentation](../v2/) for migration information.

## Environment Setup

Quick environment variable reference:

```bash
# Required
export XOA_URL="wss://your-xoa-server"
export XOA_USER="your-username"
export XOA_PASSWORD="your-password"
# Alternative to user/password
export XOA_TOKEN="your-xoa-token"

# Optional - Logging
export XOA_DEVELOPMENT="true"  # Enable debug logging

# Optional - Retries
export XOA_RETRY_MODE="backoff"
export XOA_RETRY_MAX_TIME="10m"

# Optional - Security (development only)
export XOA_INSECURE="true"
```

## Diskless cloud-init integration test

`TestCreateVmDisklessCloudConfig` requires a real XO JSON-RPC endpoint;
`xo-api-sim` only supports REST. Configure the connection above and the existing
v1 integration fixtures:

```bash
export XOA_POOL="your-pool-name"
export XOA_TEMPLATE="your-template-name"
export XOA_DISKLESS_TEMPLATE="Other install media"
# Optional: use a writable SR in the same pool, different from its default SR,
# to verify that CloudConfigSrId overrides the default.
export XOA_CLOUD_CONFIG_SR="your-other-sr-id"

go test ./client -run '^TestCreateVmDisklessCloudConfig$' -v
```

The two subtests create halted VMs without OS disks and check that their only
disk is `XO CloudConfigDrive` on the expected SR. Each test VM is deleted during
cleanup, including when creation fails after the VM has been allocated. Without
`XOA_CLOUD_CONFIG_SR`, the explicit-SR case also uses the pool's default SR.
The test is skipped when the required integration environment is absent.
It checks drive creation and placement, but does not boot a guest or verify the
contents of the drive.

## Support

For issues, questions, or contributions, please refer to the main repository documentation.
