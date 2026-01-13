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
- [Lazy v1 client loading](docs/v2/05-lazy-v1-client.md) - How and why v1 client is lazy loaded in v2
- [Integration test guide](docs/v2/06-integration-test-guide.md) - How integration tests work and how to add new ones

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