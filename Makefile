# Vocara 构建与交付流水线（Git Bash / Linux / CI）
#
# 业务区四个服务由本仓库源码构建；呼叫控制服务以预编译二进制交付
# （bin/vocara-callctrl 与 bin/vocara-callctrl.exe 随仓库提供，不随本仓库编译）。
# 版本信息经 -ldflags 注入 vocara/pkg/version。

# ---- 变量 ----
GOOS        ?= linux
GOARCH      ?= amd64
GOTOOLCHAIN ?= go1.22.12
VERSION     ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
GIT_COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME  ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

PKG       := vocara/pkg/version
LD_FLAGS  := -s -w \
	-X $(PKG).Version=$(VERSION) \
	-X $(PKG).BuildTime=$(BUILD_TIME) \
	-X $(PKG).GitCommit=$(GIT_COMMIT)

DIST      := dist
BIZ_OUT   := $(DIST)/business
STAGE     := $(DIST)/stage
PACKAGES  := $(DIST)/packages

# 业务服务 → 主包路径
src_admin       := business/services/admin/cmd
src_agentcenter := business/services/agentcenter/cmd
src_acd         := business/services/acd/cmd
src_push        := business/services/push/cmd
# 预编译交付的呼叫控制二进制（随仓库提供）
pre_callctrl    := bin/vocara-callctrl

GO_BUILD := GOTOOLCHAIN=$(GOTOOLCHAIN) CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -trimpath -ldflags "$(LD_FLAGS)"

.PHONY: all build-business build-release package package-business vet test tidy clean help

all: package ## 全量构建并打包

build-business: ## 编译业务区四个服务
	@mkdir -p $(BIZ_OUT)
	$(GO_BUILD) -o $(BIZ_OUT)/vocara-admin       ./$(src_admin)
	$(GO_BUILD) -o $(BIZ_OUT)/vocara-agentcenter ./$(src_agentcenter)
	$(GO_BUILD) -o $(BIZ_OUT)/vocara-acd         ./$(src_acd)
	$(GO_BUILD) -o $(BIZ_OUT)/vocara-push        ./$(src_push)

build-release: build-business ## 归集呼叫控制预编译二进制
	@mkdir -p $(BIZ_OUT)
	cp $(pre_callctrl) $(pre_callctrl).exe $(BIZ_OUT)/
	chmod +x $(BIZ_OUT)/vocara-callctrl

package: package-business ## 打业务交付包

package-business: build-release
	@rm -rf $(STAGE)/business && mkdir -p $(STAGE)/business/bin $(STAGE)/business/configs
	cp $(BIZ_OUT)/vocara-admin $(BIZ_OUT)/vocara-agentcenter $(BIZ_OUT)/vocara-acd $(BIZ_OUT)/vocara-push $(STAGE)/business/bin/
	cp $(BIZ_OUT)/vocara-callctrl $(BIZ_OUT)/vocara-callctrl.exe $(STAGE)/business/bin/
	cp business/services/admin/configs/config.yaml       $(STAGE)/business/configs/admin.yaml
	cp business/services/agentcenter/configs/config.yaml $(STAGE)/business/configs/agentcenter.yaml
	cp business/services/acd/configs/config.yaml         $(STAGE)/business/configs/acd.yaml
	cp business/services/push/configs/config.yaml        $(STAGE)/business/configs/push.yaml
	@sed 's#^\(\s*record_dir:\s*\).*#\1./records#' configs/callctrl.yaml > $(STAGE)/business/configs/callctrl.yaml
	@mkdir -p $(PACKAGES)
	tar -czf $(PACKAGES)/vocara-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz -C $(STAGE) business

# ---- 校验 ----
vet: ## go vet 全量模块
	go vet ./pkg/... ./contracts/... \
	  ./business/services/admin/... ./business/services/agentcenter/... \
	  ./business/services/acd/... ./business/services/push/...

test: ## go test 公共库与现有用例
	go test ./pkg/... ./business/services/acd/... ./business/services/agentcenter/...

tidy: ## 各模块 go mod tidy
	@for d in pkg contracts \
	  business/services/admin business/services/agentcenter business/services/acd business/services/push; do \
	    echo "tidy $$d"; (cd $$d && go mod tidy); \
	done

clean: ## 清理构建产物
	rm -rf $(DIST)

help: ## 显示目标列表
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-16s %s\n", $$1, $$2}'