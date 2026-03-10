# SCIM 服务 Kubernetes 部署文档

## 文档信息

| 项目 | 内容 |
|------|------|
| 服务名称 | SCIM Server |
| 镜像地址 | `registry.cn-hangzhou.aliyuncs.com/shay/scim-server` |
| 服务端口 | 8080 |
| SCIM API 路径 | `/scim/v2` |
| 文档版本 | v1.0.0 |
| 更新日期 | 2026-03-06 |

---

## 1. 环境准备要求

### 1.1 Kubernetes 集群版本要求

| 组件 | 最低版本 | 推荐版本 | 说明 |
|------|----------|----------|------|
| Kubernetes | v1.21 | v1.28+ | 支持 HPA v2 版本 |
| etcd | v3.4+ | v3.5+ | 集群状态存储 |
| kubectl | v1.21 | v1.28+ | 集群操作客户端 |
| Helm | v3.8 | v3.14+ | 包管理工具（可选） |

### 1.2 节点资源配置

#### 控制平面节点

| 资源 | 最低配置 | 推荐配置 |
|------|----------|----------|
| CPU | 2 核 | 4 核 |
| 内存 | 4 GB | 8 GB |
| 磁盘 | 50 GB | 100 GB SSD |

#### 工作节点（SCIM 服务）

| 资源 | 最低配置 | 推荐配置 | 说明 |
|------|----------|----------|------|
| CPU | 500m | 1000m | 根据请求量调整 |
| 内存 | 512Mi | 1Gi | 内存模式可选更小 |
| 磁盘 | 10 GB | 20 GB | 用于日志和临时文件 |

> **注意**：如果使用持久化存储（MySQL/PostgreSQL/Redis），需要额外配置相应的数据库资源。

### 1.3 网络策略要求

#### 必需的网络访问

| 方向 | 目标 | 端口 | 说明 |
|------|------|------|------|
| 入站 | SCIM Pod | 8080 | SCIM API 服务端口 |
| 出站 | 数据库 | 3306/5432/6379 | MySQL/PostgreSQL/Redis |
| 出站 | 外部服务 | 443 | Authing 等外部认证服务 |

#### NetworkPolicy 示例

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: scim-network-policy
  namespace: scim-system
spec:
  podSelector:
    matchLabels:
      app: scim-server
  policyTypes:
    - Ingress
    - Egress
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app: ingress-controller
      ports:
        - protocol: TCP
          port: 8080
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              name: database
      ports:
        - protocol: TCP
          port: 3306
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443
```

### 1.4 权限配置要求

#### 必需的 RBAC 权限

| 资源 | 权限 | 说明 |
|------|------|------|
| namespaces | get, list | 读取命名空间信息 |
| deployments | get, list, watch, update, patch | 部署管理 |
| services | get, list, watch, create, update, patch, delete | 服务管理 |
| configmaps | get, list, watch, create, update, patch, delete | 配置管理 |
| secrets | get, list, watch, create, update, patch, delete | 密钥管理 |
| pods | get, list, watch, log | 故障排查 |
| horizontalpodautoscalers | get, list, watch, create, update, patch, delete | 自动扩缩容 |

#### RBAC 角色定义示例

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: scim-server-role
  namespace: scim-system
rules:
  - apiGroups: [""]
    resources: ["configmaps", "secrets", "services", "pods"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: ["autoscaling"]
    resources: ["horizontalpodautoscalers"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
```

---

## 2. 部署前准备

### 2.1 镜像拉取凭证配置

如果镜像仓库需要认证，需要创建 ImagePullSecret：

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: scim-image-pull-secret
  namespace: scim-system
type: kubernetes.io/dockerconfigjson
data:
  # 执行以下命令生成 base64 编码的认证信息
  # echo -n '{"auths":{"registry.cn-hangzhou.aliyuncs.com":{"username":"your-username","password":"your-password","email":"your@email.com","auth":"base64-encoded-credentials"}}}' | base64 -w0
  .dockerconfigjson: <base64-encoded-docker-config>
```

生成认证信息的步骤：

```bash
# 1. 登录镜像仓库
docker login registry.cn-hangzhou.aliyuncs.com

# 2. 获取认证配置
cat ~/.docker/config.json | base64 -w0
```

### 2.2 命名空间创建

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: scim-system
  labels:
    name: scim-system
    environment: production
```

应用命名空间：

```bash
kubectl apply -f namespace.yaml
```

### 2.3 RBAC 权限设置

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: scim-server
  namespace: scim-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: scim-server-role
  namespace: scim-system
