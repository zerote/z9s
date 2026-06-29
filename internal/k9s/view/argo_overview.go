// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package view

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yourusername/z9s/internal/k9s/config"
	"github.com/yourusername/z9s/internal/k9s/model"
	"github.com/yourusername/z9s/internal/k9s/ui"
	"github.com/yourusername/z9s/internal/k9s/view/cmd"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const argoMaxEvents = 40

// argoApplicationGroup is the API group for ArgoCD custom resources.
const argoApplicationGroup = "argoproj.io"

// argoControllers are the reporting components used to detect ArgoCD events.
var argoControllers = map[string]bool{
	"argocd-application-controller":    true,
	"argocd-applicationset-controller": true,
	"applicationset-controller":        true,
}

// argoHealthOrder / argoHealthColors describe the Health breakdown bar
// (stacked from the bottom up).
var (
	argoHealthOrder  = []string{"Healthy", "Progressing", "Degraded", "Suspended", "Missing", "Unknown"}
	argoHealthColors = map[string]string{
		"Healthy":     "green",
		"Progressing": "dodgerblue",
		"Degraded":    "red",
		"Suspended":   "mediumpurple",
		"Missing":     "orange",
		"Unknown":     "gray",
	}
)

// argoSyncOrder / argoSyncColors describe the Sync breakdown bar.
var (
	argoSyncOrder  = []string{"Synced", "OutOfSync", "Unknown"}
	argoSyncColors = map[string]string{
		"Synced":    "green",
		"OutOfSync": "orange",
		"Unknown":   "gray",
	}
)

// argoBar is a single stacked bar (Health or Sync).
type argoBar struct {
	title string
	total int
	names []string
	count map[string]int
	color map[string]string
}

// ----------------------------------------------------------------------------
// ArgoOverview component: a Health/Sync column chart (top 1/3) plus a scrollable
// ArgoCD Events table (bottom 2/3). Mirrors FluxOverview.

type ArgoOverview struct {
	*tview.Flex

	app     *App
	chart   *tview.TextView
	events  *tview.Table
	actions *ui.KeyActions
}

// NewArgoOverview returns the ArgoCD overview page.
func NewArgoOverview(app *App) *ArgoOverview {
	o := ArgoOverview{
		Flex:    tview.NewFlex(),
		app:     app,
		chart:   tview.NewTextView(),
		events:  tview.NewTable(),
		actions: ui.NewKeyActions(),
	}
	o.actions.Bulk(ui.KeyMap{
		ui.KeyA:         ui.NewKeyAction("Applications", app.gotoArgoApplications, true),
		ui.KeyO:         ui.NewKeyAction("Refresh", o.refreshCmd, true),
		tcell.KeyEscape: ui.NewKeyAction("Back", app.PrevCmd, false),
		ui.KeyQ:         ui.NewKeyAction("Back", app.PrevCmd, false),
	})

	return &o
}

// Init initializes the view.
func (o *ArgoOverview) Init(context.Context) error {
	o.SetDirection(tview.FlexRow)

	o.chart.SetDynamicColors(true)
	o.chart.SetWrap(false)
	o.chart.SetBorder(true)
	o.chart.SetTitle(" ArgoCD Overview ")
	o.chart.SetTitleColor(tcell.ColorLightSkyBlue)
	o.chart.SetText(argoOverviewLoading())

	o.events.SetBorder(true)
	o.events.SetTitle(" ArgoCD Events ")
	o.events.SetTitleColor(tcell.ColorLightSkyBlue)
	o.events.SetSelectable(true, false)
	o.events.SetFixed(1, 0)
	o.events.SetSelectedStyle(tcell.StyleDefault.
		Background(tcell.ColorDodgerBlue).Foreground(tcell.ColorWhite))
	o.events.SetInputCapture(o.keyboard)

	o.AddItem(o.chart, 0, 1, false)
	o.AddItem(o.events, 0, 2, true)

	o.StylesChanged(o.app.Styles)

	return nil
}

func (o *ArgoOverview) keyboard(evt *tcell.EventKey) *tcell.EventKey {
	if a, ok := o.actions.Get(ui.AsKey(evt)); ok {
		return a.Action(evt)
	}

	return evt
}

func (o *ArgoOverview) refreshCmd(*tcell.EventKey) *tcell.EventKey {
	o.refresh()

	return nil
}

