# Contributing to FontGet

If you're interested in contributing to FontGet! This guide will help you get started.

## Getting Started

### Prerequisites
- Go 1.26.0 or later
- Git
- Basic understanding of Go and CLI development

### Development Setup
1. Fork the repository
2. Clone your fork: `git clone https://github.com/Graphixa/FontGet.git`
3. Navigate to the project: `cd FontGet`
4. Install dependencies: `go mod download`

## Code Structure

FontGet follows a clean architecture pattern:

- **`cmd/`** - CLI command implementations
- **`internal/`** - Internal packages (not for external use)
  - **`cmdutils/`**, **`shared/`** - CLI helpers and general utilities
  - **`config/`** - Configuration management
  - **`repo/`**, **`network/`**, **`sources/`** - Repository, downloads, built-in sources
  - **`installations/`** - Install provenance registry
  - **`platform/`** - OS-specific functionality
  - **`ui/`**, **`components/`** - Styling/theme and reusable TUI widgets
  - **`output/`**, **`logging/`** - Console output and file logging
  - **`onboarding/`**, **`update/`**, **`templates/`**, **`testutil/`** - Wizard, self-update, theme YAML templates, test helpers

For where new code should go, see [Codebase layout guidelines](development/guidelines/codebase-layout-guidelines.md) and the package map in [Codebase](development/codebase.md).

## Development Guidelines

### Code Style
- Follow Go standard formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions focused and small

### Adding New Commands
1. Create a new file in `cmd/` (e.g., `cmd/newcommand.go`), mirroring patterns in an existing command such as `cmd/version.go` or `cmd/info.go`
2. Register the command in `cmd/root.go`
3. Add documentation to `docs/usage.md`
4. See [Build guide](development/build-guide.md) for local builds

### Testing
- Write tests for new functionality
- Test on multiple platforms when possible
- Use the `--debug` flag for troubleshooting

### Documentation
- Update `docs/usage.md` for user-facing changes
- Update `docs/development/codebase.md` for architectural changes
- Run the documentation audit: `go run docs/maintenance/audit-flags.go cmd/`

## Pull Request Process

1. Create a feature branch: `git checkout -b feature/your-feature`
2. Make your changes
3. Test thoroughly
4. Update documentation if needed
5. Run the audit script to check documentation sync
6. Submit a pull request with a clear description

## Reporting Issues

When reporting issues, please include:
- Operating system and version
- FontGet version
- Steps to reproduce
- Expected vs actual behavior
- Any error messages

## Questions?

Feel free to open an issue for questions or discussions about the project.
