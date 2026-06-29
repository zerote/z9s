// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package view

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/yourusername/z9s/internal/k9s/client"
	"github.com/yourusername/z9s/internal/k9s/config"
	"github.com/yourusername/z9s/internal/k9s/model"
	"github.com/yourusername/z9s/internal/k9s/render"
	"github.com/yourusername/z9s/internal/k9s/slogs"
	"github.com/yourusername/z9s/internal/k9s/ui"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
)

var _ model.ClusterInfoListener = (*ClusterInfo)(nil)

// ClusterInfo represents a cluster info view.
type ClusterInfo struct {
	*tview.Table

	app    *App
	styles *config.Styles

	// GitOps operator detection is cached per context (discovery is relatively
	// costly) and re-run whenever the active context changes.
	opChecked  bool
	opContext  string
	flux, argo bool
}

// NewClusterInfo returns a new cluster info view.
func NewClusterInfo(app *App) *ClusterInfo {
	return &ClusterInfo{
		Table:  tview.NewTable(),
		app:    app,
		styles: app.Styles,
	}
}

// Init initializes the view.
func (c *ClusterInfo) Init() {
	c.SetBorderPadding(0, 0, 1, 0)
	c.app.Styles.AddListener(c)
	c.layout()
	c.StylesChanged(c.app.Styles)
}

// StylesChanged notifies skin changed.
func (c *ClusterInfo) StylesChanged(s *config.Styles) {
	c.styles = s
	c.SetBackgroundColor(s.BgColor())
	c.updateStyle()
}

func (c *ClusterInfo) hasMetrics() bool {
	mx := c.app.Conn().HasMetrics()
	if mx {
		auth, err := c.app.Conn().CanI("", client.NmxGVR, "", client.ListAccess)
		if err != nil {
			slog.Warn("No nodes metrics access", slogs.Error, err)
		}
		mx = auth
	}

	return mx
}

func (c *ClusterInfo) layout() {
	for row, section := range []string{"Context", "Cluster", "User", "K8s Rev", "Operator", "CPU", "MEM"} {
		c.SetCell(row, 0, c.sectionCell(section))
		c.SetCell(row, 1, c.infoCell(render.NAValue))
	}
}

func (c *ClusterInfo) sectionCell(t string) *tview.TableCell {
	cell := tview.NewTableCell(t + ":")
	cell.SetAlign(tview.AlignLeft)
	cell.SetBackgroundColor(c.app.Styles.BgColor())

	return cell
}

func (c *ClusterInfo) infoCell(t string) *tview.TableCell {
	cell := tview.NewTableCell(t)
	cell.SetExpansion(2)
	cell.SetTextColor(c.styles.K9s.Info.FgColor.Color())
	cell.SetBackgroundColor(c.app.Styles.BgColor())

	return cell
}

func (c *ClusterInfo) setCell(row int, s string) int {
	if s == "" {
		s = render.NAValue
	}
	c.GetCell(row, 1).SetText(s)
	return row + 1
}

// ClusterInfoUpdated notifies the cluster meta was updated.
func (c *ClusterInfo) ClusterInfoUpdated(data *model.ClusterMeta) {
	c.ClusterInfoChanged(data, data)
}

// operatorValue returns the GitOps operator line (FluxCD/ArgoCD on|off),
// detecting their presence via the cluster's API groups. The result is cached
// per context so switching contexts triggers a fresh detection.
func (c *ClusterInfo) operatorValue(context string) string {
	if !c.opChecked || c.opContext != context {
		c.flux, c.argo = c.detectOperators()
		c.opChecked, c.opContext = true, context
	}

	onOff := func(b bool) string {
		if b {
			return "[green::b]on[-:-:-]"
		}
		return "[red::b]off[-:-:-]"
	}

	return fmt.Sprintf("FluxCD: %s  -  ArgoCD: %s", onOff(c.flux), onOff(c.argo))
}

// detectOperators inspects the cluster's registered API groups to tell whether
// FluxCD (*.fluxcd.io) and/or ArgoCD (*.argoproj.io) CRDs are installed.
func (c *ClusterInfo) detectOperators() (flux, argo bool) {
	return detectOperators(c.app.Conn())
}

