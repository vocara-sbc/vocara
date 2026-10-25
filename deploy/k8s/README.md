# Vocara 呼叫中心系统 - Kubernetes 部署文档

## 目录结构

```
deploy/k8s/
├── namespace.yaml          # 命名空间
├── configmap.yaml          # 所有服务的配置文件
├── mysql.yaml              # MySQL 8.0（有状态）
├── redis.yaml              # Redis 7（有状态，AOF 持久化）
├── nats.yaml               # NATS JetStream（有状态）
├── vocara-admin.yaml       # 管理后台 API
├── vocara-agentcenter.yaml # 座席中心
├── vocara-acd.yaml         # ACD 智能排队
├── vocara-callctrl.yaml    # 呼叫控制
├── vocara-push.yaml        # WebSocket 实时推送
├── vocara-web.yaml         # 前端入口（Nginx：静态托管 + 五组反向代理）
├── freeswitch-sbc.yaml     # FreeSWITCH 接入侧媒体节点（hostNetwork）
├── freeswitch-agent.yaml   # FreeSWITCH 座席侧媒体节点（hostNetwork，含 5066 Service）
├── ingress.yaml            # NGINX Ingress（统一路由到 vocara-web）
└── README.md               # 本文档
```

## 架构概览

```
                  ┌─────────────────────────┐
                  │    Ingress (NGINX)      │
                  │   vocara.example.com    │
                  └────────────┬────────────┘
                               │
                  ┌────────────▼────────────┐
                  │ vocara-web (Nginx :80)  │
                  │ 静态托管 + 反向代理       │
                  └──┬─────┬─────┬────┬─────┘
                     │     │     │    │
        ┌────────────▼┐ ┌───▼───┐ │ ┌──▼──────────┐
        │ vocara-admin│ │vocara-│ │ │vocara-push  │
        │    (8080)   │ │agent- │ │ │  (8084/WS)  │
        └─────────────┘ │center │ │ └─────────────┘
                        │(8081) │ │
        ┌────────────┐  └───────┘ │
        │ vocara-    │            │
        │ callctrl   │◄───────────┤
        │  (8083)    │  /callapi  │
        └─────┬──────┘  /agentapi │
              │ ESL              │
   ┌──────────▼─────────┐        │
   │ vocara-fs-agent    │◄───────┘ /sip-ws
   │ (hostNetwork:5066) │
   └────────────────────┘

   MySQL 8.0   Redis 7   NATS JetStream（各微服务共享基础设施）
```

## 部署顺序

请严格按照以下顺序部署，确保依赖服务先启动：

### 第 1 步：创建命名空间

```bash
kubectl apply -f namespace.yaml
```

### 第 2 步：部署配置文件

```bash
kubectl apply -f configmap.yaml
```

### 第 3 步：部署基础设施（有状态服务）

```bash
# 部署 MySQL
kubectl apply -f mysql.yaml

# 部署 Redis
kubectl apply -f redis.yaml

# 部署 NATS
kubectl apply -f nats.yaml
```

等待基础设施服务全部就绪：

```bash
kubectl -n vocara get pods
# 确认 mysql/redis/nats 的 Pod 状态为 Running 且 READY 为 1/1
```

### 第 4 步：部署微服务

```bash
# 部署所有业务服务
kubectl apply -f vocara-admin.yaml
kubectl apply -f vocara-agentcenter.yaml
kubectl apply -f vocara-acd.yaml
kubectl apply -f vocara-callctrl.yaml
kubectl apply -f vocara-push.yaml
```

### 第 5 步：部署 FreeSWITCH 媒体节点

节点需预先准备 `/data/vocara/freeswitch/` 下的 hostPath 目录与配置（各清单文件头含目录与配置放置说明），并允许特权容器与 hostNetwork。

```bash
kubectl apply -f freeswitch-sbc.yaml
kubectl apply -f freeswitch-agent.yaml
```

### 第 6 步：部署前端入口

```bash
kubectl apply -f vocara-web.yaml
```

### 第 7 步：部署 Ingress

```bash
# 确保已安装 NGINX Ingress Controller
kubectl apply -f ingress.yaml
```

