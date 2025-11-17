# Agent.md

This file provides guidance to AI agents when working with code in this
repository.

## Overview

**go-qontract-reconcile** is a Go-based CLI application that provides
integrations for app-interface reconciliation. It's part of the Red Hat
App-SRE ecosystem and contains multiple specialized integrations for
validating users, managing accounts, synchronizing git partitions, and
validating PGP keys.

## Development commands

```bash
# Generate code
go generate ./...

# Build
go build

# Run tests
go test ./...

# Run specific integration
# First build the executable using following command
make gobuild
# Then run the specific integration (for example user-validator)
./go-qontract-reconcile -c config.toml user-validator

# Docker build
docker build -t go-qontract-reconcile .
```

## Architecture

### Core Components

- **Main Entry Point**: `main.go` → `cmd/` package with Cobra CLI
- **Integration Framework**: `pkg/reconcile/` - provides base
  interfaces and runners
- **Individual Integrations**: `internal/` - specific integration
  implementations
- **Shared Packages**: `pkg/` - utilities, clients, and common
  functionality

### Integration Pattern

All integrations implement the `reconcile.Integration` interface:

```go
type Integration interface {
    Setup(context.Context) error
    CurrentState(context.Context, *ResourceInventory) error
    DesiredState(context.Context, *ResourceInventory) error
    Reconcile(context.Context, *ResourceInventory) error
    LogDiff(*ResourceInventory)
}
```

## Available Integrations

### 1. User Validator (`user-validator`)

- **Location**: `internal/uservalidator/`
- **Purpose**: Validates PGP keys, usernames, and GitHub logins
- **Key Files**: `user_validator.go`, `user_validator_test.go`
- **GraphQL**: Uses generated queries via genqlient

### 2. Account Notifier (`account-notifier`)

- **Location**: `internal/accountnotifier/`
- **Purpose**: Sends PGP encrypted password notifications to users
- **Key Files**: `account_notifier.go`, `account_notifier_test.go`
- **Features**: PGP encryption, email notifications, state management

### 3. Git Partition Sync Producer (`git-partition-sync-producer`)

- **Location**: `internal/gitpartitionsync/producer/`
- **Purpose**: Produces messages for git partition synchronization
- **Key Files**: `producer.go`, `gitlab.go`, `s3.go`, `encrypt.go`,
  `tar.go`
- **Features**: GitLab integration, S3 operations, encryption (age),
  tar operations

### 4. Key Validator (`validate-key`)

- **Location**: `internal/keyvalidator/`
- **Purpose**: Validates PGP keys in user files
- **Key Files**: `key_validator.go`

## Packages

### `pkg/reconcile/`

- **integration.go**: Core integration runner with metrics and
  lifecycle management
- **validation.go**: Validation runner for validation-only operations
- **reconcile.go**: Base interfaces and configuration

### `pkg/gql/`

- GraphQL client for qontract-server communication
- Generated queries using genqlient

### `pkg/aws/`

- AWS SDK abstraction with mocking support
- Credentials management

### `pkg/vault/`

- HashiCorp Vault integration
- Multiple auth methods (token, approle, kubernetes)

### `pkg/pgp/`

- PGP operations using ProtonMail's gopenpgp
- Key validation and encryption

### `pkg/github/`

- GitHub API client integration

### `pkg/util/`

- Logging utilities (zap)
- Retry mechanisms
- Common utilities

### Integration Specific

- **genqlient**: GraphQL code generation
- **gopenpgp**: PGP operations
- **aws-sdk-go-v2**: AWS services
- **vault/api**: HashiCorp Vault
- **go-github**: GitHub API
- **go-gitlab**: GitLab API
- **age**: File encryption

## Configuration

### Configuration Methods

1. **TOML file** via `-c` flag
2. **Environment variables** (see README.md for full list)
3. **Viper-based** configuration management

## Development

### Adding New Integrations

1. **Copy Example**: Use `internal/example/` as template
2. **Implement Interface**: Implement `reconcile.Integration`
3. **GraphQL Queries**: Add to `generate.go`, update `genqlient.yaml`
4. **Generate Code**: Run `go generate ./...`
5. **Add Command**: Create new command in `cmd/`

### Testing

- **Unit Tests**: Each integration has `*_test.go` files
- **Mock Generation**: Uses `golang/mock` for AWS and other external
  services
- **Test Data**: Located in `test/data/` directory

### Code Generation

- **GraphQL**: Uses `genqlient` for type-safe GraphQL queries
- **Mocks**: Uses `golang/mock` for interface mocking
- **Command**: `go generate ./...` regenerates all generated code

### Core Dependencies

- **Cobra**: CLI framework
- **Viper**: Configuration management
- **Zap**: Structured logging
- **Prometheus**: Metrics collection
