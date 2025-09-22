#!/bin/bash

# QLink多节点一致性测试脚本
# 用于测试多节点环境下的数据一致性和网络通信

set -e

# 配置参数
NODE_COUNT=3
BASE_PORT=8080
BASE_P2P_PORT=9000
TEST_DIR="/tmp/qlink_test"
QLINK_BIN="./qlink"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}"
}

warn() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] WARNING: $1${NC}"
}

error() {
    echo -e "${RED}[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1${NC}"
}

# 清理函数
cleanup() {
    log "清理测试环境..."
    
    # 停止所有节点进程
    for i in $(seq 1 $NODE_COUNT); do
        if [ -f "$TEST_DIR/node$i/qlink.pid" ]; then
            PID=$(cat "$TEST_DIR/node$i/qlink.pid")
            if kill -0 $PID 2>/dev/null; then
                log "停止节点 $i (PID: $PID)"
                kill $PID
                sleep 1
            fi
        fi
    done
    
    # 清理测试目录
    if [ -d "$TEST_DIR" ]; then
        rm -rf "$TEST_DIR"
    fi
    
    log "清理完成"
}

# 设置信号处理
trap cleanup EXIT INT TERM

# 创建测试环境
setup_test_env() {
    log "设置测试环境..."
    
    # 创建测试目录
    mkdir -p "$TEST_DIR"
    
    # 检查QLink二进制文件
    if [ ! -f "$QLINK_BIN" ]; then
        error "QLink二进制文件不存在: $QLINK_BIN"
        exit 1
    fi
    
    log "测试环境设置完成"
}

# 生成节点配置
generate_node_config() {
    local node_id=$1
    local node_dir="$TEST_DIR/node$node_id"
    local api_port=$((BASE_PORT + node_id - 1))
    local p2p_port=$((BASE_P2P_PORT + node_id - 1))
    
    mkdir -p "$node_dir"
    
    cat > "$node_dir/config.yaml" << EOF
node:
  id: "node-$node_id"
  data_dir: "$node_dir/data"

network:
  listen_address: "127.0.0.1"
  listen_port: $p2p_port
  bootstrap_peers: []
  connection_timeout: "30s"
  max_peers: 50

api:
  listen_address: "127.0.0.1"
  listen_port: $api_port
  enable_cors: true
  rate_limit: 100

consensus:
  algorithm: "raft"
  election_timeout: "5s"
  heartbeat_interval: "1s"

cluster:
  max_nodes: 10
  heartbeat_interval: "2s"
  election_timeout: "10s"
  join_timeout: "30s"
  sync_interval: "5s"

did:
  network: "testnet"
  storage_path: "$node_dir/did_storage"

logging:
  level: "info"
  file: "$node_dir/qlink.log"
EOF

    log "生成节点 $node_id 配置文件"
}

# 启动节点
start_node() {
    local node_id=$1
    local node_dir="$TEST_DIR/node$node_id"
    
    log "启动节点 $node_id..."
    
    cd "$(dirname "$QLINK_BIN")"
    nohup "$QLINK_BIN" -config "$node_dir/config.yaml" > "$node_dir/output.log" 2>&1 &
    local pid=$!
    echo $pid > "$node_dir/qlink.pid"
    
    # 等待节点启动
    sleep 3
    
    # 检查节点是否正常启动
    if kill -0 $pid 2>/dev/null; then
        log "节点 $node_id 启动成功 (PID: $pid)"
        return 0
    else
        error "节点 $node_id 启动失败"
        cat "$node_dir/output.log"
        return 1
    fi
}

# 等待节点就绪
wait_for_node() {
    local node_id=$1
    local api_port=$((BASE_PORT + node_id - 1))
    local max_attempts=30
    local attempt=0
    
    log "等待节点 $node_id 就绪..."
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "http://127.0.0.1:$api_port/health" > /dev/null 2>&1; then
            log "节点 $node_id 已就绪"
            return 0
        fi
        
        sleep 1
        attempt=$((attempt + 1))
    done
    
    error "节点 $node_id 启动超时"
    return 1
}