// refresh fetches counts and events off the UI thread and repaints.
func (o *ArgoOverview) refresh() {
	go func() {
		bars, cerr := o.app.fetchArgoOverview()
		events, eerr := o.app.fetchArgoEvents(argoMaxEvents)
		o.app.QueueUpdateDraw(func() {
			_, _, w, h := o.chart.GetInnerRect()
			o.chart.SetText(argoOverviewChart(bars, cerr, w, h))
			o.fillEvents(events, eerr)
		})
	}()
}

func (o *ArgoOverview) fillEvents(events []fluxEvent, err error) {
	t := o.events
	t.Clear()

	headers := []string{"TYPE", "MESSAGE", "NAMESPACE", "INVOLVED OBJECT", "SOURCE", "COUNT", "AGE", "LAST SEEN"}
	expansion := []int{1, 6, 2, 3, 2, 1, 1, 1}
	for c, h := range headers {
		t.SetCell(0, c, tview.NewTableCell(h).
			SetSelectable(false).
			SetTextColor(tcell.ColorWhite).
			SetAttributes(tcell.AttrBold).
			SetExpansion(expansion[c]))
	}

	if err != nil {
		t.SetCell(1, 1, tview.NewTableCell(err.Error()).SetTextColor(tcell.ColorRed))
		return
	}
	if len(events) == 0 {
		t.SetCell(1, 1, tview.NewTableCell("No ArgoCD events found").SetTextColor(tcell.ColorGray))
		return
	}

	const celeste = tcell.ColorLightSkyBlue
	for i, e := range events {
		row := i + 1
		typeColor := tcell.ColorMediumSeaGreen
		if e.eventType == "Warning" {
			typeColor = tcell.ColorOrangeRed
		}
		t.SetCell(row, 0, tview.NewTableCell(e.eventType).SetTextColor(typeColor).SetExpansion(1))
		t.SetCell(row, 1, tview.NewTableCell(e.message).SetTextColor(celeste).SetExpansion(6).SetMaxWidth(80))
		t.SetCell(row, 2, tview.NewTableCell(e.namespace).SetTextColor(celeste).SetExpansion(2))
		t.SetCell(row, 3, tview.NewTableCell(e.object).SetTextColor(celeste).SetExpansion(3))
		t.SetCell(row, 4, tview.NewTableCell(e.source).SetTextColor(celeste).SetExpansion(2))
		t.SetCell(row, 5, tview.NewTableCell(fmt.Sprintf("%d", e.count)).SetTextColor(celeste).SetExpansion(1).SetAlign(tview.AlignRight))
		t.SetCell(row, 6, tview.NewTableCell(e.age).SetTextColor(celeste).SetExpansion(1).SetAlign(tview.AlignRight))
		t.SetCell(row, 7, tview.NewTableCell(e.lastSeen).SetTextColor(celeste).SetExpansion(1).SetAlign(tview.AlignRight))
	}
	t.Select(1, 0)
	t.ScrollToBeginning()
}

// StylesChanged notifies the skin changed.
func (o *ArgoOverview) StylesChanged(s *config.Styles) {
	bg := s.BgColor()
	o.SetBackgroundColor(bg)
	o.chart.SetBackgroundColor(bg)
	o.chart.SetTextColor(s.FgColor())
	o.chart.SetBorderColor(s.Frame().Border.FgColor.Color())
	o.chart.SetTitleColor(tcell.ColorLightSkyBlue)
	o.events.SetBackgroundColor(bg)
	o.events.SetBorderColor(s.Frame().Border.FgColor.Color())
	o.events.SetBorderFocusColor(s.Frame().Border.FocusColor.Color())
	o.events.SetTitleColor(tcell.ColorLightSkyBlue)
	o.events.SetSelectedStyle(tcell.StyleDefault.
		Foreground(s.Table().CursorFgColor.Color()).
		Background(s.Table().CursorBgColor.Color()).
		Attributes(tcell.AttrBold))
}

// Name returns the component name.
func (*ArgoOverview) Name() string { return "ArgoCD" }

// Start starts the view and triggers a data load.
func (o *ArgoOverview) Start() {
	o.app.Styles.RemoveListener(o)
	o.app.Styles.AddListener(o)
	o.refresh()
}

