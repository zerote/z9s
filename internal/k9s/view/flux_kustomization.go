// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package view

import "github.com/yourusername/z9s/internal/k9s/client"

// fluxKustomizationGVR is the GVR for Flux Kustomization custom resources.
const fluxKustomizationGVR = "kustomize.toolkit.fluxcd.io/v1/kustomizations"

// FluxKustomization represents a Flux Kustomization browser augmented with
// GitOps actions (reconcile / suspend / resume) on top of the standard k9s
// browser (which already provides edit and delete).
type FluxKustomization struct {
	ResourceViewer
}

// NewFluxKustomization returns a new Flux Kustomization viewer.
func NewFluxKustomization(gvr *client.GVR) ResourceViewer {
	k := FluxKustomization{
		ResourceViewer: NewFluxExtender(NewBrowser(gvr)),
	}

	return &k
}
