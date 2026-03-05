# SCIM 2.0 API 配置快速入门

本指南帮助您快速配置和启动 SCIM 2.0 API 服务器。

## 快速开始

### 1. 使用默认配置启动

最简单的方式，无需任何配置文件：

```bash
go run main.go
```

服务器将在 `http://localhost:8080` 启动，使用内存存储。

### 2. 使用命令行参数

快速指定关键配置：

```bash
# 指定端口和令牌
go run main.go -port 9090 -token my-secret-token

# 使用 MySQL 存储
go run main.go -storage mysql -mysql-dsn "user:pass@tcp(localhost:3306)/scim"

# 生产模式运行
go run main.go -mode release -log-level warn
```

### 3. 使用配置文件

#### 生成配置文件

```bash
# 生成开发环境配置
go run cmd/config-gen/main.go -output config.yaml

# 生成生产环境配置
go run cmd/config-gen/main.go -mode release -storage mysql -output config.yaml
```

#### 启动服务

```bash
go run main.go -config config.yaml
```

### 4. 使用环境变量

```bash
# 设置环境变量
export SCIM_PORT=8080
export SCIM_TOKEN=my-secret-token
export SCIM_STORAGE_DRIVER=redis
export SCIM_STORAGE_REDIS_URI=redis://localhost:6379/0

# 启动服务
go run main.go
```

### 5. 使用 .env 文件

创建 `.env` 文件：

```bash
cat > .env << EOF
SCIM_MODE=debug
SCIM_PORT=8080
SCIM_TOKEN=dev-token
SCIM_STORAGE_DRIVER=memory
SCIM_LOG_LEVEL=debug
EOF
```

启动服务时会自动加载：

```bash
go run main.go
```

## 配置优先级

当同时使用多种配置方式时，按以下优先级加载：

```
命令行参数 > 环境变量 > 配置文件 > .env文件 > 默认值
```

**示例**：

```bash
# 配置文件 config.yaml 中设置 port: 8080
# 环境变量设置 SCIM_PORT=9090
# 命令行参数指定 -port 3000

# 最终使用的端口是 3000（命令行参数优先级最高）
go run main.go -config config.yaml -port 3000
```

## 常用配置场景

### 场景 1: 本地开发

```bash
# 最简单的方式，使用内存存储
go run main.go

# 或指定端口和调试日志
go run main.go -port 3000 -log-level debug
```

### 场景 2: Docker 部署

```dockerfile
FROM golang:1.21-alpine
WORKDIR /app
COPY . .
RUN go build -o scim-server main.go
EXPOSE 8080
ENV SCIM_MODE=release
ENV SCIM_STORAGE_DRIVER=redis
ENV SCIM_STORAGE_REDIS_URI=redis://redis:6379/0
CMD ["./scim-server"]
```

### 场景 3: Kubernetes 部署

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: scim-config
data:
  SCIM_MODE: "release"
  SCIM_PORT: "8080"
  SCIM_STORAGE_DRIVER: "postgres"
---
apiVersion: v1
kind: Secret
metadata:
  name: scim-secrets
type: Opaque
stringData:
  SCIM_TOKEN: "your-secret-token"
  SCIM_STORAGE_POSTGRES_DSN: "host=postgres user=scim password=secret dbname=scim"
```

### 场景 4: CI/CD 流水线

```yaml
# .github/workflows/deploy.yml
- name: Deploy
  env:
    SCIM_MODE: release
    SCIM_TOKEN: ${{ secrets.SCIM_TOKEN }}
    SCIM_STORAGE_DRIVER: mysql
    SCIM_STORAGE_MYSQL_DSN: ${{ secrets.MYSQL_DSN }}
  run: |
    go run main.go
```

## 敏感信息处理

### 环境变量方式（推荐）

```bash
# 不将敏感信息写入配置文件
export SCIM_TOKEN="your-secret-token"
export SCIM_STORAGE_MYSQL_DSN="user:password@tcp(host:3306)/db"
go run main.go
```

### 加密配置（高级）

```bash
# 1. 设置加密密钥
export SCIM_ENCRYPTION_KEY="your-32-byte-encryption-key-here"

# 2. 在配置中使用加密值
cat > config.yaml << EOF
mode: release
port: "8080"
token: "enc:base64encoded_encrypted_token"
storage:
  driver: mysql
  mysql_dsn: "enc:base64encoded_encrypted_dsn"
EOF

# 3. 启动服务
go run main.go -config config.yaml
```

## 配置验证

系统启动时会自动验证配置：

```bash
# 无效配置会报错并停止启动
go run main.go -port 99999
# 输出: 配置验证失败: 无效的端口号: 99999 (必须是 1-65535 之间的整数)

go run main.go -storage oracle
# 输出: 配置验证失败: 无效的存储驱动: oracle (必须是: [memory redis mysql postgres authing])
```

## 查看配置

启动时会显示配置摘要（敏感信息已隐藏）：

```
=============================================
配置加载完成
=============================================
  port: 8080 (来源: command line)
  mode: debug (来源: default)
  token: ***隐藏*** (来源: environment variable)
  storage.driver: memory (来源: default)
  ...
=============================================
```

## 获取帮助

```bash
# 查看命令行帮助
go run main.go -help

# 查看版本信息
go run main.go -version

# 查看详细文档
cat docs/CONFIGURATION.md
```

## 故障排除

### 端口被占用

```bash
# 更换端口
go run main.go -port 9090
```

### 配置文件找不到

```bash
# 指定正确的配置文件路径
go run main.go -config /path/to/config.yaml

# 或使用默认配置（不指定配置文件）
go run main.go
```

### 存储连接失败

```bash
# 检查连接字符串
go run main.go -storage mysql -mysql-dsn "correct-dsn-format"

# 验证数据库服务
mysql -h localhost -u user -p
```

## 下一步

- 查看 [完整配置文档](CONFIGURATION.md) 了解所有配置选项
- 查看 [API 文档](../README.md) 了解 API 使用方法
- 查看 [部署指南](DEPLOYMENT.md) 了解生产环境部署

---

**提示**: 开发环境建议使用内存存储，生产环境建议使用 MySQL 或 PostgreSQL。
