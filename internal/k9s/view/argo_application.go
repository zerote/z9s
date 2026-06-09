// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package view

import (
	"github.com/yourusername/z9s/internal/k9s/client"
	"github.com/yourusername/z9s/internal/k9s/ui"
)

// argoApplicationGVR is the GVR for ArgoCD Application custom resources.
const argoApplicationGVR = "argoproj.io/v1alpha1/applications"

// ArgoApplication represents an ArgoCD Application browser. Pressing <enter>
// drills into the Application's managed resources (its details tree), and the
// ArgoExtender adds GitOps actions (sync, refresh).
type ArgoApplication struct {
	ResourceViewer
}

// NewArgoApplication returns a new ArgoCD Application viewer.
func NewArgoApplication(gvr *client.GVR) ResourceViewer {
	a := ArgoApplication{
		ResourceViewer: NewArgoExtender(NewBrowser(gvr)),
	}
	a.GetTable().SetEnterFn(a.showResources)

	return &a
}

// showResources opens the managed-resources view for the selected Application.
func (a *ArgoApplication) showResources(app *App, _ ui.Tabular, _ *client.GVR, path string) {
	ns, name := client.Namespaced(path)
	if err := app.inject(NewArgoResources(app, ns, name), false); err != nil {
		app.Flash().Err(err)
	}
}
