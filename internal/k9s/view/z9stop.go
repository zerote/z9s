// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Z9s

package view

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"github.com/yourusername/z9s/internal/k9s/client"
	"github.com/yourusername/z9s/internal/k9s/dao"
	"github.com/yourusername/z9s/internal/k9s/slogs"
	"github.com/yourusername/z9s/internal/k9s/ui"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/duration"
)

const (
	// z9sTopRefreshRate is how often the dashboard polls the cluster while visible.
	z9sTopRefreshRate = 3 * time.Second
	// z9sTopFetchTimeout bounds each refresh round-trip.
	z9sTopFetchTimeout = 15 * time.Second
	// gaugeWidth is the number of cells used to draw a usage bar.
	gaugeWidth = 12
	// celesteColor is the light-blue used for dashboard text. The default skin's
	// cadetblue/aqua read as greenish, so we pin a clearly celeste tone.
	celesteColor = "lightskyblue"
)

// z9sTopData is an immutable snapshot produced off the UI goroutine and applied
// on it. Keeping the fetch result together avoids partial renders.
type z9sTopData struct {
	nodes []v1.Node
	nmx   client.NodesMetrics
	pods  []v1.Pod
	pmx   client.PodsMetrics

	// Pod-requested (reserved) resources aggregated per node, ktop-style:
	// CPU in millicores, MEM in MB. This is what ktop shows for node CPU/MEM.
	reqCPU map[string]int64
	reqMEM map[string]int64

	// Cluster-wide aggregates fetched via the clientset.
	nsCount  int
	depReady int
	depTotal int
	pvCount  int
	pvBytes  int64
	pvcCount int
	pvcBytes int64
}

// Z9sTop is a native, ktop-inspired live dashboard rendered with the same TUI
// stack as the rest of z9s (derailed/tview). It reuses the existing k9s client
// and metrics server, so no extra dependencies or external app are needed.
type Z9sTop struct {
	*tview.Flex

	app      *App
	header   *tview.Flex
	summary  *tview.TextView
	nodes    *tview.Table
	pods     *tview.Table
	prompt   *tview.InputField
	promptOn bool
	cancel   context.CancelFunc
	focusIdx int

	// data is the last snapshot, reused by the node detail view.
	data z9sTopData
}

// NewZ9sTop builds the dashboard view.
func NewZ9sTop(app *App) *Z9sTop {
	t := Z9sTop{
		Flex: tview.NewFlex().SetDirection(tview.FlexRow),
		app:  app,
	}
	st := app.Styles

	t.summary = tview.NewTextView()
	t.summary.SetDynamicColors(true)
	t.summary.SetWrap(false)
	t.summary.SetBackgroundColor(st.BgColor())

	// Z9s logo with the version printed underneath, on the right of the
	// (borderless) header, mirroring the main cluster view header.
	logoColor := string(st.Body().LogoColor)
	logo := tview.NewTextView()
	logo.SetDynamicColors(true)
	logo.SetWrap(false)
	logo.SetTextAlign(tview.AlignCenter)
	logo.SetBackgroundColor(st.BgColor())
	var lb strings.Builder
	for i, line := range ui.LogoSmall {
		if i > 0 {
			lb.WriteString("\n")
		}
		lb.WriteString("[" + logoColor + "::b]" + line)
	}
	logo.SetText(lb.String())

	rev := tview.NewTextView()
	rev.SetDynamicColors(true)
	rev.SetWrap(false)
	rev.SetTextAlign(tview.AlignCenter)
	rev.SetBackgroundColor(st.BgColor())
	rev.SetText("[white::b]" + app.version + " ⚡️")

	right := tview.NewFlex().SetDirection(tview.FlexRow)
	right.AddItem(logo, 6, 0, false)
	right.AddItem(rev, 1, 0, false)

	// Borderless header: summary on the left, logo + version on the right.
	t.header = tview.NewFlex().SetDirection(tview.FlexColumn)
	t.header.AddItem(t.summary, 0, 1, false)
	t.header.AddItem(right, 26, 0, false)
	t.header.SetBackgroundColor(st.BgColor())

	t.nodes = t.newTable(" Nodes ")
	t.pods = t.newTable(" Pods ")

	t.nodes.SetSelectedFunc(func(row, _ int) {
		idx := row - 1
		if idx < 0 || idx >= len(t.data.nodes) {
			return
		}
		node := t.data.nodes[idx]
		t.showNodeDetail(&node)
	})

	// k9s-style command prompt, shown on ":" and dismissed with ESC.
	t.prompt = tview.NewInputField()
	t.prompt.SetBorder(true)
	t.prompt.SetTitle(" Command ")
	t.prompt.SetLabel(" 🐶 > ")
	t.prompt.SetBackgroundColor(st.BgColor())
	t.prompt.SetFieldBackgroundColor(st.BgColor())
	t.prompt.SetLabelColor(st.Frame().Title.FgColor.Color())
	t.prompt.SetFieldTextColor(st.Body().FgColor.Color())
	t.prompt.SetTitleColor(st.Frame().Title.FgColor.Color())
	t.prompt.SetBorderColor(st.Frame().Border.FocusColor.Color())
	t.prompt.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			t.hidePrompt()
		case tcell.KeyEnter:
			text := strings.TrimSpace(t.prompt.GetText())
			t.hidePrompt()
			if text != "" {
				t.runCommand(text)
			}
		}
	})

	t.SetBackgroundColor(st.BgColor())
	t.layoutItems()

	return &t
}

