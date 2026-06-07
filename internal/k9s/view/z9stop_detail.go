// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Z9s

package view

import (
	"fmt"
	"sort"
	"strings"

	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"github.com/yourusername/z9s/internal/k9s/client"
	v1 "k8s.io/api/core/v1"
)

// showNodeDetail opens a dedicated page with node information and the pods
// scheduled on it, reusing the dashboard's last data snapshot. ESC returns.
func (t *Z9sTop) showNodeDetail(node *v1.Node) {
	detail := t.buildNodeDetail(node)

	if t.app.Main.HasPage(z9sNodeDetailPageID) {
		t.app.Main.RemovePage(z9sNodeDetailPageID)
	}
	t.app.Main.AddPage(z9sNodeDetailPageID, detail.root, true, false)
	t.app.Main.SwitchToPage(z9sNodeDetailPageID)
	t.app.SetFocus(detail.pods)
}

type nodeDetail struct {
	root *tview.Flex
	pods *tview.Table
}

func (t *Z9sTop) buildNodeDetail(node *v1.Node) *nodeDetail {
	st := t.app.Styles
	sec := string(st.Frame().Title.FgColor)

	info := tview.NewTextView()
	info.SetDynamicColors(true)
	info.SetBorder(true)
	info.SetTitle(fmt.Sprintf(" Node: %s ", node.Name))
	info.SetBackgroundColor(st.BgColor())
	info.SetTitleColor(st.Frame().Title.FgColor.Color())
	info.SetBorderColor(st.Frame().Border.FocusColor.Color())

	status, _ := nodeReady(node)
	ni := node.Status.NodeInfo
	allocCPU := node.Status.Allocatable.Cpu().MilliValue()
	allocMEM := node.Status.Allocatable.Memory().Value() / (1024 * 1024)
	reqCPU, reqMEM := t.data.reqCPU[node.Name], t.data.reqMEM[node.Name]

	_, _ = fmt.Fprintf(info, " [%s::b]Status:[-:-:-] %s    [%s::b]Age:[-:-:-] %s    [%s::b]IP:[-:-:-] %s\n",
		sec, status, sec, resourceAge(node.CreationTimestamp), sec, nodeInternalIP(node))
	_, _ = fmt.Fprintf(info, " [%s::b]Pool:[-:-:-] %s    [%s::b]Instance:[-:-:-] %s    [%s::b]Zone:[-:-:-] %s\n",
		sec, nodePool(node), sec, nodeLabel(node, "node.kubernetes.io/instance-type"), sec, nodeLabel(node, "topology.kubernetes.io/zone"))
	_, _ = fmt.Fprintf(info, " [%s::b]OS:[-:-:-] %s    [%s::b]Kernel:[-:-:-] %s    [%s::b]Runtime:[-:-:-] %s\n",
		sec, ni.OSImage, sec, ni.KernelVersion, sec, ni.ContainerRuntimeVersion)
	_, _ = fmt.Fprintf(info, " [%s::b]Kubelet:[-:-:-] %s    [%s::b]Arch:[-:-:-] %s    [%s::b]Conditions:[-:-:-] %s\n",
		sec, ni.KubeletVersion, sec, ni.Architecture, sec, nodeConditions(node))
	cpuPerc := client.ToPercentage(reqCPU, allocCPU)
	memPerc := client.ToPercentage(reqMEM, allocMEM)
	_, _ = fmt.Fprintf(info, " [%s::b]CPU/REQ:[-:-:-] %s %3d%%  %dm/%dm    [%s::b]MEM/REQ:[-:-:-] %s %3d%%  %s/%s",
		sec, gauge(cpuPerc), cpuPerc, reqCPU, allocCPU,
		sec, gauge(memPerc), memPerc, fmtMB(reqMEM), fmtMB(allocMEM))

	pods := tview.NewTable()
	pods.SetBorder(true)
	pods.SetFixed(1, 0)
	pods.SetSelectable(true, false)
	pods.SetBackgroundColor(st.BgColor())
	pods.SetTitleColor(st.Frame().Title.FgColor.Color())
	pods.SetBorderColor(st.Frame().Border.FgColor.Color())

	onNode := make([]v1.Pod, 0)
	for i := range t.data.pods {
		if t.data.pods[i].Spec.NodeName == node.Name {
			onNode = append(onNode, t.data.pods[i])
		}
	}
	sort.Slice(onNode, func(i, j int) bool {
		if onNode[i].Namespace != onNode[j].Namespace {
			return onNode[i].Namespace < onNode[j].Namespace
		}
		return onNode[i].Name < onNode[j].Name
	})

	t.setHeader(pods, "NAMESPACE", "NAME", "READY", "STATUS", "RST", "CPU", "MEM")
	for i := range onNode {
		p := &onNode[i]
		row := i + 1
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
		pods.SetCell(row, 0, t.cell(p.Namespace, fg))
		pods.SetCell(row, 1, t.cell(p.Name, fg))
		pods.SetCell(row, 2, t.cell(podReady(p), fg))
		pods.SetCell(row, 3, t.cell(phase, phaseColor))
		pods.SetCell(row, 4, t.cell(fmt.Sprintf("%d", podRestarts(p)), fg))
		pods.SetCell(row, 5, t.gaugeCell(pm.CurrentCPU, allocCPU, fmt.Sprintf("%dm", pm.CurrentCPU)))
		pods.SetCell(row, 6, t.gaugeCell(pm.CurrentMEM, allocMEM, fmtMB(pm.CurrentMEM)))
	}
	pods.SetTitle(fmt.Sprintf(" Pods on node (%d) ", len(onNode)))

	pods.SetInputCapture(func(evt *tcell.EventKey) *tcell.EventKey {
		if evt.Key() == tcell.KeyEscape {
			t.app.Main.SwitchToPage(z9sTopPageID)
			t.app.SetFocus(t.nodes)
			return nil
		}
		return evt
	})

	root := tview.NewFlex().SetDirection(tview.FlexRow)
	root.SetBackgroundColor(st.BgColor())
	root.AddItem(info, 7, 0, false)
	root.AddItem(pods, 0, 1, true)

	return &nodeDetail{root: root, pods: pods}
}

// nodeInternalIP returns the node's internal IP if present.
func nodeInternalIP(n *v1.Node) string {
	for i := range n.Status.Addresses {
		if n.Status.Addresses[i].Type == v1.NodeInternalIP {
			return n.Status.Addresses[i].Address
		}
	}
	return client.NA
}

// nodePool returns the Karpenter nodepool or EKS managed nodegroup label.
func nodePool(n *v1.Node) string {
	if v := nodeLabel(n, "karpenter.sh/nodepool"); v != client.NA {
		return v
	}
	if v := nodeLabel(n, "eks.amazonaws.com/nodegroup"); v != client.NA {
		return v
	}
	return client.NA
}

func nodeLabel(n *v1.Node, key string) string {
	if v, ok := n.Labels[key]; ok && v != "" {
		return v
	}
	return client.NA
}

// nodeConditions returns a compact list of active (True) pressure conditions.
func nodeConditions(n *v1.Node) string {
	var active []string
	for i := range n.Status.Conditions {
		c := n.Status.Conditions[i]
		if c.Type == v1.NodeReady {
			continue
		}
		if c.Status == v1.ConditionTrue {
			active = append(active, string(c.Type))
		}
	}
	if len(active) == 0 {
		return "[green::]none[-:-:-]"
	}
	return "[orangered::]" + strings.Join(active, ",") + "[-:-:-]"
}