// Stop terminates the view.
func (o *ArgoOverview) Stop() { o.app.Styles.RemoveListener(o) }

// Hints returns menu hints.
func (o *ArgoOverview) Hints() model.MenuHints { return o.actions.Hints() }

// ExtraHints returns additional hints.
func (*ArgoOverview) ExtraHints() map[string]string { return nil }

// InCmdMode checks if prompt is active.
func (*ArgoOverview) InCmdMode() bool { return false }

func (*ArgoOverview) SetFilter(string, bool)                 {}
func (*ArgoOverview) SetLabelSelector(labels.Selector, bool) {}
func (*ArgoOverview) SetCommand(*cmd.Interpreter)            {}

// ----------------------------------------------------------------------------
// Data fetching.

// fetchArgoOverview lists every ArgoCD Application across all namespaces and
// buckets them by health status and sync status.
func (a *App) fetchArgoOverview() ([]argoBar, error) {
	conn := a.Conn()
	if conn == nil {
		return nil, fmt.Errorf("no cluster connection")
	}
	disc, err := conn.CachedDiscovery()
	if err != nil {
		return nil, err
	}
	dyn, err := conn.DynDial()
	if err != nil {
		return nil, err
	}

	// Resolve the served version of applications.argoproj.io (versions can drift).
	var gvr schema.GroupVersionResource
	found := false
	pref, _ := disc.ServerPreferredResources()
	for _, list := range pref {
		if list == nil {
			continue
		}
		gv, e := schema.ParseGroupVersion(list.GroupVersion)
		if e != nil || gv.Group != argoApplicationGroup {
			continue
		}
		for i := range list.APIResources {
			if list.APIResources[i].Name == "applications" {
				gvr = gv.WithResource("applications")
				found = true
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("ArgoCD Application CRD not found")
	}

	ctx, cancel := context.WithTimeout(context.Background(), conn.Config().CallTimeout())
	defer cancel()

	ul, err := dyn.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	health := argoBar{title: "Health", names: argoHealthOrder, count: map[string]int{}, color: argoHealthColors}
	sync := argoBar{title: "Sync", names: argoSyncOrder, count: map[string]int{}, color: argoSyncColors}
	for i := range ul.Items {
		h := argoHealthOf(&ul.Items[i])
		s := argoSyncOf(&ul.Items[i])
		health.count[h]++
		sync.count[s]++
		health.total++
		sync.total++
	}

	return []argoBar{health, sync}, nil
}

// fetchArgoEvents returns the most recent ArgoCD-related events (newest first).
func (a *App) fetchArgoEvents(limit int) ([]fluxEvent, error) {
	conn := a.Conn()
	if conn == nil {
		return nil, fmt.Errorf("no cluster connection")
	}
	dial, err := conn.Dial()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), conn.Config().CallTimeout())
	defer cancel()

	list, err := dial.CoreV1().Events(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	out := make([]fluxEvent, 0, len(list.Items))
	now := time.Now()
	for i := range list.Items {
		e := &list.Items[i]
		src := e.Source.Component
		if src == "" {
			src = e.ReportingController
		}
		if !argoControllers[src] && !strings.Contains(e.InvolvedObject.APIVersion, argoApplicationGroup) {
			continue
		}

		last := e.LastTimestamp.Time
		if last.IsZero() {
			last = e.EventTime.Time
		}
		if last.IsZero() {
			last = e.CreationTimestamp.Time
		}
		first := e.FirstTimestamp.Time
		if first.IsZero() {
			first = e.CreationTimestamp.Time
		}
		ns := e.InvolvedObject.Namespace
		if ns == "" {
			ns = e.Namespace
		}

		out = append(out, fluxEvent{
			eventType: e.Type,
			message:   strings.TrimSpace(e.Message),
			namespace: ns,
			object:    fmt.Sprintf("%s: %s", e.InvolvedObject.Kind, e.InvolvedObject.Name),
			source:    src,
			count:     e.Count,
			last:      last,
			age:       ageOf(now, first),
			lastSeen:  ageOf(now, last),
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].last.After(out[j].last) })
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}

	return out, nil
}

// argoHealthOf returns the Application health status, normalized.
func argoHealthOf(u *unstructured.Unstructured) string {
	s, _, _ := unstructured.NestedString(u.Object, "status", "health", "status")
	return normalizeArgoStatus(s, argoHealthOrder)
}