# 测试节点连接
test_node_connectivity() {
    log "测试节点连接性..."
    
    for i in $(seq 1 $NODE_COUNT); do
        local api_port=$((BASE_PORT + i - 1))
        
        log "测试节点 $i API连接..."
        if ! curl -s "http://127.0.0.1:$api_port/health" > /dev/null; then
            error "节点 $i API不可访问"
            return 1
        fi
        
        # 获取节点状态
        local status=$(curl -s "http://127.0.0.1:$api_port/status" | jq -r '.status // "unknown"')
        log "节点 $i 状态: $status"
    done
    
    log "所有节点连接正常"
}

# 测试集群形成
test_cluster_formation() {
    log "测试集群形成..."
    
    # 让节点2和节点3加入节点1的集群
    for i in $(seq 2 $NODE_COUNT); do
        local api_port=$((BASE_PORT + i - 1))
        local leader_port=$BASE_PORT
        
        log "节点 $i 加入集群..."
        
        local join_result=$(curl -s -X POST \
            "http://127.0.0.1:$api_port/cluster/join" \
            -H "Content-Type: application/json" \
            -d "{\"leader_address\":\"127.0.0.1\",\"leader_port\":$leader_port}" \
            | jq -r '.success // false')
        
        if [ "$join_result" = "true" ]; then
            log "节点 $i 成功加入集群"
        else
            warn "节点 $i 加入集群失败，可能已经在集群中"
        fi
        
        sleep 2
    done
    
    # 验证集群状态
    log "验证集群状态..."
    local cluster_info=$(curl -s "http://127.0.0.1:$BASE_PORT/cluster/status")
    local node_count=$(echo "$cluster_info" | jq -r '.node_count // 0')
    
    log "集群节点数量: $node_count"
    
    if [ "$node_count" -ge "$NODE_COUNT" ]; then
        log "集群形成成功"
        return 0
    else
        warn "集群节点数量不足，预期: $NODE_COUNT，实际: $node_count"
        return 1
    fi
}

# 测试DID操作一致性
test_did_consistency() {
    log "测试DID操作一致性..."
    
    local test_did="did:qlink:test123"
    local test_document='{
        "id": "'$test_did'",
        "publicKey": [{
            "id": "'$test_did'#key1",
            "type": "Ed25519VerificationKey2018",
            "publicKeyBase58": "H3C2AVvLMv6gmMNam3uVAjZpfkcJCwDwnZn6z3wXmqPV"
        }],
        "service": [{
            "id": "'$test_did'#service1",
            "type": "IdentityHub",
            "serviceEndpoint": "https://example.com/hub"
        }]
    }'
    
    # 在节点1上创建DID
    log "在节点1上创建DID: $test_did"
    local create_result=$(curl -s -X POST \
        "http://127.0.0.1:$BASE_PORT/did/create" \
        -H "Content-Type: application/json" \
        -d "$test_document")
    
    local success=$(echo "$create_result" | jq -r '.success // false')
    if [ "$success" != "true" ]; then
        error "DID创建失败"
        echo "$create_result"
        return 1
    fi
    
    log "DID创建成功，等待同步..."
    sleep 5
    
    # 在所有节点上验证DID
    for i in $(seq 1 $NODE_COUNT); do
        local api_port=$((BASE_PORT + i - 1))
        
        log "在节点 $i 上查询DID..."
        local query_result=$(curl -s "http://127.0.0.1:$api_port/did/resolve/$test_did")
        local found=$(echo "$query_result" | jq -r '.found // false')
        
        if [ "$found" = "true" ]; then
            log "节点 $i 上DID同步成功"
        else
            error "节点 $i 上DID同步失败"
            echo "$query_result"
            return 1
        fi
    done
    
    log "DID一致性测试通过"
}

