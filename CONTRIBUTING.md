# Contributing to JUNKyard

Thanks for your interest in contributing to JUNKyard! This document provides guidelines and instructions for contributing.

## Code of Conduct

Please be respectful and constructive in all interactions.

## Getting Started

1. **Fork** the repository on GitHub
2. **Clone** your fork locally:
   ```bash
   git clone https://github.com/YOUR-USERNAME/junkyard.git
   cd junkyard
   ```

3. **Create a feature branch**:
   ```bash
   git checkout -b feature/your-feature-name
   ```

## Development Setup

### Prerequisites
- Go 1.21+
- SQLite development libraries
- Docker (for cross-compilation)

### Build

```bash
# Build locally
make build

# Build for Linux (cross-compile)
make build-linux

# Run tests
make test

# Format code
make fmt

# Run linter
make lint
```

## Commit Guidelines

- Write clear, descriptive commit messages
- Reference issues when applicable: `Fixes #123`
- Keep commits focused and atomic
- Example: `feat: add query pagination to API`

## Pull Request Process

1. **Update** your fork from upstream:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Push** your changes:
   ```bash
   git push origin feature/your-feature-name
   ```

3. **Open a PR** with:
   - Clear title and description
   - Reference to related issues
   - Any relevant testing notes

4. **Respond** to review feedback promptly

## Code Style

- Follow standard Go conventions
- Run `make fmt` before committing
- Run `make vet` for static analysis
- Comments should explain *why*, not *what*
- Keep functions small and focused

## Testing

- Write tests for new functionality
- Maintain >80% code coverage
- Run full test suite: `make test`

## Documentation

- Update [docs/API.md](docs/API.md) for API changes
- Update [docs/INSTALL.md](docs/INSTALL.md) for setup changes
- Update [README.md](README.md) for feature additions
- Add examples to [docs/USAGE.md](docs/USAGE.md)

## Performance Considerations

- SQLite queries must be optimized
- Keep memory usage minimal (<50MB for 10k logs/day)
- Batch operations are preferred for bulk ingestion
- Full-text search index should be maintained

## Areas for Contribution

### High Priority
- [ ] Complete REST API handlers (Phase 4)
- [ ] Implement Web UI (Phase 4)
- [ ] Build CLI visualization (Phase 5)
- [ ] Main server integration (Phase 6)
- [ ] CI/CD automation (Phase 7)

### Medium Priority
- [ ] Performance benchmarks
- [ ] Additional ingestion formats (JSON Lines, etc.)
- [ ] Advanced filtering options
- [ ] Prometheus metrics export

### Nice to Have
- [ ] Alternative databases (PostgreSQL support)
- [ ] Grafana integration
- [ ] Mobile app
- [ ] Plugin system

## Reporting Issues

- **Bugs**: Include reproduction steps, OS, Go version
- **Features**: Describe use case and expected behavior
- **Questions**: Check docs first, then ask in issues

## License

By contributing, you agree that your work will be licensed under the MIT License.

## Questions?

- Check the [README](README.md)
- Review [API docs](docs/API.md)
- Open an issue for discussion

Thank you for helping make JUNKyard better! 🗑️
