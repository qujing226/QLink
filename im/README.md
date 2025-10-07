# QLink IM - 去中心化即时通讯系统

基于DID（去中心化身份）的安全即时通讯系统，支持端到端加密和WebSocket实时通信。

## 功能特性

- 🔐 基于DID的去中心化身份认证
- 🔒 端到端加密消息传输
- 🚀 WebSocket实时通信
- 👥 好友管理系统
- 📱 RESTful API接口
- 🛡️ 安全的密钥交换机制

## 技术架构

### 后端技术栈
- **语言**: Go 1.21+
- **Web框架**: Gin
- **数据库**: SQLite (GORM)
- **WebSocket**: Gorilla WebSocket
- **加密**: AES-256-GCM
- **认证**: JWT

### 项目结构
```
internal/
├── api/           # HTTP API处理器
├── config/        # 配置管理
├── errors/        # 统一错误处理
├── logger/        # 日志记录
├── middleware/    # 中间件
├── models/        # 数据模型
├── service/       # 业务逻辑
├── storage/       # 数据存储
└── websocket/     # WebSocket管理
```

## 快速开始

### 测试
did:qlink:TUZrd0V3WUhLb1pJemowQ0FRWUlLb1pJ
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEqLj22HGo7HuqfhBhgEELJhaYTwpTfNiVIWU3tMzC/t5nWlT4u78Jcsm5enGagquUgqDBUOSMbuga5Z4c1P1nCw==
MIGHAgEAMBMGByqGSM49AgEGCCqGSM49AwEHBG0wawIBAQQgJL6wuqMEEpHSuU6ZHU0F1vjRp5r8eT4d+hzTBCQ1J1WhRANCAASouPbYcajse6p+EGGAQQsmFphPClN82JUhZTe0zML+3mdaVPi7vwlyybl6cZqCq5SCoMFQ5Ixu6BrlnhzU/WcL


### 环境要求
- Go 1.21 或更高版本
- SQLite3

### 安装依赖
```bash
go mod tidy
```

### 配置环境变量
```bash
export SERVER_HOST=localhost
export SERVER_PORT=8080
export DATABASE_URL=./qlink.db
export JWT_SECRET=your-secret-key
export LOG_LEVEL=info
```

### 编译运行
```bash
# 编译
go build -o qlink-im .

# 运行
./qlink-im
```

### 开发模式
```bash
go run main.go
```

## API 接口

### 认证接口
- `POST /api/auth/login` - 用户登录
- `POST /api/auth/logout` - 用户登出

### 好友管理
- `GET /api/friends` - 获取好友列表
- `POST /api/friends/request` - 发送好友请求
- `POST /api/friends/accept` - 接受好友请求
- `POST /api/friends/reject` - 拒绝好友请求

### 消息接口
- `GET /api/messages/:friendDID` - 获取消息历史
- `POST /api/messages/send` - 发送消息

### WebSocket
- `GET /ws` - WebSocket连接端点

## 配置说明

### 环境变量
| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| SERVER_HOST | localhost | 服务器主机 |
| SERVER_PORT | 8080 | 服务器端口 |
| DATABASE_URL | ./qlink.db | 数据库文件路径 |
| JWT_SECRET | your-secret-key | JWT密钥 |
| LOG_LEVEL | info | 日志级别 |
| DID_NODE_URL | http://localhost:8080 | DID网关URL（推荐） |

## 开发指南

### 代码规范
- 遵循Go官方代码规范
- 使用gofmt格式化代码
- 运行go vet检查代码质量
- 添加适当的注释和文档

### 测试
```bash
# 运行所有测试
go test ./...

# 运行代码检查
go vet ./...
```

### 构建
```bash
# 本地构建
go build -o qlink-im .

# 交叉编译
GOOS=linux GOARCH=amd64 go build -o qlink-im-linux .

### 快速启动（推荐）

```
# DID-System 网关
cd ../did-system && make gateway

# IM 后端
cd ../im && make run

# 前端
cd ../im-front && npm run dev
```
```

## 安全特性

- **端到端加密**: 使用AES-256-GCM加密算法
- **密钥派生**: 基于PBKDF2的安全密钥派生
- **身份验证**: 基于DID的去中心化身份认证
- **会话管理**: 安全的JWT令牌管理
- **速率限制**: 防止API滥用的速率限制

## 许可证

MIT License

## 贡献

欢迎提交Issue和Pull Request来改进项目。

## 联系方式

如有问题或建议，请通过GitHub Issues联系我们。