rules:
  - apiGroups: [""]
    resources: ["configmaps", "secrets", "services", "pods", "pods/log"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: ["autoscaling"]
    resources: ["horizontalpodautoscalers"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: scim-server-rolebinding
  namespace: scim-system
subjects:
  - kind: ServiceAccount
    name: scim-server
    namespace: scim-system
roleRef:
  kind: Role
  name: scim-server-role
  apiGroup: rbac.authorization.k8s.io
```

应用 RBAC 配置：

```bash
kubectl apply -f rbac.yaml
```

---

## 3. 详细部署步骤

### 3.1 配置管理

#### 3.1.1 ConfigMap 配置

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: scim-config
  namespace: scim-system
data:
  config.yaml: |
    # 运行模式：debug/release
    mode: release
    # 服务端口
    port: "8080"
    # 日志级别：debug/info/warn/error
    log_level: info
    # 存储配置：memory/redis/mysql/postgres/authing
    storage:
      driver: postgres
      postgres_dsn: "host=postgres-service namespace=scim user=scim_user password=${SCIM_DB_PASSWORD} dbname=scim_db port=5432 sslmode=disable connect_timeout=5"
    # 分页配置
    pagination:
      default_count: 20
      max_count: 100
      cursor_support: true
    # SCIM协议配置
    scim:
      default_schema: "urn:ietf:params:scim:schemas:core:2.0:User"
      group_schema: "urn:ietf:params:scim:schemas:core:2.0:Group"
      error_schema: "urn:ietf:params:scim:api:messages:2.0:Error"
      list_schema: "urn:ietf:params:scim:api:messages:2.0:ListResponse"
      api_path: "/scim/v2"
    # Swagger配置（生产环境建议关闭）
    swagger:
      enabled: false
      path: "/swagger"
    # 时间精确度配置
    time_precision:
      level: second
```

#### 3.1.2 Secret 配置

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: scim-secrets
  namespace: scim-system
type: Opaque
stringData:
  # SCIM Bearer Token（生产环境请使用强随机字符串）
  SCIM_TOKEN: "your-secure-token-here"
  # 数据库密码
  SCIM_DB_PASSWORD: "your-database-password"
```

### 3.2 Deployment 资源清单

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: scim-server
  namespace: scim-system
  labels:
    app: scim-server
    version: v1.0.0
spec:
  replicas: 2
  selector:
    matchLabels:
      app: scim-server
  # 滚动更新策略
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: scim-server
        version: v1.0.0
    spec:
      serviceAccountName: scim-server
      # 镜像拉取凭证
      imagePullSecrets:
        - name: scim-image-pull-secret
      containers:
        - name: scim-server
          image: registry.cn-hangzhou.aliyuncs.com/shay/scim-server:latest
          imagePullPolicy: Always
          # 端口配置
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
          # 环境变量配置
          env:
            - name: SCIM_TOKEN
              valueFrom:
                secretKeyRef:
                  name: scim-secrets
                  key: SCIM_TOKEN
            - name: SCIM_DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: scim-secrets
                  key: SCIM_DB_PASSWORD
          # 资源配置
          resources:
            requests:
              cpu: 500m
              memory: 512Mi
            limits:
              cpu: 1000m
              memory: 1Gi
          # 配置文件挂载
          volumeMounts:
            - name: config
              mountPath: /config
              readOnly: true
          # 健康检查探针
          livenessProbe:
            httpGet:
              path: /scim/v2/ServiceProviderConfig
              port: http
            initialDelaySeconds: 30
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
            successThreshold: 1
          readinessProbe:
            httpGet:
              path: /scim/v2/ServiceProviderConfig
              port: http
            initialDelaySeconds: 10
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 3
            successThreshold: 1
          # 启动探针（可选，用于慢启动应用）
          startupProbe:
            httpGet:
              path: /scim/v2/ServiceProviderConfig
              port: http
            initialDelaySeconds: 5
            periodSeconds: 10
            timeoutSeconds: 3
            failureThreshold: 30
      # 配置文件卷
      volumes:
        - name: config
          configMap:
            name: scim-config
            items:
              - key: config.yaml
                path: config.yaml
```

#### 关键参数说明

| 参数 | 说明 | 推荐值 |
|------|------|--------|
| `replicas` | Pod 副本数 | 2-3 |
| `imagePullPolicy` | 镜像拉取策略 | Always |
| `resources.requests.cpu` | CPU 请求 | 500m |
| `resources.requests.memory` | 内存请求 | 512Mi |
| `resources.limits.cpu` | CPU 限制 | 1000m |
| `resources.limits.memory` | 内存限制 | 1Gi |
| `livenessProbe.initialDelaySeconds` | 初始延迟 | 30s |
| `livenessProbe.periodSeconds` | 检查周期 | 10s |
| `readinessProbe.initialDelaySeconds` | 初始延迟 | 10s |
| `readinessProbe.periodSeconds` | 检查周期 | 5s |
| `maxUnavailable` | 更新时最大不可用 | 0 |

### 3.3 Service 资源配置

#### ClusterIP 类型（集群内部访问）

```yaml
apiVersion: v1
kind: Service
metadata:
  name: scim-server
  namespace: scim-system
  labels:
    app: scim-server
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 8080
      targetPort: http
      protocol: TCP
  selector:
    app: scim-server
```

#### NodePort 类型（开发/测试环境）

```yaml
apiVersion: v1
kind: Service
metadata:
  name: scim-server
  namespace: scim-system
  labels:
    app: scim-server
spec:
  type: NodePort
  ports:
    - name: http
      port: 8080
      targetPort: http
      nodePort: 30080
      protocol: TCP
  selector:
    app: scim-server
```

#### LoadBalancer 类型（云环境）

```yaml
apiVersion: v1
kind: Service
metadata:
  name: scim-server
  namespace: scim-system
  labels:
    app: scim-server
  annotations:
    # 阿里云 SLB 注解示例
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-type: slb
spec:
  type: LoadBalancer
  ports:
    - name: http
      port: 8080
      targetPort: http
      protocol: TCP
  selector:
    app: scim-server
```

### 3.4 Ingress 资源配置

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: scim-server-ingress
  namespace: scim-system
  labels:
    app: scim-server
  annotations:
    # Nginx Ingress 注解
    nginx.ingress.kubernetes.io/proxy-body-size: "10m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "60"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "60"
    # 限流配置（可选）
    # nginx.ingress.kubernetes.io/limit-connections: "100"
    # nginx.ingress.kubernetes.io/limit-rps: "50"
spec:
  ingressClassName: nginx
  rules:
    - host: scim.example.com
      http:
        paths:
          - path: /scim
            pathType: Prefix
            backend:
              service:
                name: scim-server
                port:
                  number: 8080
  # TLS 配置（可选）
  # tls:
  #   - hosts:
  #       - scim.example.com
  #     secretName: scim-tls-secret
```

### 3.5 一键部署清单

将上述所有资源整合为一个文件：

```bash
cat > scim-deployment.yaml << 'EOF'
# 1. 命名空间
---
apiVersion: v1
kind: Namespace
metadata:
  name: scim-system
  labels:
    name: scim-system
    environment: production

# 2. ServiceAccount
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: scim-server
  namespace: scim-system

# 3. RBAC
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: scim-server-role
  namespace: scim-system
rules:
  - apiGroups: [""]
    resources: ["configmaps", "secrets", "services", "pods", "pods/log"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["apps"]
    resources: ["deployments"]
    verbs: ["get", "list", "watch", "create", "update", "patch"]
  - apiGroups: ["autoscaling"]
    resources: ["horizontalpodautoscalers"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: scim-server-rolebinding
  namespace: scim-system
subjects:
  - kind: ServiceAccount
    name: scim-server
    namespace: scim-system
roleRef:
  kind: Role
  name: scim-server-role
  apiGroup: rbac.authorization.k8s.io

# 4. ConfigMap
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: scim-config
  namespace: scim-system
data:
  config.yaml: |
    mode: release
    port: "8080"
    log_level: info
    storage:
      driver: postgres
      postgres_dsn: "host=postgres-service namespace=scim user=scim_user password=${SCIM_DB_PASSWORD} dbname=scim_db port=5432 sslmode=disable connect_timeout=5"
    pagination:
      default_count: 20
      max_count: 100
      cursor_support: true
    scim:
      default_schema: "urn:ietf:params:scim:schemas:core:2.0:User"
      group_schema: "urn:ietf:params:scim:schemas:core:2.0:Group"
      error_schema: "urn:ietf:params:scim:api:messages:2.0:Error"
      list_schema: "urn:ietf:params:scim:api:messages:2.0:ListResponse"
      api_path: "/scim/v2"
    swagger:
      enabled: false
    time_precision:
      level: second

# 5. Secret
---
apiVersion: v1
kind: Secret
metadata:
  name: scim-secrets
  namespace: scim-system
type: Opaque
stringData:
  SCIM_TOKEN: "your-secure-token-change-in-production"
  SCIM_DB_PASSWORD: "your-database-password"

# 6. Deployment
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: scim-server
  namespace: scim-system
  labels:
    app: scim-server
    version: v1.0.0
spec:
  replicas: 2
  selector:
    matchLabels:
      app: scim-server
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels:
        app: scim-server
        version: v1.0.0
    spec:
      serviceAccountName: scim-server
      imagePullSecrets:
        - name: scim-image-pull-secret
      containers:
        - name: scim-server
          image: registry.cn-hangzhou.aliyuncs.com/shay/scim-server:latest
          imagePullPolicy: Always
          ports:
            - name: http
              containerPort: 8080
              protocol: TCP
          env:
            - name: SCIM_TOKEN
              valueFrom:
                secretKeyRef:
                  name: scim-secrets
                  key: SCIM_TOKEN
            - name: SCIM_DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: scim-secrets
                  key: SCIM_DB_PASSWORD
          resources:
            requests:
              cpu: 500m
              memory: 512Mi
            limits:
              cpu: 1000m
              memory: 1Gi
          volumeMounts:
            - name: config
              mountPath: /config
              readOnly: true
          livenessProbe:
            httpGet:
              path: /scim/v2/ServiceProviderConfig
              port: http
            initialDelaySeconds: 30
            periodSeconds: 10
            timeoutSeconds: 5
            failureThreshold: 3
          readinessProbe:
            httpGet:
              path: /scim/v2/ServiceProviderConfig
              port: http
            initialDelaySeconds: 10
            periodSeconds: 5
            timeoutSeconds: 3
            failureThreshold: 3
      volumes:
        - name: config
          configMap:
            name: scim-config
            items:
              - key: config.yaml
                path: config.yaml

# 7. Service
---
apiVersion: v1
kind: Service
metadata:
  name: scim-server
  namespace: scim-system
  labels:
    app: scim-server
spec:
  type: ClusterIP
  ports:
    - name: http
      port: 8080
      targetPort: http
      protocol: TCP
  selector:
    app: scim-server
EOF
```

执行部署：

```bash
# 应用部署清单
kubectl apply -f scim-deployment.yaml

# 查看部署状态
kubectl get all -n scim-system
```

---

## 4. 部署验证步骤

### 4.1 检查部署状态

```bash
# 查看 Pod 状态
kubectl get pods -n scim-system -l app=scim-server

# 查看 Pod 详细信息
kubectl describe pod -n scim-system -l app=scim-server

# 查看部署状态
kubectl get deployment scim-server -n scim-system

# 查看服务状态
kubectl get service scim-server -n scim-system
```

预期输出：

```
NAME                              READY   STATUS    RESTARTS   AGE
scim-server-7b9f8d6c9-abcde      2/2     Running   0          2m
scim-server-7b9f8d6c9-fghij      2/2     Running   0          2m
```

### 4.2 查看日志

```bash
# 查看 Pod 日志
kubectl logs -n scim-system deployment/scim-server --tail=100

# 实时查看日志
kubectl logs -n scim-system -l app=scim-server -f

# 查看前一个容器的日志（重启后）
kubectl logs -n scim-system -l app=scim-server --previous

# 按时间范围查看日志
kubectl logs -n scim-system -l app=scim-server --since=1h
```

### 4.3 验证服务可用性

#### 4.3.1 验证 ServiceProviderConfig

```bash
# 集群内部验证
kubectl exec -n scim-system deployment/scim-server -- \
  wget -qO- http://localhost:8080/scim/v2/ServiceProviderConfig

# 从节点验证
kubectl run test-client --image=busybox:1.36 --rm -it --restart=Never -- \
  wget -qO- http://scim-server.scim-system:8080/scim/v2/ServiceProviderConfig
```

预期响应：

```json
{
  "schemas": ["urn:ietf:params:scim:api:messages:2.0:ServiceProviderConfig"],
  "patch": {"supported": true},
  "bulk": {"supported": false},
  "filter": {
    "supported": true,
    "maxResults": 100
  },
  "changePassword": {"supported": false},
  "sort": {"supported": false},
  "etag": {"supported": false},
  "authenticationSchemes": [
    {
      "name": "Bearer Token",
      "description": "SCIM Authentication using Bearer Token",
      "specUri": "",
      "documentationUri": ""
    }
  ]
}
```

#### 4.3.2 验证用户 API

```bash
# 创建用户测试
kubectl run test-client --image=busybox:1.36 --rm -it --restart=Never -- \
  wget -qO- -X POST http://scim-server.scim-system:8080/scim/v2/Users \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-secure-token-change-in-production" \
  -d '{"userName": "testuser", "name": {"givenName": "Test", "familyName": "User"}}'
```

#### 4.3.3 端到端验证脚本

```bash
#!/bin/bash
# verify-scim.sh - SCIM 服务验证脚本

SCIM_HOST="${SCIM_HOST:-http://scim-server.scim-system:8080}"
SCIM_TOKEN="${SCIM_TOKEN:-your-secure-token-change-in-production}"

echo "=== SCIM 服务验证 ==="
echo "服务地址: $SCIM_HOST"

# 1. 验证 ServiceProviderConfig
echo -e "\n[1] 验证 ServiceProviderConfig..."
RESPONSE=$(wget -qO- "${SCIM_HOST}/scim/v2/ServiceProviderConfig" \
  -H "Authorization: Bearer $SCIM_TOKEN" 2>&1)

if echo "$RESPONSE" | grep -q "ServiceProviderConfig"; then
  echo "✓ ServiceProviderConfig 验证通过"
else
  echo "✗ ServiceProviderConfig 验证失败"
  echo "$RESPONSE"
  exit 1
fi

# 2. 验证获取用户列表
echo -e "\n[2] 验证获取用户列表..."
RESPONSE=$(wget -qO- "${SCIM_HOST}/scim/v2/Users" \
  -H "Authorization: Bearer $SCIM_TOKEN" 2>&1)

if echo "$RESPONSE" | grep -q "totalResults"; then
  echo "✓ 用户列表 API 验证通过"
else
  echo "✗ 用户列表 API 验证失败"
  echo "$RESPONSE"
  exit 1
fi

# 3. 验证创建用户
echo -e "\n[3] 验证创建用户..."
USER_JSON='{"userName": "testuser001","name":{"givenName":"Test","familyName":"User"},"active":true}'
RESPONSE=$(wget -qO- -X POST "${SCIM_HOST}/scim/v2/Users" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $SCIM_TOKEN" \
  -d "$USER_JSON" 2>&1)

if echo "$RESPONSE" | grep -q "testuser001"; then
  echo "✓ 创建用户 API 验证通过"
else
  echo "✗ 创建用户 API 验证失败"
  echo "$RESPONSE"
  exit 1
fi

echo -e "\n=== 所有验证通过 ==="
```

---

## 5. 服务伸缩策略

### 5.1 水平自动扩缩容（HPA）配置

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: scim-server-hpa
  namespace: scim-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: scim-server
  # 最小副本数
  minReplicas: 2
  # 最大副本数
  maxReplicas: 10
  # 扩缩容指标
  metrics:
    # CPU 使用率指标
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    # 内存使用率指标
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
  # 扩缩容行为配置
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 10
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
        - type: Percent
          value: 100
          periodSeconds: 15
        - type: Pods
          value: 2
          periodSeconds: 15
      selectPolicy: Max
```

#### HPA 关键参数说明

| 参数 | 说明 | 推荐值 |
|------|------|--------|
| `minReplicas` | 最小副本数 | 2 |
| `maxReplicas` | 最大副本数 | 10 |
| `averageUtilization` (CPU) | CPU 目标使用率 | 70% |
| `averageUtilization` (Memory) | 内存目标使用率 | 80% |
| `scaleDown.stabilizationWindowSeconds` | 缩容冷却时间 | 5 分钟 |
| `scaleUp.stabilizationWindowSeconds` | 扩容冷却时间 | 0 秒 |

### 5.2 垂直自动扩缩容（VPA）配置（可选）

```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: scim-server-vpa
  namespace: scim-system
spec:
  targetRef:
    apiVersion: "apps/v1"
    kind: Deployment
    name: scim-server
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
      - containerName: scim-server
        minAllowed:
          cpu: 250m
          memory: 256Mi
        maxAllowed:
          cpu: 2000m
          memory: 2Gi
        controlledResources: ["cpu", "memory"]
```

### 5.3 手动扩缩容

```bash
# 手动扩容
kubectl scale deployment scim-server --replicas=3 -n scim-system

# 手动缩容
kubectl scale deployment scim-server --replicas=2 -n scim-system

# 查看扩缩容历史
kubectl rollout history deployment scim-server -n scim-system
```

---

## 6. 监控与告警配置

### 6.1 Prometheus 监控指标

#### 6.1.1 ServiceMonitor 配置

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: scim-server-monitor
  namespace: scim-system
  labels:
    app: scim-server
spec:
  selector:
    matchLabels:
      app: scim-server
  namespaceSelector:
    matchNames:
      - scim-system
  endpoints:
    - port: http
      path: /metrics
      interval: 30s
      scrapeTimeout: 10s
```

#### 6.1.2 关键监控指标

| 指标名称 | 类型 | 说明 | 告警阈值 |
|----------|------|------|----------|
| `scim_requests_total` | Counter | API 请求总数 | - |
| `scim_request_duration_seconds` | Histogram | 请求延迟 | P99 > 1s |
| `scim_request_errors_total` | Counter | 错误请求数 | > 10/min |
| `scim_active_connections` | Gauge | 活跃连接数 | > 1000 |
| `process_cpu_seconds_total` | Counter | CPU 使用时间 | > 80% |
| `process_resident_memory_bytes` | Gauge | 内存使用量 | > 1Gi |

### 6.2 Grafana 仪表盘

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: scim-grafana-dashboard
  namespace: monitoring
data:
  scim-dashboard.json: |
    {
      "dashboard": {
        "title": "SCIM Server 监控面板",
        "tags": ["scim", "api"],
        "timezone": "browser",
        "panels": [
          {
            "title": "API 请求 QPS",
            "type": "graph",
            "targets": [
              {
                "expr": "sum(rate(scim_requests_total[5m])) by (method, path)",
                "legendFormat": "{{method}} {{path}}"
              }
            ]
          },
          {
            "title": "请求延迟 P99",
            "type": "graph",
            "targets": [
              {
                "expr": "histogram_quantile(0.99, sum(rate(scim_request_duration_seconds_bucket[5m])) by (le))",
                "legendFormat": "P99"
              }
            ]
          },
          {
            "title": "错误率",
            "type": "graph",
            "targets": [
              {
                "expr": "sum(rate(scim_request_errors_total[5m])) by (status_code)",
                "legendFormat": "Status {{status_code}}"
              }
            ]
          },
          {
            "title": "CPU 使用率",
            "type": "graph",
            "targets": [
              {
                "expr": "rate(process_cpu_seconds_total[5m]) * 100",
                "legendFormat": "CPU %"
              }
            ]
          },
          {
            "title": "内存使用",
            "type": "graph",
            "targets": [
              {
                "expr": "process_resident_memory_bytes / 1024 / 1024",
                "legendFormat": "Memory MB"
              }
            ]
          }
        ]
      }
    }
```

### 6.3 告警规则配置

```yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: scim-server-alerts
  namespace: monitoring
spec:
  groups:
    - name: scim-server
      rules:
        # 高错误率告警
        - alert: SCIMHighErrorRate
          expr: |
            sum(rate(scim_request_errors_total[5m])) > 10
          for: 5m
          labels:
            severity: critical
          annotations:
            summary: "SCIM 服务错误率过高"
            description: "SCIM 服务 5 分钟内错误数超过 10 次"
        
        # 高延迟告警
        - alert: SCIMHighLatency
          expr: |
            histogram_quantile(0.99, sum(rate(scim_request_duration_seconds_bucket[5m])) by (le)) > 1
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "SCIM 服务延迟过高"
            description: "SCIM 服务 P99 延迟超过 1 秒"
        
        # Pod 不可用告警
        - alert: SCIMPodNotReady
          expr: |
            kube_pod_status_ready{namespace="scim-system",pod=~"scim-server.*",condition="Ready"} == 0
          for: 3m
          labels:
            severity: critical
          annotations:
            summary: "SCIM Pod 未就绪"
            description: "SCIM Pod {{ $labels.pod }} 已就绪状态为 false 超过 3 分钟"
        
        # 内存使用过高告警
        - alert: SCIMHighMemoryUsage
          expr: |
            process_resident_memory_bytes / 1024 / 1024 > 900
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "SCIM 内存使用过高"
            description: "SCIM 服务内存使用超过 900MB"
        
        # CPU 使用过高告警
        - alert: SCIMHighCPUUsage
          expr: |
            rate(process_cpu_seconds_total[5m]) * 100 > 80
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "SCIM CPU 使用过高"
            description: "SCIM 服务 CPU 使用率超过 80%"
        
        # HPA 扩容告警
        - alert: SCIMHPAScaling
          expr: |
            kube_horizontalpodautoscaler_status_desired_replicas{namespace="scim-system",horizontalpodautoscaler="scim-server-hpa"} > kube_horizontalpodautoscaler_status_current_replicas{namespace="scim-system",horizontalpodautoscaler="scim-server-hpa"}
          for: 1m
          labels:
            severity: info
          annotations:
            summary: "SCIM 正在扩容"
            description: "SCIM HPA 正在从 {{ $value }} 个副本扩容"
```

---

## 7. 备份与恢复策略

### 7.1 数据持久化方案

#### 7.1.1 使用 PostgreSQL 作为持久化存储

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: scim-postgres-pvc
  namespace: scim-system
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: ssd  # 根据实际存储类调整
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: scim-postgres
  namespace: scim-system
spec:
  replicas: 1
  selector:
    matchLabels:
      app: scim-postgres
  template:
    metadata:
      labels:
        app: scim-postgres
    spec:
      containers:
        - name: postgres
          image: postgres:15-alpine
          env:
            - name: POSTGRES_DB
              value: scim_db
            - name: POSTGRES_USER
              value: scim_user
            - name: POSTGRES_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: scim-secrets
                  key: SCIM_DB_PASSWORD
          ports:
            - containerPort: 5432
          volumeMounts:
            - name: postgres-data
              mountPath: /var/lib/postgresql/data
      volumes:
        - name: postgres-data
          persistentVolumeClaim:
            claimName: scim-postgres-pvc
---
apiVersion: v1
kind: Service
metadata:
  name: postgres-service
  namespace: scim-system
spec:
  selector:
    app: scim-postgres
  ports:
    - port: 5432
      targetPort: 5432
```

### 7.2 数据库备份

#### 7.2.1 定时备份 Job

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: scim-backup
  namespace: scim-system
spec:
  schedule: "0 2 * * *"  # 每天凌晨 2 点
  successfulJobsHistoryLimit: 7
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: backup
              image: postgres:15-alpine
              env:
                - name: POSTGRES_PASSWORD
                  valueFrom:
                    secretKeyRef:
                      name: scim-secrets
                        key: SCIM_DB_PASSWORD
              command:
                - /bin/sh
                - -c
                - |
                  # 创建备份目录
                  mkdir -p /backup
                  # 执行备份
                  pg_dump -h postgres-service -U scim_user -d scim_db > /backup/scim_$(date +%Y%m%d_%H%M%S).sql
                  # 删除 7 天前的备份
                  find /backup -name "scim_*.sql" -mtime +7 -delete
                  echo "Backup completed"
              volumeMounts:
                - name: backup-storage
                  mountPath: /backup
          volumes:
            - name: backup-storage
              persistentVolumeClaim:
                claimName: scim-backup-pvc
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: scim-backup-pvc
  namespace: scim-system
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 20Gi
```

#### 7.2.2 备份脚本（手动执行）

```bash
#!/bin/bash
# backup-scim.sh - SCIM 数据库备份脚本

BACKUP_DIR="/path/to/backup"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/scim_${DATE}.sql"

# 环境变量
export PGPASSWORD="your-database-password"

echo "=== 开始备份 SCIM 数据库 ==="

# 执行备份
pg_dump -h postgres-service \
  -U scim_user \
  -d scim_db \
  -F c \
  -b \
  -v \
  -f "${BACKUP_FILE}"

if [ $? -eq 0 ]; then
  echo "备份成功: ${BACKUP_FILE}"
  # 压缩备份文件
  gzip "${BACKUP_FILE}"
  echo "备份压缩完成: ${BACKUP_FILE}.gz"
else
  echo "备份失败"
  exit 1
fi

# 清理 7 天前的旧备份
find ${BACKUP_DIR} -name "scim_*.sql.gz" -mtime +7 -delete

echo "=== 备份完成 ==="
```

### 7.3 数据恢复

#### 7.3.1 从备份恢复

```bash
#!/bin/bash
# restore-scim.sh - SCIM 数据库恢复脚本

BACKUP_FILE="$1"

if [ -z "$BACKUP_FILE" ]; then
  echo "用法: $0 <backup_file>"
  exit 1
fi

export PGPASSWORD="your-database-password"

echo "=== 开始恢复 SCIM 数据库 ==="
echo "备份文件: ${BACKUP_FILE}"

# 停止 SCIM 服务
kubectl scale deployment scim-server --replicas=0 -n scim-system

# 恢复数据库
if [[ "${BACKUP_FILE}" == *.gz ]]; then
  gunzip -c "${BACKUP_FILE}" | psql -h postgres-service -U scim_user -d scim_db
else
  pg_restore -h postgres-service -U scim_user -d scim_db -c "${BACKUP_FILE}"
fi

# 启动 SCIM 服务
kubectl scale deployment scim-server --replicas=2 -n scim-system

echo "=== 恢复完成 ==="
```

#### 7.3.2 验证恢复

```bash
# 检查数据
kubectl exec -it <postgres-pod> -n scim-system -- \
  psql -U scim_user -d scim_db -c "SELECT COUNT(*) FROM users;"

# 重启应用使缓存生效
kubectl rollout restart deployment scim-server -n scim-system
```

---

## 8. 常见问题排查指南

### 8.1 Pod 启动失败

#### 问题现象

```
kubectl get pods -n scim-system
NAME                        READY   STATUS    RESTARTS   AGE
scim-server-xxx             0/1     ImagePullBackOff   0          5m
```

#### 排查步骤

```bash
# 1. 查看 Pod 详细状态
kubectl describe pod <pod-name> -n scim-system

# 2. 查看镜像拉取日志
kubectl events -n scim-system --reason FailedToPullImage

# 3. 检查 ImagePullSecret 是否存在
kubectl get secret scim-image-pull-secret -n scim-system

# 4. 手动测试镜像拉取
docker pull registry.cn-hangzhou.aliyuncs.com/shay/scim-server:latest
```

#### 解决方案

```bash
# 方案 1: 检查凭证
kubectl create secret docker-registry scim-image-pull-secret \
  --docker-server=registry.cn-hangzhou.aliyuncs.com \
  --docker-username=<username> \
  --docker-password=<password> \
  --docker-email=<email> \
  -n scim-system

# 方案 2: 使用公开镜像（如果有）
# 修改 Deployment 移除 imagePullSecrets
```

### 8.2 Pod 持续重启

#### 问题现象

```
NAME                        READY   STATUS    RESTARTS   AGE
scim-server-xxx             1/2     CrashLoopBackOff   5          10m
```

#### 排查步骤

```bash
# 1. 查看容器日志
kubectl logs <pod-name> -n scim-system --previous

# 2. 查看详细事件
kubectl describe pod <pod-name> -n scim-system

# 3. 检查配置文件
kubectl get configmap scim-config -n scim-system -o yaml

# 4. 检查密钥
kubectl get secret scim-secrets -n scim-system -o yaml
```

#### 常见原因及解决方案

| 原因 | 解决方案 |
|------|----------|
| 配置文件格式错误 | 验证 YAML 语法：`kubectl get configmap scim-config -o jsonpath='{.data.config\.yaml}' | python3 -c "import sys, yaml; yaml.safe_load(sys.stdin)"` |
| 数据库连接失败 | 检查数据库服务和网络连接 |
| 端口冲突 | 检查 containerPort 配置 |
| 权限不足 | 检查 ServiceAccount 和 RBAC 配置 |

### 8.3 服务无法访问

#### 问题现象

```
curl http://scim-server.scim-system:8080/scim/v2/ServiceProviderConfig
curl: (7) Failed to connect to ... Connection timed out
```

#### 排查步骤

```bash
# 1. 检查 Service 状态
kubectl get service scim-server -n scim-system

# 2. 检查 Endpoint
kubectl get endpoints scim-server -n scim-system

# 3. 检查网络策略
kubectl get networkpolicy -n scim-system

# 4. 测试 Pod 内部连通性
kubectl run test --image=busybox:1.36 --rm -it --restart=Never -- \
  wget -qO- http://<pod-ip>:8080/scim/v2/ServiceProviderConfig
```

#### 解决方案

```bash
# 方案 1: 检查 Selector 匹配
kubectl get pods -n scim-system -l app=scim-server --show-labels

# 方案 2: 重建 Service
kubectl delete service scim-server -n scim-system
kubectl apply -f service.yaml

# 方案 3: 检查防火墙/安全组
```

### 8.4 健康检查失败

#### 问题现象

```
kubectl get pods -n scim-system
NAME                        READY   STATUS    RESTARTS   AGE
scim-server-xxx             0/1     ReadinessFailed   0          5m
```

#### 排查步骤

```bash
# 1. 查看健康检查详情
kubectl describe pod <pod-name> -n scim-system | grep -A 10 "Liveness\|Readiness"

# 2. 测试健康检查端点
kubectl exec -it <pod-name> -n scim-system -- \
  wget -qO- http://localhost:8080/scim/v2/ServiceProviderConfig

# 3. 检查日志
kubectl logs <pod-name> -n scim-system
```

#### 解决方案

```yaml
# 调整健康检查参数
livenessProbe:
  httpGet:
    path: /scim/v2/ServiceProviderConfig
    port: 8080
  initialDelaySeconds: 60  # 增加初始延迟
  periodSeconds: 15        # 延长检查周期
  failureThreshold: 5      # 增加失败阈值
```

### 8.5 HPA 无法扩容

#### 问题现象

```
kubectl get hpa -n scim-system
NAME            REFERENCE          TARGETS   MINPODS   MAXPODS   REPLICAS   AGE
scim-server-hpa   Deployment/scim-server   80%/70%   2         10        2         1h
```

CPU 使用率已达到 80%，但副本数未增加。

#### 排查步骤

```bash
# 1. 检查 HPA 状态
kubectl describe hpa scim-server-hpa -n scim-system

# 2. 检查指标可用性
kubectl top pods -n scim-system

# 3. 检查 metrics-server
kubectl get apiservices | grep metrics
```

#### 解决方案

```bash
# 方案 1: 安装/重启 metrics-server
kubectl rollout restart deployment metrics-server -n kube-system

# 方案 2: 检查 Pod 是否支持扩缩容
kubectl patch deployment scim-server -n scim-system \
  -p '{"spec":{"strategy":{"type":"RollingUpdate"}}}'
```

### 8.6 数据库连接问题

#### 问题现象

```
level=error msg="postgres connect failed: dial tcp: lookup postgres-service on..."
```

#### 排查步骤

```bash
# 1. 检查数据库服务
kubectl get svc postgres-service -n scim-system

# 2. 测试数据库连接
kubectl run db-test --image=postgres:15-alpine --rm -it --restart=Never -- \
  psql -h postgres-service -U scim_user -d scim_db -c "SELECT 1"

# 3. 检查数据库凭证
kubectl get secret scim-secrets -n scim-system -o jsonpath='{.data.SCIM_DB_PASSWORD}' | base64 -d
```

#### 解决方案

```yaml
# 更新 ConfigMap 中的数据库连接字符串
kubectl patch configmap scim-config -n scim-system \
  -p '{"data":{"config.yaml":"storage:\n  driver: postgres\n  postgres_dsn: \"host=<new-host> ...\""}}'

# 重启 Pod 使配置生效
kubectl rollout restart deployment scim-server -n scim-system
```

---

## 附录

### 附录 A: 快速部署命令汇总

```bash
# 1. 创建命名空间和 RBAC
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: scim-system
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: scim-server
  namespace: scim-system
EOF

# 2. 创建密钥
kubectl create secret generic scim-secrets \
  -n scim-system \
  --from-literal=SCIM_TOKEN='your-secure-token' \
  --from-literal=SCIM_DB_PASSWORD='your-db-password'

# 3. 部署服务
kubectl apply -f scim-deployment.yaml

# 4. 部署 HPA
kubectl apply -f hpa.yaml

# 5. 验证部署
kubectl get all -n scim-system
```

### 附录 B: 常用命令速查表

```bash
# 查看日志
kubectl logs -n scim-system deployment/scim-server --tail=100 -f

# 进入容器调试
kubectl exec -it <pod-name> -n scim-system -- /bin/sh

# 查看资源使用
kubectl top pods -n scim-system

# 扩缩容
kubectl scale deployment scim-server --replicas=3 -n scim-system

# 重启服务
kubectl rollout restart deployment/scim-server -n scim-system

# 查看事件
kubectl get events -n scim-system --sort-by='.lastTimestamp'

# 端口转发进行本地测试
kubectl port-forward svc/scim-server 8080:8080 -n scim-system
```

### 附录 C: 配置参数参考

| 参数 | 环境变量 | 默认值 | 说明 |
|------|----------|--------|------|
| `port` | `SCIM_PORT` | 8080 | 服务端口 |
| `mode` | `SCIM_MODE` | release | 运行模式 |
| `log_level` | `SCIM_LOG_LEVEL` | info | 日志级别 |
| `storage.driver` | `SCIM_STORAGE_DRIVER` | memory | 存储驱动 |
| `token` | `SCIM_TOKEN` | - | 认证令牌 |

---

*文档最后更新：2026-03-06*
