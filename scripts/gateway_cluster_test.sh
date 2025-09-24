#!/bin/bash

# QLink 网关集群测试脚本
# 架构：1个API网关节点 + 3个纯共识节点

set -e

# 配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BIN_PATH="$PROJECT_ROOT/bin/qlink-node"
CONFIG_DIR="$PROJECT_ROOT/config"
DATA_DIR="$PROJECT_ROOT/data"
LOGS_DIR="$PROJECT_ROOT/logs"

# 节点配置
GATEWAY_CONFIG="$CONFIG_DIR/gateway_node.yaml"
CONSENSUS1_CONFIG="$CONFIG_DIR/consensus_node1.yaml"
CONSENSUS2_CONFIG="$CONFIG_DIR/consensus_node2.yaml"
CONSENSUS3_CONFIG="$CONFIG_DIR/consensus_node3.yaml"

# 创建必要的目录
create_directories() {
    echo "创建必要的目录..."
    mkdir -p "$DATA_DIR"/{gateway,consensus1,consensus2,consensus3}
    mkdir -p "$LOGS_DIR"
    mkdir -p "$DATA_DIR"/{gateway,consensus1,consensus2,consensus3}/storage
}

# 构建项目
build_project() {
    echo "构建项目..."
    cd "$PROJECT_ROOT"
    go build -o "$BIN_PATH" cmd/qlink-node/main.go
    echo "项目构建完成"
}

# 启动节点
start_node() {
    local node_name=$1
    local config_file=$2
    local log_file="$LOGS_DIR/${node_name}.log"
    local pid_file="$LOGS_DIR/${node_name}.pid"
    
    echo "启动 $node_name..."
    
    # 检查配置文件是否存在
    if [ ! -f "$config_file" ]; then
        echo "错误: 配置文件 $config_file 不存在"
        return 1
    fi
    
    # 启动节点
    nohup "$BIN_PATH" -config="$config_file" > "$log_file" 2>&1 &
    local pid=$!
    echo $pid > "$pid_file"
    
    echo "$node_name 已启动 (PID: $pid)"
    sleep 2
    
    # 检查进程是否还在运行
    if ! kill -0 $pid 2>/dev/null; then
        echo "错误: $node_name 启动失败"
        echo "查看日志: tail -f $log_file"
        return 1
    fi
    
    return 0
}

# 停止节点
stop_node() {
    local node_name=$1
    local pid_file="$LOGS_DIR/${node_name}.pid"
    
    if [ -f "$pid_file" ]; then
        local pid=$(cat "$pid_file")
        if kill -0 $pid 2>/dev/null; then
            echo "停止 $node_name (PID: $pid)..."
            kill $pid
            sleep 2
            
            # 强制杀死如果还在运行
            if kill -0 $pid 2>/dev/null; then
                echo "强制停止 $node_name..."
                kill -9 $pid
            fi
        fi
        rm -f "$pid_file"
    fi
}

# 启动集群
start_cluster() {
    echo "启动 QLink 网关集群..."
    
    create_directories
    build_project
    
    # 按顺序启动节点
    echo "启动共识节点..."
    start_node "consensus1" "$CONSENSUS1_CONFIG" || return 1
    sleep 3
    
    start_node "consensus2" "$CONSENSUS2_CONFIG" || return 1
    sleep 3
    
    start_node "consensus3" "$CONSENSUS3_CONFIG" || return 1
    sleep 3
    
    echo "启动API网关节点..."
    start_node "gateway" "$GATEWAY_CONFIG" || return 1
    sleep 3
    
    echo "集群启动完成！"
    echo "API网关地址: http://localhost:8080"
    echo ""
    show_status
}

# 停止集群
stop_cluster() {
    echo "停止 QLink 网关集群..."
    
    stop_node "gateway"
    stop_node "consensus1"
    stop_node "consensus2"
    stop_node "consensus3"
    
    echo "集群已停止"
}

# 显示状态
show_status() {
    echo "=== 集群状态 ==="
    
    for node in gateway consensus1 consensus2 consensus3; do
        local pid_file="$LOGS_DIR/${node}.pid"
        if [ -f "$pid_file" ]; then
            local pid=$(cat "$pid_file")
            if kill -0 $pid 2>/dev/null; then
                echo "✓ $node: 运行中 (PID: $pid)"
            else
                echo "✗ $node: 已停止"
            fi
        else
            echo "✗ $node: 未启动"
        fi
    done
    
    echo ""
    echo "API网关: http://localhost:8080"
    echo "日志目录: $LOGS_DIR"
}

