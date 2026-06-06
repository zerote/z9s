// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package app

import (
	"fmt"
	"log/slog"

	"github.com/gdamore/tcell/v2"
	k9sview "github.com/yourusername/z9s/internal/k9s/view"
)

// K9sMode wraps the k9s view.App to implement ModeApp interface
type K9sMode struct {
	app    *k9sview.App
	logger *slog.Logger
	paused bool
	name   string
}

// NewK9sMode creates a new k9s mode
func NewK9sMode(app *k9sview.App, logger *slog.Logger) *K9sMode {
	return &K9sMode{
		app:    app,
		logger: logger,
		paused: false,
		name:   "k9s",
	}
}

// Init initializes the k9s mode
func (k *K9sMode) Init() error {
	k.logger.Info("Initializing k9s mode")

	// The app should already be initialized by the caller
	// But we ensure it's ready
	if k.app == nil {
		return fmt.Errorf("k9s app is nil")
	}

	return nil
}

// Start starts the k9s mode main loop
func (k *K9sMode) Start() error {
	k.logger.Info("Starting k9s mode")

	// k9s should handle its own run loop
	// This is called when k9s becomes the active mode
	k.paused = false

	return nil
}

// Stop stops the k9s mode
func (k *K9sMode) Stop() error {
	k.logger.Info("Stopping k9s mode")

	// Try to clean up k9s gracefully
	if k.app != nil {
		// k9s cleanup would go here
	}

	return nil
}

// HandleEvent processes a keyboard or mouse event in k9s mode
func (k *K9sMode) HandleEvent(ev tcell.Event) bool {
	if k.paused || k.app == nil {
		return false
	}

	// Forward to k9s app
	// This depends on how k9s handles events internally
	// We may need to adapt this based on k9s's actual event handling

	return false // Event was not handled by this stub
}

// Pause pauses the k9s mode
func (k *K9sMode) Pause() error {
	k.logger.Info("Pausing k9s mode")
	k.paused = true

	// Save k9s state if needed
	// This could include selected item, scroll position, etc.

	return nil
}

// Resume resumes the k9s mode
func (k *K9sMode) Resume() error {
	k.logger.Info("Resuming k9s mode")
	k.paused = false

	// Restore k9s state if needed

	return nil
}

// Name returns the name of this mode
func (k *K9sMode) Name() string {
	return k.name
}

// Draw renders the k9s UI
func (k *K9sMode) Draw() error {
	if k.paused || k.app == nil {
		return nil
	}

	// k9s would render itself
	// This depends on k9s's internal rendering system

	return nil
}

// GetApp returns the underlying k9s app
func (k *K9sMode) GetApp() *k9sview.App {
	return k.app
}
