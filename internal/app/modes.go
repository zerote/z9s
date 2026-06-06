// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package app

import (
	"fmt"
)

const (
	// ModeK9s represents the k9s mode
	ModeK9s int = iota
	// ModeKtop represents the ktop mode
	ModeKtop
)

var modeNames = map[int]string{
	ModeK9s:  "k9s",
	ModeKtop: "ktop",
}

// GetModeName returns the name of the mode
func GetModeName(mode int) string {
	if name, ok := modeNames[mode]; ok {
		return name
	}
	return "unknown"
}

// Mode represents a single mode (k9s or ktop)
type Mode struct {
	ID          int
	Name        string
	Description string
	// State will hold mode-specific state
	// We'll expand this as implementation progresses
}

// ModeContext contains shared context for modes
type ModeContext struct {
	// Shared resources
	K8sConfig interface{} // Will be actual K8s config

	// Logger interface
	Logger interface{}

	// Screen and input handling
	CurrentMode int

	// Flags and options
	RefreshRate float32
}

// Toggle switches between modes
func (ctx *ModeContext) Toggle() error {
	oldMode := ctx.CurrentMode
	ctx.CurrentMode = 1 - ctx.CurrentMode

	fmt.Printf("Mode switched from %s to %s\n",
		GetModeName(oldMode),
		GetModeName(ctx.CurrentMode),
	)

	return nil
}

// GetCurrentModeName returns the name of the current mode
func (ctx *ModeContext) GetCurrentModeName() string {
	return GetModeName(ctx.CurrentMode)
}

// Available modes
var AvailableModes = []Mode{
	{
		ID:          ModeK9s,
		Name:        "k9s",
		Description: "Full Kubernetes cluster management",
	},
	{
		ID:          ModeKtop,
		Name:        "ktop",
		Description: "Kubernetes resource metrics visualization",
	},
}
