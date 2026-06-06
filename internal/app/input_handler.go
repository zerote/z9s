// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package app

import (
	"log/slog"

	"github.com/gdamore/tcell/v2"
)

// InputHandler manages keyboard input and mode switching
type InputHandler struct {
	modeContext *ModeContext
	logger      *slog.Logger
}

// NewInputHandler creates a new input handler
func NewInputHandler(ctx *ModeContext, logger *slog.Logger) *InputHandler {
	return &InputHandler{
		modeContext: ctx,
		logger:      logger,
	}
}

// IsToggleKey checks if the event is Ctrl+F10
func (h *InputHandler) IsToggleKey(ev *tcell.EventKey) bool {
	if ev == nil {
		return false
	}

	// Check for Ctrl+F10
	// In tcell, F10 = tcell.KeyF10
	// Modifiers are checked with ev.Modifiers()
	if ev.Key() == tcell.KeyF10 {
		modifiers := ev.Modifiers()
		// Check if Ctrl is pressed (bit 1)
		if modifiers&tcell.ModCtrl != 0 {
			return true
		}
	}

	return false
}

// HandleKeyEvent processes a key event
// Returns true if the event was handled (and shouldn't be forwarded to mode)
func (h *InputHandler) HandleKeyEvent(ev *tcell.EventKey) bool {
	if h.IsToggleKey(ev) {
		h.logger.Info("Toggle key pressed (Ctrl+F10)", "current_mode", h.modeContext.GetCurrentModeName())

		if err := h.modeContext.Toggle(); err != nil {
			h.logger.Error("Failed to toggle mode", "error", err)
			return true
		}

		h.logger.Info("Mode switched", "new_mode", h.modeContext.GetCurrentModeName())
		return true // Handled
	}

	// Check for quit key (q or Ctrl+C)
	if ev.Key() == tcell.KeyRune && ev.Rune() == 'q' {
		h.logger.Info("Quit key pressed")
		return false // Let the app handle it
	}

	if ev.Key() == tcell.KeyCtrlC {
		h.logger.Info("Ctrl+C pressed")
		return false // Let the app handle it
	}

	// Not handled here, forward to mode
	return false
}

// KeyBindings returns a description of available key bindings
func (h *InputHandler) KeyBindings() string {
	return `
Key Bindings:
  Ctrl+F10  - Toggle between k9s and ktop modes
  q         - Quit application
  Ctrl+C    - Force quit
`
}
