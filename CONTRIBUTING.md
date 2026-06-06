# Contributing to KT9S

First off, thanks for your interest in contributing! 🎉

## Code of Conduct

Be respectful and constructive. We're all here to make z9s better.

## How to Contribute

### Reporting Bugs

Found a bug? Open an issue with:
- Description of the bug
- Steps to reproduce
- Expected vs actual behavior
- Your environment (OS, Go version, etc.)

### Suggesting Enhancements

Want a new feature? Open an issue with:
- Clear description of the enhancement
- Use cases
- Possible implementation approaches

### Submitting Pull Requests

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests (`make test`)
5. Run linter (`make lint`)
6. Commit with clear messages
7. Push to your fork
8. Open a Pull Request with:
   - Description of changes
   - Link to related issue
   - Testing done

## Development Setup

```bash
# Clone
git clone https://github.com/your-username/z9s.git
cd z9s

# Setup
./setup.sh  # or setup.bat on Windows

# Build
make build

# Test
make test

# Run linter
make lint
```

## Code Style

- Follow Go conventions (use `gofmt`)
- Add comments for exported functions
- Keep functions focused and testable
- Use meaningful variable names

## Testing

- Write tests for new features
- Run `make test` before submitting PR
- Aim for >80% code coverage

## Documentation

- Update README.md if needed
- Add comments to complex logic
- Document any new configuration options

## Commit Messages

Use clear, descriptive commit messages:
```
feat: Add toggle animation
fix: Resolve crash on Ctrl+F10
docs: Update setup guide
test: Add AppManager tests
```

## Issues We Need Help With

Check the GitHub issues labeled:
- `good first issue` - Great for new contributors
- `help wanted` - Needs community support
- `documentation` - Documentation tasks

## Communication

- GitHub Issues for bugs/features
- GitHub Discussions for ideas
- Be respectful and inclusive

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

---

Thanks for contributing! 🚀
