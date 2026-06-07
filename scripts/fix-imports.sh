#!/usr/bin/env bash
# Fix internal import paths in the z9s merged project.
#
# The merge only swapped the module prefix
# (github.com/derailed/k9s -> github.com/yourusername/z9s) without inserting the
# new "internal/k9s/" nesting. This rewrites every k9s cross-package import to
# point at its real location under internal/k9s/.
#
# Idempotent: the negative lookahead skips already-correct paths
# (internal/k9s/, internal/ktop/, internal/app, internal/shared).
#
# Usage: ./scripts/fix-imports.sh   (run from the repo root)
set -euo pipefail

# Repo root = parent dir of this script's dir.
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# k9s: internal/<pkg> -> internal/k9s/<pkg>
find . -type f -name '*.go' -exec perl -0pi -e \
  's{github\.com/yourusername/z9s/internal/(?!k9s/|ktop/|app|shared)}{github.com/yourusername/z9s/internal/k9s/}g' {} +

# k9s root package: bare "internal" -> "internal/k9s" (package name stays "internal")
find . -type f -name '*.go' -exec perl -0pi -e \
  's{github\.com/yourusername/z9s/internal"}{github.com/yourusername/z9s/internal/k9s"}g' {} +

echo "k9s import paths rewritten."
