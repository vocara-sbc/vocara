#!/usr/bin/env bash
# ============================================================
# Vocara 开发与构建统一入口（Linux / macOS / Git Bash）
#
# 用法：./run.sh <命令> [参数]
#   dev [services|web] [local|standard]
#                              开发模式：后端 go run 源码 + 前端 vite dev（默认全部）
#                                默认配置集 local（config.local.yaml，开发联调环境），
#                                显式 standard 切回本机标准 config.yaml；
#                                也可用环境变量 VOCARA_CONFIG_PROFILE 指定
#   build [os]          构建产物包到 dist（os：本机 | linux | windows）
#   start [services|web] 产物模式：后端运行 dist 二进制 + 前端 vite dev
#   stop                停止由本脚本启动的进程
#   image <服务|web|all> 构建 Docker 镜像（在 Linux 构建平台执行）
#   help                显示帮助
#
# 基础设施（MySQL / Redis / NATS / FreeSWITCH）需自行准备并保证可达。
# ============================================================

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROC_FILE="$ROOT_DIR/.run_processes"
LOG_DIR="$ROOT_DIR/logs"
DIST_DIR="$ROOT_DIR/dist"

# 固定 Go 1.22 工具链（项目 go.mod 基线为 1.22；可用 VOCARA_GOTOOLCHAIN 覆盖）
export GOTOOLCHAIN="${VOCARA_GOTOOLCHAIN:-go1.22.12}"

# ---- 版本信息（注入 vocara/pkg/version）----
VERSION="${VOCARA_VERSION:-$(git -C "$ROOT_DIR" describe --tags --always 2>/dev/null || echo dev)}"
GIT_COMMIT="${VOCARA_GIT_COMMIT:-$(git -C "$ROOT_DIR" rev-parse --short HEAD 2>/dev/null || echo unknown)}"
BUILD_TIME="${VOCARA_BUILD_TIME:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
LD_FLAGS="-s -w \
  -X vocara/pkg/version.Version=$VERSION \
  -X vocara/pkg/version.BuildTime=$BUILD_TIME \
  -X vocara/pkg/version.GitCommit=$GIT_COMMIT"

# ---- 镜像仓库（可用环境变量覆盖）----
REGISTRY="${VOCARA_REGISTRY:-registry.example.com/vocara}"

# ---- 颜色 ----
INFO="\033[36m"; OK="\033[32m"; WARN="\033[33m"; ERR="\033[31m"; RESET="\033[0m"
info()    { echo -e "${INFO}[INFO]${RESET}  $1"; }
success() { echo -e "${OK}[OK]${RESET}    $1"; }
warn()    { echo -e "${WARN}[WARN]${RESET}  $1"; }
error()   { echo -e "${ERR}[ERROR]${RESET} $1"; }

# ---- 平台判定 ----
IS_WINDOWS=false
case "$(uname -s)" in
  MINGW*|MSYS*|CYGWIN*) IS_WINDOWS=true ;;
esac

# ---- 后端服务清单：名称|端口|源码入口|默认配置|本地联调配置|产物配置名 ----
SERVICES=(
  "vocara-admin|8080|business/services/admin/cmd|business/services/admin/configs/config.yaml|business/services/admin/configs/config.local.yaml|admin.yaml"
  "vocara-agentcenter|8081|business/services/agentcenter/cmd|business/services/agentcenter/configs/config.yaml|business/services/agentcenter/configs/config.local.yaml|agentcenter.yaml"
  "vocara-acd|8082|business/services/acd/cmd|business/services/acd/configs/config.yaml|business/services/acd/configs/config.local.yaml|acd.yaml"
  "vocara-callctrl|8083|bin/vocara-callctrl|configs/callctrl.yaml|configs/callctrl.yaml|callctrl.yaml"
  "vocara-push|8084|business/services/push/cmd|business/services/push/configs/config.yaml|business/services/push/configs/config.local.yaml|push.yaml"
)

