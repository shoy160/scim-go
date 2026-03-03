# SCIM 2.0 服务实现

## 项目概述

本项目是一个完整的 **SCIM 2.0 (System for Cross-domain Identity Management)** 服务实现，提供标准化的用户和组管理功能，支持跨域身份管理和单点登录集成。

- **符合 RFC 7644 标准**：完全实现 SCIM 2.0 协议规范
- **多存储引擎支持**：内存、Redis、MySQL、PostgreSQL
- **安全认证**：基于 Bearer Token 的认证机制
- **完整的 API 文档**：集成 Swagger UI，提供交互式 API 文档
- **容器化部署**：支持 Docker 容器化部署

## 技术栈

- **后端**：Go 1.25.0
- **Web 框架**：Gin
- **存储引擎**：内存、Redis、MySQL、PostgreSQL
- **认证**：Bearer Token
- **API 文档**：Swagger 2.0
- **容器化**：Docker

## 目录结构

```
scim-go/
├── api/             # API 接口实现
│   ├── api.go       # 路由注册
│   ├── handler.go   # 接口处理器
│   └── middleware.go # 中间件
├── config/          # 配置管理
├── docs/            # Swagger 文档
├── model/           # 数据模型
├── store/           # 存储实现
├── util/            # 工具函数
├── .dockerignore    # Docker 忽略文件
├── Dockerfile       # Docker 构建文件
├── config.yaml      # 配置文件
├── docker-compose.yml # Docker Compose 配置
├── go.mod           # Go 模块文件
├── go.sum           # 依赖校验文件
├── main.go          # 主入口
└── README.md        # 项目文档
```

## 安装步骤

### 1. 环境要求

- **Go 1.25.0** 或更高版本
- **可选**：Redis、MySQL 或 PostgreSQL（用于持久化存储）
- **可选**：Docker 和 Docker Compose（用于容器化部署）

### 2. 克隆项目

```bash
git clone <repository-url>
cd scim-go
```

### 3. 安装依赖

```bash
go mod download
go mod tidy
```

### 4. 构建项目

```bash
go build -o scim-server .
```

## 配置说明

### 配置文件（config.yaml）

```yaml
# 运行模式：debug/release
mode: release
# 服务端口
port: 8080
# SCIM Bearer Token认证密钥（生产请替换为强随机字符串）
token: "aGCIhV2JtgAezYduMpE1rK6Omy"
# 日志级别：debug/info/warn/error
log_level: info

# 存储配置：memory/redis/mysql/postgres
storage:
  driver: memory
  # redis_uri: "redis://redis:6379/0"
  # mysql_dsn: "root:password@tcp(localhost:3306)/scim?charset=utf8mb4&parseTime=True&loc=Local"
  # postgres_dsn: "host=localhost user=postgres password=postgres dbname=scim port=5432 sslmode=disable"

# 分页配置（SCIM标准）
pagination:
  default_count: 20
  max_count: 100
  # 游标分页开关（RFC 9865）
  cursor_support: true

# SCIM协议配置
scim:
  # 支持的Schema
  default_schema: "urn:ietf:params:scim:schemas:core:2.0:User"
  group_schema: "urn:ietf:params:scim:schemas:core:2.0:Group"
  error_schema: "urn:ietf:params:scim:api:messages:2.0:Error"
  list_schema: "urn:ietf:params:scim:api:messages:2.0:ListResponse"
```

### 环境变量覆盖

可以通过环境变量覆盖配置文件中的设置：

| 环境变量 | 对应配置 | 说明 |
|---------|---------|------|
| `PORT` | `port` | 服务端口 |
| `SCIM_MODE` | `mode` | 运行模式 |
| `SCIM_TOKEN` | `token` | 认证令牌 |
| `SCIM_STORAGE_DRIVER` | `storage.driver` | 存储引擎 |
| `SCIM_STORAGE_REDIS_URI` | `storage.redis_uri` | Redis 连接 URI |
| `SCIM_STORAGE_MYSQL_DSN` | `storage.mysql_dsn` | MySQL 连接 DSN |
| `SCIM_STORAGE_POSTGRES_DSN` | `storage.postgres_dsn` | PostgreSQL 连接 DSN |

## 使用方法

### 1. 启动服务

