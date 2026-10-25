# vocara 呼叫中心系统 — 技术设计文档

Vocara 是基于 Go + FreeSWITCH 的呼叫中心系统，采用微服务架构。本目录是系统的
技术设计文档集，覆盖整体架构、项目结构、ACD 排队、座席状态与 WebSocket、
ESL 交互、数据库、部署、电话条、管理后台菜单与 License 授权共十个章节。

## 文档导读

| 章节 | 内容 |
|------|------|
| [01 整体架构设计](01-整体架构设计.md) | 架构总览、服务职责、服务间通信、事件总线、技术栈 |
| [02 项目结构设计](02-项目结构设计.md) | 目录结构、代码分层、公共库与契约、工作区与构建 |
| [03 ACD 双 ZSET 方案设计](03-ACD双ZSET方案设计.md) | 双 ZSET 排队、Lua 原子脚本、事件驱动调度、优先级老化 |
| [04 座席状态与 WebSocket](04-座席状态与WebSocket.md) | 座席状态机、技能组、WebSocket 推送、多节点路由 |
| [05 FreeSWITCH ESL 交互层](05-FreeSWITCH_ESL交互层.md) | 自研 ESL 客户端、连接池、呼叫编排、事件分发 |
| [06 数据库表设计](06-数据库表设计.md) | 核心表结构、话单表与分区 |
| [07 部署方案](07-部署方案.md) | 交付物形态、Kubernetes 部署、运行期配置管理 |
| [08 电话条组件设计方案](08-电话条组件设计方案.md) | 业务 WS 与 SIP 双连接架构、原生 JavaScript SDK、管理后台集成 |
| [09 管理后台菜单设计](09-管理后台菜单设计.md) | 菜单、路由与权限码约定、各角色可见范围 |
| [10 License 授权与核心资产保护方案](10-License授权与核心资产保护方案.md) | 离线授权文件、签发体系、到期策略、核心资产保护 |

## 服务构成

| 服务 | 端口 | 职责 |
|---|---|---|
| vocara-admin | 8080 | 管理后台 API：座席、分机、技能组、队列、线路、呼入路由、租户、参数、话单、统计、License、表单 |
| vocara-agentcenter | 8081 | 座席状态机、技能组、分机、座席 Redis 会话 |
| vocara-acd | 8082 | ACD 智能排队：active/paused 双 ZSET + Lua 原子脚本 |
| vocara-callctrl | 8083 | 呼叫控制：ESL 事件、呼入呼出、录音、监听/耳语、CDR |
| vocara-push | 8084 | WebSocket 实时推送：NATS + Redis 双消费者 |

公共库 `vocara/pkg` 提供 config、database、errors、eventbus、logger、middleware、
redis、response、utils、version 等基础能力；契约模块 `vocara/contracts` 提供
跨边界事件 DTO 与 HTTP DTO。服务分层为 handler → service → repository → model，
跨服务调用走 HTTP 或事件，不直接访问对方存储。

## FreeSWITCH 双节点

```
                运营商线路            座席话机 / 浏览器软电话
                     │                   │
                     │ SIP               │ SIP / WebRTC
                     ▼                   ▼
            ┌─────────────┐  SIP   ┌─────────────┐
            │   FS-SBC    │◄──────►│  FS-Agent   │
            │  (接入侧)    │        │  (座席侧)   │
            │  客户侧媒体  │        │ 座席侧媒体  │
            └─────────────┘        └──────┬──────┘
                                         │ ESL :8021
                                         ▼
                                ┌───────────────┐
                                │ vocara-callctrl│
                                │  (呼叫控制)    │
                                └───────────────┘
```

呼叫控制由 vocara-callctrl 经单个 ESL Inbound 连接 FS-Agent 统一编排，运营商来话
由 FS-SBC 经 SIP 中继转发至 FS-Agent；FreeSWITCH 只负责媒体与协议处理。

## 技术栈

| 类别 | 选型 |
|---|---|
| HTTP 框架 | Hertz |
| ORM / 数据库 | GORM v2 / MySQL 8.0 |
| 缓存 / 客户端 | Redis 7 / go-redis v9 |
| 消息 | NATS JetStream |
| 日志 | zap，结构化输出到 stdout |
| 配置 | viper，snake_case YAML，CONFIG_PATH 可覆盖路径 |
| 鉴权 | JWT |
| ESL | 自研客户端（核心私有仓库 esl） |
| 前端 | Vue 3 + TypeScript + Naive UI + Pinia |
