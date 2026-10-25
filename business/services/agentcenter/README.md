# Vocara Agent Center (座席中心服务)

座席中心是 Vocara 呼叫中心的核心服务，负责座席的全生命周期管理：登录、签入/签出、状态管理、技能组、分机管理等。

## 功能特性

- **座席全生命周期管理**：登录/登出、签入/签出、心跳保活
- **12 种座席状态机**：严格的状态转换校验
- **技能组管理**：座席多技能组配置，支持技能等级
- **分机管理**：分机注册、状态查询、在线检测
- **空闲座席管理**：基于 Redis ZSET，按最长空闲优先排序
- **座席分配**：分布式锁保证并发安全，防止超卖
- **心跳过期检测**：定时扫描过期座席，自动置为 OFFLINE
- **JWT 鉴权**：座席接口 Token 鉴权
- **内部服务鉴权**：ACD 等内部服务专用 Token 鉴权

## 座席状态机

```
OFFLINE(0) → CHECKIN(1) → CHECKED(2) → READY(3) ←→ BUSY(9)
                                      ↑         ↓
                                      │    WAIT_SERVER(4)
                                      │         ↓
                               AFTER_WORK(8) ← RINGING(5)
                                      ↑         ↓
                                      └── CALLING(7) ← OUTBOUNDING(6)
                                     (LISTENING=71 / THREE_WAY=72 子状态)
```

### 状态说明

| 状态 | 枚举值 | 说明 | 可被ACD分配 |
|------|--------|------|-------------|
| OFFLINE | 0 | 离线 | ❌ |
| CHECKIN | 1 | 签入中 | ❌ |
| CHECKED | 2 | 已签入（未就绪） | ❌ |
| READY | 3 | 空闲就绪 | ✅ |
| WAIT_SERVER | 4 | 等待服务（分配后等待应答） | ❌ |
| RINGING | 5 | 振铃中 | ❌ |
| OUTBOUNDING | 6 | 外呼中 | ❌ |
| CALLING | 7 | 通话中 | ❌ |
| LISTENING | 71 | 监听中（子状态） | ❌ |
| THREE_WAY | 72 | 三方通话（子状态） | ❌ |
| AFTER_WORK | 8 | 事后处理 | ❌ |
| BUSY | 9 | 示忙 | ❌ |

### 状态转换规则

| 当前状态 | 可转换到 |
|----------|----------|
| OFFLINE | CHECKIN |
| CHECKIN | CHECKED, OFFLINE |
| CHECKED | READY, OFFLINE |
| READY | BUSY, WAIT_SERVER, OUTBOUNDING, OFFLINE |
| BUSY | READY, OFFLINE |
| WAIT_SERVER | RINGING, READY |
| RINGING | CALLING, AFTER_WORK, READY |
| OUTBOUNDING | CALLING, READY |
| CALLING | AFTER_WORK, LISTENING, THREE_WAY, READY |
| LISTENING | CALLING, AFTER_WORK |
| THREE_WAY | CALLING, AFTER_WORK |
| AFTER_WORK | READY, BUSY, OFFLINE |

## Redis 数据结构

```bash
# 座席会话（实时状态）
vocara:agent:session:{tenantId}:{agentId}   HASH
  字段：agentId, agentName, status, dnNumber, tenantId,
        skillGroups, lastReadyTime, currentCallId,
        lastActiveTime, checkinTime, ip

# 在线座席列表
vocara:agent:online:{tenantId}               SET
  元素：agentId

# 空闲座席（按技能组）
vocara:agent:idle:{skillGroupId}             ZSET
  score = lastReadyTimestamp  (最长空闲优先)
  member = agentId

# 座席-技能组映射
vocara:agent:skills:{agentId}                SET
  元素：skillGroupId

# 座席分布式锁
vocara:agent:lock:{agentId}                  STRING
  TTL = 5s（分配时临时锁定）
```

