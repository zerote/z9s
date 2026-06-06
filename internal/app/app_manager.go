// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package app

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gdamore/tcell/v2"
)

// ModeApp interface that both k9s and ktop apps must implement
type ModeApp interface {
	// Init initializes the mode
	Init() error

	// Start starts the mode (runs its main loop)
	Start() error

	// Stop stops the mode gracefully
	Stop() error

	// HandleEvent processes keyboard/mouse events
	HandleEvent(ev tcell.Event) bool

	// Pause pauses the mode (for switching)
	Pause() error

	// Resume resumes the mode (after switching back)
	Resume() error

	// Name returns the mode name
	Name() string

	// Draw renders the UI
	Draw() error
}

// AppManager orchestrates both modes and handles switching
type AppManager struct {
	// Modes
	k9sApp    ModeApp
	ktopApp   ModeApp
	modes     map[int]ModeApp
	activeMode int

	// Context
	ctx *ModeContext

	// Screen
	screen tcell.Screen

	// Input handler
	inputHandler *InputHandler

	// Logger
	logger *slog.Logger

	// State
	running  bool
	quitting bool

	// Event channel
	eventChan chan tcell.Event
}

// NewAppManager creates a new app manager
func NewAppManager(
	k9sApp ModeApp,
	ktopApp ModeApp,
	ctx *ModeContext,
	logger *slog.Logger,
) (*AppManager, error) {
	// Initialize screen
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("failed to create screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize screen: %w", err)
	}

	am := &AppManager{
		k9sApp:     k9sApp,
		ktopApp:    ktopApp,
		modes:      make(map[int]ModeApp),
		activeMode: ModeK9s, // Default to k9s
		ctx:        ctx,
		screen:     screen,
		logger:     logger,
		running:    false,
		quitting:   false,
		eventChan:  make(chan tcell.Event, 10),
	}

	// Register modes
	am.modes[ModeK9s] = k9sApp
	am.modes[ModeKtop] = ktopApp

	// Create input handler
	am.inputHandler = NewInputHandler(ctx, logger)

	// Set context
	ctx.CurrentMode = ModeK9s

	return am, nil
}

// Init initializes the app manager and both modes
func (am *AppManager) Init() error {
	am.logger.Info("Initializing AppManager",
		"k9s_app", am.k9sApp.Name(),
		"ktop_app", am.ktopApp.Name(),
	)

	// Initialize both modes
	if err := am.k9sApp.Init(); err != nil {
		am.logger.Error("Failed to initialize k9s mode", "error", err)
		return fmt.Errorf("k9s init failed: %w", err)
	}

	if err := am.ktopApp.Init(); err != nil {
		am.logger.Error("Failed to initialize ktop mode", "error", err)
		return fmt.Errorf("ktop init failed: %w", err)
	}

	am.logger.Info("AppManager initialized successfully")
	return nil
}

// Run starts the main application loop
func (am *AppManager) Run() error {
	defer am.Cleanup()

	am.running = true
	am.logger.Info("Starting main loop", "initial_mode", am.GetCurrentModeName())

	// Start the active mode
	if err := am.modes[am.activeMode].Start(); err != nil {
		am.logger.Error("Failed to start initial mode", "error", err)
		return err
	}

	// Start event polling goroutine
	go am.pollEvents()

	// Main loop
	return am.mainLoop()
}

// mainLoop is the core event loop
func (am *AppManager) mainLoop() error {
	ticker := time.NewTicker(50 * time.Millisecond) // ~20 FPS
	defer ticker.Stop()

	for am.running && !am.quitting {
		select {
		case ev := <-am.eventChan:
			if ev == nil {
				am.running = false
				break
			}

			// Check for quit
			if keyEv, ok := ev.(*tcell.EventKey); ok {
				if keyEv.Key() == tcell.KeyCtrlC || (keyEv.Key() == tcell.KeyRune && keyEv.Rune() == 'q') {
					am.logger.Info("Quit signal received")
					am.running = false
					break
				}
			}

			// Handle toggle key
			if keyEv, ok := ev.(*tcell.EventKey); ok {
				if am.inputHandler.IsToggleKey(keyEv) {
					if err := am.SwitchMode(); err != nil {
						am.logger.Error("Failed to switch mode", "error", err)
					}
					continue
				}
			}

			// Forward to current mode
			am.modes[am.activeMode].HandleEvent(ev)

		case <-ticker.C:
			// Render current mode
			if err := am.modes[am.activeMode].Draw(); err != nil {
				am.logger.Error("Draw error", "mode", am.GetCurrentModeName(), "error", err)
			}
		}
	}

	return nil
}

// pollEvents polls for screen events and sends them to the event channel
func (am *AppManager) pollEvents() {
	for am.running {
		ev := am.screen.PollEvent()
		if ev == nil {
			break
		}
		select {
		case am.eventChan <- ev:
		default:
			// Channel full, drop event
		}
	}
}

// SwitchMode switches between k9s and ktop modes
func (am *AppManager) SwitchMode() error {
	oldMode := am.activeMode
	oldModeName := am.GetCurrentModeName()

	// Pause current mode
	am.logger.Info("Pausing mode", "mode", oldModeName)
	if err := am.modes[oldMode].Pause(); err != nil {
		am.logger.Error("Failed to pause mode", "mode", oldModeName, "error", err)
		return err
	}

	// Switch mode
	am.activeMode = 1 - am.activeMode
	am.ctx.CurrentMode = am.activeMode

	newModeName := am.GetCurrentModeName()
	am.logger.Info("Mode switched", "from", oldModeName, "to", newModeName)

	// Resume new mode
	am.logger.Info("Resuming mode", "mode", newModeName)
	if err := am.modes[am.activeMode].Resume(); err != nil {
		am.logger.Error("Failed to resume mode", "mode", newModeName, "error", err)
		// Try to switch back
		am.activeMode = oldMode
		am.ctx.CurrentMode = oldMode
		_ = am.modes[oldMode].Resume()
		return err
	}

	// Redraw
	return am.modes[am.activeMode].Draw()
}

// GetCurrentModeName returns the name of the current mode
func (am *AppManager) GetCurrentModeName() string {
	return am.modes[am.activeMode].Name()
}

// GetCurrentMode returns the current mode app
func (am *AppManager) GetCurrentMode() ModeApp {
	return am.modes[am.activeMode]
}

// IsRunning returns true if the app is running
func (am *AppManager) IsRunning() bool {
	return am.running && !am.quitting
}

// Quit signals the app to quit
func (am *AppManager) Quit() {
	am.quitting = true
	am.running = false
}

// Cleanup cleans up resources
func (am *AppManager) Cleanup() error {
	am.logger.Info("Cleaning up AppManager")

	// Stop both modes
	if err := am.k9sApp.Stop(); err != nil {
		am.logger.Error("Error stopping k9s mode", "error", err)
	}

	if err := am.ktopApp.Stop(); err != nil {
		am.logger.Error("Error stopping ktop mode", "error", err)
	}

	// Close screen
	if am.screen != nil {
		am.screen.Fini()
	}

	// Close event channel
	close(am.eventChan)

	am.logger.Info("Cleanup complete")
	return nil
}

// GetScreen returns the tcell screen
func (am *AppManager) GetScreen() tcell.Screen {
	return am.screen
}

// GetLogger returns the logger
func (am *AppManager) GetLogger() *slog.Logger {
	return am.logger
}
