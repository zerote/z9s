// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of K9s

package model

import (
	"context"
	"sync"

	"github.com/yourusername/z9s/internal/k9s/client"
	"github.com/yourusername/z9s/internal/k9s/config"
	"github.com/yourusername/z9s/internal/k9s/dao"
)

// ClusterInfoListener registers a listener for model changes.
type ClusterInfoListener interface {
	// ClusterInfoChanged notifies the cluster meta was changed.
	ClusterInfoChanged(prev, curr *ClusterMeta)

	// ClusterInfoUpdated notifies the cluster meta was updated.
	ClusterInfoUpdated(*ClusterMeta)
}

// ClusterMeta represents cluster meta data.
type ClusterMeta struct {
	Context, Cluster    string
	User                string
	Z9sVer, Z9sLatest   string
	K8sVer              string
	Cpu, Mem, Ephemeral int
}

// NewClusterMeta returns a new instance.
func NewClusterMeta() *ClusterMeta {
	return &ClusterMeta{
		Context:   client.NA,
		Cluster:   client.NA,
		User:      client.NA,
		Z9sVer:    client.NA,
		K8sVer:    client.NA,
		Cpu:       0,
		Mem:       0,
		Ephemeral: 0,
	}
}

// Deltas diffs cluster meta return true if different, false otherwise.
func (c *ClusterMeta) Deltas(n *ClusterMeta) bool {
	if c.Cpu != n.Cpu || c.Mem != n.Mem || c.Ephemeral != n.Ephemeral {
		return true
	}

	return c.Context != n.Context ||
		c.Cluster != n.Cluster ||
		c.User != n.User ||
		c.K8sVer != n.K8sVer ||
		c.Z9sVer != n.Z9sVer ||
		c.Z9sLatest != n.Z9sLatest
}

// ClusterInfo models cluster metadata.
type ClusterInfo struct {
	cluster   *Cluster
	factory   dao.Factory
	data      *ClusterMeta
	version   string
	cfg       *config.K9s
	listeners []ClusterInfoListener
	mx        sync.RWMutex
}

// NewClusterInfo returns a new instance.
func NewClusterInfo(f dao.Factory, v string, cfg *config.K9s) *ClusterInfo {
	c := ClusterInfo{
		factory: f,
		cluster: NewCluster(f),
		data:    NewClusterMeta(),
		version: v,
		cfg:     cfg,
	}

	return &c
}

// Reset resets context and reload.
func (c *ClusterInfo) Reset(f dao.Factory) {
	if f == nil {
		return
	}

	c.mx.Lock()
	c.cluster, c.data = NewCluster(f), NewClusterMeta()
	c.mx.Unlock()

	c.Refresh()
}

// Refresh fetches the latest cluster meta.
func (c *ClusterInfo) Refresh() {
	data := NewClusterMeta()
	if c.factory.Client().ConnectionOK() {
		data.Context = c.cluster.ContextName()
		data.Cluster = c.cluster.ClusterName()
		data.User = c.cluster.UserName()
		data.K8sVer = c.cluster.Version()
		ctx, cancel := context.WithTimeout(context.Background(), c.cluster.factory.Client().Config().CallTimeout())
		defer cancel()
		var mx client.ClusterMetrics
		if err := c.cluster.Metrics(ctx, &mx); err == nil {
			data.Cpu, data.Mem, data.Ephemeral = mx.PercCPU, mx.PercMEM, mx.PercEphemeral
		}
	}
	data.Z9sVer = c.version
	v1 := NewSemVer(data.Z9sVer)

	// z9s does not track k9s upstream releases, so we never advertise a "latest"
	// revision. Only the z9s version is shown.
	data.Z9sVer, data.Z9sLatest = v1.String(), ""

	if c.data.Deltas(data) {
		c.fireMetaChanged(c.data, data)
	} else {
		c.fireNoMetaChanged(data)
	}
	c.mx.Lock()
	c.data = data
	c.mx.Unlock()
}

// AddListener adds a new model listener.
func (c *ClusterInfo) AddListener(l ClusterInfoListener) {
	c.listeners = append(c.listeners, l)
}

// RemoveListener delete a listener from the list.
func (c *ClusterInfo) RemoveListener(l ClusterInfoListener) {
	victim := -1
	for i, lis := range c.listeners {
		if lis == l {
			victim = i
			break
		}
	}

	if victim >= 0 {
		c.listeners = append(c.listeners[:victim], c.listeners[victim+1:]...)
	}
}

func (c *ClusterInfo) fireMetaChanged(prev, cur *ClusterMeta) {
	for _, l := range c.listeners {
		l.ClusterInfoChanged(prev, cur)
	}
}

func (c *ClusterInfo) fireNoMetaChanged(data *ClusterMeta) {
	for _, l := range c.listeners {
		l.ClusterInfoUpdated(data)
	}
}
