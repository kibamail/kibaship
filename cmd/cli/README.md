# Kibaship CLI

A beautiful command-line interface for managing Kibaship operator clusters, built with [Lipgloss](https://github.com/charmbracelet/lipgloss) for styled terminal output.

## Features

- 🎨 **Beautiful Interface**: Styled with colors and ASCII art
- 🚀 **Clean Output**: Beautiful help screen that prints and exits
- 📊 **Cluster Management**: Comprehensive cluster inspection and management tools
- 🔧 **Easy to Use**: Intuitive commands with helpful descriptions
- 💻 **Cross-Platform**: Works on macOS, Linux, and Windows

### Infrastructure Components

- 🌐 **Gateway API**: Custom v1.3.0 CRDs with 9 resource types
- 🔗 **Cilium CNI**: Advanced networking with Gateway API support
- 💾 **Longhorn Storage**: Distributed block storage with automatic StorageClass
- 🔐 **cert-manager**: Certificate management with HA configuration
- 🔄 **Tekton Pipelines**: Cloud-native CI/CD with 75 manifests
- 🗄️ **Valkey Operator**: Redis-compatible database operator
- 🎯 **Kibaship Operator**: Complete application lifecycle management (version matches CLI)

## Installation

Build the CLI from source:

```bash
make build-cli
```

This will create the `bin/kibaship` binary.

## Usage

### Help Screen

Run the CLI without any arguments to see the beautiful help screen:

```bash
./bin/kibaship
```

This displays a beautiful banner and help information, then exits.

### Command Mode

Use specific commands for direct operations:

```bash
# Show help
./bin/kibaship --help

# Check version
./bin/kibaship version

# Cluster management
./bin/kibaship clusters create my-cluster
./bin/kibaship clusters list
./bin/kibaship clusters destroy my-cluster

# Project management
./bin/kibaship projects create my-project
./bin/kibaship projects list
./bin/kibaship projects destroy my-project

# Application management
./bin/kibaship applications create my-app
./bin/kibaship applications list
./bin/kibaship applications destroy my-app

# Cluster destruction
./bin/kibaship clusters destroy my-cluster          # With confirmation
./bin/kibaship clusters destroy my-cluster --force  # Skip confirmation
./bin/kibaship clusters destroy --all              # Destroy all clusters
./bin/kibaship clusters destroy --all --force      # Destroy all without confirmation
```

## Available Commands

### Clusters

| Command                   | Description                                      |
| ------------------------- | ------------------------------------------------ |
| `clusters create [name]`  | Create a new Kind cluster with Kibaship operator |
| `clusters list`           | List all available clusters                      |
| `clusters destroy [name]` | Destroy a cluster and clean up resources         |
| `clusters destroy --all`  | Destroy all Kibaship clusters                    |

### Projects

| Command                   | Description                              |
| ------------------------- | ---------------------------------------- |
| `projects create [name]`  | Create a new Kibaship project            |
| `projects list`           | List all Kibaship projects               |
| `projects destroy [name]` | Destroy a project and clean up resources |

### Applications

| Command                       | Description                                   |
| ----------------------------- | --------------------------------------------- |
| `applications create [name]`  | Create a new application deployment           |
| `applications list`           | List all deployed applications                |
| `applications destroy [name]` | Destroy an application and clean up resources |

### General

| Command   | Description              |
| --------- | ------------------------ |
| `version` | Show version information |

## Development

### Running from Source

```bash
# Run the CLI directly
make run-cli

# Or use go run
go run ./cmd/cli/main.go
```

### Building

```bash
# Build the binary
make build-cli

# The binary will be available at bin/kibaship
```

## Dependencies

- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Styling and layout
- [Cobra](https://github.com/spf13/cobra) - CLI framework

## Development Builds

For development builds, you can specify which operator version to install using the `KIBASHIP_VERSION` environment variable:

```bash
# Use a specific operator version for development
export KIBASHIP_VERSION=v0.1.4
./bin/kibaship clusters create dev-cluster \
  --operator-domain dev.kibaship.com \
  --operator-webhook-url https://webhook.dev.kibaship.com/kibaship

# Or set it inline
KIBASHIP_VERSION=v0.1.2 ./bin/kibaship clusters create test-cluster \
  --operator-domain test.kibaship.com \
  --operator-webhook-url https://webhook.test.kibaship.com/kibaship
```

**Version Resolution for Development Builds:**
1. If `KIBASHIP_VERSION` environment variable is set → Use that version
2. If not set → Default to `v0.1.3`

**Production Builds:**
- Version is set via ldflags at build time
- CLI and operator versions always match exactly

## Contributing

This CLI is part of the Kibaship operator project. Contributions are welcome!

## License

This project is licensed under the same license as the Kibaship operator.
