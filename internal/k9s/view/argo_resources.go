// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package view

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

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
	"sigs.k8s.io/yaml"
)

// argoTreeChildGVRs are the resource kinds we walk to rebuild an Application's
// resource tree from the live cluster (via ownerReferences), e.g.
// Deployment -> ReplicaSet -> Pod or Service -> EndpointSlice.
var argoTreeChildGVRs = []schema.GroupVersionResource{
	{Group: "apps", Version: "v1", Resource: "replicasets"},
	{Group: "", Version: "v1", Resource: "pods"},
	{Group: "batch", Version: "v1", Resource: "jobs"},
	{Group: "apps", Version: "v1", Resource: "controllerrevisions"},
	{Group: "discovery.k8s.io", Version: "v1", Resource: "endpointslices"},
}

const argoTreeMaxDepth = 8

// argoRef points at a concrete cluster resource behind a tree node.
type argoRef struct {
	gvr       schema.GroupVersionResource
	namespace string
	name      string
	kind      string
}

// treeNode is a kind-agnostic node in the Application resource tree.
type treeNode struct {
	label    string
	color    tcell.Color
	ref      *argoRef
	children []*treeNode
}

// childObj couples a live object with the GVR it was listed under.
type childObj struct {
	obj *unstructured.Unstructured
	gvr schema.GroupVersionResource
}

// ----------------------------------------------------------------------------
// ArgoResources component: a navigable tree of every resource an ArgoCD
// Application manages, reconstructed from the live cluster (status.resources +
// ownerReferences), mirroring the ArgoCD "Application Details" tree. <enter>
// shows a resource's live manifest; <x> decodes a Secret.

type ArgoResources struct {
	*tview.TreeView

	app     *App
	ns      string
	name    string
	actions *ui.KeyActions
}

// NewArgoResources returns the managed-resources tree view for an Application.
func NewArgoResources(app *App, ns, name string) *ArgoResources {
	v := ArgoResources{
		TreeView: tview.NewTreeView(),
		app:      app,
		ns:       ns,
		name:     name,
		actions:  ui.NewKeyActions(),
	}
	v.actions.Bulk(ui.KeyMap{
		ui.KeyX:         ui.NewKeyAction("Decode Secret", v.decodeCmd, true),
		ui.KeyR:         ui.NewKeyAction("Refresh", v.refreshCmd, true),
		ui.KeySpace:     ui.NewKeyAction("Expand/Collapse", v.toggleCmd, true),
		tcell.KeyEscape: ui.NewKeyAction("Back", app.PrevCmd, false),
		ui.KeyQ:         ui.NewKeyAction("Back", app.PrevCmd, false),
	})

	return &v
}

// Init initializes the view.
func (v *ArgoResources) Init(context.Context) error {
	v.SetBorder(true)
	v.SetTitle(fmt.Sprintf(" Application: %s ", v.name))
	v.SetTitleColor(tcell.ColorLightSkyBlue)
	v.SetBorderPadding(0, 0, 1, 1)
	v.SetGraphics(true)
	v.SetGraphicsColor(tcell.ColorCadetBlue)
	v.SetSelectedFunc(v.viewNode)
	v.SetInputCapture(v.keyboard)
	v.StylesChanged(v.app.Styles)

	root := tview.NewTreeNode(argoTreeLoading()).SetSelectable(false)
	v.SetRoot(root).SetCurrentNode(root)

	return nil
}

func (v *ArgoResources) keyboard(evt *tcell.EventKey) *tcell.EventKey {
	if a, ok := v.actions.Get(ui.AsKey(evt)); ok {
		return a.Action(evt)
	}

	return evt
}

func (v *ArgoResources) refreshCmd(*tcell.EventKey) *tcell.EventKey {
	v.refresh()

	return nil
}

