# Vocara Admin Service

Vocara 呼叫中心管理后台服务，提供座席管理、队列管理、技能组管理、呼入接待配置、通话记录、统计报表等功能。

## 技术栈

- **框架**: Hertz (CloudWeGo)
- **数据库**: MySQL (GORM v2)
- **缓存**: Redis (go-redis v9)
- **认证**: JWT
- **日志**: Zap + Lumberjack
- **配置**: Viper

## 项目结构

```
services/admin/
├── cmd/
│   └── main.go              # 服务入口
├── configs/
│   └── config.yaml          # 配置文件
├── internal/
│   ├── handler/             # HTTP 处理器层
│   │   ├── auth_handler.go       # 认证
│   │   ├── agent_handler.go      # 座席管理
│   │   ├── agent_group_handler.go # 座席组管理
│   │   ├── skill_group_handler.go # 技能组管理
│   │   ├── dn_handler.go         # 分机管理
│   │   ├── queue_handler.go      # 队列管理
│   │   ├── admit_handler.go      # 呼入接待
│   │   ├── record_handler.go     # 通话记录
│   │   ├── line_handler.go       # 线路资源
│   │   ├── stats_handler.go      # 统计报表
│   │   └── system_handler.go     # 系统管理
│   ├── service/             # 业务逻辑层
│   │   ├── agent_service.go
│   │   ├── agent_group_service.go
│   │   ├── skill_group_service.go
│   │   ├── dn_service.go
│   │   ├── queue_service.go
│   │   ├── admit_service.go
│   │   ├── record_service.go
│   │   ├── line_service.go
│   │   ├── stats_service.go
│   │   ├── system_service.go
│   │   └── tenant_service.go
│   ├── repository/          # 数据访问层
│   │   ├── base_repo.go
│   │   ├── agent_repo.go
│   │   ├── agent_group_repo.go
│   │   ├── skill_group_repo.go
│   │   ├── dn_repo.go
│   │   ├── queue_repo.go
│   │   ├── admit_repo.go
│   │   ├── record_repo.go
│   │   ├── line_repo.go
│   │   └── system_repo.go
│   └── model/               # 数据模型
│       ├── models.go
│       └── stats.go
├── go.mod
└── README.md
```

## API 接口

所有接口前缀: `/api/v1/admin/`

### 认证接口

| 方法 | 路径 | 说明 | 鉴权 |
|------|------|------|------|
| POST | /auth/login | 管理员登录 | 否 |
| POST | /auth/logout | 登出 | 是 |
| POST | /auth/change_password | 修改密码 | 是 |
| GET  | /auth/userinfo | 获取用户信息 | 是 |

### 座席管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /agents/page | 分页查询 |
| GET | /agents/list | 列表查询 |
| GET | /agents/:id | 详情 |
| POST | /agents/save | 新增 |
| POST | /agents/update | 更新 |
| DELETE | /agents/:id | 删除 |
| POST | /agents/import | 批量导入 |
| GET | /agents/export | 导出 |
| POST | /agents/:id/skill | 更新技能 |
| POST | /agents/:id/bind_dn | 绑定分机 |
| POST | /agents/:id/reset_password | 重置密码 |
| POST | /agents/:id/enable | 启用 |
| POST | /agents/:id/disable | 禁用 |

### 座席组管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /agent_groups/page | 分页查询 |
| GET | /agent_groups/list | 列表查询 |
| GET | /agent_groups/:id | 详情 |
| POST | /agent_groups/save | 新增 |
| POST | /agent_groups/update | 更新 |
| DELETE | /agent_groups/:id | 删除 |
| GET | /agent_groups/:id/members | 成员列表 |
| POST | /agent_groups/:id/members/add | 添加成员 |
| POST | /agent_groups/:id/members/remove | 移除成员 |
| POST | /agent_groups/:id/leader | 设置组长 |

### 技能组管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /skill_groups/page | 分页查询 |
| GET | /skill_groups/list | 列表查询 |
| GET | /skill_groups/:id | 详情 |
| POST | /skill_groups/save | 新增 |
| POST | /skill_groups/update | 更新 |
| DELETE | /skill_groups/:id | 删除 |
| GET | /skill_groups/:id/agents | 座席列表 |
| POST | /skill_groups/:id/agents | 设置座席 |