## 数据库表

| 表名 | 说明 |
|------|------|
| tb_agent | 座席信息表 |
| tb_organization | 组织机构表 |
| tb_skill_group | 技能组表 |
| tb_agent_skill_group | 座席-技能组关系表 |
| tb_dn | 分机表 |
| tb_online_agent | 在线座席记录表 |
| tb_agent_service_record | 座席服务记录表 |

## 项目结构

```
services/agentcenter/
├── go.mod
├── cmd/
│   └── main.go                 # 服务入口
├── internal/
│   ├── handler/                # HTTP handler 层
│   │   ├── agent_handler.go    # 座席相关接口
│   │   ├── dn_handler.go       # 分机相关接口
│   │   ├── skill_handler.go    # 技能组接口
│   │   └── manage_handler.go   # 内部管理接口（供 ACD 调用）
│   ├── service/                # 业务逻辑层
│   │   ├── agent_service.go    # 座席服务（状态机核心）
│   │   ├── session_service.go  # Session 管理（Redis）
│   │   ├── dn_service.go       # 分机服务
│   │   └── skill_service.go    # 技能组服务
│   ├── repository/             # 数据访问层
│   │   ├── agent_repo.go       # 座席数据访问
│   │   ├── dn_repo.go          # 分机数据访问
│   │   └── skill_repo.go       # 技能组数据访问
│   └── model/                  # 数据模型
│       ├── agent.go            # 座席模型 & DTO
│       ├── dn.go               # 分机模型 & DTO
│       ├── skill.go            # 技能组模型 & DTO
│       └── status.go           # 状态枚举 & 状态机
├── configs/
│   └── config.yaml             # 配置文件
└── README.md
```

## API 接口

### 座席接口

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | /api/v1/agent/login | 座席登录 | 否 |
| POST | /api/v1/agent/heartbeat/{agentId} | 心跳保活 | 否 |
| POST | /api/v1/agent/logout | 座席登出 | JWT |
| POST | /api/v1/agent/checkin | 座席签入 | JWT |
| POST | /api/v1/agent/checkout | 座席签出 | JWT |
| POST | /api/v1/agent/busy | 示忙 | JWT |
| POST | /api/v1/agent/ready | 示闲 | JWT |
| GET | /api/v1/agent/info | 获取座席信息 | JWT |
| GET | /api/v1/agent/status | 获取座席状态 | JWT |

### 分机接口

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/v1/dn/list | 分机列表 | JWT |
| GET | /api/v1/dn/{dnId} | 分机详情 | JWT |
| POST | /api/v1/dn/checkonline | 批量检测分机在线 | JWT |

### 技能组接口

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| GET | /api/v1/skill/list | 技能组列表 | JWT |
| GET | /api/v1/skill/{id} | 技能组详情 | JWT |
| GET | /api/v1/skill/{id}/agents | 技能组下座席 | JWT |

### 内部管理接口（供 ACD 调用）

需要 `X-Internal-Token` 请求头鉴权。

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/internal/agent/try-acquire | 尝试分配一个空闲座席 |
| POST | /api/v1/internal/agent/ring | 通知座席振铃 |
| POST | /api/v1/internal/agent/resume-ready | 分配失败，重置为空闲 |
| POST | /api/v1/internal/agent/update-status | 更新座席状态 |

## 核心设计

### 座席分配流程

1. ACD 调用 `/api/v1/internal/agent/try-acquire`，传入 `skill_group_id`
2. 从 ZSET `vocara:agent:idle:{skillGroupId}` 中弹出 score 最小的座席（最长空闲优先）
3. 获取座席 session，检查是否确实处于 READY 状态
4. 加分布式锁（`vocara:agent:lock:{agentId}`，5秒 TTL）
5. 二次校验状态，状态转换为 WAIT_SERVER
6. 释放锁，返回座席信息
7. 如果分配失败（座席不在线/状态不对），自动重试下一个，最多重试 3 次