### 一键部署（全部）

```bash
cd deploy/k8s
kubectl apply -f namespace.yaml
kubectl apply -f configmap.yaml
kubectl apply -f mysql.yaml
kubectl apply -f redis.yaml
kubectl apply -f nats.yaml
kubectl apply -f vocara-admin.yaml
kubectl apply -f vocara-agentcenter.yaml
kubectl apply -f vocara-acd.yaml
kubectl apply -f vocara-callctrl.yaml
kubectl apply -f vocara-push.yaml
kubectl apply -f freeswitch-sbc.yaml
kubectl apply -f freeswitch-agent.yaml
kubectl apply -f vocara-web.yaml
kubectl apply -f ingress.yaml
```

## 验证部署

### 查看所有资源

```bash
# 查看命名空间下所有资源
kubectl -n vocara get all

# 查看所有 Pod 状态
kubectl -n vocara get pods -o wide

# 查看服务
kubectl -n vocara get svc

# 查看 ConfigMap
kubectl -n vocara get configmap
```

### 查看日志

```bash
# 查看某个服务的日志
kubectl -n vocara logs -f deployment/vocara-admin

# 查看 MySQL 日志
kubectl -n vocara logs -f statefulset/vocara-mysql
```

### 测试服务连通性

```bash
# 临时启动一个 Pod 进行测试
kubectl run -it --rm debug --image=busybox:1.36 --namespace=vocara --restart=Never

# 在 debug Pod 内测试各服务
# wget -qO- http://vocara-admin:8080/health
# wget -qO- http://vocara-redis:6379  # redis-cli 需要 redis 镜像
# exit
```

## 如何修改配置

### 方式一：修改 ConfigMap（推荐）

配置集中管理在 `configmap.yaml` 中，修改后重新应用：

```bash
# 编辑配置
kubectl -n vocara edit configmap vocara-admin-config

# 或修改文件后重新 apply
kubectl apply -f configmap.yaml

# 配置变更后需要重启 Pod 才能生效（滚动重启）
kubectl -n vocara rollout restart deployment/vocara-admin
```

> **提示**：生产环境建议使用 `config-hash` 注解实现配置变更自动触发滚动更新，可配合 Kustomize 或 Helm 使用。

### 方式二：环境变量覆盖

敏感配置（密码、密钥等）通过环境变量注入，在各服务的 Deployment 中修改 `env` 字段。环境变量统一使用 `VOCARA_` 前缀：`pkg/config` 经 viper AutomaticEnv 识别 `VOCARA_<配置节>_<键>` 并覆盖同名配置项。

```yaml
env:
  - name: VOCARA_MYSQL_PASSWORD
    value: "your_strong_password"
```

常用环境变量：

| 环境变量 | 覆盖配置项 |
|---|---|
| VOCARA_APP_SECRET_KEY | app.secret_key（JWT 签名与内部服务间令牌） |
| VOCARA_MYSQL_PASSWORD | mysql.password |
| VOCARA_REDIS_PASSWORD | redis.password |
| VOCARA_NATS_PASSWORD | nats.password |
| VOCARA_ESL_HOST | esl.host（座席侧媒体节点 IP） |
| VOCARA_ESL_PASSWORD | esl.password |
| VOCARA_FS_DIALPLAN_SBC_PEER | fs_dialplan.sbc_peer |
| VOCARA_FS_DIALPLAN_AGENT_EXTERNAL | fs_dialplan.agent_external |
| VOCARA_ACD_AGENT_CENTER_TOKEN | acd.agent_center_token |

### 方式三：使用 Secret（生产环境推荐）

将敏感信息存储在 Kubernetes Secret 中，通过 `secretKeyRef` 引用：

```bash
# 创建 Secret
kubectl -n vocara create secret generic vocara-secrets \
  --from-literal=app-secret-key='your_strong_secret_key' \
  --from-literal=db-password='your_db_password' \
  --from-literal=redis-password='your_redis_password' \
  --from-literal=nats-password='your_nats_password' \
  --from-literal=esl-password='your_esl_password'
```

然后在 Deployment 中引用：

