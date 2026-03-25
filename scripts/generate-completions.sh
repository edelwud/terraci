#!/usr/bin/env bash
set -euo pipefail

# Generate shell completion scripts for terraci and xterraci
# This script is called by goreleaser before building

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
COMPLETIONS_DIR="${PROJECT_ROOT}/completions"

mkdir -p "$COMPLETIONS_DIR"

generate_completions() {
  local name="$1"
  local cmd_dir="$2"

  echo "Building ${name}..."
  local bin="${PROJECT_ROOT}/${name}"
  go build -o "$bin" "${PROJECT_ROOT}/${cmd_dir}"

  echo "Generating ${name} completions..."
  "$bin" completion bash > "${COMPLETIONS_DIR}/${name}.bash"
  "$bin" completion zsh > "${COMPLETIONS_DIR}/_${name}"
  "$bin" completion fish > "${COMPLETIONS_DIR}/${name}.fish"
  "$bin" completion powershell > "${COMPLETIONS_DIR}/${name}.ps1"

  rm -f "$bin"
  echo "  ${name}: bash, zsh, fish, powershell"
}

generate_completions "terraci" "cmd/terraci"
generate_completions "xterraci" "cmd/xterraci"

echo "Shell completions generated successfully!"
