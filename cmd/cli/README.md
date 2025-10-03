# Kibaship CLI

A beautiful command-line interface for managing Kibaship operator clusters, built with [Lipgloss](https://github.com/charmbracelet/lipgloss) for styled terminal output.

## Features

- 🎨 **Beautiful Interface**: Styled with colors and ASCII art
- 🚀 **Clean Output**: Beautiful help screen that prints and exits
- 📊 **Cluster Management**: Comprehensive cluster inspection and management tools
- 🔧 **Easy to Use**: Intuitive commands with helpful descriptions
- 💻 **Cross-Platform**: Works on macOS, Linux, and Windows

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
| Command | Description |
|---------|-------------|
| `clusters create [name]` | Create a new Kind cluster with Kibaship operator |
| `clusters list` | List all available clusters |
| `clusters destroy [name]` | Destroy a cluster and clean up resources |
| `clusters destroy --all` | Destroy all Kibaship clusters |

### Projects
| Command | Description |
|---------|-------------|
| `projects create [name]` | Create a new Kibaship project |
| `projects list` | List all Kibaship projects |
| `projects destroy [name]` | Destroy a project and clean up resources |

### Applications
| Command | Description |
|---------|-------------|
| `applications create [name]` | Create a new application deployment |
| `applications list` | List all deployed applications |
| `applications destroy [name]` | Destroy an application and clean up resources |

### General
| Command | Description |
|---------|-------------|
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

## Contributing

This CLI is part of the Kibaship operator project. Contributions are welcome!

## License

This project is licensed under the same license as the Kibaship operator.
