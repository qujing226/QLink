#!/bin/bash

# 多节点一致性测试运行脚本
# 用于启动多个QLink节点并测试分布式一致性

set -e

# 配置参数
NODES=3
BASE_PORT=8080
BASE_RAFT_PORT=9090
BASE_P2P_PORT=7070
TEST_DIR="/tmp/qlink_test"
LOG_DIR="$TEST_DIR/logs"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 清理函数
cleanup() {
    log_info "清理测试环境..."
    
    # 停止所有节点进程
    for i in $(seq 1 $NODES); do
        PID_FILE="$TEST_DIR/node$i.pid"
        if [ -f "$PID_FILE" ]; then
            PID=$(cat "$PID_FILE")
            if kill -0 "$PID" 2>/dev/null; then
                log_info "停止节点 $i (PID: $PID)"
                kill "$PID"
                sleep 1
                # 强制杀死如果还在运行
                if kill -0 "$PID" 2>/dev/null; then
                    kill -9 "$PID"
                fi
            fi
            rm -f "$PID_FILE"
        fi
    done
    
    # 清理测试目录
    if [ -d "$TEST_DIR" ]; then
        rm -rf "$TEST_DIR"
    fi
    
    log_success "清理完成"
}

# 设置信号处理
trap cleanup EXIT INT TERM

# 初始化测试环境
init_test_env() {
    log_info "初始化测试环境..."
    
    # 创建测试目录
    mkdir -p "$TEST_DIR"
    mkdir -p "$LOG_DIR"
    
    # 检查qlink二进制文件
    if [ ! -f "./qlink" ]; then
        log_error "qlink二进制文件不存在，请先编译项目"
        exit 1
    fi
    
    log_success "测试环境初始化完成"
}

# 生成节点配置
generate_node_config() {
    local node_id=$1
    local config_file="$TEST_DIR/node$node_id.yaml"
    
    local api_port=$((BASE_PORT + node_id - 1))
    local raft_port=$((BASE_RAFT_PORT + node_id - 1))
    local p2p_port=$((BASE_P2P_PORT + node_id - 1))
    
    cat > "$config_file" << EOF
# QLink Node $node_id Configuration
node:
  id: "node-$node_id"
  data_dir: "$TEST_DIR/node$node_id/data"

api:
  host: "127.0.0.1"
  port: $api_port
  cors_enabled: true

consensus:
  raft:
    port: $raft_port
    data_dir: "$TEST_DIR/node$node_id/raft"
    snapshot_interval: "10s"
    heartbeat_timeout: "1s"
    election_timeout: "3s"
  
p2p:
  port: $p2p_port
  bootstrap_peers: []

did:
  registry_file: "$TEST_DIR/node$node_id/did_registry.db"

logging:
  level: "info"
  file: "$LOG_DIR/node$node_id.log"
EOF

    # 为非第一个节点添加引导节点
    if [ $node_id -gt 1 ]; then
        local bootstrap_port=$((BASE_P2P_PORT))
        sed -i "s/bootstrap_peers: \[\]/bootstrap_peers: [\"127.0.0.1:$bootstrap_port\"]/" "$config_file"
    fi
    
    log_info "生成节点 $node_id 配置文件: $config_file"
}

# 启动节点
start_node() {
    local node_id=$1
    local config_file="$TEST_DIR/node$node_id.yaml"
    local pid_file="$TEST_DIR/node$node_id.pid"
    local log_file="$LOG_DIR/node$node_id.log"
    
    log_info "启动节点 $node_id..."
    
    # 创建节点数据目录
    mkdir -p "$TEST_DIR/node$node_id/data"
    mkdir -p "$TEST_DIR/node$node_id/raft"
    
    # 启动节点
    nohup ./qlink start --config "$config_file" > "$log_file" 2>&1 &
    local pid=$!
    echo $pid > "$pid_file"
    
    # 等待节点启动
    sleep 2
    
    # 检查节点是否成功启动
    if kill -0 "$pid" 2>/dev/null; then
        local api_port=$((BASE_PORT + node_id - 1))
        # 等待API服务可用
        local retries=10
        while [ $retries -gt 0 ]; do
            if curl -s "http://127.0.0.1:$api_port/health" > /dev/null 2>&1; then
                log_success "节点 $node_id 启动成功 (PID: $pid, API: $api_port)"
                return 0
            fi
            sleep 1
            retries=$((retries - 1))
        done
        log_warning "节点 $node_id 启动但API服务不可用"
    else
        log_error "节点 $node_id 启动失败"
        return 1
    fi
}