// layoutItems (re)builds the dashboard's vertical layout, optionally inserting
// the command prompt at the top so the panels below slide down to make room.
func (t *Z9sTop) layoutItems() {
	t.Clear()
	t.AddItem(t.header, 7, 0, false)
	if t.promptOn {
		t.AddItem(t.prompt, 3, 0, true)
	}
	t.AddItem(t.nodes, 0, 2, false)
	t.AddItem(t.pods, 0, 3, false)
}

// showPrompt reveals the command prompt and focuses it.
func (t *Z9sTop) showPrompt() {
	if t.promptOn {
		return
	}
	t.promptOn = true
	t.prompt.SetText("")
	t.layoutItems()
	t.app.SetFocus(t.prompt)
}

// hidePrompt removes the command prompt and restores panel focus.
func (t *Z9sTop) hidePrompt() {
	if !t.promptOn {
		return
	}
	t.promptOn = false
	t.layoutItems()
	t.applyFocus()
}

// runCommand leaves the dashboard and runs the typed command in the main view.
// The "cluster" keyword returns to the cluster dashboard (the page we came from)
// instead of trying to resolve it as a Kubernetes resource.
func (t *Z9sTop) runCommand(c string) {
	if c == "cluster" {
		t.returnToCluster()
		return
	}
	t.Stop()
	t.app.splashReturn = ""
	t.app.gotoResource(c, "", true, true)
}

// returnToCluster switches back to the page that was active before the metrics
// dashboard was opened (defaults to the main cluster view).
func (t *Z9sTop) returnToCluster() {
	target := t.app.splashReturn
	if target == "" || !t.app.Main.HasPage(target) {
		target = mainPageID
	}
	t.Stop()
	t.app.Main.SwitchToPage(target)
	t.app.splashReturn = ""
}

func (t *Z9sTop) newTable(title string) *tview.Table {
	tbl := tview.NewTable()
	tbl.SetBorder(true)
	tbl.SetTitle(title)
	tbl.SetFixed(1, 0)
	tbl.SetSelectable(true, false)
	t.decorate(tbl.Box)

	cap := func(evt *tcell.EventKey) *tcell.EventKey {
		ctrl := evt.Modifiers()&tcell.ModCtrl != 0
		switch {
		case evt.Key() == tcell.KeyTab,
			ctrl && evt.Key() == tcell.KeyDown,
			ctrl && evt.Key() == tcell.KeyRight:
			t.cycleFocus(1)
			return nil
		case evt.Key() == tcell.KeyBacktab,
			ctrl && evt.Key() == tcell.KeyUp,
			ctrl && evt.Key() == tcell.KeyLeft:
			t.cycleFocus(-1)
			return nil
		}
		return evt
	}
	tbl.SetInputCapture(cap)

	return tbl
}

