#!/usr/bin/env bash
set -euo pipefail

# QLink DID system - start full local cluster (gateway + 3 consensus nodes)
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

mkdir -p logs .pids

declare -A CONFIGS
CONFIGS[gateway]="./config/gateway_node.yaml"
CONFIGS[node1]="./config/consensus_node1.yaml"
CONFIGS[node2]="./config/consensus_node2.yaml"
CONFIGS[node3]="./config/consensus_node3.yaml"

for name in "${!CONFIGS[@]}"; do
  cfg="${CONFIGS[$name]}"
  echo "Starting $name with $cfg ..."
  ./bin/qlink-node --config "$cfg" > "logs/${name}.log" 2>&1 &
  echo $! > ".pids/${name}.pid"
done

echo "Waiting for gateway API to come up..."
sleep 2

echo "Registering peers to gateway..."
curl -sS -X POST http://localhost:8080/api/v1/node/peers -H 'Content-Type: application/json' -d '{"id":"node1","address":"127.0.0.1:30301"}' || true
curl -sS -X POST http://localhost:8080/api/v1/node/peers -H 'Content-Type: application/json' -d '{"id":"node2","address":"127.0.0.1:30302"}' || true
curl -sS -X POST http://localhost:8080/api/v1/node/peers -H 'Content-Type: application/json' -d '{"id":"node3","address":"127.0.0.1:30303"}' || true

echo "Cluster status:"
curl -sS http://localhost:8080/api/v1/cluster/status || true

echo "Done. Logs in ./did-system/logs"