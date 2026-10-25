# vocara 呼叫中心系统 — License 授权与核心资产保护方案

## 10.1 目标与基本结论

本方案覆盖两件相互独立、需要分别建设的事情：

| 维度 | 解决的问题 |
|---|---|
| **License 授权** | 谁拥有使用权、授权给哪个客户、有效期到何时、容量多大、到期如何提示与限制 |
| **资产保护（闭源边界）** | 源代码对谁可见；核心通信资产对客户只以二进制形式交付 |

两者关系：License 是“规则”，资产保护是“围墙”。即使二进制被整体拷贝，没有匹配设备与有效期的 License，程序也无法正常工作。

基本结论：

1. 全产品作为**商业专有软件（Proprietary）**交付，以商业授权协议（EULA）约定使用权，不做公开发布式开源。
2. 资产保护分两个等级：
   - **业务层**：admin / agentcenter / acd / push，默认二进制交付；源码交付仅在商业合同（如审计、源码托管、客户二开）中单独约定。
   - **核心层**：FreeSWITCH 配置与自研模块、自研 ESL 客户端、callctrl、电话条，**任何情况下只交付二进制**，源码仅公司私有。
3. 授权形态为**纯离线 License 文件**，适配客户机房无外网的私有化部署；不提供在线激活通道。

---

## 10.2 授权模型（License 文件）

### 10.2.1 文件结构

License 文件为单个 `vocara.lic`，由“授权信封”构成，JSON 序列化、UTF-8 编码：

```json
{
  "alg": "Ed25519",
  "payload": "<base64url 编码的授权载荷>",
  "signature": "<base64url 签名>"
}
```

- `payload`：授权载荷 `LicenseClaims` 经 base64url 编码后的字符串。
- `signature`：厂商私钥对 **payload 字符串的原始字节**进行 Ed25519 签名，再 base64url 编码。
- 验签时对 `payload` 字节重新验签；通过后再 base64url 解码得到授权载荷。载荷本身不加密，字段可被读取，但任何篡改都会导致签名失效。

### 10.2.2 授权载荷字段（LicenseClaims）

| 字段 | 类型 | 必填 | 说明 |
|---|---|---|---|
| `license_id` | string | 是 | 授权唯一编号，用于续费与追溯 |
| `customer` | string | 是 | 被授权客户名称 |
| `product` | string | 是 | 产品标识，固定 `vocara` |
| `edition` | string | 是 | 版本，如 `enterprise` |
| `license_type` | string | 是 | `formal`（正式）/ `trial`（试用） |
| `issued_at` | string | 是 | 签发时间（RFC3339） |
| `not_before` | string | 是 | 生效时间（RFC3339） |
| `expire_at` | string | 否 | 到期时间（RFC3339）；永久授权省略并置 `perpetual=true` |
| `perpetual` | bool | 否 | 是否永久授权 |
| `grace_days` | int | 是 | 到期宽限天数，默认 7 |
| `fingerprint` | object | 是 | 绑定信息，见 10.5 |
| `modules` | object | 是 | 功能模块开关，见 10.2.3 |
| `limits` | object | 是 | 容量配额，见 10.2.4 |

载荷示例：

```json
{
  "license_id": "LIC-20260928-9F3A2C1B",
  "customer": "某某科技有限公司",
  "product": "vocara",
  "edition": "enterprise",
  "license_type": "formal",
  "issued_at": "2026-09-28T00:00:00Z",
  "not_before": "2026-09-28T00:00:00Z",
  "expire_at": "2027-09-28T23:59:59Z",
  "grace_days": 7,
  "fingerprint": { "mode": "host", "values": ["ab12…"] },
  "modules": { "record": true, "supervise": true, "predictive_dialer": false },
  "limits": { "agents": 100, "concurrent_calls": 50, "lines": 10, "tenants": 1 }
}
```

### 10.2.3 功能模块开关（modules）

模块开关以“缺省关闭、显式开启”为原则，未在 License 中声明的模块视为不可用：

| 模块键 | 功能 | 默认消费方 |
|---|---|---|
| `record` | 通话录音 | callctrl |
| `supervise` | 监听 / 耳语 / 强插 / 三方 | callctrl |
| `predictive_dialer` | 预测式 / 预览式外呼 | callctrl、acd |
| `report` | 统计报表导出 | admin |
| `open_api` | 对外开放 API | admin |
| `crm_form` | 动态表单 / 工单 | admin、web/form |

### 10.2.4 容量配额（limits）