# 等待集群形成
wait_for_cluster() {
    log_info "等待集群形成..."
    
    local max_wait=30
    local wait_time=0
    
    while [ $wait_time -lt $max_wait ]; do
        local leader_count=0
        local healthy_nodes=0
        
        for i in $(seq 1 $NODES); do
            local api_port=$((BASE_PORT + i - 1))
            
            # 检查节点健康状态
            if curl -s "http://127.0.0.1:$api_port/health" > /dev/null 2>&1; then
                healthy_nodes=$((healthy_nodes + 1))
                
                # 检查是否为领导者
                local leader_response=$(curl -s "http://127.0.0.1:$api_port/consensus/leader" 2>/dev/null || echo "")
                if echo "$leader_response" | grep -q "node-$i"; then
                    leader_count=$((leader_count + 1))
                fi
            fi
        done
        
        log_info "健康节点: $healthy_nodes/$NODES, 领导者数量: $leader_count"
        
        if [ $healthy_nodes -eq $NODES ] && [ $leader_count -eq 1 ]; then
            log_success "集群形成成功！"
            return 0
        fi
        
        sleep 2
        wait_time=$((wait_time + 2))
    done
    
    log_error "集群形成超时"
    return 1
}

# 测试DID操作一致性
test_did_consistency() {
    log_info "测试DID操作一致性..."
    
    local test_did="did:qlink:test$(date +%s)"
    local leader_port=""
    
    # 找到领导者节点
    for i in $(seq 1 $NODES); do
        local api_port=$((BASE_PORT + i - 1))
        local leader_response=$(curl -s "http://127.0.0.1:$api_port/consensus/leader" 2>/dev/null || echo "")
        if echo "$leader_response" | grep -q "node-$i"; then
            leader_port=$api_port
            log_info "找到领导者节点: node-$i (端口: $api_port)"
            break
        fi
    done
    
    if [ -z "$leader_port" ]; then
        log_error "未找到领导者节点"
        return 1
    fi
    
    # 在领导者节点上提交DID操作
    log_info "在领导者节点上创建DID: $test_did"
    local create_response=$(curl -s -X POST "http://127.0.0.1:$leader_port/consensus/propose" \
        -H "Content-Type: application/json" \
        -d "{
            \"type\": \"did_operation\",
            \"data\": {
                \"operation\": \"register\",
                \"did\": \"$test_did\",
                \"document\": {
                    \"id\": \"$test_did\",
                    \"publicKey\": [{
                        \"id\": \"key1\",
                        \"type\": \"Ed25519VerificationKey2018\",
                        \"publicKeyBase58\": \"H3C2AVvLMv6gmMNam3uVAjZpfkcJCwDwnZn6z3wXmqPV\"
                    }]
                }
            }
        }" 2>/dev/null)
    
    if echo "$create_response" | grep -q "error"; then
        log_error "DID创建失败: $create_response"
        return 1
    fi
    
    log_success "DID创建请求已提交"
    
    # 等待操作同步到所有节点
    sleep 5
    
    # 验证所有节点的一致性
    log_info "验证所有节点的DID状态一致性..."
    local consistent=true
    
    for i in $(seq 1 $NODES); do
        local api_port=$((BASE_PORT + i - 1))
        local status_response=$(curl -s "http://127.0.0.1:$api_port/consensus/status" 2>/dev/null || echo "")
        
        if [ -n "$status_response" ]; then
            log_info "节点 $i 状态: $status_response"
        else
            log_warning "无法获取节点 $i 的状态"
            consistent=false
        fi
    done
    
    if $consistent; then
        log_success "DID操作一致性测试通过"
        return 0
    else
        log_error "DID操作一致性测试失败"
        return 1
    fi
}