### 分机管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /dns/page | 分页查询 |
| GET | /dns/list | 列表查询 |
| GET | /dns/:id | 详情 |
| POST | /dns/save | 新增 |
| POST | /dns/update | 更新 |
| DELETE | /dns/:id | 删除 |
| POST | /dns/status_check | 状态检查 |
| POST | /dns/batch_import | 批量导入 |
| GET | /dns/available | 可用分机列表 |
| POST | /dns/:id/bind_agent | 绑定座席 |
| POST | /dns/:id/unbind_agent | 解绑座席 |

### 队列管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /queues/page | 分页查询 |
| GET | /queues/list | 列表查询 |
| GET | /queues/:id | 详情 |
| POST | /queues/save | 新增 |
| POST | /queues/update | 更新 |
| DELETE | /queues/:id | 删除 |
| GET | /queues/monitor/list | 实时监控 |
| GET | /queues/:id/agents | 队列座席 |

### 呼入接待

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /admits/page | 分页查询 |
| GET | /admits/list | 列表查询 |
| GET | /admits/:id | 详情 |
| POST | /admits/save | 新增 |
| POST | /admits/update | 更新 |
| DELETE | /admits/:id | 删除 |
| GET | /admits/holiday/page | 节假日分页 |
| GET | /admits/holiday/list | 节假日列表 |
| GET | /admits/holiday/:id | 节假日详情 |
| POST | /admits/holiday/save | 新增节假日 |
| POST | /admits/holiday/update | 更新节假日 |
| DELETE | /admits/holiday/:id | 删除节假日 |

### 通话记录

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /records/inbound/page | 呼入记录分页 |
| GET | /records/outbound/page | 呼出记录分页 |
| GET | /records/:id | 通话详情 |
| GET | /records/export | 导出 |
| GET | /records/today | 今日统计 |
| GET | /records/uncalled | 未接列表 |
| GET | /records/:id/recording | 录音信息 |

### 线路资源

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /lines/page | 分页查询 |
| GET | /lines/list | 列表查询 |
| GET | /lines/:id | 详情 |
| POST | /lines/save | 新增 |
| POST | /lines/update | 更新 |
| DELETE | /lines/:id | 删除 |
| GET | /lines/status/info | 状态统计 |

### 统计报表

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /stats/agent | 座席话务统计 |
| GET | /stats/queue | 队列统计 |
| GET | /stats/line | 线路统计 |
| GET | /stats/trend | 趋势统计 |
| GET | /stats/realtime | 实时统计 |

### 系统管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /system/param/list | 参数配置列表 |
| POST | /system/param/save | 保存配置 |
| DELETE | /system/param/:id | 删除配置 |
| GET | /system/operation_log/page | 操作日志分页 |
| GET | /system/fs/page | FS实例分页 |
| GET | /system/fs/list | FS实例列表 |
| GET | /system/fs/:id | FS实例详情 |
| POST | /system/fs/save | 新增FS实例 |
| POST | /system/fs/update | 更新FS实例 |
| DELETE | /system/fs/:id | 删除FS实例 |
| GET | /system/tenant/page | 租户分页 |
| GET | /system/tenant/list | 租户列表 |
| GET | /system/tenant/:id | 租户详情 |
| POST | /system/tenant/save | 新增租户 |
| POST | /system/tenant/update | 更新租户 |
| DELETE | /system/tenant/:id | 删除租户 |

## 中间件

请求经过以下中间件链:

1. **TraceID** - 生成/透传追踪 ID
2. **Recovery** - Panic 恢复
3. **CORS** - 跨域处理
4. **RequestLogger** - 请求日志
5. **JWT** - 鉴权（登录接口跳过）

## 多租户设计

- 所有业务表都有 `tenant_id` 字段
- 管理员登录后获取默认租户
- 查询操作自动携带租户隔离条件
- 系统级数据（FS实例、全局配置）不限制租户

## 启动

```bash
# 安装依赖
go mod download

# 构建
go build -o admin ./cmd/main.go

# 运行
./admin
```

## 配置

参见 `configs/config.yaml`（本机标准配置），支持环境变量覆盖（前缀 `VOCARA_`）。开发联调环境使用 `configs/config.local.yaml`（指向共享联调主机，不进版本库），通过 `CONFIG_PATH=configs/config.local.yaml` 加载。