### 心跳保活机制

- 座席会话 key 带 TTL（默认 120 秒）
- 座席客户端定时调用心跳接口（建议 30 秒一次）
- 心跳接口刷新 session TTL 和 lastActiveTime
- 后台定时扫描器（默认 30 秒）扫描所有在线集合
- 发现 session 已过期的座席，自动清理并置为 OFFLINE

### 空闲座席 ZSET

- 每个技能组一个 ZSET：`vocara:agent:idle:{skillGroupId}`
- score = lastReadyTimestamp（进入 READY 状态的时间戳）
- 使用 `ZPOPMIN` 弹出最早就绪的座席（最长空闲优先）
- 座席离开 READY 状态时从所有相关 ZSET 中移除

## 配置说明

配置文件位置：`configs/config.yaml`，为本机标准配置（中间件指向 `127.0.0.1` 标准端口），支持通过环境变量 `CONFIG_PATH` 指定路径。开发联调环境配置为 `configs/config.local.yaml`（指向共享联调主机，不进版本库），启动时显式指定：

```bash
CONFIG_PATH=configs/config.local.yaml go run cmd/main.go
```

```yaml
server:
  host: "0.0.0.0"
  port: 8080

mysql:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "123456"
  database: "vocara_agent"

redis:
  host: "127.0.0.1"
  port: 6379
  db: 0

jwt:
  secret: "your-jwt-secret"
  expire_hour: 24

agent:
  session_ttl: 120       # 会话过期时间（秒）
  scan_interval: 30000   # 过期扫描间隔（毫秒）

internal:
  token: "internal-token"  # 内部服务鉴权Token
```

## 快速开始

### 环境要求

- Go 1.22+
- MySQL 5.7+ / 8.0+
- Redis 6.0+

### 启动服务

```bash
# 进入项目目录
cd services/agentcenter

# 安装依赖
go mod tidy

# 启动服务
go run cmd/main.go

# 或指定配置文件
CONFIG_PATH=/path/to/config.yaml go run cmd/main.go
```

### 构建

```bash
# 构建二进制
go build -o bin/agent-center cmd/main.go

# 运行
./bin/agent-center
```

### 健康检查

```bash
curl http://localhost:8080/health
# {"status":"ok","service":"agent-center"}
```

## 错误码

| 错误码 | 说明 |
|--------|------|
| 0 | 成功 |
| 40000 | 参数错误 |
| 40100 | 未授权 |
| 40101 | Token 无效 |
| 40102 | Token 已过期 |
| 40103 | Token 缺失 |
| 40300 | 禁止访问 |
| 40400 | 资源不存在 |
| 50000 | 系统错误 |
| 50002 | 数据库错误 |
| 50003 | Redis 错误 |
| 10000 | 座席不存在 |
| 10001 | 座席未签入 |
| 10002 | 座席已签入 |
| 10003 | 座席忙 |
| 10004 | 座席已离线 |
| 10005 | 座席状态无效 |
| 10006 | 密码错误 |
| 10007 | 座席已在线 |
| 10008 | 座席未就绪 |
| 10009 | 状态转换不合法 |
| 10010 | 无空闲座席 |
| 10011 | 分机不存在 |
| 10012 | 分机不可用 |
| 10013 | 获取分布式锁失败 |

## 依赖

- [Hertz](https://github.com/cloudwego/hertz) - HTTP 框架
- [GORM](https://gorm.io/) - ORM 框架
- [go-redis](https://github.com/redis/go-redis) - Redis 客户端
- [golang-jwt](https://github.com/golang-jwt/jwt) - JWT 库
- [zap](https://github.com/uber-go/zap) - 日志库
- [viper](https://github.com/spf13/viper) - 配置管理
- [bcrypt](https://golang.org/x/crypto/bcrypt) - 密码加密

## License

Vocara