func (t *Z9sTop) decorate(b *tview.Box) {
	st := t.app.Styles
	b.SetBackgroundColor(st.BgColor())
	b.SetTitleColor(st.Frame().Title.FgColor.Color())
	b.SetBorderColor(st.Frame().Border.FgColor.Color())
}

// cycleFocus moves keyboard focus between the Nodes and Pods panels.
func (t *Z9sTop) cycleFocus(delta int) {
	t.focusIdx = (t.focusIdx + delta + 2) % 2
	t.applyFocus()
}

// applyFocusStyles paints the active panel's cursor blue and the inactive
// panel's cursor celeste. Re-applied on every render since Clear() can reset it.
func (t *Z9sTop) applyFocusStyles() {
	st := t.app.Styles
	focus, blur := st.Frame().Border.FocusColor.Color(), st.Frame().Border.FgColor.Color()
	focusSel := tcell.StyleDefault.Background(tcell.GetColor("dodgerblue")).Foreground(tcell.ColorWhite)
	blurSel := tcell.StyleDefault.Background(tcell.GetColor(celesteColor)).Foreground(tcell.ColorBlack)

	t.nodes.SetBorderColor(blur)
	t.pods.SetBorderColor(blur)
	t.nodes.SetSelectedStyle(blurSel)
	t.pods.SetSelectedStyle(blurSel)
	if t.focusIdx == 0 {
		t.nodes.SetBorderColor(focus)
		t.nodes.SetSelectedStyle(focusSel)
	} else {
		t.pods.SetBorderColor(focus)
		t.pods.SetSelectedStyle(focusSel)
	}
}

func (t *Z9sTop) applyFocus() {
	t.applyFocusStyles()
	if t.focusIdx == 0 {
		t.app.SetFocus(t.nodes)
	} else {
		t.app.SetFocus(t.pods)
	}
}

// Start kicks off the periodic refresh loop and performs an immediate refresh.
func (t *Z9sTop) Start() {
	t.focusIdx = 0
	t.applyFocus()
	if t.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.cancel = cancel

	t.refresh()
	go func() {
		tick := time.NewTicker(z9sTopRefreshRate)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				t.refresh()
			}
		}
	}()
}

// Stop halts the refresh loop.
func (t *Z9sTop) Stop() {
	if t.cancel != nil {
		t.cancel()
		t.cancel = nil
	}
}

