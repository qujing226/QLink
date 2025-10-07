# QLink 项目快速指南

本仓库包含三部分：
- `did-system/` 去中心化身份与区块链节点（提供 HTTP API）
- `im/` Go 后端（即时通信服务，消费 DID-System API）
- `im-front/` 前端（Vue 应用，与后端和 DID-System 联调）

## 启动区块链与DID API（推荐）

进入 `did-system` 目录并使用 Make 命令：

```
cd did-system

# 启动仅网关 API（http://localhost:8080）
make gateway

# 启动完整本地集群（网关 + 3 共识节点）
make cluster

# 查看状态
make status

# 停止所有本地节点
make stop
```

说明：日志在 `did-system/logs/`，进程 PID 在 `did-system/.pids/`。

## 启动 Go 后端（IM）

在 `im/` 目录：

```
cd im
make run
# 或编译
make build && ./bin/im-server
```

配置环境变量示例：

```
export SERVER_HOST=localhost
export SERVER_PORT=8081
export DATABASE_URL=./qlink.db
export JWT_SECRET=your-secret-key
export LOG_LEVEL=info
export DID_NODE_URL=http://localhost:8080
```

## 启动前端

在 `im-front/` 目录：

```
cd im-front
npm install
npm run dev
```

前端默认连接后端与区块链 API，请确保 DID-System 网关已在 `8080` 启动。

## 项目结构优化说明

- 提供 `did-system/scripts/` 与 `Makefile` 简化本地启动与停止
- 清理冗余配置：移除 `config/unified.yaml`、`config/node1.yaml`、`config/validator_node.yaml`
- 文档补充：新增 `did-system/QUICKSTART.md` 与本仓库根 `README.md`

建议进一步优化：
- 统一配置入口：仅保留 `gateway_node.yaml` 与 `consensus_node{1..3}.yaml`
- 标准化 API BaseURL：统一使用 `http://localhost:8080` 作为 DID-System 网关地址
- 为 CI 增加健康检查与集群启动脚本，便于自动化测试