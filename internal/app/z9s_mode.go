// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package app

import (
	"fmt"
	"log/slog"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"

	"github.com/yourusername/z9s/internal/shared"
)

// Z9sMode implements a functional z9s interface with context and node viewing
type Z9sMode struct {
	app            *tview.Application
	logger         *slog.Logger
	k8sClient      *shared.SharedK8sClient
	mainView       *tview.Flex
	contextList    *tview.List
	nodeList       *tview.List
	infoPanel      *tview.TextArea
	paused         bool
	name           string
	selectedMode   int // 0 = contexts, 1 = nodes
	kubeconfig     *api.Config
	currentContext string
}

// NewZ9sMode creates a new z9s mode with real functionality
func NewZ9sMode(k8sClient *shared.SharedK8sClient, logger *slog.Logger) *Z9sMode {
	app := tview.NewApplication()

	k := &Z9sMode{
		app:       app,
		logger:    logger,
		k8sClient: k8sClient,
		paused:    false,
		name:      "z9s",
	}

	k.setupUI()
	return k
}

// setupUI initializes the user interface
func (k *Z9sMode) setupUI() {
	// Contexts list on the left
	k.contextList = tview.NewList()
	k.contextList.SetTitle("Contexts").SetBorder(true)
	k.contextList.ShowSecondaryText(false)

	// Nodes list in the middle
	k.nodeList = tview.NewList()
	k.nodeList.SetTitle("Nodes").SetBorder(true)
	k.nodeList.ShowSecondaryText(false)

	// Info panel on the right
	k.infoPanel = tview.NewTextArea()
	k.infoPanel.SetTitle("Info").SetBorder(true)

	// Layout
	leftRight := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(k.contextList, 0, 1, true).
		AddItem(k.nodeList, 0, 1, false).
		AddItem(k.infoPanel, 0, 1, false)

	k.mainView = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(leftRight, 0, 1, true)

	// Load contexts
	k.refreshContexts()

	// Set up event handling
	k.setupEventHandlers()
}

// setupEventHandlers sets up keyboard and mouse handlers
func (k *Z9sMode) setupEventHandlers() {
	k.contextList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEnter:
			// Switch context
			idx := k.contextList.GetCurrentItem()
			if idx >= 0 {
				_, contextName := k.contextList.GetItemText(idx)
				k.switchContext(contextName)
			}
			return nil
		case tcell.KeyTab:
			// Move to nodes list
			k.selectedMode = 1
			k.app.SetFocus(k.nodeList)
			return nil
		}
		return event
	})

	k.nodeList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			// Move to contexts list
			k.selectedMode = 0
			k.app.SetFocus(k.contextList)
			return nil
		}
		return event
	})
}

// refreshContexts loads and displays available Kubernetes contexts
func (k *Z9sMode) refreshContexts() {
	k.contextList.Clear()

	// Load kubeconfig
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{})
	config, err := kubeconfig.RawConfig()
	if err != nil {
		k.logger.Error("Failed to load kubeconfig", "error", err)
		k.contextList.AddItem("Error loading kubeconfig", "", 0, nil)
		return
	}

	k.kubeconfig = &config

	// Get current namespace
	var currentNamespace string
	if ns, _, err := kubeconfig.Namespace(); err == nil {
		currentNamespace = ns
	} else {
		currentNamespace = "default"
	}

	k.logger.Info("Current context", "context", k.currentContext, "namespace", currentNamespace)

	// Add all contexts
	for name := range config.Contexts {
		marker := " "
		if name == k.currentContext {
			marker = "★" // Mark current context
		}
		k.contextList.AddItem(fmt.Sprintf("%s %s", marker, name), "", 0, nil)
	}

	// Load nodes for current context
	k.refreshNodes()
}

// refreshNodes loads and displays nodes for the current context
func (k *Z9sMode) refreshNodes() {
	k.nodeList.Clear()
	k.nodeList.SetTitle(fmt.Sprintf("Nodes [%s]", k.currentContext))

	nodes, err := k.k8sClient.GetNodes()
	if err != nil {
		k.logger.Error("Failed to get nodes", "error", err)
		k.nodeList.AddItem("Error loading nodes", "", 0, nil)
		return
	}

	for _, node := range nodes.Items {
		name := node.Name
		status := "Unknown"
		for _, cond := range node.Status.Conditions {
			if cond.Type == "Ready" {
				if cond.Status == "True" {
					status = "Ready"
				} else {
					status = "NotReady"
				}
				break
			}
		}
		k.nodeList.AddItem(fmt.Sprintf("%s [%s]", name, status), "", 0, nil)
	}

	if k.nodeList.GetItemCount() == 0 {
		k.nodeList.AddItem("No nodes found", "", 0, nil)
	}
}

// switchContext switches to a different Kubernetes context
func (k *Z9sMode) switchContext(contextName string) {
	k.logger.Info("Switching context", "context", contextName)

	// Switch using kubectl
	if err := k.k8sClient.SwitchContext(contextName); err != nil {
		k.logger.Error("Failed to switch context", "context", contextName, "error", err)
		k.infoPanel.SetText(fmt.Sprintf("Error: %v", err), false)
		return
	}

	k.currentContext = contextName
	k.infoPanel.SetText(fmt.Sprintf("Switched to context: %s", contextName), false)
	k.refreshContexts()
}

// Init initializes the z9s mode
func (k *Z9sMode) Init() error {
	k.logger.Info("Initializing z9s mode")
	return nil
}

// Start starts the z9s mode main loop
func (k *Z9sMode) Start() error {
	k.logger.Info("Starting z9s mode")
	k.paused = false

	if err := k.app.SetRoot(k.mainView, true).Run(); err != nil {
		return fmt.Errorf("error running app: %w", err)
	}
	return nil
}

// Stop stops the z9s mode
func (k *Z9sMode) Stop() error {
	k.logger.Info("Stopping z9s mode")
	k.app.Stop()
	return nil
}

// HandleEvent processes keyboard and mouse events
func (k *Z9sMode) HandleEvent(ev tcell.Event) bool {
	if k.paused || k.app == nil {
		return false
	}

	switch event := ev.(type) {
	case *tcell.EventKey:
		// Handle key events
		switch event.Key() {
		case tcell.KeyEscape:
			return true // Signal to switch modes
		}
	}
	return false
}

// Pause pauses the z9s mode
func (k *Z9sMode) Pause() error {
	k.logger.Info("Pausing z9s mode")
	k.paused = true
	return nil
}

// Resume resumes the z9s mode
func (k *Z9sMode) Resume() error {
	k.logger.Info("Resuming z9s mode")
	k.paused = false
	return nil
}

// Name returns the name of this mode
func (k *Z9sMode) Name() string {
	return k.name
}

// Draw renders the z9s UI
func (k *Z9sMode) Draw() error {
	if k.paused || k.app == nil {
		return nil
	}
	// tview handles rendering internally
	return nil
}