# 预编译交付的服务：不随本仓库源码 go build / go run，直接运行 bin/ 下随仓库提供的二进制。
is_prebuilt() { [ "$1" = "vocara-callctrl" ]; }

# Docker 镜像构建时各服务的编译入口
image_pkg() {
  case "$1" in
    admin)       echo "./business/services/admin/cmd" ;;
    agentcenter) echo "./business/services/agentcenter/cmd" ;;
    acd)         echo "./business/services/acd/cmd" ;;
    push)        echo "./business/services/push/cmd" ;;
    *) return 1 ;;
  esac
}

mkdir -p "$LOG_DIR"

# 端口占用检测（lsof / netstat 双实现，兼容 Git Bash）
port_in_use() {
  local port="$1"
  if command -v lsof >/dev/null 2>&1; then
    lsof -i ":$port" >/dev/null 2>&1
  elif command -v netstat >/dev/null 2>&1; then
    netstat -ano 2>/dev/null | grep -qE ":$port[[:space:]].*LISTENING"
  else
    return 1
  fi
}

# Windows 下按监听端口取真实 PID（Git Bash 的 $! 为 MSYS 虚拟 PID，taskkill 无法识别）。
# 无监听时输出空串且返回 0，避免 set -e 下命令替换的非零状态终止调用方。
win_pid_by_port() {
  local port="$1" pid=""
  pid="$(netstat -ano 2>/dev/null \
    | grep -E ":$port[[:space:]].*LISTENING" \
    | awk '{print $NF}' | sort -u | head -1)" || true
  printf '%s' "$pid"
}

# 跨平台停止进程树
stop_pid_tree() {
  local pid="$1" port="$2"
  if $IS_WINDOWS; then
    # 优先按端口定位真实 PID 杀进程树；端口已无监听时兜底按 MSYS PID 终止
    local winpid
    winpid="$(win_pid_by_port "$port")"
    if [ -n "$winpid" ]; then
      # // 前缀由 MSYS 转为 /，避免单斜杠参数被当作 POSIX 路径
      taskkill //PID "$winpid" //T //F >/dev/null 2>&1 || true
    else
      kill "$pid" 2>/dev/null || true
    fi
  else
    kill "$pid" 2>/dev/null || true
  fi
}

# ============================================================
# dev：源码开发模式（go run + vite）
# ============================================================
start_dev_services() {
  info "开发模式：go run 启动后端微服务..."
  command -v go >/dev/null 2>&1 || { error "未检测到 Go 环境，请先安装 Go 1.22+"; exit 1; }

  if [ "$CONFIG_PROFILE" = "local" ]; then
    info "配置集：开发联调环境（config.local.yaml）"
  else
    info "配置集：本机标准环境（config.yaml）"
  fi

  > "$PROC_FILE"
  local started=0
  for row in "${SERVICES[@]}"; do
    IFS='|' read -r name port entry config local_config _ <<< "$row"

    if port_in_use "$port"; then
      warn "端口 $port 已被占用，跳过 $name"
      continue
    fi

    local cfg="$config"
    [ "$CONFIG_PROFILE" = "local" ] && cfg="$local_config"
    if [ ! -f "$ROOT_DIR/$cfg" ]; then
      error "配置文件不存在：$ROOT_DIR/$cfg"
      exit 1
    fi

    info "启动 $name (端口 $port)..."
    # 应用日志由 zap 写入 logs/<服务短名>.log；startup.log 仅承载 zap 初始化前的
    # 早期致命输出（配置/日志初始化失败等），正常启动时为空。
    cd "$ROOT_DIR"
    if is_prebuilt "$name"; then
      local bin="$ROOT_DIR/$entry"
      $IS_WINDOWS && bin="$bin.exe"
      [ -f "$bin" ] || { error "预编译二进制缺失：$bin（请先在 core 仓库执行 make sync）"; exit 1; }
      CONFIG_PATH="$ROOT_DIR/$cfg" nohup "$bin" >> "$LOG_DIR/${name#vocara-}.startup.log" 2>&1 &
    else
      CONFIG_PATH="$ROOT_DIR/$cfg" nohup go run "./$entry" >> "$LOG_DIR/${name#vocara-}.startup.log" 2>&1 &
    fi
    local pid=$!
    echo "$name $port $pid" >> "$PROC_FILE"
    started=$((started + 1))
    sleep 0.5
  done

  success "已启动 $started 个后端服务"
  show_service_urls
}

