// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package ui

import (
	"fmt"
	"strings"

	"github.com/yourusername/z9s/internal/k9s/config"
	"github.com/derailed/tview"
)

// LogoSmall KT9s small log (merged k9s + ktop).
var LogoSmall = []string{
	`__________________       `,
	`\____    /   __   \______`,
	`  /     /\____    /  ___/`,
	` /     /_   /    /\___ \ `,
	`/_______ \ /____//____  >`,
	`        \/            \/ `,
}

// LogoBig KT9s big logo for splash page (merged k9s + ktop).
var LogoBig = []string{
	`__________________         _______  ____     ___ `,
	`\____    /   __   \______/   ___ \|    |   |   |`,
	`  /     /\____    /  ___/    \  \/|    |   |   |`,
	` /     /_   /    /\___ \      \___|    |___|   |`,
	`/_______ \ /____//____  >\______  /_______ \___|`,
	`        \/            \/        \/        \/    `,
}

// Splash represents a splash screen.
type Splash struct {
	*tview.Flex
}

// NewSplash instantiates a new splash screen with product and company info.
func NewSplash(styles *config.Styles, version string) *Splash {
	s := Splash{Flex: tview.NewFlex()}
	s.SetBackgroundColor(styles.BgColor())

	logo := tview.NewTextView()
	logo.SetDynamicColors(true)
	logo.SetTextAlign(tview.AlignCenter)
	s.layoutLogo(logo, styles)

	vers := tview.NewTextView()
	vers.SetDynamicColors(true)
	vers.SetTextAlign(tview.AlignCenter)
	s.layoutRev(vers, version, styles)

	s.SetDirection(tview.FlexRow)
	s.AddItem(logo, 10, 1, false)
	s.AddItem(vers, 1, 1, false)

	return &s
}

func (*Splash) layoutLogo(t *tview.TextView, styles *config.Styles) {
	logo := strings.Join(LogoBig, fmt.Sprintf("\n[%s::b]", styles.Body().LogoColor))
	_, _ = fmt.Fprintf(t, "%s[%s::b]%s\n",
		strings.Repeat("\n", 2),
		styles.Body().LogoColor,
		logo)
}

func (*Splash) layoutRev(t *tview.TextView, rev string, styles *config.Styles) {
	_, _ = fmt.Fprintf(t, "[%s::b]Revision [red::b]%s", styles.Body().FgColor, rev)
}