```yaml
env:
  - name: VOCARA_MYSQL_PASSWORD
    valueFrom:
      secretKeyRef:
        name: vocara-secrets
        key: db-password
```

### 修改镜像版本

修改各服务 Deployment 中的 `image` 字段：

```yaml
image: registry.example.com/vocara/vocara-admin:v1.0.0
```

或使用命令快速更新：

```bash
kubectl -n vocara set image deployment/vocara-admin \
  vocara-admin=registry.example.com/vocara/vocara-admin:v1.0.0
```

## 如何扩缩容

### 手动扩缩容

```bash
# 扩容 vocara-admin 到 5 副本
kubectl -n vocara scale deployment vocara-admin --replicas=5

# 缩容 vocara-push 到 2 副本（不建议低于 2）
kubectl -n vocara scale deployment vocara-push --replicas=2
```

### 自动扩缩容（HPA）

根据 CPU/内存使用率自动扩缩容（需要安装 metrics-server）：

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: vocara-admin-hpa
  namespace: vocara
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: vocara-admin
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

### WebSocket 服务扩缩容注意事项

- `vocara-push` 是 WebSocket 长连接服务，扩容容易，缩容难
- 缩容时会导致连接断开，建议设置较长的 `terminationGracePeriodSeconds`
- 生产环境建议使用基于连接数的自定义指标进行扩缩容
- 避免频繁扩缩容

## 资源配置参考

以下是默认资源配置，可根据实际负载调整：

| 服务 | CPU 请求 | CPU 限制 | 内存请求 | 内存限制 | 副本数 |
|------|---------|---------|---------|---------|-------|
| vocara-admin | 100m | 500m | 128Mi | 512Mi | 2 |
| vocara-agentcenter | 100m | 500m | 128Mi | 512Mi | 2 |
| vocara-acd | 100m | 500m | 128Mi | 512Mi | 2 |
| vocara-callctrl | 100m | 500m | 128Mi | 512Mi | 2 |
| vocara-push | 100m | 500m | 256Mi | 512Mi | 3 |
| MySQL | 250m | 1000m | 512Mi | 1Gi | 1 |
| Redis | 100m | 500m | 128Mi | 512Mi | 1 |
| NATS | 100m | 500m | 128Mi | 512Mi | 1 |

> **说明**：Go 语言服务内存占用通常较低，128Mi-512Mi 对于中小规模场景足够。
> 高并发场景下请适当调整 CPU 和内存限制。

## 注意事项

### 1. 生产环境安全

- **务必修改所有默认密码与密钥**：MySQL root 密码、应用密钥（app.secret_key，承担 JWT 签名）等
- 使用 Kubernetes **Secret** 管理敏感信息，不要明文写在 YAML 中
- 启用 **HTTPS**：为 Ingress 配置 TLS 证书
- 配置 **NetworkPolicy** 限制服务间访问
- MySQL 不使用 root 用户连接，创建专用数据库用户

### 2. 数据持久化

- MySQL、Redis、NATS 均使用 PVC 持久化存储
- 删除 StatefulSet 不会自动删除 PVC，数据会保留
- 如需彻底删除数据，请手动删除 PVC
- 生产环境建议使用高可用存储类（StorageClass）

### 3. 高可用建议

- 当前模板为单节点的 MySQL/Redis/NATS，仅适合开发测试
- 生产环境建议：
  - MySQL：主从复制或使用云数据库（RDS 等）
  - Redis：Redis Cluster 或哨兵模式
  - NATS：3 节点集群
  - 所有微服务：至少 2 副本起步

### 4. WebSocket 相关

- Ingress 已配置 WebSocket 协议升级
- 确保 proxy-read-timeout 设置足够长（默认 3600s）
- WebSocket 服务使用长连接，滚动更新时注意连接 draining
- 如需会话保持，可在 Service 中启用 `sessionAffinity: ClientIP`

### 5. 数据库初始化与表结构