| 配额键 | 含义 | 0 值语义 |
|---|---|---|
| `agents` | 可同时处于就绪态的座席数 | 0 表示不限制 |
| `concurrent_calls` | 并发通话通道数 | 0 表示不限制 |
| `lines` | 可配置的中继线路数 | 0 表示不限制 |
| `tenants` | 可开通租户数 | 0 表示不限制 |

容量统一以 0 表示“不限制”，正数表示硬上限。

---

## 10.3 密码学与签发体系

### 10.3.1 算法与密钥

- 签名算法采用 **Ed25519**：签名长度固定 64 字节，验签速度快，实现简单。
- **私钥**：厂商在离线环境生成并保管（离线机器或硬件密钥设备），不进入任何代码仓库、构建产物与客户环境。
- **公钥**：32 字节，以常量形式编译进各服务二进制与闭源 callctrl。安全性建立在“私钥不泄露”的密码学前提上，不依赖 License 校验代码本身是否可见。

### 10.3.2 签发工具 license-cli（厂商内部使用）

命令行工具位于核心私有仓库 `tools/license-cli`，仅供厂商内部签发使用：

| 子命令 | 功能 |
|---|---|
| `keygen` | 生成 Ed25519 密钥对，输出私钥文件（妥善保管）与需内置的公钥常量 |
| `fingerprint` | 在目标主机上采集并输出设备指纹（客户侧操作，亦可由独立小程序完成） |
| `sign` | 读取客户提交的指纹申请文件与授权参数，签发 `vocara.lic` |
| `inspect` | 用公钥验签并展示 License 的完整载荷，用于核对 |

### 10.3.3 申请—签发流程

```
客户环境                                   厂商（离线）
─────────────                             ─────────────
license-cli fingerprint                   license-cli sign \
  → 生成指纹申请文件 vocara.req   ──┐        -key keys/vocara.key \
                                   ├──→      -customer "某某公司" \
提交申请文件给厂商                  │        -type formal \
                                   │        -expire 2027-09-28 \
                                   │        -req vocara.req \
导入 vocara.lic，验签生效      ←───┘        -modules … -limits …
                                            → vocara.lic
```

申请文件 `vocara.req` 仅包含客户名称、采集到的指纹与采集时间，不含任何敏感凭据。

---

## 10.4 校验架构

### 10.4.1 原则：本地验签，无中心单点

每个服务在本进程内独立完成 License 校验，不依赖任何其他在线服务，避免“授权组件故障导致全业务停摆”：

```
   admin（授权管理面：install_id 生成 / License 导入 / 状态展示）
                    │
                    │ License 与 install_id 写入 Redis 下发
                    ▼
   每个服务内置 pkg/license：读 Redis → 内置公钥验签 → 指纹/到期/容量判定 → 内存缓存状态
```

- 各服务（admin / agentcenter / acd / push + 闭源 callctrl）启动时读取 Redis 下发的 License 并验签，运行期按固定间隔重读复校；admin 经管理接口导入时即时加载并写入 Redis。
- admin 提供授权管理面（install_id 展示、License 导入、复校重载，端点 `/api/v1/admin/license/info`、`/import`、`/reload`），不设跨节点状态上报。Redis 不可达时各节点依据本地快照/文件兜底独立运行。

### 10.4.2 License 分发

| 部署形态 | 分发方式 | 更新生效 |
|---|---|---|
| Kubernetes / 集群 | admin 页面上传后写入 Redis 并广播变更通知，各服务订阅后即时复校 | 上传即广播，其余服务即时生效；断线/冷启动由复校周期兜底 |
| 单机私有化 | 固定路径文件 | 替换文件后下个复校周期生效；admin 经导入接口即时加载 |

### 10.4.3 服务配置节

各服务 `configs/config.yaml` 增加 `license` 节（snake_case，统一走 `pkg/config.Load`）：

```yaml
license:
  file: ./license/vocara.lic   # License 文件路径（单机兜底；Kubernetes 下由 Redis 提供）
  check_interval: 3600         # 运行期复校间隔（秒）
  fail_open: false             # 指纹/签名异常时是否放行（仅排障临时使用，正式环境保持 false）
```

验签公钥内置在二进制中，不通过配置提供。

### 10.4.4 License 管理器（pkg/license.Manager）

License 管理器作为各服务的统一接入组件：