#### 内存存储模式（默认）

```bash
./scim-server
```

#### Redis 存储模式

```bash
SCIM_STORAGE_DRIVER=redis SCIM_STORAGE_REDIS_URI="redis://localhost:6379/0" ./scim-server
```

#### MySQL 存储模式

```bash
SCIM_STORAGE_DRIVER=mysql SCIM_STORAGE_MYSQL_DSN="root:password@tcp(localhost:3306)/scim?charset=utf8mb4&parseTime=True&loc=Local" ./scim-server
```

#### PostgreSQL 存储模式

```bash
SCIM_STORAGE_DRIVER=postgres SCIM_STORAGE_POSTGRES_DSN="host=localhost user=postgres password=postgres dbname=scim port=5432 sslmode=disable" ./scim-server
```

### 2. 访问 API 文档

服务启动后，可通过以下地址访问 Swagger API 文档：

**地址**：`http://localhost:8080/swagger/index.html`

**认证**：需要在右上角点击 "Authorize" 按钮，输入 Bearer Token（默认：`aGCIhV2JtgAezYduMpE1rK6Omy`）

### 3. 调用 API 示例

#### 获取用户列表

```bash
curl -H "Authorization: Bearer aGCIhV2JtgAezYduMpE1rK6Omy" http://localhost:8080/scim/v2/Users
```

#### 创建用户

```bash
curl -X POST -H "Authorization: Bearer aGCIhV2JtgAezYduMpE1rK6Omy" -H "Content-Type: application/json" -d '{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:User"],
  "userName": "johndoe",
  "name": {
    "givenName": "John",
    "familyName": "Doe"
  },
  "emails": [
    {
      "value": "john.doe@example.com",
      "type": "work",
      "primary": true
    }
  ],
  "active": true
}' http://localhost:8080/scim/v2/Users
```

#### 获取服务配置

```bash
curl http://localhost:8080/scim/v2/ServiceProviderConfig
```

## API 文档

### Swagger 文档生成

本项目集成了 Swagger 2.0 文档系统，提供交互式 API 文档。以下是生成和访问 Swagger 文档的详细步骤：

#### 1. 安装依赖包

```bash
# 安装 Swagger 核心依赖
go get -u github.com/swaggo/swag/cmd/swag
go get -u github.com/swaggo/gin-swagger
go get -u github.com/swaggo/files

# 安装 Swagger 命令行工具
go install github.com/swaggo/swag/cmd/swag@latest
```

#### 2. 生成 Swagger 文档

在项目根目录执行以下命令：

```bash
# 使用 swag 命令生成文档
swag init

# 或使用 go run 方式（如果 swag 命令未安装到 PATH）
go run github.com/swaggo/swag/cmd/swag@latest init
```

生成的文档文件会存储在 `docs/` 目录中：
- `docs/docs.go` - Go 代码形式的文档
- `docs/swagger.json` - JSON 格式的文档
- `docs/swagger.yaml` - YAML 格式的文档

#### 3. 配置说明

Swagger 文档的配置信息定义在 `main.go` 文件的注释中：

```go
// @title SCIM 2.0 API
// @version 1.0.0
// @description SCIM (System for Cross-domain Identity Management) 2.0 实现，提供用户和组管理功能
// @termsOfService https://example.com/terms

// @contact.name API Support
// @contact.url https://example.com/support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8000
// @BasePath /scim/v2

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description 请输入 "Bearer {token}" 格式的认证令牌
```

#### 4. 访问 Swagger 文档

服务启动后，可通过以下地址访问 Swagger UI：

**地址**：`http://localhost:8080/swagger/index.html`

**认证**：需要在右上角点击 "Authorize" 按钮，输入 Bearer Token（默认：`aGCIhV2JtgAezYduMpE1rK6Omy`）

**Swagger JSON 文档**：`http://localhost:8080/swagger/doc.json`

#### 5. 自动生成说明

- **API 注释**：所有 API 接口都已添加了 Swagger 注释，包括接口路径、请求方法、参数说明、响应格式等
- **模型定义**：数据模型（如 User、Group、Email 等）会自动从代码中生成
- **文档更新**：每次修改 API 或模型后，需要重新运行 `swag init` 命令更新文档

### 核心 API 端点

