// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package view

import (
	"context"
	"fmt"
	"strings"

	"github.com/yourusername/z9s/internal/k9s/client"
	"github.com/yourusername/z9s/internal/k9s/config"
	"github.com/yourusername/z9s/internal/k9s/model"
	"github.com/yourusername/z9s/internal/k9s/ui"
	"github.com/yourusername/z9s/internal/k9s/view/cmd"
	"github.com/derailed/tcell/v2"
	"github.com/derailed/tview"
	"k8s.io/apimachinery/pkg/labels"
)

// menuPage is a lightweight, keyboard-driven landing page used for the operators
// navigation (Operators -> FluxCD -> ...). It is pushed onto the content stack
// like any other view, so it reuses the shared k9s header and menu.
type menuPage struct {
	*tview.Flex

	text    *tview.TextView
	app     *App
	title   string
	actions *ui.KeyActions
}

func newMenuPage(app *App, title string) *menuPage {
	p := menuPage{
		Flex:    tview.NewFlex(),
		text:    tview.NewTextView(),
		app:     app,
		title:   title,
		actions: ui.NewKeyActions(),
	}
	p.text.SetDynamicColors(true)
	p.text.SetWrap(false)
	p.AddItem(p.text, 0, 1, true)

	return &p
}

// Init initializes the view.
func (p *menuPage) Init(context.Context) error {
	p.SetBorder(true)
	p.SetBorderPadding(0, 0, 1, 1)
	p.SetInputCapture(p.keyboard)
	p.updateTitle()
	p.StylesChanged(p.app.Styles)

	return nil
}

func (p *menuPage) keyboard(evt *tcell.EventKey) *tcell.EventKey {
	if a, ok := p.actions.Get(ui.AsKey(evt)); ok {
		return a.Action(evt)
	}

	return evt
}

// StylesChanged notifies the skin changed.
func (p *menuPage) StylesChanged(s *config.Styles) {
	p.SetBackgroundColor(s.BgColor())
	p.text.SetBackgroundColor(s.BgColor())
	p.text.SetTextColor(s.FgColor())
	p.SetTitleColor(s.Frame().Title.FgColor.Color())
	p.SetBorderColor(s.Frame().Border.FgColor.Color())
	p.SetBorderFocusColor(s.Frame().Border.FocusColor.Color())
}

func (p *menuPage) updateTitle() {
	p.SetTitle(fmt.Sprintf(" [aqua::b]%s ", p.title))
}

// Name returns the component name.
func (p *menuPage) Name() string { return p.title }

// Start starts the view.
func (p *menuPage) Start() {
	p.app.Styles.RemoveListener(p)
	p.app.Styles.AddListener(p)
	p.StylesChanged(p.app.Styles)
}

// Stop terminates the view.
func (p *menuPage) Stop() { p.app.Styles.RemoveListener(p) }

// Hints returns menu hints.
func (p *menuPage) Hints() model.MenuHints { return p.actions.Hints() }

// ExtraHints returns additional hints.
func (*menuPage) ExtraHints() map[string]string { return nil }

// InCmdMode checks if prompt is active.
func (*menuPage) InCmdMode() bool { return false }

func (*menuPage) SetFilter(string, bool)                 {}
func (*menuPage) SetLabelSelector(labels.Selector, bool) {}
func (*menuPage) SetCommand(*cmd.Interpreter)            {}

// ----------------------------------------------------------------------------
// Operators landing page.

// NewOperators returns the operators landing page.
func NewOperators(app *App) *menuPage {
	p := newMenuPage(app, "Operators")
	p.actions.Bulk(ui.KeyMap{
		ui.KeyF:         ui.NewKeyAction("FluxCD", app.gotoFlux, true),
		ui.KeyA:         ui.NewKeyAction("ArgoCD", app.gotoArgo, true),
		tcell.KeyEscape: ui.NewKeyAction("Back", app.PrevCmd, false),
		ui.KeyQ:         ui.NewKeyAction("Back", app.PrevCmd, false),
	})
	flux, argo := detectOperators(app.Conn())
	p.text.SetText(operatorsBody(flux, argo))

	return p
}

func operatorsBody(flux, argo bool) string {
	dot := func(on bool) string {
		if on {
			return "[green::b]●[-:-:-]"
		}
		return "[red::b]●[-:-:-]"
	}
	state := func(on bool) string {
		if on {
			return "[green::b]detected[-:-:-]"
		}
		return "[red::b]not found[-:-:-]"
	}

	var b strings.Builder
	fmt.Fprint(&b, "\n  [aqua::b]GitOps Operators[-:-:-]\n\n")
	fmt.Fprintf(&b, "   %s  FluxCD   %s\n", dot(flux), state(flux))
	fmt.Fprintf(&b, "   %s  ArgoCD   %s\n", dot(argo), state(argo))

	return b.String()
}

// ----------------------------------------------------------------------------
// FluxCD overview page.

// NewFluxOverview returns the FluxCD overview page. Enabled commands live in the
// header menu only (<o> Overview, <k> Kustomizations).
func NewFluxOverview(app *App) *menuPage {
	p := newMenuPage(app, "FluxCD")
	p.actions.Bulk(ui.KeyMap{
		ui.KeyK:         ui.NewKeyAction("Kustomizations", app.gotoFluxKustomizations, true),
		tcell.KeyEscape: ui.NewKeyAction("Back", app.PrevCmd, false),
		ui.KeyQ:         ui.NewKeyAction("Back", app.PrevCmd, false),
	})
	p.actions.Add(ui.KeyO, ui.NewKeyAction("Overview", func(*tcell.EventKey) *tcell.EventKey {
		p.text.SetText(fluxOverviewBody())
		return nil
	}, true))
	p.text.SetText(fluxOverviewBody())

	return p
}

func fluxOverviewBody() string {
	var b strings.Builder
	fmt.Fprint(&b, "\n  [aqua::b]FluxCD Overview[-:-:-]\n\n")
	fmt.Fprint(&b, "   GitOps reconciliation status for this cluster.\n\n")
	fmt.Fprint(&b, "   [gray::-](overview metrics coming soon)[-:-:-]\n")

	return b.String()
}

// ----------------------------------------------------------------------------
// App navigation handlers.

// operatorsCmd opens the operators navigation (Ctrl-O). When a single operator
// is installed it jumps straight to that operator's overview.
func (a *App) operatorsCmd(evt *tcell.EventKey) *tcell.EventKey {
	if a.Prompt().InCmdMode() {
		return evt
	}
	flux, argo := detectOperators(a.Conn())
	if flux && !argo {
		return a.gotoFlux(evt)
	}
	if err := a.inject(NewOperators(a), false); err != nil {
		a.Flash().Err(err)
	}

	return nil
}

func (a *App) gotoFlux(*tcell.EventKey) *tcell.EventKey {
	if err := a.inject(NewFluxOverview(a), false); err != nil {
		a.Flash().Err(err)
	}

	return nil
}

func (a *App) gotoArgo(*tcell.EventKey) *tcell.EventKey {
	a.Flash().Info("ArgoCD support is coming soon")

	return nil
}

func (a *App) gotoFluxKustomizations(*tcell.EventKey) *tcell.EventKey {
	v := NewFluxKustomization(client.NewGVR(fluxKustomizationGVR))
	v.SetCommand(cmd.NewInterpreter("kustomizations"))
	if err := a.inject(v, false); err != nil {
		a.Flash().Err(err)
	}

	return nil
}
