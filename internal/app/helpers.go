// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of z9s

package app

import (
	"log/slog"

	"github.com/yourusername/z9s/internal/shared"
)

// GetSharedK8sClient returns the singleton K8s client instance
func GetSharedK8sClient(logger *slog.Logger) *shared.SharedK8sClient {
	return shared.GetInstance(logger)
}