func (v *ArgoResources) toggleCmd(*tcell.EventKey) *tcell.EventKey {
	n := v.GetCurrentNode()
	if n != nil && len(n.GetChildren()) > 0 {
		n.SetExpanded(!n.IsExpanded())
	}

	return nil
}

// refresh fetches the Application's resource tree off the UI thread.
func (v *ArgoResources) refresh() {
	go func() {
		root, err := v.app.fetchArgoTree(v.ns, v.name)
		v.app.QueueUpdateDraw(func() {
			v.rebuild(root, err)
		})
	}()
}

func (v *ArgoResources) rebuild(root *treeNode, err error) {
	if err != nil {
		n := tview.NewTreeNode("Error: " + err.Error()).
			SetColor(tcell.ColorRed).SetSelectable(false)
		v.SetRoot(n).SetCurrentNode(n)
		return
	}

	rn := toTviewNode(root)
	v.SetRoot(rn).SetCurrentNode(rn)
}

// viewNode opens the live manifest of the resource behind a node (<enter>).
func (v *ArgoResources) viewNode(node *tview.TreeNode) {
	if node == nil {
		return
	}
	ref, ok := node.GetReference().(*argoRef)
	if !ok || ref == nil {
		return
	}
	go func() {
		raw, err := v.app.fetchManifest(ref)
		v.app.QueueUpdateDraw(func() {
			if err != nil {
				v.app.Flash().Err(err)
				return
			}
			title := fmt.Sprintf("%s/%s", ref.kind, ref.name)
			d := NewDetails(v.app, "YAML", title, contentYAML, true).Update(raw)
			if e := v.app.inject(d, false); e != nil {
				v.app.Flash().Err(e)
			}
		})
	}()
}

// decodeCmd decodes the selected Secret's data (<x>).
func (v *ArgoResources) decodeCmd(*tcell.EventKey) *tcell.EventKey {
	node := v.GetCurrentNode()
	if node == nil {
		return nil
	}
	ref, ok := node.GetReference().(*argoRef)
	if !ok || ref == nil {
		return nil
	}
	if ref.kind != "Secret" {
		v.app.Flash().Info("Decode (<x>) only applies to Secrets. Use <enter> for the manifest.")
		return nil
	}
	go func() {
		raw, err := v.app.fetchDecodedSecret(ref)
		v.app.QueueUpdateDraw(func() {
			if err != nil {
				v.app.Flash().Err(err)
				return
			}
			title := fmt.Sprintf("%s/%s", ref.kind, ref.name)
			d := NewDetails(v.app, "Secret Decoder", title, contentYAML, true).Update(raw)
			if e := v.app.inject(d, false); e != nil {
				v.app.Flash().Err(e)
			}
		})
	}()

	return nil
}

// StylesChanged notifies the skin changed.
func (v *ArgoResources) StylesChanged(s *config.Styles) {
	v.SetBackgroundColor(s.BgColor())
	v.SetBorderColor(s.Frame().Border.FgColor.Color())
	v.SetBorderFocusColor(s.Frame().Border.FocusColor.Color())
	v.SetTitleColor(tcell.ColorLightSkyBlue)
}

// Name returns the component name.
func (v *ArgoResources) Name() string { return v.name }

// Start starts the view and triggers a data load.
func (v *ArgoResources) Start() {
	v.app.Styles.RemoveListener(v)
	v.app.Styles.AddListener(v)
	v.refresh()
}

// Stop terminates the view.
func (v *ArgoResources) Stop() { v.app.Styles.RemoveListener(v) }

// Hints returns menu hints.
func (v *ArgoResources) Hints() model.MenuHints { return v.actions.Hints() }

// ExtraHints returns additional hints.
func (*ArgoResources) ExtraHints() map[string]string { return nil }

// InCmdMode checks if prompt is active.
func (*ArgoResources) InCmdMode() bool { return false }

func (*ArgoResources) SetFilter(string, bool)                 {}
func (*ArgoResources) SetLabelSelector(labels.Selector, bool) {}
func (*ArgoResources) SetCommand(*cmd.Interpreter)            {}