- 启动与复校：优先读运行期来源（Redis 下发的 `vocara:license:current`），来源不可用时回退本地文件与快照；验签 → 解码载荷 → 指纹比对 → 得到当前授权状态，缓存在内存。
- 运行期：按 `check_interval` 周期从来源重读复校；admin 经管理接口导入/重载时由 `ReloadFromBytes` 即时加载本节点 License、写入 Redis 并广播 `vocara:license:changed`，其余服务订阅后立刻复校，无需等待下个周期。
- 本地快照：每次成功校验后，将 License 加密写入本地快照。Redis 与文件均不可读时，以最近一次有效快照继续运行；密钥由设备特征派生。
- 对外能力：查询当前状态、到期剩余天数、模块是否可用、容量是否超限、提供拦截判定方法。

---

## 10.5 设备与集群绑定

### 10.5.1 指纹模式

| 模式 | 绑定对象 | 适用场景 |
|---|---|---|
| `host` | 媒体服务器（callctrl / FreeSWITCH 所在主机）硬件指纹 | 1~2 台主机的典型私有化 |
| `install` | admin 首次部署生成的 `install_id`（主存系统配置表，经 Redis 下发，页面展示） | 集群 / Kubernetes，节点可变 |
| `none` | 不绑定设备 | 仅短期试用授权 |

私有化交付默认绑定媒体服务器——核心通信资产与并发通道都在该主机，绑定此处最有意义。集群环境绑定 `install_id`，避免节点扩缩或换机触发售后。

### 10.5.2 主机指纹采集

采集以下信息的组合，做 SHA-256 后取十六进制作为指纹值：

- 主机名
- CPU 型号与核数
- 主板序列号与 product_uuid
- 首个在用物理网卡 MAC 地址

采集按 Linux / Windows 分别实现，虚拟环境下取不到的字段自动忽略。指纹申请文件中同时上送各原始项，便于签发侧核对。

### 10.5.3 防时间回拨

每次成功校验时，将“最后见到的时间”加密写入本地快照。运行时若发现当前系统时间明显早于记录值，判定为时钟回拨，授权状态置为异常（`invalid`），防止通过回拨系统时钟延长使用期。恢复正确时钟后，服务于下个复校周期恢复正常。

---

## 10.6 到期状态机与提示

### 10.6.1 授权状态

呼叫中心是生产系统，到期采用**分级施压**而非直接硬停，既推动续费，又避免客户业务瞬间中断：

| 状态 | 触发条件 | 系统行为 |
|---|---|---|
| `valid` | 距到期 > 30 天，或永久授权 | 全功能 |
| `expiring` | 到期前 30 天内 | 全功能，黄色横幅提示剩余天数 |
| `grace` | 到期后 `grace_days` 天内 | 全功能，红色横幅 + 登录弹窗强提示 |
| `expired` | 超过宽限期 | 禁止新建外呼与写操作；**保留来话接听与历史查询**；前端显示到期遮罩 |
| `invalid` | 文件缺失、签名错误、指纹不符、时钟回拨 | 同 `expired`，仅保留登录、授权管理与健康检查能力 |

### 10.6.2 提示落点

- **前端横幅**：布局层全局常驻横幅（与电话条同层），`expiring` 为黄色、`grace` 为红色；进入 `expired` 后展示授权遮罩。
- **登录与定时弹窗**：`grace` 期登录后弹窗提示；`expiring` 期每次进入首页轻提示。
- **后端接口**：被限制的操作返回授权错误码（见 10.6.3），前端请求拦截器统一识别并转为提示。
- **状态接口**：admin 授权管理页与前端启动时查询 `/license/info` 获取状态。

### 10.6.3 授权错误码（700xx 段）

| 错误码 | 常量 | 含义 | HTTP 状态 |
|---|---|---|---|
| 70001 | `CodeLicenseMissing` | License 文件缺失 | 402 |
| 70002 | `CodeLicenseInvalid` | 签名错误 / 载荷非法 | 402 |
| 70003 | `CodeLicenseExpired` | 授权已过有效期 | 402 |
| 70004 | `CodeLicenseFingerprintMismatch` | 设备指纹不符 | 403 |
| 70005 | `CodeLicenseLimitExceeded` | 超出容量配额 | 403 |
| 70006 | `CodeLicenseModuleDisabled` | 功能模块未授权 | 403 |
| 70007 | `CodeLicenseClockRollback` | 检测到时钟回拨 | 403 |

错误码在 `pkg/errors/code.go` 中定义；前端拦截器对 700xx 统一处理。表中 HTTP 状态为授权接口的状态契约。

---

