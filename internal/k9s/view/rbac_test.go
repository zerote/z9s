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

func TestRbacNew(t *testing.T) {
	v := view.NewRbac(client.RbacGVR)

	require.NoError(t, v.Init(makeCtx(t)))
	assert.Equal(t, "Rbac", v.Name())
	assert.Len(t, v.Hints(), 6)
}