// refresh fetches nodes/pods + metrics off the UI goroutine and renders them.
func (t *Z9sTop) refresh() {
	if t.app.factory == nil || t.app.Conn() == nil || !t.app.Conn().ConnectionOK() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), z9sTopFetchTimeout)
		defer cancel()

		var data z9sTopData

		nn, err := dao.FetchNodes(ctx, t.app.factory, "")
		if err != nil || nn == nil {
			slog.Warn("z9sTop: fetch nodes failed", slogs.Error, err)
			return
		}
		data.nodes = nn.Items

		mx := client.DialMetrics(t.app.Conn())
		data.nmx = make(client.NodesMetrics)
		if nmx, merr := mx.FetchNodesMetrics(ctx); merr == nil && nmx != nil {
			mx.NodesMetrics(nn, nmx, data.nmx)
		}

		if dial, derr := t.app.Conn().Dial(); derr == nil {
			if pl, perr := dial.CoreV1().Pods("").List(ctx, metav1.ListOptions{}); perr == nil && pl != nil {
				data.pods = pl.Items
			} else if perr != nil {
				slog.Warn("z9sTop: list pods failed", slogs.Error, perr)
			}
			if nsl, nerr := dial.CoreV1().Namespaces().List(ctx, metav1.ListOptions{}); nerr == nil && nsl != nil {
				data.nsCount = len(nsl.Items)
			}
			if dl, derr2 := dial.AppsV1().Deployments("").List(ctx, metav1.ListOptions{}); derr2 == nil && dl != nil {
				for i := range dl.Items {
					data.depTotal += int(dl.Items[i].Status.Replicas)
					data.depReady += int(dl.Items[i].Status.ReadyReplicas)
				}
			}
			if pvl, pverr := dial.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{}); pverr == nil && pvl != nil {
				data.pvCount = len(pvl.Items)
				for i := range pvl.Items {
					if q := pvl.Items[i].Spec.Capacity.Storage(); q != nil {
						data.pvBytes += q.Value()
					}
				}
			}
			if pvcl, pvcerr := dial.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{}); pvcerr == nil && pvcl != nil {
				data.pvcCount = len(pvcl.Items)
				for i := range pvcl.Items {
					if q := pvcl.Items[i].Spec.Resources.Requests.Storage(); q != nil {
						data.pvcBytes += q.Value()
					}
				}
			}
		}
		data.pmx = make(client.PodsMetrics)
		if pmx, perr := mx.FetchPodsMetrics(ctx, ""); perr == nil && pmx != nil {
			mx.PodsMetrics(pmx, data.pmx)
		}

		// Aggregate pod resource requests per node (skip terminal pods).
		data.reqCPU = make(map[string]int64, len(data.nodes))
		data.reqMEM = make(map[string]int64, len(data.nodes))
		for i := range data.pods {
			p := &data.pods[i]
			if p.Spec.NodeName == "" {
				continue
			}
			if p.Status.Phase == v1.PodSucceeded || p.Status.Phase == v1.PodFailed {
				continue
			}
			cpu, mem := podRequests(p)
			data.reqCPU[p.Spec.NodeName] += cpu
			data.reqMEM[p.Spec.NodeName] += mem
		}

		t.app.QueueUpdateDraw(func() {
			t.render(data)
		})
	}()
}

// render applies a snapshot. Must run on the UI goroutine.
func (t *Z9sTop) render(data z9sTopData) {
	sort.Slice(data.nodes, func(i, j int) bool { return data.nodes[i].Name < data.nodes[j].Name })
	sort.Slice(data.pods, func(i, j int) bool {
		if data.pods[i].Namespace != data.pods[j].Namespace {
			return data.pods[i].Namespace < data.pods[j].Namespace
		}
		return data.pods[i].Name < data.pods[j].Name
	})
	t.data = data

	t.renderSummary()
	t.renderNodes()
	t.renderPods()
	t.applyFocusStyles()
}