| 方法 | 路径 | 描述 | 认证 |
|------|------|------|------|
| `GET` | `/scim/v2/ServiceProviderConfig` | 获取服务配置 | 否 |
| `GET` | `/scim/v2/ResourceTypes` | 获取资源类型 | 是 |
| `GET` | `/scim/v2/Schemas` | 获取 schema 定义 | 是 |
| `GET` | `/scim/v2/Users` | 列出用户 | 是 |
| `POST` | `/scim/v2/Users` | 创建用户 | 是 |
| `GET` | `/scim/v2/Users/{id}` | 获取用户 | 是 |
| `PUT` | `/scim/v2/Users/{id}` | 更新用户 | 是 |
| `PATCH` | `/scim/v2/Users/{id}` | 补丁更新用户 | 是 |
| `DELETE` | `/scim/v2/Users/{id}` | 删除用户 | 是 |
| `GET` | `/scim/v2/Groups` | 列出组 | 是 |
| `POST` | `/scim/v2/Groups` | 创建组 | 是 |
| `GET` | `/scim/v2/Groups/{id}` | 获取组 | 是 |
| `PUT` | `/scim/v2/Groups/{id}` | 更新组 | 是 |
| `PATCH` | `/scim/v2/Groups/{id}` | 补丁更新组 | 是 |
| `DELETE` | `/scim/v2/Groups/{id}` | 删除组 | 是 |
| `POST` | `/scim/v2/Groups/{id}/members` | 添加用户到组 | 是 |
| `DELETE` | `/scim/v2/Groups/{id}/members/{userId}` | 从组移除用户 | 是 |

### 健康检查

- **地址**：`http://localhost:8080/health`
- **方法**：`GET`
- **认证**：否
- **响应**：`{"status":"ok","service":"scim-server","version":"1.0.0"}`

## Docker 部署

### 1. 构建 Docker 镜像

```bash
docker build -t scim-go .
```

### 2. 运行容器

```bash
docker run -d -p 8080:8080 --name scim-go \
  -e SCIM_STORAGE_DRIVER=memory \
  scim-go
```

### 3. 使用 Docker Compose

```bash
docker-compose up -d
```

## 常见问题解决

### 1. 端口被占用

**问题**：`listen tcp :8080: bind: address already in use`

**解决**：使用不同端口启动服务

```bash
PORT=8000 ./scim-server
```

### 2. 数据库连接失败

**问题**：`mysql connect failed: dial tcp 127.0.0.1:3306: connect: connection refused`

**解决**：
- 确保数据库服务正在运行
- 检查连接字符串是否正确
- 尝试使用内存存储模式进行测试

### 3. 认证失败

**问题**：`{"schemas":"urn:ietf:params:scim:api:messages:2.0:Error","detail":"Invalid or missing Bearer Token","status":401,"scimType":"invalidToken"}`

**解决**：
- 确保请求头中包含正确的 `Authorization: Bearer {token}`
- 检查配置文件中的 `token` 值

### 4. Swagger 文档访问失败

**问题**：Swagger UI 无法加载或提示认证错误

**解决**：
- 确保服务正在运行
- 在 Swagger UI 中正确输入 Bearer Token
- 检查浏览器控制台是否有网络错误

## 注意事项

1. **安全配置**：
   - 生产环境中请修改默认的 `token` 值为强随机字符串
   - 避免使用内存存储模式，建议使用 Redis 或数据库存储

2. **性能优化**：
   - 对于大规模部署，建议使用 Redis 存储以提高性能
   - 合理设置分页参数，避免一次性返回过多数据

3. **扩展性**：
   - 项目采用模块化设计，可根据需要扩展新的存储引擎
   - 支持添加自定义中间件和处理器

4. **兼容性**：
   - 完全符合 RFC 7644 标准，与主流 SCIM 客户端兼容
   - 支持大小写不敏感的路径处理

## 许可证

本项目采用 **Apache 2.0** 许可证。详情请参阅 [LICENSE](LICENSE) 文件。

## 贡献

欢迎提交问题和 pull request 来改进这个项目！

## 联系方式

- **项目维护者**：SCIM API Support
- **Email**：shoy160@qq.com
- **文档**：http://localhost:8080/swagger/index.html
