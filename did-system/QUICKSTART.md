# QLink DID-System Quick Start

This quick start provides a simplified way to run the local blockchain+DID API.

## Prerequisites
- Built binaries exist in `did-system/bin/` (e.g. `qlink-node`).

## Start Gateway API only
```
make gateway
make status
```

Gateway API available at `http://localhost:8080/`.

## Start full local cluster
```
make cluster
make status
```

Starts gateway plus three consensus nodes, then auto-registers peers.

## Stop all nodes
```
make stop
```

Logs are written to `did-system/logs`. PIDs are tracked under `did-system/.pids`.