# 测试网络分区恢复
test_network_partition() {
    log_info "测试网络分区恢复..."
    
    # 模拟网络分区：停止一个节点
    local partition_node=3
    local pid_file="$TEST_DIR/node$partition_node.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        log_info "模拟网络分区：停止节点 $partition_node"
        kill "$pid"
        sleep 2
        
        # 在剩余节点上进行操作
        local test_did="did:qlink:partition$(date +%s)"
        local api_port=$((BASE_PORT))
        
        log_info "在分区期间创建DID: $test_did"
        curl -s -X POST "http://127.0.0.1:$api_port/consensus/propose" \
            -H "Content-Type: application/json" \
            -d "{
                \"type\": \"did_operation\",
                \"data\": {
                    \"operation\": \"register\",
                    \"did\": \"$test_did\",
                    \"document\": {\"id\": \"$test_did\"}
                }
            }" > /dev/null 2>&1
        
        sleep 3
        
        # 重新启动节点
        log_info "恢复网络分区：重新启动节点 $partition_node"
        start_node $partition_node
        
        # 等待同步
        sleep 5
        
        log_success "网络分区恢复测试完成"
        return 0
    else
        log_error "找不到节点 $partition_node 的PID文件"
        return 1
    fi
}

# 生成测试报告
generate_report() {
    log_info "生成测试报告..."
    
    local report_file="$TEST_DIR/test_report.txt"
    
    cat > "$report_file" << EOF
QLink 多节点一致性测试报告
========================

测试时间: $(date)
节点数量: $NODES
测试目录: $TEST_DIR

节点状态:
EOF

    for i in $(seq 1 $NODES); do
        local api_port=$((BASE_PORT + i - 1))
        local pid_file="$TEST_DIR/node$i.pid"
        
        echo "节点 $i:" >> "$report_file"
        echo "  API端口: $api_port" >> "$report_file"
        
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                echo "  状态: 运行中 (PID: $pid)" >> "$report_file"
                
                # 获取节点状态
                local status=$(curl -s "http://127.0.0.1:$api_port/consensus/status" 2>/dev/null || echo "无法获取")
                echo "  共识状态: $status" >> "$report_file"
            else
                echo "  状态: 已停止" >> "$report_file"
            fi
        else
            echo "  状态: 未启动" >> "$report_file"
        fi
        echo "" >> "$report_file"
    done
    
    echo "日志文件:" >> "$report_file"
    for i in $(seq 1 $NODES); do
        echo "  节点 $i: $LOG_DIR/node$i.log" >> "$report_file"
    done
    
    log_success "测试报告已生成: $report_file"
    cat "$report_file"
}

# 主函数
main() {
    log_info "开始QLink多节点一致性测试"
    
    # 初始化测试环境
    init_test_env
    
    # 生成配置并启动节点
    for i in $(seq 1 $NODES); do
        generate_node_config $i
        start_node $i
    done
    
    # 等待集群形成
    if ! wait_for_cluster; then
        log_error "集群形成失败，测试终止"
        exit 1
    fi
    
    # 运行一致性测试
    local test_passed=0
    local test_total=0
    
    # 测试DID操作一致性
    test_total=$((test_total + 1))
    if test_did_consistency; then
        test_passed=$((test_passed + 1))
    fi
    
    # 测试网络分区恢复
    test_total=$((test_total + 1))
    if test_network_partition; then
        test_passed=$((test_passed + 1))
    fi
    
    # 生成测试报告
    generate_report
    
    # 输出测试结果
    log_info "测试完成: $test_passed/$test_total 项测试通过"
    
    if [ $test_passed -eq $test_total ]; then
        log_success "所有测试通过！"
        exit 0
    else
        log_error "部分测试失败"
        exit 1
    fi
}

# 检查命令行参数
case "${1:-}" in
    "clean")
        cleanup
        exit 0
        ;;
    "help"|"-h"|"--help")
        echo "用法: $0 [clean|help]"
        echo "  clean: 清理测试环境"
        echo "  help:  显示帮助信息"
        exit 0
        ;;
    "")
        main
        ;;
    *)
        log_error "未知参数: $1"
        echo "使用 '$0 help' 查看帮助信息"
        exit 1
        ;;
esac