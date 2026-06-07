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

func TestScreenDumpNew(t *testing.T) {
	po := view.NewScreenDump(client.SdGVR)

	require.NoError(t, po.Init(makeCtx(t)))
	assert.Equal(t, "ScreenDumps", po.Name())
	assert.Len(t, po.Hints(), 7)
}
