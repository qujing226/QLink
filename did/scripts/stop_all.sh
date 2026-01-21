#!/usr/bin/env bash
set -euo pipefail

# QLink DID system - stop all locally started nodes using recorded PIDs
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

if [ -d ".pids" ]; then
  for pidfile in .pids/*.pid; do
    [ -e "$pidfile" ] || continue
    pid=$(cat "$pidfile" || true)
    if [ -n "${pid:-}" ] && kill -0 "$pid" 2>/dev/null; then
      echo "Stopping process $pid ($pidfile)"
      kill "$pid" || true
    fi
    rm -f "$pidfile"
  done
fi

echo "Stopped recorded QLink node processes."