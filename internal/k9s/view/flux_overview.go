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
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/duration"
	"k8s.io/apimachinery/pkg/labels"
)

const fluxMaxEvents = 40

// fluxKind describes a Flux resource type shown on the overview.
type fluxKind struct {
	label string
	short string
	group string
	res   string
}

// fluxOverviewKinds mirrors the resource breakdown shown by Lens.
var fluxOverviewKinds = []fluxKind{
	{"Kustomizations", "Kustomize", "kustomize.toolkit.fluxcd.io", "kustomizations"},
	{"Helm Releases", "HelmRel", "helm.toolkit.fluxcd.io", "helmreleases"},
	{"Git Repositories", "GitRepo", "source.toolkit.fluxcd.io", "gitrepositories"},
	{"Helm Repositories", "HelmRepo", "source.toolkit.fluxcd.io", "helmrepositories"},
	{"Helm Charts", "HelmChart", "source.toolkit.fluxcd.io", "helmcharts"},
	{"OCI Repositories", "OCIRepo", "source.toolkit.fluxcd.io", "ocirepositories"},
}

// fluxControllers are the reporting components used to detect Flux events.
var fluxControllers = map[string]bool{
	"source-controller":           true,
	"kustomize-controller":        true,
	"helm-controller":             true,
	"notification-controller":     true,
	"image-reflector-controller":  true,
	"image-automation-controller": true,
}

// statusColors lists the stacked segments from the bottom of the bar upward.
var statusColors = []string{"green", "orange", "red", "dodgerblue", "gray"}

// fluxCounts aggregates resource status buckets.
type fluxCounts struct {
	ready, notReady, inProgress, suspended, unknown, total int
	installed                                              bool
}

// fluxEvent is a parsed Flux-related Kubernetes event.
type fluxEvent struct {
	eventType, message, namespace, object, source, age, lastSeen string
	count                                                        int32
	last                                                         time.Time
}

// ----------------------------------------------------------------------------
// FluxOverview component: a status column chart (top 1/3) plus a scrollable
// Flux Events table (bottom 2/3).

type FluxOverview struct {
	*tview.Flex

	app     *App
	chart   *tview.TextView
	events  *tview.Table
	actions *ui.KeyActions
}

// NewFluxOverview returns the FluxCD overview page.
func NewFluxOverview(app *App) *FluxOverview {
	o := FluxOverview{
		Flex:    tview.NewFlex(),
		app:     app,
		chart:   tview.NewTextView(),
		events:  tview.NewTable(),
		actions: ui.NewKeyActions(),
	}
	o.actions.Bulk(ui.KeyMap{
		ui.KeyK:         ui.NewKeyAction("Kustomizations", app.gotoFluxKustomizations, true),
		ui.KeyO:         ui.NewKeyAction("Refresh", o.refreshCmd, true),
		tcell.KeyEscape: ui.NewKeyAction("Back", app.PrevCmd, false),
		ui.KeyQ:         ui.NewKeyAction("Back", app.PrevCmd, false),
	})

	return &o
}