func (t *Z9sTop) renderSummary() {
	st := t.app.Styles
	// Cluster-header aesthetic: gold/orange labels and white values.
	sec := string(st.K9s.Info.FgColor)

	// Node-derived aggregates: requested (req) and live usage (use) vs allocatable.
	var reqCPU, allCPU, reqMEM, allMEM, useCPU, useMEM int64
	var ready, volsInUse, pressure int
	uptime := time.Time{}
	for i := range t.data.nodes {
		n := &t.data.nodes[i]
		if _, ok := nodeReady(n); ok {
			ready++
		}
		if !n.CreationTimestamp.IsZero() && (uptime.IsZero() || n.CreationTimestamp.Time.Before(uptime)) {
			uptime = n.CreationTimestamp.Time
		}
		volsInUse += len(n.Status.VolumesInUse)
		if nodeHasPressure(n) {
			pressure++
		}
		nm := t.data.nmx[n.Name]
		reqCPU += t.data.reqCPU[n.Name]
		allCPU += n.Status.Allocatable.Cpu().MilliValue()
		reqMEM += t.data.reqMEM[n.Name]
		allMEM += n.Status.Allocatable.Memory().Value() / (1024 * 1024)
		useCPU += nm.CurrentCPU
		useMEM += nm.CurrentMEM
	}

	// Pod-derived aggregates.
	var running, restarts, failed, evicted int
	for i := range t.data.pods {
		p := &t.data.pods[i]
		switch p.Status.Phase {
		case v1.PodRunning:
			running++
		case v1.PodFailed:
			failed++
			if p.Status.Reason == "Evicted" {
				evicted++
			}
		}
		restarts += int(podRestarts(p))
	}

	k8sVer := client.NA
	if info, err := t.app.Conn().ServerVersion(); err == nil && info != nil {
		k8sVer = info.GitVersion
	}
	metricsState := "[red::b]not connected[-:-:-]"
	if t.app.Conn().HasMetrics() {
		metricsState = "[green::b]metrics-server[-:-:-]"
	}

	uptimeStr := client.NA
	if !uptime.IsZero() {
		uptimeStr = duration.HumanDuration(time.Since(uptime))
	}

	// Tag helpers. Labels use the accent color; values are white. Pipes line up
	// across rows thanks to fixed-width fields.
	const bw = 22 // bar width
	const uw = 21 // fixed width for the "use" value so the "req" block aligns
	lbl := func(s string) string { return "[" + sec + "::b]" + s + "[-:-:-]" }
	white := func(s string) string { return "[white::]" + s + "[-:-:-]" }
	// valW renders a white value padded to a fixed visible width.
	valW := func(s string, width int) string {
		out := white(s)
		if pad := width - len(s); pad > 0 {
			out += strings.Repeat(" ", pad)
		}
		return out
	}
	colw := func(c, s string) string { return "[" + c + "::]" + s + "[-:-:-]" }
	pipe := "[" + sec + "::b]│[-:-:-] "
	bar := func(perc int) string {
		return "[" + sec + "::b]│[-:-:-]" + gaugeN(perc, bw) + "[" + sec + "::b]│[-:-:-]"
	}
	// field renders a fixed visible-width "label + value" cell (plain is the
	// uncolored value used only to compute padding; rendered is what's shown).
	field := func(label, plain, rendered string, width int) string {
		s := lbl(label) + rendered
		if pad := width - len(label) - len(plain); pad > 0 {
			s += strings.Repeat(" ", pad)
		}
		return s
	}
	// Shared widths: cols 0-3 are common to inventory and health so their first
	// pipes align vertically.
	w := []int{14, 13, 12, 18, 15, 10, 18, 18}

	nodesStr := fmt.Sprintf("%d/%d", ready, len(t.data.nodes))
	nsStr := fmt.Sprintf("%d", t.data.nsCount)
	depStr := fmt.Sprintf("%d/%d", t.data.depReady, t.data.depTotal)
	podsStr := fmt.Sprintf("%d/%d", running, len(t.data.pods))
	volsStr := fmt.Sprintf("%d", volsInUse)
	pvStr := fmt.Sprintf("%d (%s)", t.data.pvCount, fmtBytes(t.data.pvBytes))
	pvcStr := fmt.Sprintf("%d (%s)", t.data.pvcCount, fmtBytes(t.data.pvcBytes))
	rstStr := fmt.Sprintf("%d", restarts)
	failStr := fmt.Sprintf("%d", failed)
	evStr := fmt.Sprintf("%d", evicted)
	presStr := fmt.Sprintf("%d", pressure)

	t.summary.Clear()
	// Block 1: identity.
	_, _ = fmt.Fprintln(t.summary, " "+
		lbl("Context: ")+white(t.app.Conn().ActiveContext())+"   "+
		lbl("K8s: ")+white(k8sVer)+"   "+
		lbl("Metrics: ")+metricsState)
	_, _ = fmt.Fprintln(t.summary, "")
	// Block 2: inventory + health (aligned columns).
	_, _ = fmt.Fprintln(t.summary, " "+
		field("Uptime: ", uptimeStr, white(uptimeStr), w[0])+pipe+
		field("Nodes: ", nodesStr, white(nodesStr), w[1])+pipe+
		field("NS: ", nsStr, white(nsStr), w[2])+pipe+
		field("Deploys: ", depStr, colw(ratioColor(t.data.depReady, t.data.depTotal), depStr), w[3])+pipe+
		field("Pods: ", podsStr, colw(ratioColor(running, len(t.data.pods)), podsStr), w[4])+pipe+
		field("Vols: ", volsStr, white(volsStr), w[5])+pipe+
		field("PVs: ", pvStr, white(pvStr), w[6])+pipe+
		field("PVCs: ", pvcStr, white(pvcStr), w[7]))
	_, _ = fmt.Fprintln(t.summary, " "+
		field("Restarts: ", rstStr, colw(countColor(restarts, 50, 100), rstStr), w[0])+pipe+
		field("Failures: ", failStr, colw(zeroBadColor(failed), failStr), w[1])+pipe+
		field("Evicted: ", evStr, colw(zeroWarnColor(evicted), evStr), w[2])+pipe+
		field("Pressure: ", presStr, colw(zeroBadColor(pressure), presStr), w[3]))
	_, _ = fmt.Fprintln(t.summary, "")
	// Block 3: cluster CPU/MEM, live usage and requested with delimited bars.
	_, _ = fmt.Fprintln(t.summary, " "+
		lbl("CPU  ")+
		lbl("use ")+bar(client.ToPercentage(useCPU, allCPU))+valW(fmt.Sprintf(" %dm/%dm", useCPU, allCPU), uw)+
		lbl("req ")+bar(client.ToPercentage(reqCPU, allCPU))+white(fmt.Sprintf(" %dm/%dm", reqCPU, allCPU)))
	_, _ = fmt.Fprint(t.summary, " "+
		lbl("MEM  ")+
		lbl("use ")+bar(client.ToPercentage(useMEM, allMEM))+valW(fmt.Sprintf(" %s/%s", fmtMB(useMEM), fmtMB(allMEM)), uw)+
		lbl("req ")+bar(client.ToPercentage(reqMEM, allMEM))+white(fmt.Sprintf(" %s/%s", fmtMB(reqMEM), fmtMB(allMEM))))
}