// argoSyncOf returns the Application sync status, normalized.
func argoSyncOf(u *unstructured.Unstructured) string {
	s, _, _ := unstructured.NestedString(u.Object, "status", "sync", "status")
	return normalizeArgoStatus(s, argoSyncOrder)
}

func normalizeArgoStatus(s string, known []string) string {
	for _, k := range known {
		if k == s {
			return k
		}
	}

	return "Unknown"
}

func argoOverviewLoading() string {
	return "\n  [gray::]Loading ArgoCD overview...[-:-:-]\n"
}

// ----------------------------------------------------------------------------
// Chart rendering: one stacked vertical bar per dimension (Health, Sync), each
// with its own labeled segments to the left of the bar.

type argoCellVal struct {
	txt, color string
}

func argoOverviewChart(bars []argoBar, err error, width, height int) string {
	if err != nil {
		return fmt.Sprintf("\n [red::]Unable to load overview: %v[-:-:-]\n", err)
	}
	if len(bars) == 0 {
		return "\n  [gray::]No ArgoCD Applications found[-:-:-]\n"
	}
	if width <= 0 {
		width = 110
	}
	if height <= 0 {
		height = 18
	}

	const labelW = 18
	n := len(bars)
	usableW := width - 2
	if usableW < n*(labelW+8) {
		usableW = n * (labelW + 8)
	}
	cellW := usableW / n
	barW := cellW - labelW - 4
	if barW > 16 {
		barW = 16
	}
	if barW < 2 {
		barW = 2
	}
	blockW := labelW + 1 + barW
	leftPad := (cellW - blockW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	rightPad := cellW - leftPad - blockW
	if rightPad < 0 {
		rightPad = 0
	}
	barCenter := leftPad + labelW + 1 + barW/2

	plotH := height - 3
	if plotH < 5 {
		plotH = 5
	}
	if plotH > 30 {
		plotH = 30
	}

	// Per-bar segment heights + per-segment label/value rows.
	segs := make([][]int, n)
	colorsByBar := make([][]string, n)
	rowVals := make([]map[int]argoCellVal, n)
	for i, b := range bars {
		vals := make([]int, len(b.names))
		cols := make([]string, len(b.names))
		for j, name := range b.names {
			vals[j] = b.count[name]
			cols[j] = b.color[name]
		}
		seg := allocSegments(vals, plotH)
		segs[i] = seg
		colorsByBar[i] = cols
		rv := make(map[int]argoCellVal, len(seg))
		start := 0
		for s := range seg {
			if seg[s] > 0 {
				center := start + (seg[s]-1)/2
				rv[center] = argoCellVal{fmt.Sprintf("%s %d", b.names[s], vals[s]), cols[s]}
			}
			start += seg[s]
		}
		rowVals[i] = rv
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, " [gray::](all namespaces · updated %s)[-:-:-]\n\n", time.Now().Format("15:04:05"))

	// Title row: dimension + total, centered over each bar.
	sb.WriteString(strings.Repeat(" ", 1))
	for _, b := range bars {
		title := fmt.Sprintf("%s (%d)", b.title, b.total)
		sb.WriteString(centerAt(title, "#87cefa", cellW, barCenter))
	}
	sb.WriteByte('\n')

	// Plot rows (full height), top to bottom.
	for r := plotH - 1; r >= 0; r-- {
		sb.WriteByte(' ')
		for i := range bars {
			sb.WriteString(strings.Repeat(" ", leftPad))
			if v, ok := rowVals[i][r]; ok {
				sb.WriteString(leftColored(v.txt, v.color, labelW))
			} else {
				sb.WriteString(strings.Repeat(" ", labelW))
			}
			sb.WriteByte(' ')
			if color := argoColorAtRow(segs[i], colorsByBar[i], r); color != "" {
				sb.WriteString(tag(color) + strings.Repeat("█", barW) + "[-:-:-]")
			} else {
				sb.WriteString(strings.Repeat(" ", barW))
			}
			sb.WriteString(strings.Repeat(" ", rightPad))
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}

// argoColorAtRow returns the segment color covering plot row r (0 = bottom).
func argoColorAtRow(seg []int, colors []string, r int) string {
	cum := 0
	for i, h := range seg {
		cum += h
		if r < cum {
			return colors[i]
		}
	}

	return ""
}
