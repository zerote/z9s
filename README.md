# z9s - K9s + Ktop Unified CLI

> **A unified Kubernetes CLI tool combining the power of k9s (cluster management) and ktop (metrics visualization)**

## 🚀 What is z9s?

**z9s** merges two powerful Kubernetes CLI tools into one seamless experience:

- **k9s** 🐶: Full-featured Kubernetes cluster management and inspection
- **ktop** 📊: Real-time Kubernetes resource metrics and visualization

## ✨ Key Features

- **Unified Interface**: Single binary for both tools
- **Mode Toggle**: Press `Ctrl+F10` to instantly switch between k9s and ktop modes
- **Persistent State**: Your context and selections are preserved when switching modes
- **Full Compatibility**: All features from both original tools work as expected
- **Apache 2.0 Licensed**: Open source, free to use and modify

## 🎯 Use Cases

- **DevOps Engineers**: Switch between cluster management and metrics analysis without restarting
- **Kubernetes Operators**: Quick context switching for different troubleshooting scenarios
- **SREs**: Unified dashboard for both config and performance monitoring

## 📋 Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/zerote/z9s.git
cd z9s

# Build from source
go build -o z9s

# Run
./z9s
```

### Key Bindings

| Key | Action |
|-----|--------|
| `Ctrl+F10` | Toggle between k9s and ktop modes |
| `q` | Quit application |
| `Ctrl+C` | Force quit |

## 🏗️ Project Structure

```
z9s/
├── cmd/                 # Command line interface
├── internal/
│   ├── app/            # Application core (modes, toggle logic)
│   ├── k9s/            # K9s mode implementation
│   ├── ktop/           # Ktop mode implementation
│   └── shared/         # Shared code (K8s client, etc.)
├── docs/               # Documentation
└── main.go             # Entry point
```

## 🔧 Development

### Prerequisites

- Go 1.25 or higher
- kubectl configured
- Access to a Kubernetes cluster

### Building

```bash
# Simple build
go build

# With version info
go build -ldflags="-X main.version=v1.0.0"
```

### Running Tests

```bash
go test ./...
```

## 📚 Documentation

- **Architecture**: See `INTEGRATION_GUIDE.md` or `KT9S_ANALYSIS.md`
- **Contributing**: See `CONTRIBUTING.md` (coming soon)
- **Original Projects**:
  - [k9s](https://github.com/derailed/k9s)
  - [ktop](https://github.com/vladimirvivien/ktop)

## 🤝 Contributing

This is a community-driven project. Contributions are welcome!

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the **Apache License 2.0** - see the [LICENSE](LICENSE) file for details.

**Attribution**: This project builds upon the excellent work of:
- [K9s](https://github.com/derailed/k9s) by Fernand Galiana (@derailed)
- [Ktop](https://github.com/vladimirvivien/ktop) by Vladimir Vivien (@vladimirvivien)

## 🐛 Issues & Support

Found a bug? Have a feature request?

- [Report an Issue](https://github.com/zerote/z9s/issues)
- [Start a Discussion](https://github.com/zerote/z9s/discussions)

## 🎓 Roadmap

- [ ] Merge k9s and ktop codebases
- [ ] Implement mode toggle (Ctrl+F10)
- [ ] Test both modes thoroughly
- [ ] Add configuration for default mode
- [ ] Add combined help page
- [ ] Performance optimization
- [ ] Advanced metrics overlays
- [ ] Plugin system compatibility

## 📞 Contact

- **Author**: @zerote (@zerote)
- **Email**: dev@z9s.dev

---

**Note**: This is an active development project. Features and APIs may change before v1.0 release.

Last Updated: June 6, 2026
