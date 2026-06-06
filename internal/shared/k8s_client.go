// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package shared

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// SharedK8sClient provides a singleton K8s client for both modes
type SharedK8sClient struct {
	mu sync.RWMutex

	// REST config
	restConfig *rest.Config

	// Standard K8s clientset
	clientset kubernetes.Interface

	// Metrics clientset
	metricsClientset metricsv.Interface

	// Connection state
	connected bool

	// Logger
	logger *slog.Logger

	// Current context
	currentContext string
}

var (
	instance *SharedK8sClient
	once     sync.Once
)

// GetInstance returns the singleton instance of SharedK8sClient
func GetInstance(logger *slog.Logger) *SharedK8sClient {
	once.Do(func() {
		instance = &SharedK8sClient{
			logger:    logger,
			connected: false,
		}
	})
	return instance
}

// Initialize initializes the K8s client
func (c *SharedK8sClient) Initialize(kubeconfig string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("Initializing shared K8s client")

	// Load kubeconfig
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// Try to load from KUBECONFIG env var or default paths
		config, err = clientcmd.BuildConfigFromFlags("", "")
	}

	if err != nil {
		c.logger.Error("Failed to load kubeconfig", "error", err)
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	c.restConfig = config

	// Create K8s clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		c.logger.Error("Failed to create K8s clientset", "error", err)
		return fmt.Errorf("failed to create clientset: %w", err)
	}
	c.clientset = clientset

	// Create metrics clientset
	metricsClientset, err := metricsv.NewForConfig(config)
	if err != nil {
		c.logger.Warn("Failed to create metrics clientset (optional)", "error", err)
		// Don't fail here - metrics are optional
	}
	c.metricsClientset = metricsClientset

	// Get current context
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err == nil {
		c.currentContext = kubeConfig.ClientConfig.CurrentContext
	}

	c.connected = true
	c.logger.Info("Shared K8s client initialized successfully", "context", c.currentContext)

	return nil
}

// GetRestConfig returns the REST config
func (c *SharedK8sClient) GetRestConfig() *rest.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.restConfig
}

// GetClientset returns the K8s clientset
func (c *SharedK8sClient) GetClientset() kubernetes.Interface {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.clientset
}

// GetMetricsClientset returns the metrics clientset
func (c *SharedK8sClient) GetMetricsClientset() metricsv.Interface {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.metricsClientset
}

// IsConnected returns true if connected to K8s
func (c *SharedK8sClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// CheckConnectivity checks if we can connect to the K8s cluster
func (c *SharedK8sClient) CheckConnectivity() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.clientset == nil {
		return false
	}

	// Try to get server version
	_, err := c.clientset.Discovery().ServerVersion()
	return err == nil
}

// GetNodes retrieves all nodes from the cluster
func (c *SharedK8sClient) GetNodes() (*v1.NodeList, error) {
	c.mu.RLock()
	clientset := c.clientset
	c.mu.RUnlock()

	if clientset == nil {
		return nil, fmt.Errorf("not connected to cluster")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		c.logger.Error("Failed to get nodes", "error", err)
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	return nodes, nil
}

// SwitchContext switches to a different Kubernetes context
func (c *SharedK8sClient) SwitchContext(contextName string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("Switching context", "context", contextName)

	// Use kubectl to switch context
	cmd := exec.Command("kubectl", "config", "use-context", contextName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		c.logger.Error("Failed to switch context", "context", contextName, "error", err)
		return fmt.Errorf("failed to switch context: %w", err)
	}

	c.currentContext = contextName

	// Reinitialize the client with the new context
	return c.reinitializeWithNewContext()
}

// reinitializeWithNewContext reinitializes the client after context switch
func (c *SharedK8sClient) reinitializeWithNewContext() error {
	// Load kubeconfig from default paths
	config, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		c.logger.Error("Failed to load kubeconfig after switch", "error", err)
		return err
	}

	c.restConfig = config

	// Create new clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		c.logger.Error("Failed to create clientset after switch", "error", err)
		return err
	}
	c.clientset = clientset

	// Create new metrics clientset
	metricsClientset, err := metricsv.NewForConfig(config)
	if err != nil {
		c.logger.Warn("Failed to create metrics clientset after switch (optional)", "error", err)
	}
	c.metricsClientset = metricsClientset

	c.logger.Info("Client reinitialized with new context", "context", c.currentContext)
	return nil
}

// GetCurrentContext returns the current context
func (c *SharedK8sClient) GetCurrentContext() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentContext
}

// Close closes the connection
func (c *SharedK8sClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.logger.Info("Closing shared K8s client")
	c.connected = false

	return nil
}

// Reconnect reconnects to the K8s cluster
func (c *SharedK8sClient) Reconnect(kubeconfig string) error {
	if err := c.Close(); err != nil {
		c.logger.Error("Error closing client", "error", err)
	}

	return c.Initialize(kubeconfig)
}
