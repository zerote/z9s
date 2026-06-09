// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package view

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/yourusername/z9s/internal/k9s/client"
	"github.com/yourusername/z9s/internal/k9s/ui"
	"github.com/yourusername/z9s/internal/k9s/ui/dialog"
	"github.com/derailed/tcell/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// ArgoExtender adds ArgoCD GitOps actions (sync, refresh) to a resource viewer.
// Sync is triggered the same way `argocd app sync` does under the hood: by
// setting the Application's `operation` field, which the application-controller
// then executes. Refresh re-polls git via the refresh annotation.
type ArgoExtender struct {
	ResourceViewer
}

// NewArgoExtender returns a new ArgoCD extender.
func NewArgoExtender(v ResourceViewer) ResourceViewer {
	a := ArgoExtender{ResourceViewer: v}
	v.AddBindKeysFn(a.bindKeys)

	return &a
}

func (a *ArgoExtender) bindKeys(aa *ui.KeyActions) {
	if a.App().Config.IsReadOnly() {
		return
	}
	aa.Bulk(ui.KeyMap{
		ui.KeyS: ui.NewKeyAction("Sync", a.syncCmd, true),
		ui.KeyR: ui.NewKeyAction("Refresh", a.refreshCmd, true),
	})
}

func (a *ArgoExtender) syncCmd(evt *tcell.EventKey) *tcell.EventKey {
	path := a.GetTable().GetSelectedItem()
	if path == "" {
		return evt
	}
	d := a.App().Styles.Dialog()
	dialog.ShowConfirm(&d, a.App().Content.Pages, "Confirm Sync",
		fmt.Sprintf("Sync application %s?", path),
		func() {
			if err := a.triggerSync(path); err != nil {
				a.App().Flash().Err(err)
				return
			}
			a.App().Flash().Infof("Sync triggered for %s", path)
		},
		func() {},
	)

	return nil
}

func (a *ArgoExtender) refreshCmd(evt *tcell.EventKey) *tcell.EventKey {
	path := a.GetTable().GetSelectedItem()
	if path == "" {
		return evt
	}
	patch := []byte(`{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"normal"}}}`)
	if err := a.patch(path, patch); err != nil {
		a.App().Flash().Err(err)
		return nil
	}
	a.App().Flash().Infof("Refresh requested for %s", path)

	return nil
}

func (a *ArgoExtender) triggerSync(path string) error {
	body := map[string]any{
		"operation": map[string]any{
			"initiatedBy": map[string]any{"username": "z9s"},
			"info":        []any{map[string]any{"name": "Reason", "value": "Triggered by z9s"}},
			"sync":        map[string]any{"syncStrategy": map[string]any{"hook": map[string]any{}}},
		},
	}
	patch, err := json.Marshal(body)
	if err != nil {
		return err
	}

	return a.patch(path, patch)
}

func (a *ArgoExtender) patch(path string, patch []byte) error {
	ns, name := client.Namespaced(path)
	dyn, err := a.App().Conn().DynDial()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), a.App().Conn().Config().CallTimeout())
	defer cancel()

	_, err = dyn.Resource(a.GVR().GVR()).Namespace(ns).Patch(
		ctx, name, types.MergePatchType, patch, metav1.PatchOptions{FieldManager: "z9s"},
	)

	return err
}
