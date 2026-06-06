// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package app

import (
	"fmt"
	"log/slog"

	"github.com/gdamore/tcell/v2"
	ktopapp "github.com/yourusername/z9s/internal/ktop/application"
)

// KtopMode wraps the ktop application.App to implement ModeApp interface
type KtopMode struct {
	app    *ktopapp.App
	logger *slog.Logger
	paused bool
	name   string
}

// NewKtopMode creates a new ktop mode
func NewKtopMode(app *ktopapp.App, logger *slog.Logger) *KtopMode {
	return &KtopMode{
		app:    app,
		logger: logger,
		paused: false,
		name:   "ktop",
	}
}

// Init initializes the ktop mode
func (k *KtopMode) Init() error {
	k.logger.Info("Initializing ktop mode")

	// The app should already be initialized by the caller
	if k.app == nil {
		return fmt.Errorf("ktop app is nil")
	}

	return nil
}

// Start starts the ktop mode main loop
func (k *KtopMode) Start() error {
	k.logger.Info("Starting ktop mode")

	// ktop should handle its own run loop
	k.paused = false

	return nil
}

// Stop stops the ktop mode
func (k *KtopMode) Stop() error {
	k.logger.Info("Stopping ktop mode")

	// Try to clean up ktop gracefully
	if k.app != nil {
		// ktop cleanup would go here
	}

	return nil
}

// HandleEvent processes a keyboard or mouse event in ktop mode
func (k *KtopMode) HandleEvent(ev tcell.Event) bool {
	if k.paused || k.app == nil {
		return false
	}

	// Forward to ktop app
	// This depends on how ktop handles events internally

	return false // Event was not handled by this stub
}

// Pause pauses the ktop mode
func (k *KtopMode) Pause() error {
	k.logger.Info("Pausing ktop mode")
	k.paused = true

	// Save ktop state if needed

	return nil
}

// Resume resumes the ktop mode
func (k *KtopMode) Resume() error {
	k.logger.Info("Resuming ktop mode")
	k.paused = false

	// Restore ktop state if needed

	return nil
}

// Name returns the name of this mode
func (k *KtopMode) Name() string {
	return k.name
}

// Draw renders the ktop UI
func (k *KtopMode) Draw() error {
	if k.paused || k.app == nil {
		return nil
	}

	// ktop would render itself
	// This depends on ktop's internal rendering system

	return nil
}

// GetApp returns the underlying ktop app
func (k *KtopMode) GetApp() *ktopapp.App {
	return k.app
}
