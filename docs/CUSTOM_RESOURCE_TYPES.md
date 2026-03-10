# Custom Resource Types 使用指南

## 概述

Custom Resource Types 是 SCIM 2.0 协议的扩展功能，允许系统管理员定义和管理非标准的 SCIM 资源类型，以满足特定业务需求。本指南将详细介绍如何使用 Custom Resource Types 功能。

## 功能特性

- **自定义资源类型管理**：创建、读取、更新和删除自定义资源类型
- **自定义资源管理**：对自定义资源执行 CRUD 操作
- **模式验证**：确保自定义资源类型和资源符合 SCIM 2.0 协议规范
- **与现有资源集成**：支持引用现有的 SCIM 资源（如 User、Group）
- **完整的错误处理**：提供符合 SCIM 2.0 规范的错误响应

## API 端点

### 自定义资源类型管理

| 方法 | 端点 | 描述 |
|------|------|------|
| POST | `/scim/v2/CustomResourceTypes` | 创建新的自定义资源类型 |
| GET | `/scim/v2/CustomResourceTypes/{id}` | 获取指定的自定义资源类型 |
| GET | `/scim/v2/CustomResourceTypes` | 列出自定义资源类型 |
| PUT | `/scim/v2/CustomResourceTypes/{id}` | 更新自定义资源类型 |
| DELETE | `/scim/v2/CustomResourceTypes/{id}` | 删除自定义资源类型 |

### 自定义资源管理

| 方法 | 端点 | 描述 |
|------|------|------|
| GET | `/scim/v2/{resourceType}` | 列出指定类型的自定义资源 |
| POST | `/scim/v2/{resourceType}` | 创建新的自定义资源 |
| GET | `/scim/v2/{resourceType}/{resourceID}` | 获取指定的自定义资源 |
| PUT | `/scim/v2/{resourceType}/{resourceID}` | 更新自定义资源 |
| PATCH | `/scim/v2/{resourceType}/{resourceID}` | 补丁更新自定义资源 |
| DELETE | `/scim/v2/{resourceType}/{resourceID}` | 删除自定义资源 |

## 使用示例

### 1. 创建自定义资源类型

**请求**：
```json
POST /scim/v2/CustomResourceTypes
Content-Type: application/json

{
  "id": "Device",
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:ResourceType"],
  "name": "Device",
  "endpoint": "/Devices",
  "description": "A device managed by the system",
  "schema": "urn:example:params:scim:schemas:core:2.0:Device"
}
```

**响应**：
```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:ResourceType"],
  "id": "Device",
  "name": "Device",
  "endpoint": "http://localhost:8080/scim/v2/Devices",
  "description": "A device managed by the system",
  "schema": "urn:example:params:scim:schemas:core:2.0:Device",
  "meta": {
    "resourceType": "ResourceType",
    "location": "http://localhost:8080/scim/v2/ResourceTypes/Device"
  }
}
```

### 2. 创建自定义资源

**请求**：
```json
POST /scim/v2/Devices
Content-Type: application/json

{
  "schemas": ["urn:example:params:scim:schemas:core:2.0:Device"],
  "externalId": "DEV-001",
  "attributes": {
    "name": "Laptop",
    "model": "MacBook Pro",
    "serialNumber": "ABC123",
    "owner": {
      "value": "user123",
      "$ref": "http://localhost:8080/scim/v2/Users/user123",
      "type": "User"
    }
  }
}
```

**响应**：
```json
{
  "schemas": ["urn:example:params:scim:schemas:core:2.0:Device"],
  "id": "device123",
  "externalId": "DEV-001",
  "resourceType": "Device",
  "attributes": {
    "name": "Laptop",
    "model": "MacBook Pro",
    "serialNumber": "ABC123",
    "owner": {
      "value": "user123",
      "$ref": "http://localhost:8080/scim/v2/Users/user123",
      "type": "User"
    }
  },
  "meta": {
    "resourceType": "Device",
    "location": "http://localhost:8080/scim/v2/Devices/device123"
  }
}
```

### 3. 获取自定义资源

**请求**：
```
GET /scim/v2/Devices/device123
```

**响应**：
```json
{
  "schemas": ["urn:example:params:scim:schemas:core:2.0:Device"],
  "id": "device123",
  "externalId": "DEV-001",
  "resourceType": "Device",
  "attributes": {
    "name": "Laptop",
    "model": "MacBook Pro",
    "serialNumber": "ABC123",
    "owner": {
      "value": "user123",
      "$ref": "http://localhost:8080/scim/v2/Users/user123",
      "type": "User"
    }
  },
  "meta": {
    "resourceType": "Device",
    "location": "http://localhost:8080/scim/v2/Devices/device123"
  }
}
```

### 4. 更新自定义资源