# ============================================================
# start：产物二进制模式（前端仍走 vite 代理）
# ============================================================
start_release_services() {
  if [ ! -d "$DIST_DIR/bin" ]; then
    error "未找到产物目录 $DIST_DIR/bin，请先执行：./run.sh build"
    exit 1
  fi

  info "产物模式：启动 dist 二进制..."
  > "$PROC_FILE"
  local started=0
  for row in "${SERVICES[@]}"; do
    IFS='|' read -r name port _ _ _ cfgname <<< "$row"

    if port_in_use "$port"; then
      warn "端口 $port 已被占用，跳过 $name"
      continue
    fi

    local bin="$DIST_DIR/bin/$name"
    $IS_WINDOWS && bin="$bin.exe"
    if [ ! -x "$bin" ]; then
      error "二进制不存在或不可执行：$bin"
      exit 1
    fi

    info "启动 $name (端口 $port)..."
    # 工作目录为产物根：配置中的 ./logs、./license、./records 相对路径均以此为基准。
    cd "$DIST_DIR"
    CONFIG_PATH="$DIST_DIR/configs/$cfgname" nohup "$bin" >/dev/null 2>&1 &
    local pid=$!
    echo "$name $port $pid" >> "$PROC_FILE"
    started=$((started + 1))
    sleep 0.5
  done

  success "已启动 $started 个后端服务（产物二进制）"
  show_service_urls
}

show_service_urls() {
  info "服务端口一览："
  echo "  admin       -> http://localhost:8080"
  echo "  agentcenter -> http://localhost:8081"
  echo "  acd         -> http://localhost:8082"
  echo "  callctrl    -> http://localhost:8083"
  echo "  push        -> ws://localhost:8084/ws"
  info "停止： ./run.sh stop"
}

# ============================================================
# 前端 vite dev
# ============================================================
start_web() {
  info "启动前端 vite dev (端口 5173)..."
  local web_dir="$ROOT_DIR/business/web/admin"
  [ -d "$web_dir" ] || { warn "前端目录不存在：$web_dir"; return; }
  command -v node >/dev/null 2>&1 || { warn "未检测到 Node.js，跳过前端"; return; }

  if [ ! -d "$web_dir/node_modules" ]; then
    info "首次启动，安装前端依赖..."
    (cd "$web_dir" && npm install)
  fi
  if port_in_use 5173; then
    warn "端口 5173 已被占用，跳过前端"
    return
  fi

  cd "$web_dir"
  NO_COLOR=1 nohup npm run dev > "$LOG_DIR/web-admin.log" 2>&1 &
  local pid=$!
  echo "web-admin 5173 $pid" >> "$PROC_FILE"
  success "前端已启动 (PID $pid)"
  info "访问地址： https://localhost:5173"
}

# ============================================================
# stop
# ============================================================
# 固定端口清单：五个后端服务与 vite，供 stop 兜底清理进程记录之外的游离进程
KNOWN_PORTS="8080 8081 8082 8083 8084 5173"

# 按固定端口清理游离进程（端口被占用但不在进程记录中的进程）。
# 清理数量写入全局 ORPHAN_CLEANED，函数始终返回 0，规避 set -e 下 return 非零
# 导致脚本提前退出。
cleanup_orphan_ports() {
  ORPHAN_CLEANED=0
  for port in $KNOWN_PORTS; do
    if $IS_WINDOWS; then
      local winpid
      winpid="$(win_pid_by_port "$port")"
      [ -z "$winpid" ] && continue
      taskkill //PID "$winpid" //T //F >/dev/null 2>&1 || true
    else
      command -v lsof >/dev/null 2>&1 || continue
      lsof -ti ":$port" >/dev/null 2>&1 || continue
      lsof -ti ":$port" | xargs -r kill 2>/dev/null || true
    fi
    info "已释放端口 $port 的游离进程"
    ORPHAN_CLEANED=$((ORPHAN_CLEANED + 1))
  done
  return 0
}

