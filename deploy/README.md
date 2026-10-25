# Vocara 产物包说明

`dist/` 为构建产物包，包含五个后端服务二进制、前端构建产物、配置模板、Windows 版 Nginx 运行时与一键启停脚本。
产物包不含 Go 源码，呼叫控制（callctrl）以二进制交付。

## 目录结构

```
dist/
├── bin/          # vocara-admin / agentcenter / acd / callctrl / push
├── configs/      # 五份服务配置（按实际环境修改）
├── web/          # 管理后台前端构建产物
├── nginx/        # Windows 版 Nginx 运行时（统一入口，五组反向代理）
├── license/      # License 目录（vocara.lic 单独交付后放入）
├── records/      # 通话录音目录
├── logs/         # 运行日志目录
├── start.cmd     # Windows 一键启动（五个服务 + Nginx）
└── stop.cmd      # Windows 一键停止
```

## 一、一键启动（Windows）

1. 按需修改 `configs/` 中各服务的 MySQL / Redis / NATS / ESL 地址；
2. 双击 `start.cmd`，脚本以产物根为工作目录依次启动五个后端服务与 Nginx；
3. 浏览器访问 http://localhost ；
4. 停止时双击 `stop.cmd`。

端口被占用时对应服务自动跳过，避免重复启动。

## 二、License 放置

`vocara.lic` 经厂商单独交付，与目标机器指纹匹配。收到后放入 `dist/license/vocara.lic`，
服务启动时按配置 `license.file` 加载。

## 三、非 Windows 裸机运行

1. 修改 `configs/` 中各服务的 MySQL / Redis / NATS / ESL 地址；
2. 执行 `./start.sh` 启动五个后端服务（`./stop.sh` 停止）；
3. 前端入口：将 `web/` 作为站点根目录，把 `nginx/nginx.conf` 站点片段纳入系统 Nginx 的 http 块，
   按实际地址调整其中上游地址（或在渲染配置时提供 `VOCARA_*_ADDR` 环境变量）；
4. 浏览器访问 Nginx 站点（默认 80 端口），TLS 由站点或前置负载均衡终止。

## 四、基础设施要求

- MySQL 8.0：库表经 `db/migrations/001_init_schema.sql` 初始化；
- Redis 7、NATS JetStream、FreeSWITCH（WebRTC WebSocket 端口 5066）。

镜像构建文件位于 `deploy/docker/`，Kubernetes 部署清单位于 `deploy/k8s/`。
