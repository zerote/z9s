// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package view_test

import (
	"testing"

	"github.com/yourusername/z9s/internal/k9s/client"
	"github.com/yourusername/z9s/internal/k9s/view"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPortForwardNew(t *testing.T) {
	pf := view.NewPortForward(client.PfGVR)

	require.NoError(t, pf.Init(makeCtx(t)))
	assert.Equal(t, "PortForwards", pf.Name())
	assert.Len(t, pf.Hints(), 11)
}