func toTviewNode(n *treeNode) *tview.TreeNode {
	tn := tview.NewTreeNode(n.label).SetColor(n.color).SetExpanded(true)
	if n.ref != nil {
		tn.SetReference(n.ref)
	}
	for _, c := range n.children {
		tn.AddChild(toTviewNode(c))
	}

	return tn
}

// ----------------------------------------------------------------------------
// Data fetching: status.resources gives the top-level managed resources; the
// rest of the tree is rebuilt from the live cluster via ownerReferences.

func (a *App) fetchArgoTree(ns, name string) (*treeNode, error) {
	conn := a.Conn()
	if conn == nil {
		return nil, fmt.Errorf("no cluster connection")
	}
	dyn, err := conn.DynDial()
	if err != nil {
		return nil, err
	}

	appGVR := schema.GroupVersionResource{
		Group:    argoApplicationGroup,
		Version:  "v1alpha1",
		Resource: "applications",
	}

	ctx, cancel := context.WithTimeout(context.Background(), conn.Config().CallTimeout())
	defer cancel()

	app, err := dyn.Resource(appGVR).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	kindToGVR := a.buildKindToGVR()

	type topRes struct {
		group, kind, namespace, name, sync, health string
	}
	items, _, _ := unstructured.NestedSlice(app.Object, "status", "resources")
	tops := make([]topRes, 0, len(items))
	nsSet := map[string]bool{}
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		t := topRes{
			group:     asString(m["group"]),
			kind:      asString(m["kind"]),
			namespace: asString(m["namespace"]),
			name:      asString(m["name"]),
			sync:      asString(m["status"]),
		}
		if h, ok := m["health"].(map[string]any); ok {
			t.health = asString(h["status"])
		}
		tops = append(tops, t)
		if t.namespace != "" {
			nsSet[t.namespace] = true
		}
	}
	if dest, _, _ := unstructured.NestedString(app.Object, "spec", "destination", "namespace"); dest != "" {
		nsSet[dest] = true
	}

	// Build an ownerKey -> children map from candidate child resources.
	childrenByOwner := map[string][]childObj{}
	for childNS := range nsSet {
		for _, gvr := range argoTreeChildGVRs {
			list, lerr := dyn.Resource(gvr).Namespace(childNS).List(ctx, metav1.ListOptions{})
			if lerr != nil {
				continue
			}
			for i := range list.Items {
				o := &list.Items[i]
				for _, or := range o.GetOwnerReferences() {
					key := ownerKey(o.GetNamespace(), or.Kind, or.Name)
					childrenByOwner[key] = append(childrenByOwner[key], childObj{obj: o, gvr: gvr})
				}
			}
		}
	}

	root := &treeNode{
		label: fmt.Sprintf("Application  %s", name),
		color: argoNodeColor("", argoHealthOf(app), ""),
		ref:   &argoRef{gvr: appGVR, namespace: ns, name: name, kind: "Application"},
	}
	sort.Slice(tops, func(i, j int) bool {
		if tops[i].kind != tops[j].kind {
			return tops[i].kind < tops[j].kind
		}
		return tops[i].name < tops[j].name
	})
	for _, t := range tops {
		status := t.sync
		if t.health != "" {
			status = t.health
		}
		node := &treeNode{
			label: argoNodeLabel(t.kind, t.name, status),
			color: argoNodeColor(t.sync, t.health, ""),
			ref:   refFor(kindToGVR, t.group, t.kind, t.namespace, t.name),
		}
		addLiveChildren(node, t.namespace, t.kind, t.name, childrenByOwner, 1)
		root.children = append(root.children, node)
	}

	return root, nil
}

