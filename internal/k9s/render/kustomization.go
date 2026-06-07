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

var defaultKustomizationHeader = model1.Header{
	model1.HeaderColumn{Name: "NAME"},
	model1.HeaderColumn{Name: "NAMESPACE"},
	model1.HeaderColumn{Name: "STATUS"},
	model1.HeaderColumn{Name: "READY MESSAGE"},
	model1.HeaderColumn{Name: "AGE", Attrs: model1.Attrs{Time: true}},
}

// Kustomization renders a Flux Kustomization to screen.
type Kustomization struct {
	Base
}

// Header returns a header row.
func (k Kustomization) Header(_ string) model1.Header {
	return k.doHeader(defaultKustomizationHeader)
}

// Render renders a Flux Kustomization to screen.
func (k Kustomization) Render(o any, _ string, row *model1.Row) error {
	raw, ok := o.(*unstructured.Unstructured)
	if !ok {
		return fmt.Errorf("expected Unstructured, but got %T", o)
	}
	if err := k.defaultRow(raw, row); err != nil {
		return err
	}
	if k.specs.isEmpty() {
		return nil
	}
	cols, err := k.specs.realize(raw, defaultKustomizationHeader, row)
	if err != nil {
		return err
	}
	cols.hydrateRow(row)

	return nil
}

func (Kustomization) defaultRow(raw *unstructured.Unstructured, r *model1.Row) error {
	status, message := fluxReadyState(raw)

	r.ID = client.FQN(raw.GetNamespace(), raw.GetName())
	r.Fields = model1.Fields{
		raw.GetName(),
		raw.GetNamespace(),
		status,
		message,
		ToAge(raw.GetCreationTimestamp()),
	}

	return nil
}

// ColorerFunc colors a Kustomization row based on its STATUS.
func (Kustomization) ColorerFunc() model1.ColorerFunc {
	return func(ns string, h model1.Header, re *model1.RowEvent) tcell.Color {
		idx, ok := h.IndexOf("STATUS", true)
		if !ok {
			return model1.DefaultColorer(ns, h, re)
		}
		switch strings.TrimSpace(re.Row.Fields[idx]) {
		case "Suspended":
			return model1.PendingColor
		case "Ready":
			return model1.StdColor
		case "Reconciling", "Unknown":
			return model1.AddColor
		default:
			return model1.ErrColor
		}
	}
}

// fluxReadyState derives a Lens-like status plus the Ready condition message
// from a Flux resource (suspend flag + status.conditions[type=Ready]).
func fluxReadyState(raw *unstructured.Unstructured) (status, message string) {
	if suspended, _, _ := unstructured.NestedBool(raw.Object, "spec", "suspend"); suspended {
		return "Suspended", "Suspended"
	}

	conds, _, _ := unstructured.NestedSlice(raw.Object, "status", "conditions")
	var readyStatus, readyReason string
	for _, c := range conds {
		cm, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if t, _ := cm["type"].(string); t != "Ready" {
			continue
		}
		readyStatus, _ = cm["status"].(string)
		readyReason, _ = cm["reason"].(string)
		message, _ = cm["message"].(string)
		break
	}

	switch readyStatus {
	case "True":
		return "Ready", message
	case "False":
		if readyReason == "" {
			return "Failed", message
		}
		return readyReason, message
	default:
		return "Unknown", message
	}
}
