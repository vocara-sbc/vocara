# Vocara 呼叫中心系统

> Vocal + Atelier — 专业语音工作坊

智能呼叫中心系统，基于 Go 语言 + FreeSWITCH 构建，采用微服务架构。


## 微服务列表

| 服务 | 端口 | 职责 |
|------|------|------|
| vocara-admin | 8080 | 管理后台 API |
| vocara-agentcenter | 8081 | 座席中心 |
| vocara-acd | 8082 | ACD 智能排队 |
| vocara-callctrl | 8083 | 呼叫控制 |
| vocara-push | 8084 | WebSocket 实时推送 |

## 技术栈

- **语言**：Go 1.22
- **Web 框架**：Hertz
- **ORM**：GORM v2
- **缓存**：Redis 7 + go-redis v9
- **消息队列**：NATS JetStream
- **数据库**：MySQL 8.0
- **FS 交互**：自研 ESL 客户端（核心私有仓库 `esl`，随 callctrl 二进制交付）

## 目录结构

```
vocara/
├── contracts/    # 跨边界契约
├── pkg/          # 公共库
├── business/     # 业务区：四个微服务 + web
├── bin/          # 预编译核心二进制：vocara-callctrl（linux + windows）
├── configs/      # 核心运行期配置模板：callctrl.yaml
├── db/           # 数据库脚本
├── deploy/k8s/   # Kubernetes 部署清单
├── docs/         # 设计文档
├── Makefile      # 二进制交付流水线
├── run.cmd       # Windows 统一入口
├── run.sh        # Linux / macOS 统一入口
└── go.work
```

核心源码（callctrl、ESL、电话条、FreeSWITCH、license-cli）位于独立私有仓库 `core`（与 `vocara` 同级），本仓库仅接收其发布产物：`bin/vocara-callctrl(.exe)`、混淆电话条与 `configs/callctrl.yaml`。共享库 `pkg` 与契约 `contracts` 以本仓库为单一事实来源，核心仓库经 go.work 引用，依赖方向严格为私有 → 公开。

## 统一入口脚本

`run.cmd`（Windows）与 `run.sh`（Linux / macOS，Windows 下经 Git Bash 执行）是开发与构建的统一入口，同一组子命令：

```bash
# Windows
run.cmd <命令> [参数]

# Linux / macOS
./run.sh <命令> [参数]
```

| 命令 | 作用 |
|------|------|
| `dev [services\|web]` | 开发模式：后端 `go run` 源码 + 前端 vite dev（默认全部启动） |
| `build [linux\|windows\|darwin]` | 构建统一产物包到 `dist/`（默认本机平台） |
| `start [services\|web]` | 产物模式：后端运行 `dist/` 二进制 + 前端 vite dev |
| `stop` | 停止由脚本启动的进程 |
| `image <服务\|web\|all>` | 构建 Docker 镜像（在 Linux 平台执行） |
| `help` | 显示帮助 |

### 开发调试

```bash
./run.sh dev          # 启动五个后端服务与前端，访问 https://localhost:5173
./run.sh dev services # 仅启动后端服务
```

### 产物构建与联调

```bash
./run.sh build        # 产物输出到 dist/：bin、configs、web、nginx 等
./run.sh start        # 后端运行产物二进制，前端仍走 vite dev
```

Windows 产物双击 `dist/start.cmd` 即可启动全部服务，浏览器访问 http://localhost ，停止双击 `stop.cmd`。

`dist/` 不含 Go 源码，callctrl 以二进制交付。`vocara.lic` 经单独交付后放入 `dist/license/` 目录；`license-cli` 为厂商内部签发工具，不进入产物包。

基础设施（MySQL、Redis、NATS、FreeSWITCH）地址在各服务配置或 `dist/configs/` 中调整。

### Docker 镜像

```bash
./run.sh image all    # 构建五个后端镜像与前端镜像（Linux 平台）
```

镜像构建文件位于 `deploy/docker/`，部署清单位于 `deploy/k8s/`，详见 [docs/07-部署方案.md](docs/07-部署方案.md)。