## 10.7 授权粒度与容量拦截点

各服务读取同一份 License，在自己负责的操作入口独立判定与拦截：

| 控制点 | 所在服务 | 判定时机 | 超限 / 未授权行为 |
|---|---|---|---|
| 就绪座席数 | agentcenter | 座席签到 / 切换就绪 | 拒绝新增就绪，在座席保持工作 |
| 并发通话数 | callctrl | 外呼发起 / 来话接入 | 拒绝新建通话 |
| 中继线路数 | admin | 保存线路配置 | 拒绝新增 |
| 租户数 | admin | 新建租户 | 拒绝新增 |
| 录音 | callctrl | 启动录音 | 返回模块未授权 |
| 监听 / 耳语 | callctrl | 发起监听类操作 | 返回模块未授权 |
| 预测式外呼 | callctrl / acd | 启动预测任务 | 返回模块未授权 |
| 报表导出 | admin | 导出请求 | 返回模块未授权 |

容量计数以运行态实际数量为准；被拦截操作返回 `CodeLicenseLimitExceeded` 或 `CodeLicenseModuleDisabled`。

---

## 10.8 核心资产保护与仓库边界

### 10.8.1 双仓库边界与独立构建

系统由两个独立 git 仓库构成：公开仓库 `vocara` 与私有仓库 `core`。`vocara` 承载业务区
`business/`、契约 `contracts/` 与公共库 `pkg/`，并存放核心发布的预编译产物；`core` 承载核心区源码。

