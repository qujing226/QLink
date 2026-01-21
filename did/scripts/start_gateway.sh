#!/usr/bin/env bash
set -euo pipefail

# QLink DID system - start gateway API node
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p logs .pids
CONFIG="./config/gateway_node.yaml"

echo "Starting QLink gateway API using $CONFIG ..."
./bin/qlink-node --config "$CONFIG" > logs/gateway.log 2>&1 &
echo $! > .pids/gateway.pid

echo "Gateway started. API: http://localhost:8080/"