func addLiveChildren(parent *treeNode, ns, kind, name string, m map[string][]childObj, depth int) {
	if depth > argoTreeMaxDepth {
		return
	}
	kids := m[ownerKey(ns, kind, name)]
	sort.Slice(kids, func(i, j int) bool {
		if kids[i].obj.GetKind() != kids[j].obj.GetKind() {
			return kids[i].obj.GetKind() < kids[j].obj.GetKind()
		}
		return kids[i].obj.GetName() < kids[j].obj.GetName()
	})
	for _, c := range kids {
		o := c.obj
		status, color := liveStatus(o)
		cn := &treeNode{
			label: argoNodeLabel(o.GetKind(), o.GetName(), status),
			color: color,
			ref:   &argoRef{gvr: c.gvr, namespace: o.GetNamespace(), name: o.GetName(), kind: o.GetKind()},
		}
		parent.children = append(parent.children, cn)
		addLiveChildren(cn, o.GetNamespace(), o.GetKind(), o.GetName(), m, depth+1)
	}
}

// buildKindToGVR maps "group|Kind" to a served GVR using discovery.
func (a *App) buildKindToGVR() map[string]schema.GroupVersionResource {
	out := map[string]schema.GroupVersionResource{}
	conn := a.Conn()
	if conn == nil {
		return out
	}
	disc, err := conn.CachedDiscovery()
	if err != nil {
		return out
	}
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
			r := &list.APIResources[i]
			if strings.Contains(r.Name, "/") {
				continue
			}
			out[gv.Group+"|"+r.Kind] = gv.WithResource(r.Name)
		}
	}

	return out
}

func refFor(kindToGVR map[string]schema.GroupVersionResource, group, kind, ns, name string) *argoRef {
	gvr, ok := kindToGVR[group+"|"+kind]
	if !ok {
		return nil
	}

	return &argoRef{gvr: gvr, namespace: ns, name: name, kind: kind}
}

// fetchManifest returns the cleaned live YAML for a resource.
func (a *App) fetchManifest(ref *argoRef) (string, error) {
	conn := a.Conn()
	if conn == nil {
		return "", fmt.Errorf("no cluster connection")
	}
	dyn, err := conn.DynDial()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), conn.Config().CallTimeout())
	defer cancel()

	res := dyn.Resource(ref.gvr)
	var o *unstructured.Unstructured
	if ref.namespace != "" {
		o, err = res.Namespace(ref.namespace).Get(ctx, ref.name, metav1.GetOptions{})
	} else {
		o, err = res.Get(ctx, ref.name, metav1.GetOptions{})
	}
	if err != nil {
		return "", err
	}

	unstructured.RemoveNestedField(o.Object, "metadata", "managedFields")
	b, err := yaml.Marshal(o.Object)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// fetchDecodedSecret returns a Secret's data base64-decoded as YAML.
