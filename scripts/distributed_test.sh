#!/bin/bash

# QLink 分布式区块链系统测试脚本
# 用于启动多节点集群并进行全面测试

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置参数
PROJECT_ROOT="/home/peninsula/go/src/QLink"
NODE_BINARY="$PROJECT_ROOT/qlink-node"
CONFIG_DIR="$PROJECT_ROOT/config"
DATA_DIR="$PROJECT_ROOT/data"
LOG_DIR="$PROJECT_ROOT/logs"

# 节点配置
NODE1_PORT=8080
NODE2_PORT=8082
NODE3_PORT=8084
NODE1_P2P_PORT=30303
NODE2_P2P_PORT=30304
NODE3_P2P_PORT=30305

# 创建必要的目录
create_directories() {
    echo -e "${BLUE}创建必要的目录...${NC}"
    mkdir -p "$DATA_DIR/node1" "$DATA_DIR/node2" "$DATA_DIR/node3"
    mkdir -p "$LOG_DIR"
    echo -e "${GREEN}目录创建完成${NC}"
}

# 生成节点配置文件
generate_node_configs() {
    echo -e "${BLUE}生成节点配置文件...${NC}"
    
    # Node 1 配置 (主节点)
    cat > "$CONFIG_DIR/local_node1.yaml" << EOF
node:
  id: "node1"
  name: "QLink Node 1"
  version: "1.0.0"
  data_dir: "$DATA_DIR/node1"

network:
  port: $NODE1_P2P_PORT
  
api:
  port: $NODE1_PORT
  host: "localhost"

storage:
  type: "local"
  path: "$DATA_DIR/node1/storage"

sync:
  enabled: true
  bootstrap_nodes: []
  
logging:
  level: "info"
  file: "$LOG_DIR/node1.log"
EOF

    # Node 2 配置 (副本节点)
    cat > "$CONFIG_DIR/local_node2.yaml" << EOF
node:
  id: "node2"
  name: "QLink Node 2"
  version: "1.0.0"
  data_dir: "$DATA_DIR/node2"

network:
  port: $NODE2_P2P_PORT
  
api:
  port: $NODE2_PORT
  host: "localhost"

storage:
  type: "local"
  path: "$DATA_DIR/node2/storage"

sync:
  enabled: true
  bootstrap_nodes: ["localhost:$NODE1_P2P_PORT"]
  
logging:
  level: "info"
  file: "$LOG_DIR/node2.log"
EOF

    # Node 3 配置 (副本节点)
    cat > "$CONFIG_DIR/local_node3.yaml" << EOF
node:
  id: "node3"
  name: "QLink Node 3"
  version: "1.0.0"
  data_dir: "$DATA_DIR/node3"

network:
  port: $NODE3_P2P_PORT
  
api:
  port: $NODE3_PORT
  host: "localhost"

storage:
  type: "local"
  path: "$DATA_DIR/node3/storage"

sync:
  enabled: true
  bootstrap_nodes: ["localhost:$NODE1_P2P_PORT"]
  
logging:
  level: "info"
  file: "$LOG_DIR/node3.log"
EOF

    echo -e "${GREEN}节点配置文件生成完成${NC}"
}

# 启动节点
start_node() {
    local node_id=$1
    local config_file=$2
    local port=$3
    
    echo -e "${BLUE}启动节点 $node_id (端口: $port)...${NC}"
    
    # 检查端口是否被占用
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo -e "${YELLOW}端口 $port 已被占用，尝试停止现有进程...${NC}"
        pkill -f "qlink-node.*$config_file" || true
        sleep 2
    fi
    
    # 启动节点
    nohup "$NODE_BINARY" --config="$config_file" > "$LOG_DIR/${node_id}_startup.log" 2>&1 &
    local pid=$!
    echo $pid > "$LOG_DIR/${node_id}.pid"
    
    # 等待节点启动
    echo -e "${YELLOW}等待节点 $node_id 启动...${NC}"
    for i in {1..30}; do
        if curl -s "http://localhost:$port/health" >/dev/null 2>&1; then
            echo -e "${GREEN}节点 $node_id 启动成功 (PID: $pid)${NC}"
            return 0
        fi
        sleep 1
    done
    
    echo -e "${RED}节点 $node_id 启动失败${NC}"
    return 1
}

# 停止所有节点
stop_all_nodes() {
    echo -e "${BLUE}停止所有节点...${NC}"
    
    for node in node1 node2 node3; do
        if [ -f "$LOG_DIR/${node}.pid" ]; then
            local pid=$(cat "$LOG_DIR/${node}.pid")
            if kill -0 $pid 2>/dev/null; then
                echo -e "${YELLOW}停止节点 $node (PID: $pid)...${NC}"
                kill $pid
                sleep 2
                if kill -0 $pid 2>/dev/null; then
                    kill -9 $pid
                fi
            fi
            rm -f "$LOG_DIR/${node}.pid"
        fi
    done
    
    # 清理可能残留的进程
    pkill -f "qlink-node" || true
    echo -e "${GREEN}所有节点已停止${NC}"
}