stop_all() {
  info "停止由本脚本启动的进程..."
  local killed=0
  if [ -f "$PROC_FILE" ]; then
    while read -r name port pid _; do
      [ -z "${name:-}" ] && continue
      stop_pid_tree "$pid" "$port"
      info "已停止 $name (端口 $port)"
      killed=$((killed + 1))
    done < "$PROC_FILE"
  else
    warn "未找到进程记录文件，改按固定端口清理"
  fi
  rm -f "$PROC_FILE"

  # 兜底：清理因端口占用跳过启动而未写入记录的游离进程（如旧 vite）
  cleanup_orphan_ports
  local orphans="$ORPHAN_CLEANED"
  local total=$((killed + orphans))
  if [ "$total" -eq 0 ]; then
    info "没有需要停止的进程"
  else
    success "共停止 $total 个进程（记录内 $killed，端口兜底 $orphans）"
  fi
}

# ============================================================
# build：产物包
# ============================================================
build_web() {
  info "构建前端管理后台..."
  local web_dir="$ROOT_DIR/business/web/admin"
  if [ ! -d "$web_dir/node_modules" ]; then
    (cd "$web_dir" && npm install)
  fi
  (cd "$web_dir" && npm run build)
}

# 渲染 nginx 配置：envsubst 可用时按限定变量列表渲染，否则 sed 逐变量替换
render_nginx() {
  local tpl="$1" out="$2"
  local vars=(
    VOCARA_ADMIN_ADDR VOCARA_AGENTCENTER_ADDR VOCARA_ACD_ADDR
    VOCARA_CALLCTRL_ADDR VOCARA_PUSH_ADDR VOCARA_SIP_ADDR VOCARA_WEB_ROOT
  )
  if command -v envsubst >/dev/null 2>&1; then
    local names=""; for v in "${vars[@]}"; do names="$names \${$v}"; done
    envsubst "$names" < "$tpl" > "$out"
  else
    local sed_args=()
    for v in "${vars[@]}"; do
      sed_args+=("-e" "s|\${$v}|${!v}|g")
    done
    sed "${sed_args[@]}" < "$tpl" > "$out"
  fi
}

# 统一渲染 server 片段（上游指向本机各端口）
render_local_fragment() {
  local out="$1" web_root="$2"
  VOCARA_ADMIN_ADDR="${VOCARA_ADMIN_ADDR:-127.0.0.1:8080}" \
  VOCARA_AGENTCENTER_ADDR="${VOCARA_AGENTCENTER_ADDR:-127.0.0.1:8081}" \
  VOCARA_ACD_ADDR="${VOCARA_ACD_ADDR:-127.0.0.1:8082}" \
  VOCARA_CALLCTRL_ADDR="${VOCARA_CALLCTRL_ADDR:-127.0.0.1:8083}" \
  VOCARA_PUSH_ADDR="${VOCARA_PUSH_ADDR:-127.0.0.1:8084}" \
  VOCARA_SIP_ADDR="${VOCARA_SIP_ADDR:-127.0.0.1:5066}" \
  VOCARA_WEB_ROOT="$web_root" \
    render_nginx "$ROOT_DIR/deploy/docker/nginx/nginx.conf.template" "$out"
}

