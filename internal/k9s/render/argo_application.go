// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package render

import (
	"fmt"
	"strings"

	"github.com/yourusername/z9s/internal/k9s/client"
	"github.com/yourusername/z9s/internal/k9s/model1"
	"github.com/derailed/tcell/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var defaultArgoApplicationHeader = model1.Header{
	model1.HeaderColumn{Name: "NAME"},
	model1.HeaderColumn{Name: "PROJECT"},
	model1.HeaderColumn{Name: "SOURCE"},
	model1.HeaderColumn{Name: "DESTINATION"},
	model1.HeaderColumn{Name: "REVISION"},
	model1.HeaderColumn{Name: "SYNC"},
	model1.HeaderColumn{Name: "HEALTH"},
	model1.HeaderColumn{Name: "AGE", Attrs: model1.Attrs{Time: true}},
}

// ArgoApplication renders an ArgoCD Application to screen.
type ArgoApplication struct {
	Base
}

// Header returns a header row.
func (a ArgoApplication) Header(_ string) model1.Header {
	return a.doHeader(defaultArgoApplicationHeader)
}

// Render renders an ArgoCD Application to screen.
func (a ArgoApplication) Render(o any, _ string, row *model1.Row) error {
	raw, ok := o.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("expected Unstructured, but got %T", o)
	}
	if err := a.defaultRow(raw, row); err != nil {
		return err
	}
	if a.specs.isEmpty() {
		return nil
	}
	cols, err := a.specs.realize(raw, defaultArgoApplicationHeader, row)
	if err != nil {
		return err
	}
	cols.hydrateRow(row)

	return nil
}

func (ArgoApplication) defaultRow(raw *unstructured.Unstructured, r *model1.Row) error {
	project, _, _ := unstructured.NestedString(raw.Object, "spec", "project")
	sync, _, _ := unstructured.NestedString(raw.Object, "status", "sync", "status")
	health, _, _ := unstructured.NestedString(raw.Object, "status", "health", "status")
	destNS := argoDestination(raw)
	repo := argoSource(raw)
	revision := argoRevision(raw)

	r.ID = client.FQN(raw.GetNamespace(), raw.GetName())
	r.Fields = model1.Fields{
		raw.GetName(),
		emptyDash(project),
		emptyDash(repo),
		emptyDash(destNS),
		emptyDash(revision),
		argoSyncCell(sync),
		argoHealthCell(health),
		ToAge(raw.GetCreationTimestamp()),
	}

	return nil
}

// ColorerFunc keeps rows neutral so the per-cell colored SYNC/HEALTH icons
// stand out, mirroring the ArgoCD web UI look.
func (ArgoApplication) ColorerFunc() model1.ColorerFunc {
	return func(string, model1.Header, *model1.RowEvent) tcell.Color {
		return model1.StdColor
	}
}

// argoHealthCell returns a colored, icon-prefixed Health status cell.
func argoHealthCell(status string) string {
	switch strings.TrimSpace(status) {
	case "Healthy":
		return "[green::b]\u2665 Healthy[::-]"
	case "Progressing":
		return "[dodgerblue::b]\u21bb Progressing[::-]"
	case "Degraded":
		return "[red::b]\u2717 Degraded[::-]"
	case "Suspended":
		return "[mediumpurple::b]\u2016 Suspended[::-]"
	case "Missing":
		return "[orange::b]? Missing[::-]"
	case "":
		return "[gray::b]? Unknown[::-]"
	default:
		return "[gray::b]? " + status + "[::-]"
	}
}

// argoSyncCell returns a colored, icon-prefixed Sync status cell.
func argoSyncCell(status string) string {
	switch strings.TrimSpace(status) {
	case "Synced":
		return "[green::b]\u2713 Synced[::-]"
	case "OutOfSync":
		return "[orange::b]\u2260 OutOfSync[::-]"
	case "":
		return "[gray::b]? Unknown[::-]"
	default:
		return "[gray::b]? " + status + "[::-]"
	}
}

// argoDestination returns the deploy target namespace for an Application.
func argoDestination(raw *unstructured.Unstructured) string {
	ns, _, _ := unstructured.NestedString(raw.Object, "spec", "destination", "namespace")

	return ns
}

// argoSource returns a compact owner/repo for an Application source.
func argoSource(raw *unstructured.Unstructured) string {
	repo, _, _ := unstructured.NestedString(raw.Object, "spec", "source", "repoURL")
	if repo == "" {
		if sources, ok, _ := unstructured.NestedSlice(raw.Object, "spec", "sources"); ok && len(sources) > 0 {
			if m, ok := sources[0].(map[string]any); ok {
				if v, ok := m["repoURL"].(string); ok {
					repo = v
				}
			}
		}
	}

	return shortRepo(repo)
}

// argoRevision returns the human-friendly target revision (branch/tag/HEAD).
func argoRevision(raw *unstructured.Unstructured) string {
	rev, _, _ := unstructured.NestedString(raw.Object, "spec", "source", "targetRevision")
	if rev == "" {
		if sources, ok, _ := unstructured.NestedSlice(raw.Object, "spec", "sources"); ok && len(sources) > 0 {
			if m, ok := sources[0].(map[string]any); ok {
				if v, ok := m["targetRevision"].(string); ok {
					rev = v
				}
			}
		}
	}

	return strings.TrimSpace(rev)
}

func emptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}

	return s
}

// shortRepo trims a git URL down to a compact owner/repo form.
func shortRepo(u string) string {
	u = strings.TrimSpace(u)
	if u == "" {
		return ""
	}
	u = strings.TrimSuffix(u, ".git")
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	u = strings.TrimPrefix(u, "git@")

	parts := strings.Split(u, "/")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], "/")
	}

	return u
}