| 仓库 | 内容 | 独立构建 |
|---|---|---|
| **vocara/**（公开） | admin / agentcenter / acd / push；web/admin、web/form；contracts、pkg；核心预编译产物（bin/、configs/） | 根 `go.work`（业务模块 + pkg + contracts） |
| **core/**（私有） | 自研 ESL（`esl`）、callctrl、电话条、FreeSWITCH 配置、license-cli | `core/go.work`，经 `../vocara/pkg`、`../vocara/contracts` 引用共享模块 |

`contracts/`（module `vocara/contracts`）零第三方依赖，承载跨边界事件与 HTTP DTO，是双仓库共享、编译期稳定的接口面：`vocara` 是 `pkg` 与 `contracts` 的单一事实来源，`core` 只读引用（go.work → `../vocara`），不复制代码。依赖方向严格为私有 → 公开：`core` 引用 `vocara` 的 pkg/contracts，而 `core` 进入 `vocara` 的唯一方式是通过 `make sync` 提交编译产物（`bin/vocara-callctrl(.exe)`、混淆电话条、`configs/callctrl.yaml`），不存在公开 → 私有的源码依赖。

边界事实：自研 ESL 仅被 callctrl 引用，业务服务不直接依赖；业务侧不通过 Go HTTP 调用 callctrl，acd 与 callctrl 之间以 `queue.dequeue` 事件协作，callctrl 反向以 HTTP（内部令牌）调用业务服务。业务区可独立编译。

### 10.8.2 FreeSWITCH

- FreeSWITCH 主程序为开源软件，随产品构建其官方版本；私有化资产是**对接逻辑、配置（sip profile / directory）与拨号规则**。
- **拨号规则不落媒体节点磁盘**：dialplan 编译进管理后台，经 mod_xml_curl 在每通呼叫实时下发并与 License 联动——超宽限 / 失效时外呼出局规则替换为挂断阻断段，来话接听、分机互拨与座席振铃规则始终下发。
- 交付构成为“开源 FreeSWITCH 主程序 + 闭源配置包（fs-sbc / fs-agent 的 conf，含 sip profile / directory）+ 闭源管理后台（承载拨号规则）+ 闭源 callctrl（含自研 Go ESL）+ 混淆电话条”。
- 自研 FreeSWITCH 动态加载模块（`.so`）按具体媒体能力需求单独立项；当前交付不含 `.so`，不预置占位模块。

### 10.8.3 callctrl 二进制化

- Go 静态编译为单个可执行文件，仅交付二进制与配置；构建以 `-trimpath -ldflags "-s -w"` 产出去除本机路径与符号表的成品，版本信息注入 `vocara/pkg/version`。
- 二进制内置 License 校验与媒体主机指纹采集。
- **接口契约固化**：`contracts/` 模块定义跨边界 DTO——事件 `AllocatedCall`（queue.dequeue）、`CDRData`（cdr.created），以及 callctrl HTTP DTO（`contracts/call/v1`，inbound 回调已接线）；契约随模块版本化。业务侧据此开发，编译期不依赖核心源码。
- **三类路由鉴权**：健康检查公开；业务 / 查询路由经 JWT；FreeSWITCH inbound 回调（`/api/v1/call/inbound/*`）经内部令牌（X-Internal-Token），不继承 JWT。
- **CDR 统一口径**：话单经事件总线以统一信封发布 `cdr.created`（主题 `vocara.event.cdr.created`，NATS 与 Redis 双通道）；持久化模型经 `ToEvent()` 转为契约 DTO，不存在无信封裸主题路径。

### 10.8.4 电话条

前端 JavaScript 天然以明文下发浏览器，无法做到与服务端同等的闭源强度，工程上采用三层保护：

1. **核心逻辑下沉**：呼叫控制、ESL、FreeSWITCH 交互全部位于闭源 callctrl，前端不包含核心通信资产。
2. **代码混淆**：电话条独立构建，自有 SDK 经 javascript-obfuscator 做标识符混淆、控制流扁平化与字符串加密（jssip 依赖段仅压缩）。
3. **分机凭据动态化**：分机密码由后端动态下发并可轮换，单个分机凭据泄露的影响被限制在该分机。

电话条保持 **WebRTC 浏览器分机**模式（浏览器即分机），媒体在浏览器与 FreeSWITCH 之间建立、不占用服务器媒体资源；核心通信控制与 ESL 已全部闭源于服务端，满足资产保护目标。

---

## 10.9 交付形态与构建流水线

标准交付物：

1. 业务层服务二进制（admin / agentcenter / acd / push）
2. 核心层二进制（callctrl）+ FreeSWITCH 配置（fs-sbc / fs-agent conf，由运维直接部署）
3. 前端产物（web/admin、电话条、动态表单 SDK），电话条经混淆保护自有代码
4. 与客户、设备绑定的 `vocara.lic`
5. Kubernetes 部署清单与配置文件

构建流水线（vocara 顶层 Makefile + 核心仓库 Makefile）职责：

- `build-business`（vocara）与 `build` / `build-callctrl`（核心）分别构建；核心层以 `-trimpath -s -w` 产出去除路径与符号的成品。
- 核心侧 `build-phonebar` 生成与客户无关的混淆电话条通用产物。
- License 由签发侧单独生成，不进入构建产物本身（出厂可预置试用 License）。
- 核心侧 `make sync` 将 callctrl 二进制（linux + windows）、混淆电话条、callctrl 配置模板发布到 vocara；FreeSWITCH 配置（fs-sbc / fs-agent conf）由运维从核心仓库 `freeswitch/` 目录直接部署；vocara 侧 `make package` 打业务交付包。
- 不提供 docker-compose 部署；FreeSWITCH 镜像构建文件位于核心仓库 `freeswitch/Dockerfile`，业务微服务由 `deploy/docker/Dockerfile.backend`、callctrl 由 `deploy/docker/Dockerfile.callctrl` 构建，编排以 Kubernetes 为准。

---

## 10.10 定稿决策

| 决策点 | 定稿 |
|---|---|
| 授权模式 | 纯离线 License 文件，不提供在线激活 |
| 签名算法 | Ed25519，公钥内置、私钥厂商离线保管 |
| 设备绑定 | 私有化绑媒体主机指纹（host），集群绑 install_id（install） |
| 到期策略 | 到期前 30 天提示 + 7 天宽限；过期后限外呼与写操作、保留接听与查询 |
| 授权粒度 | 整体到期 + 功能模块开关 + 座席 / 并发 / 线路 / 租户容量 |
| 电话条 | 保持 WebRTC 浏览器分机；逻辑下沉、混淆构建、凭据动态化 |
| 软件形态 | 全产品商业专有；业务层默认二进制（源码按合同），核心层仅二进制 |
| 授权对象 | 以单实例授权单个客户为主；多租户通过 `limits.tenants` 容量预留 |
| 仓库边界 | 公开仓库 vocara（business + contracts + pkg + 核心预编译产物）与私有仓库 core（核心源码），各自独立构建，共享 vocara/contracts |
| 电话条混淆 | 自有 SDK 强混淆（标识符 + 控制流扁平化 + 字符串加密），jssip 仅压缩；产物与客户无关，授权由 License 统一控制 |
| 自研 FreeSWITCH 模块 | 当前交付不含 .so，按具体媒体能力需求单独立项 |
| 部署编排 | Kubernetes + 二进制流水线；不提供 docker-compose |
