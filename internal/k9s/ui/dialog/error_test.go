// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package dialog

import (
	"testing"

	"github.com/yourusername/z9s/internal/k9s/config"
	"github.com/yourusername/z9s/internal/k9s/ui"
	"github.com/derailed/tview"
	"github.com/stretchr/testify/assert"
)

func TestErrorDialog(t *testing.T) {
	p := ui.NewPages()

	ShowError(new(config.Dialog), p, "Yo")

	d := p.GetPrimitive(dialogKey).(*tview.ModalForm)
	assert.NotNil(t, d)
	dismiss(p)
	assert.Nil(t, p.GetPrimitive(dialogKey))
}