// Init initializes the view.
func (o *FluxOverview) Init(context.Context) error {
	o.SetDirection(tview.FlexRow)

	o.chart.SetDynamicColors(true)
	o.chart.SetWrap(false)
	o.chart.SetBorder(true)
	o.chart.SetTitle(" FluxCD Overview ")
	o.chart.SetTitleColor(tcell.ColorLightSkyBlue)
	o.chart.SetText(fluxOverviewLoading())

	o.events.SetBorder(true)
	o.events.SetTitle(" Flux Events ")
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

func (o *FluxOverview) keyboard(evt *tcell.EventKey) *tcell.EventKey {
	if a, ok := o.actions.Get(ui.AsKey(evt)); ok {
		return a.Action(evt)
	}

	return evt
}

func (o *FluxOverview) refreshCmd(*tcell.EventKey) *tcell.EventKey {
	o.refresh()

	return nil
}

// refresh fetches counts and events off the UI thread and repaints.
func (o *FluxOverview) refresh() {
	go func() {
		counts, cerr := o.app.fetchFluxOverview()
		events, eerr := o.app.fetchFluxEvents(fluxMaxEvents)
		o.app.QueueUpdateDraw(func() {
			_, _, w, h := o.chart.GetInnerRect()
			o.chart.SetText(fluxOverviewChart(counts, cerr, w, h))
			o.fillEvents(events, eerr)
		})
	}()
}

func (o *FluxOverview) fillEvents(events []fluxEvent, err error) {
	t := o.events
	t.Clear()

	headers := []string{"TYPE", "MESSAGE", "NAMESPACE", "INVOLVED OBJECT", "SOURCE", "COUNT", "AGE", "LAST SEEN"}
	// Every column expands so the leftover width is spread evenly, giving wide
	// separation between columns (like the cluster dashboard table).
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
		t.SetCell(1, 1, tview.NewTableCell("No Flux events found").SetTextColor(tcell.ColorGray))
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
func (o *FluxOverview) StylesChanged(s *config.Styles) {
	bg := s.BgColor()
	o.SetBackgroundColor(bg)
	o.chart.SetBackgroundColor(bg)
	o.chart.SetTextColor(s.FgColor())
	o.events.SetBackgroundColor(bg)
	o.chart.SetBorderColor(s.Frame().Border.FgColor.Color())
	o.chart.SetTitleColor(tcell.ColorLightSkyBlue)
	o.events.SetBorderColor(s.Frame().Border.FgColor.Color())
	o.events.SetBorderFocusColor(s.Frame().Border.FocusColor.Color())
	o.events.SetTitleColor(tcell.ColorLightSkyBlue)
	// Match the selection color used by every other table (driven by the skin).
	o.events.SetSelectedStyle(tcell.StyleDefault.
		Foreground(s.Table().CursorFgColor.Color()).
		Background(s.Table().CursorBgColor.Color()).
		Attributes(tcell.AttrBold))
}

// Name returns the component name.
func (*FluxOverview) Name() string { return "FluxCD" }

// Start starts the view and triggers a data load.
func (o *FluxOverview) Start() {
	o.app.Styles.RemoveListener(o)
	o.app.Styles.AddListener(o)
	o.refresh()
}

// Stop terminates the view.
func (o *FluxOverview) Stop() { o.app.Styles.RemoveListener(o) }

// Hints returns menu hints.
func (o *FluxOverview) Hints() model.MenuHints { return o.actions.Hints() }

// ExtraHints returns additional hints.
func (*FluxOverview) ExtraHints() map[string]string { return nil }

// InCmdMode checks if prompt is active.
func (*FluxOverview) InCmdMode() bool { return false }

func (*FluxOverview) SetFilter(string, bool)                 {}
func (*FluxOverview) SetLabelSelector(labels.Selector, bool) {}
func (*FluxOverview) SetCommand(*cmd.Interpreter)            {}

// ----------------------------------------------------------------------------
// Data fetching.

// fetchFluxOverview lists every Flux resource type across all namespaces and
// buckets them by status.
func (a *App) fetchFluxOverview() (map[string]fluxCounts, error) {
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

	// Resolve the served version for each Flux resource (versions drift across
	// Flux releases, so never hardcode them).
	gvrByKey := make(map[string]schema.GroupVersionResource)
	pref, _ := disc.ServerPreferredResources()
	for _, list := range pref {
		if list == nil {
			continue
		}
		gv, e := schema.ParseGroupVersion(list.GroupVersion)
		if e != nil {
			continue
		}
		for i := range list.APIResources {
			name := list.APIResources[i].Name
			if strings.Contains(name, "/") {
				continue
			}
			gvrByKey[gv.Group+"/"+name] = gv.WithResource(name)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), conn.Config().CallTimeout())
	defer cancel()

	out := make(map[string]fluxCounts, len(fluxOverviewKinds))
	for _, k := range fluxOverviewKinds {
		gvr, ok := gvrByKey[k.group+"/"+k.res]
		if !ok {
			out[k.label] = fluxCounts{}
			continue
		}
		ul, e := dyn.Resource(gvr).List(ctx, metav1.ListOptions{})
		if e != nil {
			out[k.label] = fluxCounts{installed: true}
			continue
		}
		c := fluxCounts{installed: true}
		for i := range ul.Items {
			switch fluxCategory(&ul.Items[i]) {
			case "Suspended":
				c.suspended++
			case "InProgress":
				c.inProgress++
			case "Ready":
				c.ready++
			case "NotReady":
				c.notReady++
			default:
				c.unknown++
			}
			c.total++
		}
		out[k.label] = c
	}

	return out, nil
}

// fetchFluxEvents returns the most recent Flux-related events (newest first).
func (a *App) fetchFluxEvents(limit int) ([]fluxEvent, error) {
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
		if !fluxControllers[src] && !strings.Contains(e.InvolvedObject.APIVersion, "fluxcd.io") {
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

func ageOf(now, t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	return duration.HumanDuration(now.Sub(t))
}

// fluxCategory classifies a Flux resource into a single status bucket.
func fluxCategory(u *unstructured.Unstructured) string {
	if susp, _, _ := unstructured.NestedBool(u.Object, "spec", "suspend"); susp {
		return "Suspended"
	}

	conds, _, _ := unstructured.NestedSlice(u.Object, "status", "conditions")
	var readyStatus, readyReason, reconciling string
	for _, cc := range conds {
		cm, ok := cc.(map[string]any)
		if !ok {
			continue
		}
		switch t, _ := cm["type"].(string); t {
		case "Ready":
			readyStatus, _ = cm["status"].(string)
			readyReason, _ = cm["reason"].(string)
		case "Reconciling":
			reconciling, _ = cm["status"].(string)
		}
	}

	if reconciling == "True" {
		return "InProgress"
	}
	switch readyStatus {
	case "True":
		return "Ready"
	case "False":
		if isProgressingReason(readyReason) {
			return "InProgress"
		}
		return "NotReady"
	default:
		return "Unknown"
	}
}

func isProgressingReason(reason string) bool {
	switch reason {
	case "Progressing", "ProgressingWithRetry", "Reconciling", "DependencyNotReady":
		return true
	default:
		return false
	}
}

func fluxOverviewLoading() string {
	return "\n  [gray::]Loading FluxCD overview...[-:-:-]\n"
}

// ----------------------------------------------------------------------------
// Chart rendering: vertical, stacked status columns (one per resource type).

// statusNames are the row labels shown once in the left gutter.
var statusNames = []string{"Ready", "InProgress", "NotReady", "Suspended", "Unknown"}

type cellVal struct {
	txt, color string
}

func fluxOverviewChart(counts map[string]fluxCounts, err error, width, height int) string {
	if err != nil {
		return fmt.Sprintf("\n [red::]Unable to load overview: %v[-:-:-]\n", err)
	}

	if width <= 0 {
		width = 110
	}
	if height <= 0 {
		height = 18
	}

	const (
		gutterW = 11
		valW    = 4
	)
	n := len(fluxOverviewKinds)
	usableW := width - gutterW - 2
	if usableW < n*10 {
		usableW = n * 10
	}
	cellW := usableW / n
	// Bars capped at 12 chars with a generous reserve so the inter-column gaps
	// are wide and even. The value column sits just before each bar and the whole
	// block is centered within its cell (like the cluster dashboard columns).
	barW := cellW - valW - 12
	if barW > 12 {
		barW = 12
	}
	if barW < 2 {
		barW = 2
	}
	blockW := valW + 1 + barW
	leftPad := (cellW - blockW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	rightPad := cellW - leftPad - blockW
	if rightPad < 0 {
		rightPad = 0
	}
	// Column offset (within the cell) of the bar's center, used to align titles.
	barCenter := leftPad + valW + 1 + barW/2

	// Bars use the full height of the panel (reserve only the heading rows).
	plotH := height - 3
	if plotH < 5 {
		plotH = 5
	}
	if plotH > 30 {
		plotH = 30
	}

	// Per-column segment heights plus the per-segment value labels.
	segs := make([][]int, n)
	rowVals := make([]map[int]cellVal, n)
	for i, k := range fluxOverviewKinds {
		c := counts[k.label]
		vals := []int{c.ready, c.inProgress, c.notReady, c.suspended, c.unknown}
		seg := allocSegments(vals, plotH)
		segs[i] = seg
		rv := make(map[int]cellVal, len(seg))
		start := 0
		for s := range seg {
			if seg[s] > 0 {
				center := start + (seg[s]-1)/2
				rv[center] = cellVal{fmt.Sprintf("%d", vals[s]), statusColors[s]}
			}
			start += seg[s]
		}
		rowVals[i] = rv
	}

	var b strings.Builder
	fmt.Fprintf(&b, " [gray::](all namespaces · updated %s)[-:-:-]\n\n", time.Now().Format("15:04:05"))

	// Title row: short label + total, centered over each bar.
	b.WriteString(strings.Repeat(" ", gutterW+2))
	for _, k := range fluxOverviewKinds {
		c := counts[k.label]
		title := fmt.Sprintf("%s %d", k.short, c.total)
		if !c.installed {
			title = k.short + " n/a"
		}
		b.WriteString(centerAt(title, "#87cefa", cellW, barCenter))
	}
	b.WriteByte('\n')

	// Plot rows (full height), top to bottom.
	for r := plotH - 1; r >= 0; r-- {
		b.WriteByte(' ')
		legendIdx := (plotH - 1) - r
		if legendIdx < len(statusNames) {
			b.WriteString(leftColored(statusNames[legendIdx], statusColors[legendIdx], gutterW))
		} else {
			b.WriteString(strings.Repeat(" ", gutterW))
		}
		b.WriteByte(' ')

		for i := range fluxOverviewKinds {
			b.WriteString(strings.Repeat(" ", leftPad))
			if v, ok := rowVals[i][r]; ok {
				b.WriteString(rightColored(v.txt, v.color, valW))
			} else {
				b.WriteString(strings.Repeat(" ", valW))
			}
			b.WriteByte(' ')
			if color := colorAtRow(segs[i], r); color != "" {
				b.WriteString(tag(color) + strings.Repeat("█", barW) + "[-:-:-]")
			} else {
				b.WriteString(strings.Repeat(" ", barW))
			}
			b.WriteString(strings.Repeat(" ", rightPad))
		}
		b.WriteByte('\n')
	}

	return b.String()
}

// tag builds a well-formed color tag (a bare color name is not parsed).
func tag(color string) string {
	if !strings.Contains(color, ":") {
		color += "::"
	}

	return "[" + color + "]"
}

// centerAt centers plain text on column `at` within a cellW-wide cell, clamping
// so the text stays inside the cell, and colors it.
func centerAt(plain, color string, cellW, at int) string {
	r := []rune(plain)
	if len(r) > cellW {
		plain = string(r[:cellW])
		r = []rune(plain)
	}
	start := at - len(r)/2
	if start < 0 {
		start = 0
	}
	if start+len(r) > cellW {
		start = cellW - len(r)
	}

	return strings.Repeat(" ", start) + tag(color) + plain + "[-:-:-]" + strings.Repeat(" ", cellW-start-len(r))
}

// leftColored left-aligns plain text within width w and colors it.
func leftColored(plain, color string, w int) string {
	r := []rune(plain)
	if len(r) > w {
		plain = string(r[:w])
		r = []rune(plain)
	}

	return tag(color) + plain + "[-:-:-]" + strings.Repeat(" ", w-len(r))
}

// rightColored right-aligns plain text within width w and colors it.
func rightColored(plain, color string, w int) string {
	r := []rune(plain)
	if len(r) > w {
		plain = string(r[:w])
		r = []rune(plain)
	}

	return strings.Repeat(" ", w-len(r)) + tag(color) + plain + "[-:-:-]"
}

// colorAtRow returns the segment color covering plot row r (0 = bottom).
func colorAtRow(seg []int, r int) string {
	cum := 0
	for i, h := range seg {
		cum += h
		if r < cum {
			return statusColors[i]
		}
	}

	return ""
}

// allocSegments turns per-status counts into stacked bar heights summing to H,
// keeping every non-zero status visible (at least one cell).
func allocSegments(vals []int, H int) []int {
	h := make([]int, len(vals))
	total := 0
	for _, v := range vals {
		total += v
	}
	if total == 0 || H <= 0 {
		return h
	}

	acc := 0
	for i, v := range vals {
		h[i] = v * H / total
		acc += h[i]
	}
	for acc < H {
		bi, bn := -1, 0
		for i, v := range vals {
			if v > bn {
				bn, bi = v, i
			}
		}
		if bi < 0 {
			break
		}
		h[bi]++
		acc++
	}
	for i, v := range vals {
		if v > 0 && h[i] == 0 {
			tallest := 0
			for j := range h {
				if h[j] > h[tallest] {
					tallest = j
				}
			}
			if h[tallest] > 1 {
				h[tallest]--
				h[i] = 1
			}
		}
	}

	return h
}