// detectOperators inspects the cluster's registered API groups to tell whether
// FluxCD (*.fluxcd.io) and/or ArgoCD (*.argoproj.io) CRDs are installed.
func detectOperators(conn client.Connection) (flux, argo bool) {
	if conn == nil {
		return false, false
	}
	dial, err := conn.CachedDiscovery()
	if err != nil {
		slog.Warn("Operator detection: discovery failed", slogs.Error, err)
		return false, false
	}
	groups, err := dial.ServerGroups()
	if err != nil || groups == nil {
		slog.Warn("Operator detection: server groups failed", slogs.Error, err)
		return false, false
	}
	for i := range groups.Groups {
		name := groups.Groups[i].Name
		switch {
		case strings.Contains(name, "fluxcd.io"):
			flux = true
		case strings.Contains(name, "argoproj.io"):
			argo = true
		}
	}

	return flux, argo
}

func (*ClusterInfo) warnCell(s string, w bool) string {
	if w {
		return fmt.Sprintf("[orangered::b]%s", s)
	}

	return s
}

// ClusterInfoChanged notifies the cluster meta was changed.
func (c *ClusterInfo) ClusterInfoChanged(prev, curr *model.ClusterMeta) {
	c.app.QueueUpdateDraw(func() {
		c.Clear()
		c.layout()

		context := curr.Context
		if ic := ui.ROIndicator(c.app.Config.IsReadOnly(), c.app.Config.K9s.UI.NoIcons); ic != "" {
			context += " " + ic
		}
		row := c.setCell(0, context)
		row = c.setCell(row, curr.Cluster)
		row = c.setCell(row, curr.User)
		row = c.setCell(row, curr.K8sVer)
		row = c.setCell(row, c.operatorValue(curr.Context))
		if c.hasMetrics() {
			row = c.setCell(row, ui.AsPercDelta(prev.Cpu, curr.Cpu))
			_ = c.setCell(row, ui.AsPercDelta(prev.Mem, curr.Mem))
			c.setDefCon(curr.Cpu, curr.Mem)
		} else {
			row = c.setCell(row, c.warnCell(render.NAValue, true))
			_ = c.setCell(row, c.warnCell(render.NAValue, true))
		}
		c.updateStyle()
	})
}

const defconFmt = "%s %s level!"

func (c *ClusterInfo) setDefCon(cpu, mem int) {
	var set bool
	l := c.app.Config.K9s.Thresholds.LevelFor(config.CPU, cpu)
	if l > config.SeverityLow {
		c.app.Status(flashLevel(l), fmt.Sprintf(defconFmt, flashMessage(l), "CPU"))
		set = true
	}
	l = c.app.Config.K9s.Thresholds.LevelFor(config.MEM, mem)
	if l > config.SeverityLow {
		c.app.Status(flashLevel(l), fmt.Sprintf(defconFmt, flashMessage(l), "Memory"))
		set = true
	}
	if !set && !c.app.IsBenchmarking() {
		c.app.ClearStatus(true)
	}
}

func (c *ClusterInfo) updateStyle() {
	for row := range c.GetRowCount() {
		c.GetCell(row, 0).SetTextColor(c.styles.K9s.Info.FgColor.Color())
		c.GetCell(row, 0).SetBackgroundColor(c.styles.BgColor())
		var s tcell.Style
		s = s.Bold(true)
		s = s.Foreground(c.styles.K9s.Info.SectionColor.Color())
		s = s.Background(c.styles.BgColor())
		c.GetCell(row, 1).SetStyle(s)
	}
}

// ----------------------------------------------------------------------------
// Helpers...

func flashLevel(l config.SeverityLevel) model.FlashLevel {
	//nolint:exhaustive
	switch l {
	case config.SeverityHigh:
		return model.FlashErr
	case config.SeverityMedium:
		return model.FlashWarn
	default:
		return model.FlashInfo
	}
}

func flashMessage(l config.SeverityLevel) string {
	//nolint:exhaustive
	switch l {
	case config.SeverityHigh:
		return "Critical"
	case config.SeverityMedium:
		return "Warning"
	default:
		return "OK"
	}
}
