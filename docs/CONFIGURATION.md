# SCIM 2.0 API 配置文档

本文档详细说明 SCIM 2.0 API 服务器的配置系统，包括配置参数、加载优先级、验证规则和安全机制。

## 目录

- [配置概述](#配置概述)
- [配置优先级](#配置优先级)
- [配置方式](#配置方式)
- [配置参数详解](#配置参数详解)
- [敏感配置加密](#敏感配置加密)
- [配置验证](#配置验证)
- [配置示例](#配置示例)
- [故障排除](#故障排除)

## 配置概述

SCIM 2.0 API 采用灵活的多层配置系统，支持以下配置方式：

1. **命令行参数** - 最高优先级，适合临时覆盖
2. **环境变量** - 适合容器化部署和 CI/CD 环境
3. **配置文件** - 支持 YAML 和 JSON 格式
4. **环境文件** - `.env` 文件，适合本地开发
5. **默认值** - 内置安全默认值

## 配置优先级

配置按以下优先级加载（高优先级覆盖低优先级）：

```
命令行参数 > 环境变量 > 配置文件 > .env文件 > 默认值
```

### 优先级说明

1. **命令行参数** (`-port`, `-token` 等)
   - 最高优先级
   - 适合临时测试和调试
   - 立即生效，无需重启服务

2. **环境变量** (`SCIM_PORT`, `SCIM_TOKEN` 等)
   - 第二优先级
   - 适合 Docker/Kubernetes 部署
   - 支持敏感信息的安全传递

3. **配置文件** (`config.yaml`, `config.json`)
   - 第三优先级
   - 适合生产环境的复杂配置
   - 支持结构化配置和注释

4. **环境文件** (`.env`, `.env.local`)
   - 第四优先级
   - 适合本地开发环境
   - 不提交到版本控制

5. **默认值**
   - 最低优先级
   - 确保系统可用性
   - 开发环境友好

## 配置方式

### 1. 命令行参数

```bash
# 基本用法
go run main.go -port 8080 -token my-secret-token

# 完整示例
go run main.go \
  -config ./config.yaml \
  -port 9090 \
  -storage mysql \
  -mysql-dsn "user:pass@tcp(localhost:3306)/scim" \
  -mode release \
  -log-level info

# 显示帮助
go run main.go -help

# 显示版本
go run main.go -version
```

#### 支持的命令行参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-config` | string | `""` | 配置文件路径 |
| `-port` | string | `"8080"` | 服务端口号 |
| `-token` | string | `""` | 身份验证令牌 |
| `-storage` | string | `"memory"` | 存储驱动类型 |
| `-mode` | string | `"debug"` | 运行模式 |
| `-log-level` | string | `"info"` | 日志级别 |
| `-redis-uri` | string | `""` | Redis 连接 URI |
| `-mysql-dsn` | string | `""` | MySQL DSN |
| `-postgres-dsn` | string | `""` | PostgreSQL DSN |
| `-swagger` | bool | `false` | 启用 Swagger |
| `-help` | bool | `false` | 显示帮助 |
| `-version` | bool | `false` | 显示版本 |

### 2. 环境变量

所有配置项都支持通过环境变量设置，前缀为 `SCIM_`：

```bash
# 基础配置
export SCIM_MODE=release
export SCIM_PORT=8080
export SCIM_TOKEN=your-secret-token
export SCIM_LOG_LEVEL=info

# 存储配置
export SCIM_STORAGE_DRIVER=mysql
export SCIM_STORAGE_REDIS_URI=redis://localhost:6379/0
export SCIM_STORAGE_MYSQL_DSN="user:password@tcp(localhost:3306)/scim?charset=utf8mb4"
export SCIM_STORAGE_POSTGRES_DSN="host=localhost user=postgres password=postgres dbname=scim"

# 分页配置
export SCIM_PAGINATION_DEFAULT_COUNT=20
export SCIM_PAGINATION_MAX_COUNT=100
export SCIM_PAGINATION_CURSOR_SUPPORT=true

# Swagger 配置
export SCIM_SWAGGER_ENABLED=true
export SCIM_SWAGGER_PATH=/swagger/*any

# 加密密钥（用于解密敏感配置）
export SCIM_ENCRYPTION_KEY=your-32-byte-encryption-key
```

### 3. 配置文件

支持 YAML 和 JSON 两种格式：

#### YAML 格式 (`config.yaml`)

```yaml
# 基础配置
mode: release
port: "8080"
token: "your-secret-token"
log_level: info

# 存储配置
storage:
  driver: mysql
  redis_uri: "redis://localhost:6379/0"
  mysql_dsn: "user:password@tcp(localhost:3306)/scim?charset=utf8mb4&parseTime=True&loc=Local"
  postgres_dsn: "host=localhost user=postgres password=postgres dbname=scim port=5432 sslmode=disable"
  authing_host: "https://api.authing.cn"
  authing_user_pool_id: "your-user-pool-id"
  authing_access_key: "your-access-key"
  authing_access_secret: "your-access-secret"

# 分页配置
pagination:
  default_count: 20
  max_count: 100
  cursor_support: true

# SCIM 配置
scim:
  default_schema: "urn:ietf:params:scim:schemas:core:2.0:User"
  group_schema: "urn:ietf:params:scim:schemas:core:2.0:Group"
  error_schema: "urn:ietf:params:scim:api:messages:2.0:Error"
  list_schema: "urn:ietf:params:scim:api:messages:2.0:ListResponse"

# Swagger 配置
swagger:
  enabled: true
  path: "/swagger/*any"
```

#### JSON 格式 (`config.json`)

```json
{
  "mode": "release",
  "port": "8080",
  "token": "your-secret-token",
  "log_level": "info",
  "storage": {
    "driver": "mysql",
    "mysql_dsn": "user:password@tcp(localhost:3306)/scim?charset=utf8mb4&parseTime=True&loc=Local"
  },
  "pagination": {
    "default_count": 20,
    "max_count": 100,
    "cursor_support": true
  },
  "scim": {
    "default_schema": "urn:ietf:params:scim:schemas:core:2.0:User",
    "group_schema": "urn:ietf:params:scim:schemas:core:2.0:Group"
  },
  "swagger": {
    "enabled": true,
    "path": "/swagger/*any"
  }
}
```

### 4. 环境文件 (.env)

创建 `.env` 文件（不提交到版本控制）：

```bash
# 复制示例文件
cp .env.example .env

# 编辑 .env 文件
SCIM_MODE=debug
SCIM_PORT=8080
SCIM_TOKEN=dev-token
SCIM_STORAGE_DRIVER=memory
SCIM_LOG_LEVEL=debug
```

支持的环境文件：
- `.env` - 基础环境配置
- `.env.local` - 本地覆盖配置
- `.env.{mode}` - 模式特定配置（如 `.env.debug`, `.env.release`）

## 配置参数详解

### 基础配置

#### mode
- **类型**: string
- **默认值**: `debug`
- **可选值**: `debug`, `test`, `release`
- **说明**: 应用程序运行模式
  - `debug`: 开发模式，启用详细日志和调试信息
  - `test`: 测试模式，优化测试性能
  - `release`: 生产模式，最小化日志输出，优化性能

#### port
- **类型**: string
- **默认值**: `8080`
- **范围**: 1-65535
- **说明**: HTTP 服务器监听端口

#### token
- **类型**: string
- **默认值**: `""`
- **敏感**: 是
- **说明**: API 身份验证令牌
- **安全**: 支持加密格式 `enc:base64encoded`

#### log_level
- **类型**: string
- **默认值**: `info`
- **可选值**: `debug`, `info`, `warn`, `error`
- **说明**: 日志输出级别

### 存储配置

#### storage.driver
- **类型**: string
- **默认值**: `memory`
- **可选值**: `memory`, `redis`, `mysql`, `postgres`, `authing`
- **说明**: 数据存储后端驱动

#### storage.redis_uri
- **类型**: string
- **默认值**: `""`
- **敏感**: 是
- **说明**: Redis 连接 URI
- **格式**: `redis://[user:password@]host:port/db`

#### storage.mysql_dsn
- **类型**: string
- **默认值**: `""`
- **敏感**: 是
- **说明**: MySQL 数据源名称
- **格式**: `user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True`

#### storage.postgres_dsn
- **类型**: string
- **默认值**: `""`
- **敏感**: 是
- **说明**: PostgreSQL 数据源名称
- **格式**: `host=localhost user=postgres password=postgres dbname=scim port=5432 sslmode=disable`

### 分页配置

#### pagination.default_count
- **类型**: int
- **默认值**: `20`
- **范围**: > 0
- **说明**: 默认分页大小

#### pagination.max_count
- **类型**: int
- **默认值**: `100`
- **范围**: > 0
- **说明**: 最大分页大小

#### pagination.cursor_support
- **类型**: bool
- **默认值**: `true`
- **说明**: 是否启用游标分页

### SCIM 配置

#### scim.default_schema
- **类型**: string
- **默认值**: `urn:ietf:params:scim:schemas:core:2.0:User`
- **说明**: 默认 SCIM Schema

#### scim.group_schema
- **类型**: string
- **默认值**: `urn:ietf:params:scim:schemas:core:2.0:Group`
- **说明**: 组资源 Schema

### Swagger 配置

#### swagger.enabled
- **类型**: bool
- **默认值**: `true`
- **说明**: 是否启用 Swagger 文档

#### swagger.path
- **类型**: string
- **默认值**: `/swagger/*any`
- **说明**: Swagger UI 访问路径

## 敏感配置加密

### 加密机制

系统支持对敏感配置项进行 AES-GCM 加密：

1. **加密格式**: `enc:base64encoded_ciphertext`
2. **加密算法**: AES-256-GCM
3. **密钥来源**: `SCIM_ENCRYPTION_KEY` 环境变量

### 加密步骤

```bash
# 1. 设置加密密钥（32字节）
export SCIM_ENCRYPTION_KEY="your-32-byte-encryption-key-here"

# 2. 使用加密工具加密敏感信息
# （需要实现加密工具函数）

# 3. 在配置中使用加密值
# config.yaml
token: "enc:base64encoded_encrypted_token"
```

### 解密流程

1. 系统启动时检查 `SCIM_ENCRYPTION_KEY` 环境变量
2. 识别以 `enc:` 开头的配置值
3. 使用 AES-GCM 解密算法解密
4. 将解密后的值用于运行时配置

### 安全建议

1. **密钥管理**: 使用密钥管理系统（KMS）或密钥保险库
2. **密钥轮换**: 定期更换加密密钥
3. **访问控制**: 限制对加密密钥的访问权限
4. **传输安全**: 使用 HTTPS 传输敏感配置

## 配置验证

### 验证规则

系统在启动时自动验证配置：

#### 端口号验证
- 必须是 1-65535 之间的整数
- 端口不能被其他服务占用

#### 运行模式验证
- 必须是 `debug`, `test`, `release` 之一

#### 日志级别验证
- 必须是 `debug`, `info`, `warn`, `error` 之一

#### 存储驱动验证
- 必须是 `memory`, `redis`, `mysql`, `postgres`, `authing` 之一
- 根据驱动类型验证必需的连接参数

#### 分页配置验证
- `default_count` 必须大于 0
- `max_count` 必须大于 0
- `default_count` 不能大于 `max_count`

#### 存储特定验证

**Redis 驱动**:
- 必须提供 `redis_uri`
- URI 格式必须正确

**MySQL 驱动**:
- 必须提供 `mysql_dsn`
- DSN 格式必须正确

**PostgreSQL 驱动**:
- 必须提供 `postgres_dsn`
- DSN 格式必须正确

### 验证错误处理

当配置验证失败时：

1. **错误收集**: 收集所有验证错误
2. **错误报告**: 输出详细的错误信息
3. **启动终止**: 阻止服务启动，避免运行时不稳定

示例错误输出：
```
配置验证失败: 无效的端口号: abc (必须是 1-65535 之间的整数); 
无效的存储驱动: oracle (必须是: [memory redis mysql postgres authing]); 
使用 MySQL 存储时必须提供 MySQL DSN
```

## 配置示例

### 开发环境配置

```yaml
# config.yaml
mode: debug
port: "8080"
token: "dev-token"
log_level: debug
storage:
  driver: memory
swagger:
  enabled: true
```

```bash
# 或使用命令行
go run main.go -mode debug -port 8080 -token dev-token -storage memory
```

### 生产环境配置

```yaml
# config.yaml
mode: release
port: "443"
token: "enc:base64encoded_token"
log_level: warn
storage:
  driver: mysql
  mysql_dsn: "enc:base64encoded_dsn"
pagination:
  default_count: 50
  max_count: 200
swagger:
  enabled: false
```

```bash
# 环境变量
export SCIM_ENCRYPTION_KEY="your-production-key"
export SCIM_MODE=release
export SCIM_PORT=443
export SCIM_STORAGE_DRIVER=mysql

# 启动服务
go run main.go -config ./config.yaml
```

### Docker 部署配置

```dockerfile
# Dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o scim-server main.go
EXPOSE 8080
CMD ["./scim-server", "-config", "/app/config.yaml"]
```

```yaml
# docker-compose.yml
version: '3.8'
services:
  scim:
    build: .
    ports:
      - "8080:8080"
    environment:
      - SCIM_MODE=release
      - SCIM_STORAGE_DRIVER=mysql
      - SCIM_STORAGE_MYSQL_DSN=user:pass@tcp(mysql:3306)/scim
    depends_on:
      - mysql
  
  mysql:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=password
      - MYSQL_DATABASE=scim
```

### Kubernetes 配置

```yaml
# configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: scim-config
data:
  mode: "release"
  port: "8080"
  log_level: "info"
  storage_driver: "postgres"
---
# secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: scim-secrets
type: Opaque
stringData:
  token: "your-secret-token"
  postgres_dsn: "host=postgres user=scim password=secret dbname=scim"
---
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: scim-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: scim-server
  template:
    metadata:
      labels:
        app: scim-server
    spec:
      containers:
      - name: scim
        image: scim-server:latest
        ports:
        - containerPort: 8080
        env:
        - name: SCIM_MODE
          valueFrom:
            configMapKeyRef:
              name: scim-config
              key: mode
        - name: SCIM_TOKEN
          valueFrom:
            secretKeyRef:
              name: scim-secrets
              key: token
```

## 故障排除

### 常见问题

#### 1. 配置文件加载失败

**症状**: `加载配置文件失败: 读取配置文件失败: open ./config.yaml: no such file or directory`

**解决方案**:
```bash
# 检查文件是否存在
ls -la config.yaml

# 指定正确的配置文件路径
go run main.go -config /path/to/config.yaml

# 或使用默认配置（不指定配置文件）
go run main.go
```

#### 2. 配置验证失败

**症状**: `配置验证失败: 无效的端口号: 99999`

**解决方案**:
```bash
# 检查端口号范围
go run main.go -port 8080

# 查看帮助了解有效值
go run main.go -help
```

#### 3. 敏感配置解密失败

**症状**: `解密敏感配置失败: 解密 token 失败: cipher: message authentication failed`

**解决方案**:
```bash
# 检查加密密钥
export SCIM_ENCRYPTION_KEY="your-correct-key"

# 验证加密值格式
echo "enc:base64encoded_value"  # 必须以 enc: 开头
```

#### 4. 存储连接失败

**症状**: `mysql connect failed: dial tcp: connect: connection refused`

**解决方案**:
```bash
# 检查数据库服务状态
mysql -h localhost -u user -p

# 验证 DSN 格式
go run main.go -mysql-dsn "correct-dsn-format"

# 检查网络连接
telnet localhost 3306
```

### 调试配置

启用调试日志查看配置加载过程：

```bash
# 设置日志级别为 debug
export SCIM_LOG_LEVEL=debug

# 启动服务查看详细日志
go run main.go
```

### 配置验证工具

```bash
# 验证配置文件语法
go run main.go -config ./config.yaml

# 测试配置加载（不启动服务）
SCIM_TEST_CONFIG=true go run main.go -config ./config.yaml
```

### 获取帮助

```bash
# 显示命令行帮助
go run main.go -help

# 显示版本信息
go run main.go -version
```

## 最佳实践

### 1. 环境分离

- 开发环境: `.env` 文件 + 内存存储
- 测试环境: 环境变量 + 测试数据库
- 生产环境: 配置文件 + 加密密钥 + 生产数据库

### 2. 敏感信息保护

- 使用加密格式存储敏感配置
- 通过环境变量传递加密密钥
- 定期轮换加密密钥
- 使用密钥管理系统（KMS）

### 3. 配置版本控制

- 提交示例配置文件（`config.example.yaml`）
- 忽略实际配置文件（`config.yaml`）
- 使用模板工具管理环境特定配置

### 4. 监控和告警

- 监控配置加载失败事件
- 设置敏感配置访问告警
- 记录配置变更审计日志

---

如有其他问题，请参考项目文档或提交 Issue。
