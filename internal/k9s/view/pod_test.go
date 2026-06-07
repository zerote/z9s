// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package view_test

import (
	"context"
	"testing"

	"github.com/yourusername/z9s/internal/k9s"
	"github.com/yourusername/z9s/internal/k9s/client"
	"github.com/yourusername/z9s/internal/k9s/config/mock"
	"github.com/yourusername/z9s/internal/k9s/view"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPodNew(t *testing.T) {
	po := view.NewPod(client.PodGVR)

	require.NoError(t, po.Init(makeCtx(t)))
	assert.Equal(t, "Pods", po.Name())
	assert.Len(t, po.Hints(), 19)
}

// Helpers...

func makeCtx(t testing.TB) context.Context {
	cfg := mock.NewMockConfig(t)
	return context.WithValue(context.Background(), internal.KeyApp, view.NewApp(cfg))
}