// ratioColor returns a color name based on ready/total health.
func ratioColor(ready, total int) string {
	if total <= 0 {
		return "green"
	}
	r := float64(ready) / float64(total)
	switch {
	case r <= 0.5:
		return "red"
	case r < 0.8:
		return "yellow"
	default:
		return "green"
	}
}

func countColor(n, warn, bad int) string {
	switch {
	case n >= bad:
		return "red"
	case n >= warn:
		return "yellow"
	default:
		return "green"
	}
}

func zeroWarnColor(n int) string {
	if n > 0 {
		return "yellow"
	}
	return "green"
}

func zeroBadColor(n int) string {
	if n > 0 {
		return "red"
	}
	return "green"
}

// nodeHasPressure reports whether a node has any active pressure condition.
func nodeHasPressure(n *v1.Node) bool {
	for i := range n.Status.Conditions {
		c := n.Status.Conditions[i]
		if c.Status != v1.ConditionTrue {
			continue
		}
		switch c.Type {
		case v1.NodeMemoryPressure, v1.NodeDiskPressure, v1.NodePIDPressure:
			return true
		}
	}
	return false
}

// fmtBytes renders a byte quantity as a human readable size.
func fmtBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.0f%ci", float64(b)/float64(div), "KMGTPE"[exp])
}