# package_windows_runtime 归集 Windows 版 Nginx 运行时（lib/nginx-*），
# 组装完整配置（events/http）并生成 start.cmd / stop.cmd
package_windows_runtime() {
  local nginx_src
  nginx_src="$(find "$ROOT_DIR/lib" -maxdepth 1 -type d -name 'nginx-*' | sort | tail -1)"
  [ -n "$nginx_src" ] && [ -f "$nginx_src/nginx.exe" ] \
    || { error "未找到 Windows 版 Nginx（lib/nginx-*/nginx.exe）"; exit 1; }

  info "归集 Nginx 运行时..."
  rm -rf "$DIST_DIR/nginx"
  cp -r "$nginx_src" "$DIST_DIR/nginx"

  info "渲染 Nginx 配置..."
  local fragment="$DIST_DIR/nginx/conf/.server.conf"
  # root 相对 Nginx prefix（dist/nginx），指向 dist/web
  render_local_fragment "$fragment" "../web"

  # 模板是容器 conf.d 片段（map + server），Windows Nginx 需要完整配置，
  # 外包 events/http 并引入发行版自带 mime.types
  {
    echo '# Vocara 前端入口 Nginx 完整配置（Windows 运行时）'
    echo 'events {'
    echo '    worker_connections 4096;'
    echo '}'
    echo ''
    echo 'http {'
    echo '    include       mime.types;'
    echo '    default_type  application/octet-stream;'
    echo '    sendfile      on;'
    echo '    keepalive_timeout 65;'
    sed 's/^/    /' "$fragment"
    echo '}'
  } > "$DIST_DIR/nginx/conf/nginx.conf"
  rm -f "$fragment"

  generate_windows_scripts
}

# package_posix_runtime 输出裸机站点片段（供系统 Nginx 纳入 http 块）
# 与 start.sh / stop.sh
package_posix_runtime() {
  info "渲染裸机 Nginx 配置..."
  mkdir -p "$DIST_DIR/nginx"
  # root 相对站点片段被 include 时的 prefix，裸机部署按实际目录调整
  render_local_fragment "$DIST_DIR/nginx/nginx.conf" "../../web"
  generate_posix_scripts
}

# generate_windows_scripts 生成 dist/start.cmd 与 dist/stop.cmd（GBK + CRLF）
generate_windows_scripts() {
  info "生成启动命令 start.cmd / stop.cmd..."
  local tmp; tmp="$(mktemp -d)"

  cat > "$tmp/start.cmd" <<'CMDEOF'
@echo off
chcp 936 >nul
setlocal
rem Vocara 一体化产物启动：五个后端服务 + Nginx 前端入口
cd /d "%~dp0"
set "ROOT=%~dp0"

if not exist logs mkdir logs
if not exist records mkdir records

call :start_svc vocara-admin       8080 admin.yaml
call :start_svc vocara-agentcenter 8081 agentcenter.yaml
call :start_svc vocara-acd         8082 acd.yaml
call :start_svc vocara-callctrl    8083 callctrl.yaml
call :start_svc vocara-push        8084 push.yaml

tasklist /fi "imagename eq nginx.exe" | findstr /i /c:"nginx.exe" >nul
if errorlevel 1 (
  cd /d "%ROOT%nginx"
  start "vocara-nginx" /min nginx.exe
  cd /d "%ROOT%"
  echo [OK] Nginx 已启动
) else (
  echo [WARN] nginx.exe 已在运行，跳过
)

echo.
echo [OK] Vocara 已启动，访问地址： http://localhost
echo      停止服务： 双击 stop.cmd
exit /b 0

:start_svc
set "BIN=%~1"
set "PORT=%~2"
set "CFG=%~3"
netstat -ano | findstr ":%PORT% " | findstr LISTENING >nul
if not errorlevel 1 (
  echo [WARN] 端口 %PORT% 已被占用，跳过 %BIN%
  goto :eof
)
set "CONFIG_PATH=%ROOT%configs\%CFG%"
start "%BIN%" /min "%ROOT%bin\%BIN%.exe"
goto :eof
CMDEOF

  cat > "$tmp/stop.cmd" <<'CMDEOF'
@echo off
chcp 936 >nul
setlocal
cd /d "%~dp0"

echo 停止 Nginx...
cd /d "%~dp0nginx"
nginx.exe -s quit >nul 2>nul
cd /d "%~dp0"

echo 停止后端服务...
for %%P in (8080 8081 8082 8083 8084) do (
  for /f "tokens=5" %%a in ('netstat -ano ^| findstr ":%%P " ^| findstr LISTENING') do (
    taskkill /pid %%a /t /f >nul 2>&1 && echo 端口 %%P ^(PID %%a^) 已停止
  )
)

tasklist /fi "imagename eq nginx.exe" | findstr /i /c:"nginx.exe" >nul
if not errorlevel 1 taskkill /im nginx.exe /t /f >nul 2>&1

echo [OK] Vocara 已停止
CMDEOF

  # Windows 控制台按 GBK 解析，统一 CRLF 行尾
  iconv -f UTF-8 -t GBK "$tmp/start.cmd" | sed 's/$/\r/' > "$DIST_DIR/start.cmd"
  iconv -f UTF-8 -t GBK "$tmp/stop.cmd"  | sed 's/$/\r/' > "$DIST_DIR/stop.cmd"
  rm -rf "$tmp"
}

