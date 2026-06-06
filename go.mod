module github.com/yourusername/z9s

go 1.25

replace (
	github.com/yourusername/z9s/internal/k9s => ./internal/k9s
	github.com/yourusername/z9s/internal/ktop => ./internal/ktop
	github.com/yourusername/z9s/internal/shared => ./internal/shared
)

require (
	// === Core Dependencies ===
	github.com/spf13/cobra v1.10.2
	k8s.io/client-go v0.32.0
	k8s.io/api v0.32.0
	k8s.io/metrics v0.32.0
	k8s.io/cli-runtime v0.32.0
	k8s.io/klog/v2 v2.135.0
	
	// === UI Dependencies (tcell & tview) ===
	github.com/gdamore/tcell/v2 v2.7.6
	github.com/derailed/tcell/v2 v2.3.1-rc.4
	github.com/derailed/tview v0.8.5
	github.com/lucasb-eyer/go-colorful v1.3.0
	github.com/mattn/go-runewidth v0.0.21
	github.com/mattn/go-colorable v0.1.14
	
	// === K9s Specific ===
	github.com/adrg/xdg v0.5.3
	github.com/fatih/color v1.19.0
	github.com/lmittmann/tint v1.1.3
	github.com/sahilm/fuzzy v0.1.1
	github.com/olekukonko/tablewriter v1.1.4
	
	// === Ktop Specific (Metrics) ===
	github.com/prometheus/client_golang v1.17.0
	
	// === Shared Utilities ===
	github.com/spf13/pflag v1.0.6-0.20210604193023-d5e0c0615ace
	github.com/fsnotify/fsnotify v1.9.0
	github.com/atotto/clipboard v0.1.4
	
	// === Testing ===
	github.com/stretchr/testify v1.11.1
)

// This is a merged project combining:
// - k9s: https://github.com/derailed/k9s
// - ktop: https://github.com/vladimirvivien/ktop
//
// License: Apache 2.0

