# z9s - K9s con métricas de cluster

> **Un fork de [k9s](https://github.com/derailed/k9s) con modificaciones propias y un dashboard de métricas de cluster extraídas desde Prometheus.**

## 🚀 ¿Qué es z9s?

**z9s** es un fork de **k9s** que mantiene su look & feel y todas sus capacidades de gestión e inspección de clusters Kubernetes, y le agrega:

- **Dashboard de métricas de cluster** construido de forma nativa sobre el mismo stack TUI de k9s (`derailed/tview` + `derailed/tcell`), sin depender de proyectos externos.
- **Scraping de métricas desde Prometheus** para series históricas de uso (CPU/MEM y más) además de metrics-server.
- **Toggle rápido** (`Ctrl+N`) entre la vista actual y el dashboard, preservando el estado de la vista.

## ✨ Características

- **Todo k9s**: navegación con `:`, contextos, namespaces, recursos, skins y atajos tal cual k9s.
- **Dashboard z9sTop**: paneles con Cluster Summary, Nodes y Pods, con gauges de CPU/MEM.
- **Navegación del dashboard**: `Tab` / `Ctrl+flechas` para moverte entre paneles, flechas dentro de cada panel.
- **Detalle de nodo**: `Enter` sobre un nodo abre una pantalla con su info y los pods que corren en él.
- **Métricas desde Prometheus**: histórico de uso del cluster además del valor puntual de metrics-server.
- **Licencia Apache 2.0**.

## 📋 Quick Start

```bash
# Clonar el repositorio
git clone https://github.com/zerote/z9s.git
cd z9s

# Build
./start.sh        # o: go build -o z9s .

# Ejecutar
./z9s
```

### Atajos principales

| Tecla | Acción |
|-------|--------|
| `Ctrl+N` | Toggle entre la vista actual y el dashboard de métricas (z9sTop) |
| `Tab` / `Ctrl+↑↓←→` | Moverse entre paneles del dashboard |
| `Enter` (sobre un nodo) | Abrir el detalle del nodo |
| `ESC` | Volver desde el detalle |
| `:` | Comandos de k9s (contextos, recursos, etc.) |
| `Ctrl+C` | Salir |

## 🔧 Desarrollo

### Requisitos

- Go 1.24 o superior
- `kubectl` configurado
- Acceso a un cluster Kubernetes (con metrics-server y/o Prometheus para métricas)

### Build

```bash
# Build simple (toma la versión del código)
go build -o z9s .

# Con info de versión vía ldflags
make build        # usa VERSION del Makefile
```

### Tests

```bash
go test ./...
```

## 📝 Licencia

Este proyecto está licenciado bajo **Apache License 2.0** — ver el archivo [LICENSE](LICENSE).

**Atribución**: z9s es un fork basado en el excelente trabajo de [k9s](https://github.com/derailed/k9s) de Fernand Galiana (@derailed).

## 📞 Contacto

- **Autor**: @zerote
- **Email**: condezero@gmail.com

---

**Nota**: Proyecto en desarrollo activo. Las features y APIs pueden cambiar antes de la v1.0.