# 检查节点状态
check_nodes_status() {
    echo -e "${BLUE}检查节点状态...${NC}"
    
    local all_healthy=true
    
    for port in $NODE1_PORT $NODE2_PORT $NODE3_PORT; do
        if curl -s "http://localhost:$port/health" >/dev/null 2>&1; then
            echo -e "${GREEN}节点 (端口 $port): 健康${NC}"
        else
            echo -e "${RED}节点 (端口 $port): 不健康${NC}"
            all_healthy=false
        fi
    done
    
    if [ "$all_healthy" = true ]; then
        echo -e "${GREEN}所有节点都处于健康状态${NC}"
        return 0
    else
        echo -e "${RED}部分节点不健康${NC}"
        return 1
    fi
}

# 测试API功能
test_api_functions() {
    echo -e "${BLUE}开始API功能测试...${NC}"
    
    local base_url="http://localhost:$NODE1_PORT"
    local test_did="did:qlink:distributed-test-$(date +%s)"
    
    # 测试健康检查
    echo -e "${YELLOW}测试健康检查API...${NC}"
    if curl -s "$base_url/health" | grep -q "ok"; then
        echo -e "${GREEN}✓ 健康检查API正常${NC}"
    else
        echo -e "${RED}✗ 健康检查API失败${NC}"
        return 1
    fi
    
    # 测试节点信息
    echo -e "${YELLOW}测试节点信息API...${NC}"
    if curl -s "$base_url/api/v1/node/info" | grep -q "node_id"; then
        echo -e "${GREEN}✓ 节点信息API正常${NC}"
    else
        echo -e "${RED}✗ 节点信息API失败${NC}"
    fi
    
    # 测试DID注册
    echo -e "${YELLOW}测试DID注册API...${NC}"
    local register_response=$(curl -s -X POST "$base_url/api/v1/did/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"did\": \"$test_did\",
            \"document\": {
                \"@context\": \"https://www.w3.org/ns/did/v1\",
                \"id\": \"$test_did\",
                \"verificationMethod\": [{
                    \"id\": \"$test_did#key1\",
                    \"type\": \"Ed25519VerificationKey2018\",
                    \"controller\": \"$test_did\",
                    \"publicKeyBase58\": \"H3C2AVvLMv6gmMNam3uVAjZpfkcJCwDwnZn6z3wXmqPV\"
                }]
            },
            \"signature\": \"test_signature\"
        }")
    
    if echo "$register_response" | grep -q "success\|registered"; then
        echo -e "${GREEN}✓ DID注册API正常${NC}"
    else
        echo -e "${YELLOW}⚠ DID注册API响应: $register_response${NC}"
    fi
    
    # 测试DID解析
    echo -e "${YELLOW}测试DID解析API...${NC}"
    sleep 2  # 等待数据同步
    local resolve_response=$(curl -s "$base_url/api/v1/did/resolve/$test_did")
    if echo "$resolve_response" | grep -q "$test_did"; then
        echo -e "${GREEN}✓ DID解析API正常${NC}"
    else
        echo -e "${YELLOW}⚠ DID解析API响应: $resolve_response${NC}"
    fi
    
    echo -e "${GREEN}API功能测试完成${NC}"
}

# 测试分布式一致性
test_distributed_consistency() {
    echo -e "${BLUE}开始分布式一致性测试...${NC}"
    
    local test_did="did:qlink:consistency-test-$(date +%s)"
    
    # 在节点1注册DID
    echo -e "${YELLOW}在节点1注册DID...${NC}"
    curl -s -X POST "http://localhost:$NODE1_PORT/api/v1/did/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"did\": \"$test_did\",
            \"document\": {
                \"@context\": \"https://www.w3.org/ns/did/v1\",
                \"id\": \"$test_did\",
                \"verificationMethod\": [{
                    \"id\": \"$test_did#key1\",
                    \"type\": \"Ed25519VerificationKey2018\",
                    \"controller\": \"$test_did\",
                    \"publicKeyBase58\": \"H3C2AVvLMv6gmMNam3uVAjZpfkcJCwDwnZn6z3wXmqPV\"
                }]
            },
            \"signature\": \"test_signature\"
        }" > /dev/null
    
    # 等待数据同步
    echo -e "${YELLOW}等待数据同步...${NC}"
    sleep 5
    
    # 在所有节点上查询DID
    local consistency_check=true
    for port in $NODE1_PORT $NODE2_PORT $NODE3_PORT; do
        echo -e "${YELLOW}检查节点 (端口 $port) 的数据一致性...${NC}"
        local response=$(curl -s "http://localhost:$port/api/v1/did/resolve/$test_did")
        if echo "$response" | grep -q "$test_did"; then
            echo -e "${GREEN}✓ 节点 (端口 $port) 数据一致${NC}"
        else
            echo -e "${RED}✗ 节点 (端口 $port) 数据不一致${NC}"
            consistency_check=false
        fi
    done
    
    if [ "$consistency_check" = true ]; then
        echo -e "${GREEN}✓ 分布式一致性测试通过${NC}"
        return 0
    else
        echo -e "${RED}✗ 分布式一致性测试失败${NC}"
        return 1
    fi
}

