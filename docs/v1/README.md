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

## Support

For issues, questions, or contributions, please refer to the main repository documentation.