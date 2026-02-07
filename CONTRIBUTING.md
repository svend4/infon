# Contributing to TVCP

Thank you for your interest in contributing to TVCP (Terminal Video Communication Platform)!

## 🚀 Project Status

TVCP is currently in **pre-alpha** stage. We're building the foundation and welcome early contributors.

## 🎯 Areas Where We Need Help

### High Priority
- [ ] **Go Development** — Core codec and networking implementation
- [ ] **Video Encoding** — Optimize .babe codec performance
- [ ] **Terminal Rendering** — Improve rendering quality and compatibility
- [ ] **Testing** — Test on various terminals and platforms
- [ ] **Documentation** — API docs, tutorials, examples

### Medium Priority
- [ ] **Audio Codecs** — G.722, Opus, Codec2 integration
- [ ] **Network Stack** — Yggdrasil integration, congestion control
- [ ] **CI/CD** — GitHub Actions workflows
- [ ] **Package Management** — Homebrew, apt, pacman packages

### Future
- [ ] Mobile clients (iOS, Android)
- [ ] Web portal
- [ ] Enterprise features

## 🛠️ Development Setup

### Prerequisites
- Go 1.21 or later
- Git
- Linux, macOS, or Windows
- Terminal with True Color support (recommended)

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/svend4/infon.git
cd infon

# Initialize dependencies
go mod tidy

# Build
make build

# Run
./bin/tvcp version
```

## 📝 Contribution Workflow

1. **Fork** the repository
2. **Create a branch** for your feature/fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes** and commit:
   ```bash
   git add .
   git commit -m "Add feature: description"
   ```
4. **Push** to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```
5. **Open a Pull Request** against `main` branch

## ✅ Code Guidelines

### Go Code Style
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` to format code
- Run `go vet` before committing
- Add tests for new functionality

### Commit Messages
Use conventional commits format:
```
feat: add new feature
fix: fix bug in codec
docs: update README
test: add tests for encoder
refactor: restructure network package
```

### Pull Request Guidelines
- Clear description of what you're changing and why
- Reference any related issues
- Ensure tests pass
- Update documentation if needed

## 🧪 Testing

```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/codec/babe/...

# Run with coverage
go test -cover ./...
```

## 📚 Documentation

When adding new features:
- Add godoc comments to exported functions
- Update relevant README files
- Add examples if applicable

## 🐛 Reporting Bugs

Open an issue with:
- Clear title
- Steps to reproduce
- Expected vs actual behavior
- Terminal type and OS
- TVCP version (`tvcp version`)

## 💡 Feature Requests

We welcome feature ideas! Please:
- Check existing issues first
- Describe the use case
- Explain expected behavior
- Consider implementation complexity

## 📖 Resources

- [Business Plan](tvcp-business-plan.md) — Full project vision and roadmap
- [Technical Appendix](tvcp-appendix.md) — Deep technical details
- [Repository Review](REPOSITORY_REVIEW.md) — Current status and recommendations

## 🙏 Code of Conduct

Be respectful and constructive. We're building this together.

## 📧 Questions?

- Open a GitHub Discussion
- File an issue
- Email: stefan.engel.de@gmail.com

Thank you for contributing to TVCP! 🎉
