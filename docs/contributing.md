# Contributing to FontGet

Thank you for your interest in contributing to FontGet! This guide will help you get started.

## Getting Started

### Prerequisites
- Go 1.24.4 or later
- Git
- Basic understanding of Go and CLI development

### Development Setup
1. Fork the repository
2. Clone your fork: `git clone https://github.com/yourusername/FontGet.git`
3. Navigate to the project: `cd FontGet`
4. Install dependencies: `go mod download`

## Code Structure

FontGet follows a clean architecture pattern:

- **`cmd/`** - CLI command implementations
- **`internal/`** - Internal packages (not for external use)
  - **`config/`** - Configuration management
  - **`repo/`** - Font repository operations
  - **`platform/`** - OS-specific functionality
  - **`ui/`** - User interface components
  - **`output/`** - Output management (verbose/debug)

## Development Guidelines

### Code Style
- Follow Go standard formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for exported functions
- Keep functions focused and small

### Adding New Commands
1. Create a new file in `cmd/` (e.g., `cmd/newcommand.go`)
2. Use the template in `internal/templates/command_template.go`
3. Register the command in `cmd/root.go`
4. Add documentation to `docs/help.md`

### Testing
- Write tests for new functionality
- Test on multiple platforms when possible
- Use the `--debug` flag for troubleshooting

### Documentation
- Update `docs/help.md` for user-facing changes
- Update `docs/codebase.md` for architectural changes
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