func (a *App) fetchDecodedSecret(ref *argoRef) (string, error) {
	conn := a.Conn()
	if conn == nil {
		return "", fmt.Errorf("no cluster connection")
	}
	dyn, err := conn.DynDial()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), conn.Config().CallTimeout())
	defer cancel()

	o, err := dyn.Resource(ref.gvr).Namespace(ref.namespace).Get(ctx, ref.name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	data, _, _ := unstructured.NestedMap(o.Object, "data")
	decoded := map[string]string{}
	for k, val := range data {
		s := asString(val)
		if b, derr := base64.StdEncoding.DecodeString(s); derr == nil {
			decoded[k] = string(b)
		} else {
			decoded[k] = s
		}
	}
	if len(decoded) == 0 {
		return "# (no data keys to decode)\n", nil
	}

	b, err := yaml.Marshal(map[string]any{"data": decoded})
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func ownerKey(ns, kind, name string) string {
	return ns + "|" + kind + "|" + name
}

func argoNodeLabel(kind, name, status string) string {
	if strings.TrimSpace(status) == "" {
		return fmt.Sprintf("%s  %s", kind, name)
	}

	return fmt.Sprintf("%s  %s   (%s)", kind, name, status)
}

// liveStatus derives a short status string and color for a live object.
func liveStatus(o *unstructured.Unstructured) (string, tcell.Color) {
	switch o.GetKind() {
	case "Pod":
		phase, _, _ := unstructured.NestedString(o.Object, "status", "phase")
		ready, total := podReadiness(o)
		txt := phase
		if total > 0 {
			txt = fmt.Sprintf("%s %d/%d", phase, ready, total)
		}
		return txt, podPhaseColor(phase)
	case "ReplicaSet", "Deployment", "StatefulSet", "DaemonSet":
		ready, _, _ := unstructured.NestedInt64(o.Object, "status", "readyReplicas")
		desired, ok, _ := unstructured.NestedInt64(o.Object, "spec", "replicas")
		if !ok {
			desired, _, _ = unstructured.NestedInt64(o.Object, "status", "replicas")
		}
		color := tcell.ColorOrange
		if desired == 0 || ready == desired {
			color = tcell.ColorGreen
		}
		return fmt.Sprintf("%d/%d", ready, desired), color
	case "Job":
		succeeded, _, _ := unstructured.NestedInt64(o.Object, "status", "succeeded")
		failed, _, _ := unstructured.NestedInt64(o.Object, "status", "failed")
		if failed > 0 {
			return "failed", tcell.ColorRed
		}
		if succeeded > 0 {
			return "complete", tcell.ColorGreen
		}
		return "running", tcell.ColorDodgerBlue
	default:
		return "", tcell.ColorLightSkyBlue
	}
}

func podReadiness(o *unstructured.Unstructured) (ready, total int) {
	cs, _, _ := unstructured.NestedSlice(o.Object, "status", "containerStatuses")
	for _, c := range cs {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		total++
		if r, ok := m["ready"].(bool); ok && r {
			ready++
		}
	}

	return ready, total
}

func podPhaseColor(phase string) tcell.Color {
	switch phase {
	case "Running":
		return tcell.ColorGreen
	case "Succeeded":
		return tcell.ColorGray
	case "Pending":
		return tcell.ColorOrange
	case "Failed":
		return tcell.ColorRed
	default:
		return tcell.ColorOrange
	}
}

// argoNodeColor picks a node color from sync/health/phase, health first.
func argoNodeColor(sync, health, _ string) tcell.Color {
	if health != "" {
		_, c := argoHealthGlyph(health)
		return c
	}
	if sync != "" {
		_, c := argoSyncGlyph(sync)
		return c
	}

	return tcell.ColorLightSkyBlue
}

func argoTreeLoading() string { return "Loading resource tree..." }

// ----------------------------------------------------------------------------
// Status glyph helpers (shared with the Application list look).

// argoHealthGlyph maps a health status to an icon-prefixed label and color.
func argoHealthGlyph(s string) (string, tcell.Color) {
	switch strings.TrimSpace(s) {
	case "Healthy":
		return "\u2665 Healthy", tcell.ColorGreen
	case "Progressing":
		return "\u21bb Progressing", tcell.ColorDodgerBlue
	case "Degraded":
		return "\u2717 Degraded", tcell.ColorRed
	case "Suspended":
		return "\u2016 Suspended", tcell.ColorMediumPurple
	case "Missing":
		return "? Missing", tcell.ColorOrange
	case "":
		return "-", tcell.ColorGray
	default:
		return "? " + s, tcell.ColorGray
	}
}

// argoSyncGlyph maps a sync status to an icon-prefixed label and color.
func argoSyncGlyph(s string) (string, tcell.Color) {
	switch strings.TrimSpace(s) {
	case "Synced":
		return "\u2713 Synced", tcell.ColorGreen
	case "OutOfSync":
		return "\u2260 OutOfSync", tcell.ColorOrange
	case "":
		return "-", tcell.ColorGray
	default:
		return "? " + s, tcell.ColorGray
	}
}

func asString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}

	return ""
}