func (t *Z9sTop) renderNodes() {
	prevRow, _ := t.nodes.GetSelection()

	t.nodes.Clear()
	t.setHeader(t.nodes, "NAME", "STATUS", "PODS", "CPU/USE", "CPU/REQ", "MEM/USE", "MEM/REQ")

	// pods per node
	podCount := make(map[string]int, len(t.data.nodes))
	for i := range t.data.pods {
		podCount[t.data.pods[i].Spec.NodeName]++
	}

	for i := range t.data.nodes {
		n := &t.data.nodes[i]
		row := i + 1
		status, ok := nodeReady(n)

		allocCPU := n.Status.Allocatable.Cpu().MilliValue()
		allocMEM := n.Status.Allocatable.Memory().Value() / (1024 * 1024)
		rc, rm := t.data.reqCPU[n.Name], t.data.reqMEM[n.Name]
		nm := t.data.nmx[n.Name]

		fg := tcell.GetColor(celesteColor)
		t.nodes.SetCell(row, 0, t.cell(n.Name, fg))
		statusCell := t.cell(status, tcell.ColorGreen)
		if !ok {
			statusCell = t.cell(status, tcell.ColorRed)
		}
		t.nodes.SetCell(row, 1, statusCell)
		t.nodes.SetCell(row, 2, t.cell(fmt.Sprintf("%d", podCount[n.Name]), fg))
		t.nodes.SetCell(row, 3, t.gaugeCell(nm.CurrentCPU, allocCPU, fmt.Sprintf("%dm", nm.CurrentCPU)))
		t.nodes.SetCell(row, 4, t.gaugeCell(rc, allocCPU, fmt.Sprintf("%dm", rc)))
		t.nodes.SetCell(row, 5, t.gaugeCell(nm.CurrentMEM, allocMEM, fmtMB(nm.CurrentMEM)))
		t.nodes.SetCell(row, 6, t.gaugeCell(rm, allocMEM, fmtMB(rm)))
	}

	if prevRow > 0 && prevRow <= len(t.data.nodes) {
		t.nodes.Select(prevRow, 0)
	}
	t.nodes.SetTitle(fmt.Sprintf(" Nodes (%d) ", len(t.data.nodes)))
}

func (t *Z9sTop) renderPods() {
	prevRow, _ := t.pods.GetSelection()

	t.pods.Clear()
	t.setHeader(t.pods, "NAMESPACE", "NAME", "READY", "STATUS", "RST", "NODE", "CPU/USE", "MEM/USE")

	for i := range t.data.pods {
		p := &t.data.pods[i]
		row := i + 1
		nm := t.data.nmx[p.Spec.NodeName]
		pm := t.data.pmx[p.Namespace+"/"+p.Name]

		fg := tcell.GetColor(celesteColor)
		phase := string(p.Status.Phase)
		phaseColor := fg
		switch phase {
		case "Running", "Succeeded":
			phaseColor = tcell.ColorGreen
		case "Pending":
			phaseColor = tcell.ColorYellow
		case "Failed":
			phaseColor = tcell.ColorRed
		}

		t.pods.SetCell(row, 0, t.cell(p.Namespace, fg))
		t.pods.SetCell(row, 1, t.cell(p.Name, fg))
		t.pods.SetCell(row, 2, t.cell(podReady(p), fg))
		t.pods.SetCell(row, 3, t.cell(phase, phaseColor))
		t.pods.SetCell(row, 4, t.cell(fmt.Sprintf("%d", podRestarts(p)), fg))
		t.pods.SetCell(row, 5, t.cell(p.Spec.NodeName, fg))
		t.pods.SetCell(row, 6, t.gaugeCell(pm.CurrentCPU, nm.AllocatableCPU, fmt.Sprintf("%dm", pm.CurrentCPU)))
		t.pods.SetCell(row, 7, t.gaugeCell(pm.CurrentMEM, nm.AllocatableMEM, fmtMB(pm.CurrentMEM)))
	}

	if prevRow > 0 && prevRow <= len(t.data.pods) {
		t.pods.Select(prevRow, 0)
	}
	ns := t.app.Config.ActiveNamespace()
	if ns == "" {
		ns = "all"
	}
	t.pods.SetTitle(fmt.Sprintf(" Pods · %s (%d) ", ns, len(t.data.pods)))
}