# 性能压力测试
run_stress_test() {
    echo -e "${BLUE}开始性能压力测试...${NC}"
    
    local base_url="http://localhost:$NODE1_PORT"
    local concurrent_requests=10
    local total_requests=100
    
    echo -e "${YELLOW}执行 $total_requests 个并发请求 (并发数: $concurrent_requests)...${NC}"
    
    # 创建临时测试脚本
    cat > /tmp/stress_test.sh << 'EOF'
#!/bin/bash
base_url=$1
request_id=$2

test_did="did:qlink:stress-test-$request_id-$(date +%s)"

# 注册DID
curl -s -X POST "$base_url/api/v1/did/register" \
    -H "Content-Type: application/json" \
    -d "{
        \"did\": \"$test_did\",
        \"document\": {
            \"@context\": \"https://www.w3.org/ns/did/v1\",
            \"id\": \"$test_did\",
            \"verificationMethod\": [{
                \"id\": \"$test_did#key1\",
                \"type\": \"Ed25519VerificationKey2018\",
                \"controller\": \"$test_did\",
                \"publicKeyBase58\": \"H3C2AVvLMv6gmMNam3uVAjZpfkcJCwDwnZn6z3wXmqPV\"
            }]
        },
        \"signature\": \"test_signature\"
    }" > /dev/null

# 解析DID
sleep 1
curl -s "$base_url/api/v1/did/resolve/$test_did" > /dev/null

echo "Request $request_id completed"
EOF

    chmod +x /tmp/stress_test.sh
    
    # 执行并发测试
    local start_time=$(date +%s)
    for i in $(seq 1 $total_requests); do
        /tmp/stress_test.sh "$base_url" "$i" &
        
        # 控制并发数
        if [ $((i % concurrent_requests)) -eq 0 ]; then
            wait
        fi
    done
    wait
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo -e "${GREEN}压力测试完成${NC}"
    echo -e "${BLUE}总请求数: $total_requests${NC}"
    echo -e "${BLUE}总耗时: ${duration}秒${NC}"
    echo -e "${BLUE}平均TPS: $((total_requests / duration))${NC}"
    
    # 清理临时文件
    rm -f /tmp/stress_test.sh
}

# 显示使用帮助
show_help() {
    echo "QLink 分布式区块链系统测试脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  start       启动分布式节点集群"
    echo "  stop        停止所有节点"
    echo "  status      检查节点状态"
    echo "  test        运行完整测试套件"
    echo "  api-test    仅运行API功能测试"
    echo "  consistency 仅运行一致性测试"
    echo "  stress      仅运行压力测试"
    echo "  restart     重启所有节点"
    echo "  logs        显示节点日志"
    echo "  clean       清理所有数据和日志"
    echo "  help        显示此帮助信息"
}

# 显示节点日志
show_logs() {
    echo -e "${BLUE}显示节点日志...${NC}"
    
    for node in node1 node2 node3; do
        echo -e "${YELLOW}=== $node 日志 ===${NC}"
        if [ -f "$LOG_DIR/${node}.log" ]; then
            tail -n 20 "$LOG_DIR/${node}.log"
        else
            echo "日志文件不存在"
        fi
        echo ""
    done
}

# 清理数据和日志
clean_all() {
    echo -e "${BLUE}清理所有数据和日志...${NC}"
    
    stop_all_nodes
    
    rm -rf "$DATA_DIR/node1" "$DATA_DIR/node2" "$DATA_DIR/node3"
    rm -rf "$LOG_DIR"/*
    rm -f "$CONFIG_DIR/local_node"*.yaml
    
    echo -e "${GREEN}清理完成${NC}"
}

# 主函数
main() {
    case "${1:-help}" in
        "start")
            create_directories
            generate_node_configs
            start_node "node1" "$CONFIG_DIR/local_node1.yaml" $NODE1_PORT
            sleep 3
            start_node "node2" "$CONFIG_DIR/local_node2.yaml" $NODE2_PORT
            sleep 3
            start_node "node3" "$CONFIG_DIR/local_node3.yaml" $NODE3_PORT
            sleep 3
            check_nodes_status
            ;;
        "stop")
            stop_all_nodes
            ;;
        "status")
            check_nodes_status
            ;;
        "test")
            echo -e "${GREEN}开始完整测试套件...${NC}"
            test_api_functions
            test_distributed_consistency
            run_stress_test
            echo -e "${GREEN}所有测试完成${NC}"
            ;;
        "api-test")
            test_api_functions
            ;;
        "consistency")
            test_distributed_consistency
            ;;
        "stress")
            run_stress_test
            ;;
        "restart")
            stop_all_nodes
            sleep 2
            main start
            ;;
        "logs")
            show_logs
            ;;
        "clean")
            clean_all
            ;;
        "help"|*)
            show_help
            ;;
    esac
}

# 执行主函数
main "$@"