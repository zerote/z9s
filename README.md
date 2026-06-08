# z9s - K9s with cluster metrics & GitOps

> **A fork of [k9s](https://github.com/derailed/k9s) with custom features: a native cluster metrics dashboard and GitOps operator management (FluxCD).**

## 🚀 What is z9s?

**z9s** is a fork of **k9s** that keeps its look & feel and all of its Kubernetes cluster management and inspection capabilities, while adding:

- **Cluster metrics dashboard** built natively on top of the same TUI stack as k9s (`derailed/tview` + `derailed/tcell`), with no external project dependencies.
- **Prometheus metrics scraping** for historical usage series (CPU/MEM and more) on top of metrics-server.
- **GitOps operators**: detection and management of **FluxCD** directly from the TUI, with an **ArgoCD** integration planned (see below).
- **Fast toggle** (`Ctrl+N`) between the current view and the dashboards, preserving view state.

## ✨ Features

### Core
- **All of k9s**: navigation with `:`, contexts, namespaces, resources, skins and shortcuts, exactly like k9s.
- **z9sTop dashboard**: panels for Cluster Summary, Nodes and Pods, with CPU/MEM gauges.
- **Dashboard navigation**: `Tab` / `Ctrl+arrows` to move between panels, arrow keys within each panel.
- **Node detail**: pressing `Enter` on a node opens a screen with its info and the pods running on it.
- **Prometheus metrics**: historical cluster usage in addition to the point-in-time value from metrics-server.
- **Operator detection**: the cluster header shows the status of `FluxCD` and `ArgoCD` (on/off) at startup.

### GitOps — FluxCD
- **Operators page** (`Ctrl+O`): landing page listing the GitOps operators detected on the cluster. If only one operator is installed, it jumps straight to its Overview.
- **FluxCD Overview** (`<o>`): a Lens-like dashboard with:
  - **Vertical stacked bars** per resource type (Kustomizations, Helm Releases, Git Repositories, Helm Repositories, Helm Charts, OCI Repositories).
  - Status breakdown per type: **Ready**, **InProgress**, **NotReady**, **Suspended**, **Unknown**.
  - A **Flux Events** panel with the latest Flux-related Kubernetes events (Type, Message, Namespace, Involved Object, Source, Count, Age, Last Seen), scrollable with the arrow keys.
- **Kustomization management** (`<k>`): list of Kustomizations with columns **Name, Namespace, Status, Ready Message, Age** and real actions against FluxCD:
  - **Reconcile** (`<r>`) — force a reconciliation.
  - **Suspend** (`<s>`) — pause reconciliation (`spec.suspend`).
  - **Resume** (`<u>`) — resume reconciliation.

### GitOps — ArgoCD (preview / not ready yet)
> ⚠️ **Work in progress.** z9s already **detects** whether ArgoCD is installed on the cluster (shown in the header and on the Operators page via `<a>`), but the management views (Applications, sync/refresh actions, etc.) are **not implemented yet**. The goal is to offer an experience equivalent to the FluxCD integration: application overview, status breakdown, events, and actions like sync, refresh and rollback.

- **Apache 2.0 license**.

## 📋 Quick Start

```bash
# Install via Homebrew
brew tap zerote/z9s
brew install z9s

# Run
z9s
```

### Build from source

```bash
git clone https://github.com/zerote/z9s.git
cd z9s
go build -o z9s .   # or: ./start.sh
./z9s
```

### Main shortcuts

| Key | Action |
|-----|--------|
| `Ctrl+N` | Toggle between the current view and the metrics dashboard (z9sTop) |
| `Ctrl+O` | Open the GitOps Operators page (FluxCD / ArgoCD) |
| `Tab` / `Ctrl+↑↓←→` | Move between dashboard panels |
| `Enter` (on a node) | Open the node detail |
| `ESC` | Go back from a detail/page |
| `:` | k9s commands (contexts, resources, etc.) |
| `:metrics` / `:cluster` | Switch between the metrics and cluster dashboards |
| `Ctrl+C` | Quit |

### FluxCD shortcuts

| Key | Context | Action |
|-----|---------|--------|
| `<f>` | Operators page | Open the FluxCD section |
| `<o>` | FluxCD | Overview (status chart + Flux events) |
| `<k>` | FluxCD | Kustomizations list |
| `<r>` | Kustomizations | Reconcile the selected resource |
| `<s>` | Kustomizations | Suspend the selected resource |
| `<u>` | Kustomizations | Resume the selected resource |

## 🔧 Development

### Requirements

- Go 1.24 or higher
- `kubectl` configured
- Access to a Kubernetes cluster (with metrics-server and/or Prometheus for metrics)
- FluxCD installed on the cluster to use the GitOps features

### Build

```bash
# Simple build (takes the version from the code)
go build -o z9s .

# With version info via ldflags
make build        # uses VERSION from the Makefile
```

### Tests

```bash
go test ./...
```

## 📝 License

This project is licensed under the **Apache License 2.0** — see the [LICENSE](LICENSE) file.

**Attribution**: z9s is a fork based on the excellent work of [k9s](https://github.com/derailed/k9s) by Fernand Galiana (@derailed).

## 📞 Contact

- **Author**: @zerote
- **Email**: condezero@gmail.com

---

**Note**: Project under active development. Features and APIs may change before v1.0.
