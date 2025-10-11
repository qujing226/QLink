#!/usr/bin/env bash
set -euo pipefail

# QLink DID system - stop all locally started nodes using recorded PIDs
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

timeout_secs=10

# 1) Stop recorded PIDs gracefully, then force if needed
if [ -d ".pids" ]; then
  for pidfile in .pids/*.pid; do
    [ -e "$pidfile" ] || continue
    pid=$(cat "$pidfile" || true)
    if [ -n "${pid:-}" ] && kill -0 "$pid" 2>/dev/null; then
      echo "Stopping process $pid ($pidfile) with SIGTERM"
      kill -TERM "$pid" || true
      # Wait for exit
      for i in $(seq 1 $timeout_secs); do
        if ! kill -0 "$pid" 2>/dev/null; then
          break
        fi
        sleep 1
      done
      # Force kill if still alive
      if kill -0 "$pid" 2>/dev/null; then
        echo "Process $pid still alive, sending SIGKILL"
        kill -KILL "$pid" || true
      fi
    fi
    rm -f "$pidfile"
  done
fi

# 2) Additionally, kill any qlink-node processes not tracked in .pids
echo "Scanning for stray qlink-node processes..."
ps -eo pid,comm,args | awk '/qlink-node/ && !/awk/ {print $1}' | while read -r stray; do
  if [ -n "$stray" ] && kill -0 "$stray" 2>/dev/null; then
    echo "Stopping stray qlink-node PID $stray"
    kill -TERM "$stray" || true
    sleep 2
    if kill -0 "$stray" 2>/dev/null; then
      kill -KILL "$stray" || true
    fi
  fi
done || true

# 3) Free known ports if any process still listening
echo "Ensuring ports are free (8080, 30301-30303)..."
ports=(8080 30301 30302 30303)
for port in "${ports[@]}"; do
  # ss prints like: LISTEN 0 128 127.0.0.1:8080 users:(('qlink-node',pid=123,fd=3))
  if command -v ss >/dev/null 2>&1; then
    ss -ltnp 2>/dev/null | grep ":$port " | sed -n 's/.*pid=\([0-9]\+\).*/\1/p' | while read -r p; do
      echo "Killing PID $p listening on port $port"
      kill -TERM "$p" || true
      sleep 2
      kill -KILL "$p" || true
    done || true
  fi
done

echo "All QLink node processes stopped."