**请求**：
```json
PUT /scim/v2/Devices/device123
Content-Type: application/json

{
  "schemas": ["urn:example:params:scim:schemas:core:2.0:Device"],
  "id": "device123",
  "resourceType": "Device",
  "externalId": "DEV-001",
  "attributes": {
    "name": "Laptop",
    "model": "MacBook Pro 2023",
    "serialNumber": "ABC123",
    "owner": {
      "value": "user123",
      "$ref": "http://localhost:8080/scim/v2/Users/user123",
      "type": "User"
    }
  }
}
```

**响应**：
```json
{
  "schemas": ["urn:example:params:scim:schemas:core:2.0:Device"],
  "id": "device123",
  "resourceType": "Device",
  "externalId": "DEV-001",
  "attributes": {
    "name": "Laptop",
    "model": "MacBook Pro 2023",
    "serialNumber": "ABC123",
    "owner": {
      "value": "user123",
      "$ref": "http://localhost:8080/scim/v2/Users/user123",
      "type": "User"
    }
  },
  "meta": {
    "resourceType": "Device",
    "location": "http://localhost:8080/scim/v2/Devices/device123"
  }
}
```

### 5. 补丁更新自定义资源

**请求**：
```json
PATCH /scim/v2/Devices/device123
Content-Type: application/json

[
  {
    "op": "replace",
    "path": "attributes.model",
    "value": "MacBook Pro 2024"
  }
]
```

**响应**：
```json
{
  "schemas": ["urn:example:params:scim:schemas:core:2.0:Device"],
  "id": "device123",
  "resourceType": "Device",
  "externalId": "DEV-001",
  "attributes": {
    "name": "Laptop",
    "model": "MacBook Pro 2024",
    "serialNumber": "ABC123",
    "owner": {
      "value": "user123",
      "$ref": "http://localhost:8080/scim/v2/Users/user123",
      "type": "User"
    }
  },
  "meta": {
    "resourceType": "Device",
    "location": "http://localhost:8080/scim/v2/Devices/device123"
  }
}
```

### 6. 删除自定义资源

**请求**：
```
DELETE /scim/v2/Devices/device123
```

**响应**：
```
204 No Content
```

### 7. 列出自定义资源

**请求**：
```
GET /scim/v2/Devices?startIndex=1&count=10
```

**响应**：
```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:ListResponse"],
  "totalResults": 1,
  "startIndex": 1,
  "itemsPerPage": 10,
  "Resources": [
    {
      "schemas": ["urn:example:params:scim:schemas:core:2.0:Device"],
      "id": "device123",
      "resourceType": "Device",
      "externalId": "DEV-001",
      "attributes": {
        "name": "Laptop",
        "model": "MacBook Pro 2024",
        "serialNumber": "ABC123",
        "owner": {
          "value": "user123",
          "$ref": "http://localhost:8080/scim/v2/Users/user123",
          "type": "User"
        }
      },
      "meta": {
        "resourceType": "Device",
        "location": "http://localhost:8080/scim/v2/Devices/device123"
      }
    }
  ]
}
```

## 错误处理

Custom Resource Types 功能提供了符合 SCIM 2.0 规范的错误处理机制，以下是常见的错误响应：

### 400 Bad Request

当请求参数无效时返回：

```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Error"],
  "status": 400,
  "scimType": "invalidValue",
  "detail": "Invalid value"
}
```

### 404 Not Found

当资源不存在时返回：

```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Error"],
  "status": 404,
  "scimType": "notFound",
  "detail": "Resource not found"
}
```

### 500 Internal Server Error

当服务器内部错误时返回：

```json
{
  "schemas": ["urn:ietf:params:scim:schemas:core:2.0:Error"],
  "status": 500,
  "scimType": "internalError",
  "detail": "Internal server error"
}
```

## 最佳实践

1. **命名规范**：自定义资源类型的 ID 和名称应使用 PascalCase 命名法（如 `Device`、`Application`）
2. **模式设计**：自定义资源的 schema URI 应遵循 SCIM 2.0 规范，使用 `urn:` 前缀
3. **端点设计**：自定义资源的端点应使用复数形式（如 `/Devices`、`/Applications`）
4. **引用处理**：在自定义资源中引用其他 SCIM 资源时，应使用 `CustomResourceReference` 结构
5. **错误处理**：客户端应正确处理不同类型的错误响应

## 注意事项

1. **权限控制**：Custom Resource Types 功能需要适当的权限控制，确保只有授权用户能够创建和管理自定义资源类型
2. **数据一致性**：删除自定义资源类型时，应确保相关的自定义资源也被正确处理
3. **性能考虑**：对于大量自定义资源的场景，应考虑分页和过滤功能的使用
4. **兼容性**：自定义资源类型的设计应考虑与现有 SCIM 客户端的兼容性

## 结论

Custom Resource Types 功能为 SCIM 2.0 协议提供了强大的扩展能力，允许系统管理员根据特定业务需求定义和管理非标准的 SCIM 资源类型。通过本指南的示例和最佳实践，您可以有效地使用这一功能来满足您的业务需求。