- `vocara-mysql-initdb` ConfigMap 挂入 `/docker-entrypoint-initdb.d/`，MySQL 首启执行其中的建库语句
- 库表基准 DDL 位于仓库 `db/migrations/001_init_schema.sql`，交付时可将该 SQL 加入 initdb ConfigMap 随首启执行
- 各服务经 GORM AutoMigrate 维护自身表结构，由配置项 `mysql.auto_migrate` 控制（vocara-admin 默认开启，vocara-callctrl 启动期维护话单表）

### 6. FreeSWITCH 媒体节点

- FreeSWITCH 承担媒体与协议处理，包含 FS-SBC（运营商接入侧）与 FS-Agent（座席侧）两个节点
- 部署清单为 `freeswitch-sbc.yaml`、`freeswitch-agent.yaml`，以 hostNetwork + 特权容器运行，配置与录音经 hostPath 挂载
- vocara-callctrl 经 ESL（8021）连接 FS-Agent，节点 IP 由 `VOCARA_ESL_HOST` 注入，ESL 口令由 `VOCARA_ESL_PASSWORD` 注入
- SIP 端口：FS-SBC internal 5060 / external 5080；FS-Agent 5060/5080 与 WebRTC 5066/7443
- 节点配置目录内容来自 core 仓库 `freeswitch/` 下对应节点的 conf 目录

### 7. 监控和日志

- 所有服务日志输出到 stdout，可通过 `kubectl logs` 查看
- 生产环境建议部署：
  - **Prometheus + Grafana**：监控指标
  - **ELK / Loki**：日志收集
  - **Jaeger / Zipkin**：链路追踪

### 8. 镜像仓库

- 默认使用占位符 `registry.example.com/vocara/`
- 请替换为实际的镜像仓库地址
- 如使用私有仓库，需配置 `imagePullSecrets`

### 9. 版本兼容性

- Kubernetes 版本要求：1.20+（使用 `apps/v1`、`networking.k8s.io/v1` 等稳定 API）
- NGINX Ingress Controller 版本：1.0+

### 10. Service 环境变量注入

所有工作负载的 Pod 均关闭了 service 环境变量注入（`enableServiceLinks: false`）。服务发现统一走 CoreDNS 的 Service 名解析（config 中 `vocara-mysql:3306`、`vocara-redis:6379`、`vocara-nats:4222` 等），不依赖 kubelet 注入的 `*_SERVICE_HOST`/`*_PORT` 变量。这样可避免 kubelet 为 `vocara-mysql` Service 注入的 `VOCARA_MYSQL_PORT=tcp://<host>:3306`（legacy 组合形式）被 viper `AutomaticEnv` 回映射为 `mysql.port`（int）导致配置解析失败；显式声明的 `VOCARA_*` 环境变量不受影响。

## 常见问题

### Q: Pod 一直处于 Pending 状态？

```bash
# 查看事件
kubectl -n vocara describe pod <pod-name>
```

常见原因：
- 节点资源不足
- PVC 无法绑定（没有可用的 PV / StorageClass）
- 镜像拉取失败（私有仓库凭证问题）

### Q: 服务启动失败，连接不上 MySQL？

1. 确认 MySQL Pod 是否 Running
2. 查看 MySQL 日志确认初始化是否完成
3. 检查数据库密码配置是否一致
4. 确认服务名 `vocara-mysql` 在 vocara 命名空间内可解析

### Q: WebSocket 连接断开？

1. 检查 Ingress 的 WebSocket 配置是否正确
2. 确认 `proxy-read-timeout` 足够长
3. 检查 push 服务心跳配置是否正常
4. 查看服务日志是否有错误信息

### Q: 如何删除所有资源？

```bash
# 删除命名空间（会删除该命名空间下所有资源）
kubectl delete namespace vocara

# 注意：PVC 可能不会被自动删除，需要手动清理
kubectl -n vocara delete pvc --all
```

## 进一步优化建议

1. 使用 **Helm Chart** 管理部署（将模板化参数化）
2. 引入 **Kustomize** 管理多环境配置（dev/staging/prod）
3. 配置 **PodDisruptionBudget** 保障服务可用性
4. 配置 **Pod Security Admission** / **SecurityContext** 加固安全
5. 实现 **蓝绿部署** / **金丝雀发布**
6. 接入 **Service Mesh**（如 Istio）做流量管理
