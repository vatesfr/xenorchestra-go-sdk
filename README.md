# <p align="center">Golang client for XenOrchestra API</p>
  
This is a Golang module for the [XenOrchestra](https://github.com/vatesfr/xen-orchestra) API. It provides two client implementations:

- **v1**: Uses the JSON-RPC API (legacy)
- **v2**: Uses the REST API (WIP, should be used in parallel with v1 for missing endpoints, until v2 is fully released)

## 📚 Documentation 

### v1 Documentation

The v1 client uses the JSON-RPC API and is primarily used in the [terraform-provider-xenorchestra](https://github.com/vatesfr/terraform-provider-xenorchestra) Terraform provider.

Comprehensive documentation is available in the `docs/v1` directory:

- [Overview](docs/v1/01-overview.md) - Introduction, features, and basic usage

### v2 Documentation

The v2 client uses the REST API and provides a more modern, type-safe interface. Comprehensive documentation is available in the `docs/v2` directory:

- [Overview](docs/v2/01-overview.md) - Introduction and key features
- [Architecture](docs/v2/02-architecture.md) - Design patterns and components
- [Migration Guide](docs/v2/03-migration-guide.md) - How to migrate from v1 to v2
- [Service Implementation Guide](docs/v2/04-service-implementation.md) - How to add new services

### v2 Integration Tests Documentation

Complete documentation for writing and running integration tests is available in the `v2/integration` directory:

- [README.md](v2/integration/README.md) - Overview and quick start
- [QUICK_REFERENCE.md](v2/integration/QUICK_REFERENCE.md) - Cheat sheet with essential commands and snippets
- [INTEGRATION_TESTS_GUIDE.md](v2/integration/INTEGRATION_TESTS_GUIDE.md) - Complete guide for adding tests
- [LEARNING_PATH.md](v2/integration/LEARNING_PATH.md) - Personalized learning path by profile
- [CONFIGURATION.md](v2/integration/CONFIGURATION.md) - Environment variables and configuration
- [ARCHITECTURE.md](v2/integration/ARCHITECTURE.md) - Technical deep dive into how tests work
- [EXAMPLES.md](v2/integration/EXAMPLES.md) - Practical examples and patterns
- [CODE_ANALYSIS.md](v2/integration/CODE_ANALYSIS.md) - Analysis of existing test code
- [INDEX.md](v2/integration/INDEX.md) - Complete index and navigation guide

## 🧑🏻‍💻 Usage

```shell
go get github.com/vatesfr/xenorchestra-go-sdk
```

### Examples

The SDK includes examples for both v1 and v2 clients:

- [v1 Examples](examples/v1) - Examples using the JSON-RPC API
- [v2 Examples](examples/v2) - Examples using the REST API

## 🍰 Contributing    

Contributions are what make the open source community such an amazing place to be learn, inspire, and create. Any contributions you make are **greatly appreciated**.

### Development

This project includes a Makefile to help with common development tasks:

```shell
# Run tests
make test

# Run specific client tests
make test-v1
make test-v2

# Run integration tests for v2
make test-integration

# Run linter
make lint

# Generate mocks
make mock

# Run examples
make example-v1
make example-v2

# See all available commands
make help
```

## License

MIT License