func (t *Z9sTop) setHeader(tbl *tview.Table, cols ...string) {
	c := t.app.Styles.Table().Header.FgColor.Color()
	for i, h := range cols {
		tbl.SetCell(0, i, tview.NewTableCell(h).
			SetTextColor(c).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold).
			SetExpansion(1))
	}
}

func (t *Z9sTop) cell(s string, color tcell.Color) *tview.TableCell {
	return tview.NewTableCell(s).SetTextColor(color).SetExpansion(1)
}

// gaugeCell renders a colored usage bar plus a raw value, e.g. "███░░ 45%  120m".
func (t *Z9sTop) gaugeCell(cur, total int64, value string) *tview.TableCell {
	perc := client.ToPercentage(cur, total)
	txt := fmt.Sprintf("%s [%s::]%3d%%  %s[-:-:-]", gauge(perc), celesteColor, perc, value)
	return tview.NewTableCell(txt).SetExpansion(2)
}

// gauge renders a two-tone usage bar of the default width.
func gauge(perc int) string {
	return gaugeN(perc, gaugeWidth)
}

// gaugeN renders a two-tone usage bar of the given width using derailed/tview
// color tags. The filled portion is colored by severity; the empty portion is a
// solid gray block (instead of a dotted character).
func gaugeN(perc, width int) string {
	if perc < 0 {
		perc = 0
	}
	if perc > 100 {
		perc = 100
	}
	filled := perc * width / 100

	color := "green"
	switch {
	case perc >= 90:
		color = "red"
	case perc >= 70:
		color = "orange"
	}

	var b strings.Builder
	b.WriteString("[" + color + "::]")
	b.WriteString(strings.Repeat("█", filled))
	b.WriteString("[black::]")
	b.WriteString(strings.Repeat("█", width-filled))
	b.WriteString("[-:-:-]")
	return b.String()
}

// fmtMB renders a MB quantity as MB or GB for readability.
func fmtMB(mb int64) string {
	if mb >= 1024 {
		return fmt.Sprintf("%.1fGi", float64(mb)/1024)
	}
	return fmt.Sprintf("%dMi", mb)
}

// nodeReady reports the node Ready condition and whether it is healthy.
func nodeReady(n *v1.Node) (string, bool) {
	for i := range n.Status.Conditions {
		c := n.Status.Conditions[i]
		if c.Type == v1.NodeReady {
			if c.Status == v1.ConditionTrue {
				return "Ready", true
			}
			return "NotReady", false
		}
	}
	return "Unknown", false
}

// podReady returns the ready/total container count as "r/t".
func podReady(p *v1.Pod) string {
	ready := 0
	for i := range p.Status.ContainerStatuses {
		if p.Status.ContainerStatuses[i].Ready {
			ready++
		}
	}
	return fmt.Sprintf("%d/%d", ready, len(p.Status.ContainerStatuses))
}

// podRequests returns the pod's aggregated resource requests as ktop does:
// the sum of regular container requests plus pod overhead. Returns CPU in
// millicores and memory in MB.
func podRequests(p *v1.Pod) (cpuMilli, memMB int64) {
	var cpu, mem int64
	for i := range p.Spec.Containers {
		r := p.Spec.Containers[i].Resources.Requests
		cpu += r.Cpu().MilliValue()
		mem += r.Memory().Value()
	}
	if p.Spec.Overhead != nil {
		cpu += p.Spec.Overhead.Cpu().MilliValue()
		mem += p.Spec.Overhead.Memory().Value()
	}
	return cpu, mem / (1024 * 1024)
}

// podRestarts sums container restart counts.
func podRestarts(p *v1.Pod) int32 {
	var n int32
	for i := range p.Status.ContainerStatuses {
		n += p.Status.ContainerStatuses[i].RestartCount
	}
	return n
}

// nodeAge returns a humanized age for a node.
func resourceAge(ts metav1.Time) string {
	if ts.IsZero() {
		return client.NA
	}
	return duration.HumanDuration(time.Since(ts.Time))
}
