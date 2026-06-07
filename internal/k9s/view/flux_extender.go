// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package view

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/z9s/internal/k9s/client"
	"github.com/yourusername/z9s/internal/k9s/ui"
	"github.com/yourusername/z9s/internal/k9s/ui/dialog"
	"github.com/derailed/tcell/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// FluxExtender adds Flux GitOps actions (reconcile, suspend, resume) to a
// resource viewer by patching the selected resource through the dynamic client,
// mirroring what `flux reconcile|suspend|resume` does under the hood.
type FluxExtender struct {
	ResourceViewer
}

// NewFluxExtender returns a new Flux extender.
func NewFluxExtender(v ResourceViewer) ResourceViewer {
	f := FluxExtender{ResourceViewer: v}
	v.AddBindKeysFn(f.bindKeys)

	return &f
}

func (f *FluxExtender) bindKeys(aa *ui.KeyActions) {
	if f.App().Config.IsReadOnly() {
		return
	}
	aa.Bulk(ui.KeyMap{
		ui.KeyR: ui.NewKeyAction("Reconcile", f.reconcileCmd, true),
		ui.KeyS: ui.NewKeyActionWithOpts("Suspend", f.suspendCmd,
			ui.ActionOpts{Visible: true, Dangerous: true},
		),
		ui.KeyU: ui.NewKeyAction("Resume", f.resumeCmd, true),
	})
}

func (f *FluxExtender) reconcileCmd(*tcell.EventKey) *tcell.EventKey {
	path := f.GetTable().GetSelectedItem()
	if path == "" {
		return nil
	}
	d := f.App().Styles.Dialog()
	dialog.ShowConfirm(&d, f.App().Content.Pages, "Confirm Reconcile",
		fmt.Sprintf("Reconcile %s %s?", singularize(f.GVR().R()), path),
		func() {
			ts := time.Now().UTC().Format(time.RFC3339Nano)
			patch := []byte(fmt.Sprintf(
				`{"metadata":{"annotations":{"reconcile.fluxcd.io/requestedAt":%q}}}`, ts,
			))
			if err := f.patch(path, patch); err != nil {
				f.App().Flash().Err(err)
				return
			}
			f.App().Flash().Infof("Reconcile requested for %s", path)
		},
		func() {},
	)

	return nil
}

func (f *FluxExtender) suspendCmd(*tcell.EventKey) *tcell.EventKey {
	return f.setSuspend(true)
}

func (f *FluxExtender) resumeCmd(*tcell.EventKey) *tcell.EventKey {
	return f.setSuspend(false)
}

func (f *FluxExtender) setSuspend(suspend bool) *tcell.EventKey {
	path := f.GetTable().GetSelectedItem()
	if path == "" {
		return nil
	}
	verb := "Resume"
	if suspend {
		verb = "Suspend"
	}
	d := f.App().Styles.Dialog()
	dialog.ShowConfirm(&d, f.App().Content.Pages, "Confirm "+verb,
		fmt.Sprintf("%s %s %s?", verb, singularize(f.GVR().R()), path),
		func() {
			patch := []byte(fmt.Sprintf(`{"spec":{"suspend":%t}}`, suspend))
			if err := f.patch(path, patch); err != nil {
				f.App().Flash().Err(err)
				return
			}
			f.App().Flash().Infof("%s applied to %s", verb, path)
		},
		func() {},
	)

	return nil
}

func (f *FluxExtender) patch(path string, patch []byte) error {
	ns, name := client.Namespaced(path)
	dyn, err := f.App().Conn().DynDial()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), f.App().Conn().Config().CallTimeout())
	defer cancel()

	_, err = dyn.Resource(f.GVR().GVR()).Namespace(ns).Patch(
		ctx, name, types.MergePatchType, patch, metav1.PatchOptions{FieldManager: "z9s"},
	)

	return err
}