# generate_posix_scripts 生成 dist/start.sh 与 dist/stop.sh（管理五个后端服务）
generate_posix_scripts() {
  info "生成启动命令 start.sh / stop.sh..."

  cat > "$DIST_DIR/start.sh" <<'SHEOF'
#!/usr/bin/env bash
# Vocara 产物启动：五个后端服务（前端入口由系统 Nginx 纳入 nginx/nginx.conf）
set -u
cd "$(dirname "$0")"
mkdir -p logs records run

SERVICES=(
  "vocara-admin|admin.yaml"
  "vocara-agentcenter|agentcenter.yaml"
  "vocara-acd|acd.yaml"
  "vocara-callctrl|callctrl.yaml"
  "vocara-push|push.yaml"
)

for row in "${SERVICES[@]}"; do
  IFS='|' read -r name cfg <<< "$row"
  pidfile="run/$name.pid"
  if [ -f "$pidfile" ] && kill -0 "$(cat "$pidfile")" 2>/dev/null; then
    echo "[WARN] $name 已在运行，跳过"
    continue
  fi
  CONFIG_PATH="configs/$cfg" nohup "bin/$name" >> "logs/$name.log" 2>&1 &
  echo $! > "$pidfile"
  echo "[OK] 已启动 $name"
done
SHEOF

  cat > "$DIST_DIR/stop.sh" <<'SHEOF'
#!/usr/bin/env bash
# Vocara 产物停止：按 pidfile 终止五个后端服务
set -u
cd "$(dirname "$0")"

for pidfile in run/*.pid; do
  [ -f "$pidfile" ] || continue
  pid="$(cat "$pidfile")"
  name="$(basename "$pidfile" .pid)"
  kill "$pid" 2>/dev/null && echo "[OK] 已停止 $name"
  rm -f "$pidfile"
done
SHEOF

  chmod +x "$DIST_DIR/start.sh" "$DIST_DIR/stop.sh"
}

build_release() {
  local target="${1:-}"
  local goos="" goarch="" exe=""
  case "$target" in
    "")         goos=""; goarch="" ;;
    linux)      goos="linux"; goarch="amd64" ;;
    windows)    goos="windows"; goarch="amd64" ;;
    darwin)     goos="darwin"; goarch="arm64" ;;
    *) error "不支持的目标平台：$target（可选 linux | windows | darwin）"; exit 1 ;;
  esac
  # 目标为 Windows，或本机平台构建且当前系统为 Windows 时，二进制带 .exe 后缀
  # （go build 经 -o 指定输出名时不自动补后缀）
  if [ "$goos" = "windows" ] || { [ -z "$goos" ] && $IS_WINDOWS; }; then
    exe=".exe"
  fi

  command -v go >/dev/null 2>&1 || { error "未检测到 Go 环境"; exit 1; }

  build_web

  info "编译后端服务二进制..."
  mkdir -p "$DIST_DIR/bin" "$DIST_DIR/configs" "$DIST_DIR/web" \
           "$DIST_DIR/nginx" "$DIST_DIR/license" "$DIST_DIR/records"

  for row in "${SERVICES[@]}"; do
    IFS='|' read -r name _ entry _ _ _ <<< "$row"
    if is_prebuilt "$name"; then
      info "归集 $name（预编译二进制）..."
      local pre="$ROOT_DIR/$entry$exe"
      [ -f "$pre" ] || { error "预编译二进制缺失：$pre（请先在 core 仓库执行 make sync）"; exit 1; }
      cp "$pre" "$DIST_DIR/bin/$name$exe"
      [ -n "$exe" ] || chmod +x "$DIST_DIR/bin/$name"
    else
      info "编译 $name..."
      if [ -n "$goos" ]; then
        (cd "$ROOT_DIR" && CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
          go build -trimpath -ldflags "$LD_FLAGS" -o "$DIST_DIR/bin/$name$exe" "./$entry")
      else
        (cd "$ROOT_DIR" && CGO_ENABLED=0 \
          go build -trimpath -ldflags "$LD_FLAGS" -o "$DIST_DIR/bin/$name$exe" "./$entry")
      fi
    fi
  done

  # license-cli 是厂商内部签发工具（含签发与密钥对生成能力），不进入产物包。

  info "归集配置..."
  for row in "${SERVICES[@]}"; do
    IFS='|' read -r name _ _ src_config _ cfg_name <<< "$row"
    if [ "$name" = "vocara-callctrl" ]; then
      # 产物录音目录为产物根下的 ./records（配置模板中的绝对路径仅适用于固定部署环境）
      sed 's#^\(\s*record_dir:\s*\).*#\1./records#' "$ROOT_DIR/$src_config" \
        > "$DIST_DIR/configs/$cfg_name"
    else
      cp "$ROOT_DIR/$src_config" "$DIST_DIR/configs/$cfg_name"
    fi
  done

  info "归集前端与说明..."
  cp -r "$ROOT_DIR/business/web/admin/dist/." "$DIST_DIR/web/"
  cp "$ROOT_DIR/deploy/README.md" "$DIST_DIR/README.md"

  # License 经厂商单独交付，由接收方放入 license/ 目录，不随产物包构建。

  if [ -n "$exe" ]; then
    package_windows_runtime
  else
    package_posix_runtime
  fi

  # 产物包不得混入 Go 源码（core 资产保护）
  if find "$DIST_DIR" -name '*.go' | grep -q .; then
    error "产物目录中存在 Go 源码，中止"
    find "$DIST_DIR" -name '*.go'
    exit 1
  fi

  success "产物包构建完成：$DIST_DIR"
  echo "  版本：$VERSION ($GIT_COMMIT @ $BUILD_TIME)"
  if [ -n "$exe" ]; then
    echo "  直接运行： 双击 start.cmd（停止双击 stop.cmd），访问 http://localhost"
  else
    echo "  直接运行： ./start.sh（停止 ./stop.sh），前端入口见 nginx/nginx.conf"
  fi
  echo "  License：  vocara.lic 单独交付，放入 license/ 目录"
  echo "  说明：     见 $DIST_DIR/README.md"
}

# ============================================================
# image：Docker 镜像（Linux 构建平台）
# ============================================================
build_backend_image() {
  local svc="$1" pkg
  pkg="$(image_pkg "$svc")" || { error "未知服务：$svc"; exit 1; }

  info "构建镜像 vocara-$svc:$VERSION ..."
  cd "$ROOT_DIR"
  docker build -f deploy/docker/Dockerfile.backend \
    --build-arg SERVICE="$svc" \
    --build-arg PKG="$pkg" \
    --build-arg VERSION="$VERSION" \
    --build-arg GIT_COMMIT="$GIT_COMMIT" \
    --build-arg BUILD_TIME="$BUILD_TIME" \
    -t "$REGISTRY/vocara-$svc:$VERSION" \
    -t "$REGISTRY/vocara-$svc:latest" \
    .
  success "镜像完成：$REGISTRY/vocara-$svc:$VERSION"
}

build_web_image() {
  info "构建镜像 vocara-web:$VERSION ..."
  cd "$ROOT_DIR"
  docker build -f deploy/docker/Dockerfile.web \
    --build-arg VERSION="$VERSION" \
    -t "$REGISTRY/vocara-web:$VERSION" \
    -t "$REGISTRY/vocara-web:latest" \
    .
  success "镜像完成：$REGISTRY/vocara-web:$VERSION"
}

build_callctrl_image() {
  info "构建镜像 vocara-callctrl:$VERSION（预编译二进制）..."
  cd "$ROOT_DIR"
  docker build -f deploy/docker/Dockerfile.callctrl \
    --build-arg VERSION="$VERSION" \
    --build-arg GIT_COMMIT="$GIT_COMMIT" \
    --build-arg BUILD_TIME="$BUILD_TIME" \
    -t "$REGISTRY/vocara-callctrl:$VERSION" \
    -t "$REGISTRY/vocara-callctrl:latest" \
    .
  success "镜像完成：$REGISTRY/vocara-callctrl:$VERSION"
}

image_cmd() {
  local target="${1:-}"
  command -v docker >/dev/null 2>&1 || { error "未检测到 Docker（镜像构建在 Linux 平台执行）"; exit 1; }

  case "$target" in
    web) build_web_image ;;
    all)
      for svc in admin agentcenter acd push; do build_backend_image "$svc"; done
      build_callctrl_image
      build_web_image
      ;;
    admin|agentcenter|acd|push) build_backend_image "$target" ;;
    callctrl) build_callctrl_image ;;
    "") error "用法：./run.sh image <admin|agentcenter|acd|callctrl|push|web|all>"; exit 1 ;;
    *)  error "未知目标：$target"; exit 1 ;;
  esac
}

# ============================================================
# help
# ============================================================
show_help() {
  echo ""
  echo -e "${INFO}Vocara 开发与构建统一入口${RESET}"
  echo ""
  echo "用法： ./run.sh <命令> [参数]"
  echo ""
  echo "命令："
  echo -e "  ${OK}dev [services|web] [local|standard]${RESET} 开发模式：go run + vite（默认 local 联调配置，standard 走本机标准配置）"
  echo -e "  ${OK}build [linux|windows|darwin]${RESET}   构建产物包到 dist（默认本机平台）"
  echo -e "  ${OK}start [services|web]${RESET}           产物模式：dist 二进制 + vite dev"
  echo -e "  ${OK}stop${RESET}                           停止由本脚本启动的进程"
  echo -e "  ${OK}image <服务|web|all>${RESET}           构建 Docker 镜像（Linux 构建平台）"
  echo -e "  ${OK}help${RESET}                           显示此帮助"
  echo ""
}

# ---- 主逻辑 ----
cd "$ROOT_DIR"
CMD="${1:-dev}"
case "$CMD" in
  dev)
    ARG="${2:-all}"
    # 默认开发联调配置集 local；显式指定 standard 可切回本机标准配置
    CONFIG_PROFILE="${3:-${VOCARA_CONFIG_PROFILE:-local}}"
    case "$CONFIG_PROFILE" in
      local|standard) ;;
      *) error "未知配置集：$CONFIG_PROFILE（可选 local / standard）"; exit 1 ;;
    esac
    echo -e "${OK}=== Vocara 开发模式 ===${RESET}"
    case "$ARG" in
      all)      start_dev_services; start_web ;;
      services) start_dev_services ;;
      web)      start_web ;;
      *) error "未知参数：$ARG"; exit 1 ;;
    esac
    ;;
  build)
    echo -e "${OK}=== 构建产物包 ===${RESET}"
    build_release "${2:-}"
    ;;
  start)
    ARG="${2:-all}"
    echo -e "${OK}=== Vocara 产物模式 ===${RESET}"
    case "$ARG" in
      all)      start_release_services; start_web ;;
      services) start_release_services ;;
      web)      start_web ;;
      *) error "未知参数：$ARG"; exit 1 ;;
    esac
    ;;
  stop)  stop_all ;;
  image) image_cmd "${2:-}" ;;
  help)  show_help ;;
  *)
    error "未知命令：$CMD"
    show_help
    exit 1
    ;;
esac