# 查看日志
show_logs() {
    local node_name=${1:-"gateway"}
    local log_file="$LOGS_DIR/${node_name}.log"
    
    if [ -f "$log_file" ]; then
        echo "=== $node_name 日志 ==="
        tail -f "$log_file"
    else
        echo "日志文件不存在: $log_file"
    fi
}

# 测试API
test_api() {
    echo "测试API功能..."
    
    # 等待API服务启动
    echo "等待API服务启动..."
    sleep 5
    
    # 测试健康检查
    echo "1. 测试健康检查..."
    if curl -s http://localhost:8080/health > /dev/null; then
        echo "✓ 健康检查通过"
    else
        echo "✗ 健康检查失败"
        return 1
    fi
    
    # 测试DID注册
    echo "2. 测试DID注册..."
    local test_did="did:qlink:test$(date +%s)"
    local response=$(curl -s -X POST http://localhost:8080/api/v1/did/register \
        -H "Content-Type: application/json" \
        -d "{\"did\":\"$test_did\",\"document\":{\"id\":\"$test_did\"}}")
    
    if echo "$response" | grep -q "success\|registered"; then
        echo "✓ DID注册成功: $test_did"
    else
        echo "✗ DID注册失败: $response"
    fi
    
    # 测试批量注册
    echo "3. 测试批量DID注册..."
    local batch_response=$(curl -s -X POST http://localhost:8080/api/v1/did/batch-register \
        -H "Content-Type: application/json" \
        -d "{\"dids\":[\"did:qlink:batch1\",\"did:qlink:batch2\"]}")
    
    if echo "$batch_response" | grep -q "success\|registered"; then
        echo "✓ 批量DID注册成功"
    else
        echo "✗ 批量DID注册失败: $batch_response"
    fi
    
    echo "API测试完成"
}

# 压力测试
stress_test() {
    echo "开始压力测试..."
    
    local concurrent_requests=10
    local total_requests=100
    
    echo "并发请求数: $concurrent_requests"
    echo "总请求数: $total_requests"
    
    # 使用ab进行压力测试
    if command -v ab > /dev/null; then
        echo "使用Apache Bench进行压力测试..."
        ab -n $total_requests -c $concurrent_requests http://localhost:8080/health
    else
        echo "Apache Bench未安装，使用curl进行简单测试..."
        for i in $(seq 1 $total_requests); do
            curl -s http://localhost:8080/health > /dev/null &
            if [ $((i % concurrent_requests)) -eq 0 ]; then
                wait
            fi
        done
        wait
        echo "完成 $total_requests 个请求"
    fi
}

# 清理数据
clean_data() {
    echo "清理数据..."
    
    stop_cluster
    
    rm -rf "$DATA_DIR"/{gateway,consensus1,consensus2,consensus3}
    rm -f "$LOGS_DIR"/*.log
    rm -f "$LOGS_DIR"/*.pid
    
    echo "数据清理完成"
}

# 主函数
main() {
    case "${1:-help}" in
        "start")
            start_cluster
            ;;
        "stop")
            stop_cluster
            ;;
        "restart")
            stop_cluster
            sleep 2
            start_cluster
            ;;
        "status")
            show_status
            ;;
        "logs")
            show_logs "$2"
            ;;
        "test")
            test_api
            ;;
        "stress")
            stress_test
            ;;
        "clean")
            clean_data
            ;;
        "help"|*)
            echo "QLink 网关集群测试脚本"
            echo ""
            echo "用法: $0 <command> [options]"
            echo ""
            echo "命令:"
            echo "  start     启动集群"
            echo "  stop      停止集群"
            echo "  restart   重启集群"
            echo "  status    显示集群状态"
            echo "  logs      查看日志 [node_name]"
            echo "  test      测试API功能"
            echo "  stress    压力测试"
            echo "  clean     清理数据"
            echo "  help      显示帮助"
            echo ""
            echo "架构说明:"
            echo "  - gateway: API网关节点 (端口8080)"
            echo "  - consensus1-3: 纯共识节点 (端口30301-30303)"
            echo ""
            ;;
    esac
}

# 执行主函数
main "$@"