# 测试网络分区恢复
test_network_partition_recovery() {
    log "测试网络分区恢复..."
    
    # 模拟网络分区：暂停节点3
    local node3_pid=$(cat "$TEST_DIR/node3/qlink.pid")
    log "暂停节点3 (PID: $node3_pid)"
    kill -STOP $node3_pid
    
    sleep 5
    
    # 在剩余节点上进行操作
    local test_did="did:qlink:partition_test"
    local test_document='{
        "id": "'$test_did'",
        "publicKey": [{
            "id": "'$test_did'#key1",
            "type": "Ed25519VerificationKey2018",
            "publicKeyBase58": "H3C2AVvLMv6gmMNam3uVAjZpfkcJCwDwnZn6z3wXmqPV"
        }]
    }'
    
    log "在分区期间创建DID..."
    curl -s -X POST \
        "http://127.0.0.1:$BASE_PORT/did/create" \
        -H "Content-Type: application/json" \
        -d "$test_document" > /dev/null
    
    sleep 2
    
    # 恢复节点3
    log "恢复节点3"
    kill -CONT $node3_pid
    
    # 等待同步
    log "等待分区恢复同步..."
    sleep 10
    
    # 验证节点3上的数据
    local query_result=$(curl -s "http://127.0.0.1:$((BASE_PORT + 2))/did/resolve/$test_did")
    local found=$(echo "$query_result" | jq -r '.found // false')
    
    if [ "$found" = "true" ]; then
        log "网络分区恢复测试通过"
        return 0
    else
        error "网络分区恢复测试失败"
        return 1
    fi
}

# 生成测试报告
generate_test_report() {
    log "生成测试报告..."
    
    local report_file="$TEST_DIR/test_report.json"
    
    cat > "$report_file" << EOF
{
    "test_time": "$(date -Iseconds)",
    "node_count": $NODE_COUNT,
    "test_results": {
        "connectivity": true,
        "cluster_formation": true,
        "did_consistency": true,
        "partition_recovery": true
    },
    "node_logs": [
EOF

    for i in $(seq 1 $NODE_COUNT); do
        if [ $i -gt 1 ]; then
            echo "," >> "$report_file"
        fi
        echo "        {" >> "$report_file"
        echo "            \"node_id\": $i," >> "$report_file"
        echo "            \"log_file\": \"$TEST_DIR/node$i/qlink.log\"" >> "$report_file"
        echo "        }" >> "$report_file"
    done

    cat >> "$report_file" << EOF
    ]
}
EOF

    log "测试报告已生成: $report_file"
}

# 主测试流程
main() {
    log "开始QLink多节点一致性测试"
    log "节点数量: $NODE_COUNT"
    log "API端口范围: $BASE_PORT-$((BASE_PORT + NODE_COUNT - 1))"
    log "P2P端口范围: $BASE_P2P_PORT-$((BASE_P2P_PORT + NODE_COUNT - 1))"
    
    # 设置测试环境
    setup_test_env
    
    # 生成配置并启动节点
    for i in $(seq 1 $NODE_COUNT); do
        generate_node_config $i
        start_node $i
        wait_for_node $i
    done
    
    # 运行测试
    test_node_connectivity
    test_cluster_formation
    test_did_consistency
    test_network_partition_recovery
    
    # 生成报告
    generate_test_report
    
    log "所有测试完成！"
    log "测试结果保存在: $TEST_DIR"
    
    # 保持运行以便手动检查
    log "按Ctrl+C停止测试环境"
    while true; do
        sleep 10
        log "测试环境运行中... (节点数: $NODE_COUNT)"
    done
}

# 检查依赖
check_dependencies() {
    if ! command -v curl &> /dev/null; then
        error "curl命令未找到，请安装curl"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        error "jq命令未找到，请安装jq"
        exit 1
    fi
}

# 脚本入口
if [ "$1" = "cleanup" ]; then
    cleanup
    exit 0
fi

check_dependencies
main