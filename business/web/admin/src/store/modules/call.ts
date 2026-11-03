// 通话全局状态 Store（基于 VocaraPhoneBar 原生 JS 电话条 SDK）
// SDK 通过 index.html 的 <script src="/phonebar/vocara-phonebar.js"> 加载，
// 暴露 window.VocaraPhoneBar；本 store 持有其实例（模块级单例），
// 任何组件通过 useCallStore 访问通话状态和发起操作。

import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { notification } from '@/utils/naive'
import { useUserStore } from '@/store/modules/user'
import { getMyDnApi, type MyDnInfo } from '@/api/auth'

// ===== 通话类型 =====

/** 通话状态枚举 */
export type CallStatus =
  | 'idle'       // 空闲
  | 'dialing'    // 呼出中（发起呼叫，未振铃）
  | 'ringing'    // 振铃中（呼入来电振铃 / 呼出对方振铃）
  | 'talking'    // 通话中
  | 'holding'    // 保持中
  | 'transferring' // 转接中
  | 'hangup'     // 已挂断（过渡状态，即将回到 idle）

export type CallDirection = 'inbound' | 'outbound' | 'internal'

/** 通话信息（字段命名与后端 CallEventData 对齐） */
export interface CallInfo {
  callUuid: string         // 通话唯一标识
  agentId: number          // 座席ID
  tenantId: number         // 租户ID
  queueId: number          // 队列ID（0 表示非队列来电）
  direction: CallDirection // 通话方向
  callerNum: string        // 主叫号码
  calledNum: string        // 被叫号码
  ani: string              // 主叫号码显示（可能带姓名）
  dnis: string             // 被叫号码显示
  startTime: number        // 通话开始时间戳（毫秒）
  answerTime?: number      // 接听时间戳（毫秒）
  endTime?: number         // 结束时间戳（毫秒）
  hangupCause?: string     // 挂机原因
  duration: number         // 通话时长（秒）
  holdCount: number        // 保持次数
  status: CallStatus       // 当前通话状态
  recording: boolean       // 是否录音中
  muted: boolean           // 是否静音
  held: boolean            // 是否保持
}

// ===== VocaraPhoneBar SDK 类型声明 =====

/** SDK 实例方法（与 web/phonebar README 实例方法表一致） */
interface VocaraPhoneBarInstance {
  connect(): void
  disconnect(): void
  ready(): void
  busy(reason?: string): void
  answer(): void
  hangup(): void
  dial(number: string): void
  mute(muted: boolean): void
  destroy(): void
}

/** SDK init 参数 */
interface VocaraPhoneBarOptions {
  container?: Element | null
  wsUrl: string
  token: string
  extension: string
  sip?: { wsUri: string; domain: string; password: string; extension?: string }
  onStatusChange?: (status: string) => void
  onSignin?: () => void
  onSipRegistered?: () => void
  onSipUnregistered?: () => void
  onSipRegistrationFailed?: (cause: string) => void
  onIncoming?: (call: { id: string; caller: string; callee: string }) => void
  onAccepted?: () => void
  onEnded?: () => void
  onKick?: (reason: string) => void
  onError?: (err: Error) => void
  onEvent?: (msg: any) => void
}

declare global {
  interface Window {
    VocaraPhoneBar?: { init(opts: VocaraPhoneBarOptions): VocaraPhoneBarInstance }
  }
}

// 座席话务工作状态
export type AgentWorkStatus = 'offline' | 'checked' | 'ready' | 'busy' | 'wrapup'

// SDK 实例（模块级单例，再次 init 会销毁既有实例）
let bar: VocaraPhoneBarInstance | null = null
// PhoneBar 组件挂载时注册的容器元素
let barContainer: Element | null = null

// 铃声音频单例
let ringAudio: HTMLAudioElement | null = null

function getRingAudio(): HTMLAudioElement {
  if (!ringAudio) {
    ringAudio = new Audio()
    ringAudio.loop = true
  }
  return ringAudio
}

function playRing() {
  const audio = getRingAudio()
  audio.src = '/sounds/ring1.mp3'
  audio.play().catch(() => {
    // 浏览器自动播放策略可能阻止，忽略
  })
}

function stopRing() {
  const audio = getRingAudio()
  audio.pause()
  audio.currentTime = 0
}

export const useCallStore = defineStore('call', () => {
  // ===== 状态 =====
  const connected = ref(false)
  const currentCall = ref<CallInfo | null>(null)
  const callHistory = ref<CallInfo[]>([])
  const maxHistory = 20

  // 座席话务状态（由 SDK 的 status_change 权威事件驱动）
  const workStatus = ref<AgentWorkStatus>('offline')
  // 座席登录信息
  const agentAuth = ref<{ agentId: string; tenantId: string; extension: string } | null>(null)

  // 当前登录账号绑定的分机；未绑定分机时为 null，电话条据此决定是否显示
  const myDn = ref<MyDnInfo | null>(null)

  // 退出登录（token 清空）时销毁电话条并复位会话级状态，避免跨账号状态残留
  watch(
    () => useUserStore().token,
    (t) => {
      if (!t) destroyAdapter()
    },
  )

  // 本地静音标记（SDK 为 set 语义，toggle 由本 store 维护）
  const mutedFlag = ref(false)

  // ===== 计算属性 =====

  const callStatus = computed(() => currentCall.value?.status || 'idle')
  const isIdle = computed(() => !currentCall.value || currentCall.value.status === 'idle' || currentCall.value.status === 'hangup')
  const isRinging = computed(() => currentCall.value?.status === 'ringing')
  const isTalking = computed(() => currentCall.value?.status === 'talking')
  const isHolding = computed(() => currentCall.value?.status === 'holding')
  const isDialing = computed(() => currentCall.value?.status === 'dialing')
  const isInbound = computed(() => currentCall.value?.direction === 'inbound')
  const isOutbound = computed(() => currentCall.value?.direction === 'outbound')

  /** 显示用的主叫号码（呼入时是对方号码，呼出时是拨打的号码） */
  const displayNumber = computed(() => {
    if (!currentCall.value) return ''
    if (currentCall.value.direction === 'inbound') {
      return currentCall.value.callerNum
    }
    return currentCall.value.calledNum
  })

  /** 显示用的主叫名称 */
  const displayName = computed(() => {
    if (!currentCall.value) return ''
    if (currentCall.value.direction === 'inbound') {
      return currentCall.value.ani
    }
    return currentCall.value.dnis
  })

  /** 格式化通话时长 mm:ss */
  const formattedDuration = computed(() => {
    const sec = currentCall.value?.duration || 0
    const m = Math.floor(sec / 60).toString().padStart(2, '0')
    const s = (sec % 60).toString().padStart(2, '0')
    return `${m}:${s}`
  })

  // 通话中每秒计时（由状态变化自动启停）
  let durationTimer: number | null = null
  watch(isTalking, (talking) => {
    if (talking) {
      if (durationTimer === null) {
        durationTimer = window.setInterval(() => {
          if (currentCall.value && currentCall.value.status === 'talking') {
            currentCall.value = { ...currentCall.value, duration: currentCall.value.duration + 1 }
          }
        }, 1000)
      }
    } else if (durationTimer !== null) {
      clearInterval(durationTimer)
      durationTimer = null
    }
  })

  // ===== SDK 挂载 =====

  /**
   * 注册电话条容器（由 PhoneBar 组件 onMounted 调用）。
   * 容器就绪后若尚未签入，先创建离线态实例渲染"签入"按钮，
   * 签入/签出统一在电话条上操作。
   */
  function setPhoneBarContainer(el: Element | null) {
    barContainer = el
    if (el) ensureOfflineBar()
  }

  // 签入流程标志：会话建立（CHECKED）后自动示闲，避免座席停在 CHECKED 而不进入 ACD 空闲池
  let autoReadyAfterSignIn = false

  /** SDK 状态回调（离线实例与签入实例共用） */
  function handleSdkStatus(s: string) {
    const key = (s || '').toUpperCase()
    const mapped = mapSdkStatus(s)
    if (mapped) workStatus.value = mapped
    connected.value = key !== 'OFFLINE'
    if (autoReadyAfterSignIn && key === 'CHECKED') {
      autoReadyAfterSignIn = false
      agentSetReady().catch(() => { /* ack 乐观更新，失败由后续 status_change 纠正 */ })
    }
    if (key === 'OFFLINE') autoReadyAfterSignIn = false
  }

  /** SDK 错误回调（离线实例与签入实例共用） */
  function handleSdkError(err: Error) {
    console.error('[callStore] 电话条错误:', err)
    notification.error({ title: '通话异常', content: err.message || '未知错误' })
  }

  /**
   * 处理 push 透传的业务事件（SDK onEvent）。
   * SIP INVITE 是振铃主通道；WS 的 call.ringing 用于补充通话元数据，
   * 并在 SIP 未注册（收不到 INVITE）时兜底展示来电，避免"有呼叫无感知"。
   */
  function handleBusinessEvent(msg: any) {
    const evt = msg?.event
    const data = msg?.data
    if (!evt || !data) return

    if (evt === 'call.ringing') {
      // 已有通话卡（SIP onIncoming 已创建）时仅忽略，SIP 卡为准
      if (currentCall.value) return
      currentCall.value = {
        callUuid: String(data.call_uuid || ''),
        agentId: Number(data.agent_id) || 0,
        tenantId: Number(data.tenant_id) || 0,
        queueId: Number(data.queue_id) || 0,
        direction: data.direction === 'outbound' ? 'outbound' : 'inbound',
        callerNum: String(data.caller_num || ''),
        calledNum: String(data.called_num || ''),
        ani: String(data.ani || data.caller_num || ''),
        dnis: String(data.dnis || ''),
        startTime: Number(data.start_time) || Date.now(),
        duration: 0,
        holdCount: 0,
        status: 'ringing',
        recording: false,
        muted: false,
        held: false,
      }
      notification.info({
        title: '来电振铃',
        content: `主叫 ${data.caller_num || '未知'}（WS 事件兜底，SIP 通道未振铃请检查软电话注册）`,
        duration: 4000,
      })
    }
    if (evt === 'call.hangup' || evt === 'call.abandoned') {
      // 仅清理 WS 兜底创建的卡；SIP 卡（callUuid=sip-*）由 SIP 路径自行清理
      if (
        currentCall.value &&
        !currentCall.value.callUuid.startsWith('sip-') &&
        String(data.call_uuid || '') === currentCall.value.callUuid
      ) {
        callHistory.value.unshift({ ...currentCall.value, status: 'hangup', endTime: Date.now() })
        currentCall.value = null
      }
    }
  }

  // SIP 注册状态（签入/示闲只依赖业务 WS，SIP 独立；false 时来电无法振铃）
  const sipRegistered = ref(false)
  // 上次 SIP 告警时间：jssip 重试期间会反复触发失败回调，30 秒内只提示一次
  let lastSipWarnAt = 0
  function warnSipOnce(title: string, content: string) {
    const now = Date.now()
    if (now - lastSipWarnAt < 30000) return
    lastSipWarnAt = now
    notification.warning({ title, content, duration: 6000 })
  }

  /**
   * 统一构造 phonebar 全部回调。
   * 关键：离线 bar（ensureOfflineBar）与在线 init（agentSignIn）必须用同一套回调，
   * 否则离线实例 connect 复用后缺少 onIncoming/onAccepted，通话中 currentCall 永远为 null。
   */
  function buildPhoneBarHandlers(extension: string) {
    return {
      onStatusChange: handleSdkStatus,
      onError: handleSdkError,
      // 外呼拨号：立即建立 currentCall（dialing 态），与 inbound 一致，拨号即显示表单
      onDial: (num: string) => {
        const info = useUserStore().userInfo
        currentCall.value = {
          callUuid: `sip-out-${Date.now()}`,
          agentId: Number(info?.user_id) || 0,
          tenantId: Number(info?.tenant_id) || 0,
          queueId: 0,
          direction: 'outbound',
          callerNum: info?.username || '',
          calledNum: num,
          ani: info?.username || '',
          dnis: num,
          startTime: Date.now(),
          duration: 0,
          holdCount: 0,
          status: 'dialing',
          recording: false,
          muted: false,
          held: false,
        }
      },
      onIncoming: (call: any) => {
        // 浏览器直接收到 FS INVITE：以 SIP 会话为准振铃；用户信息就地获取
        const info = useUserStore().userInfo
        currentCall.value = {
          callUuid: `sip-${call.id}`,
          agentId: Number(info?.user_id) || 0,
          tenantId: Number(info?.tenant_id) || 0,
          queueId: 0,
          direction: 'inbound',
          callerNum: call.caller,
          ani: call.caller,
          dnis: call.callee,
          calledNum: call.callee,
          startTime: Date.now(),
          duration: 0,
          holdCount: 0,
          status: 'ringing',
          recording: false,
          muted: false,
          held: false,
        }
        playRing()
        requestNotificationPermission()
      },
      onAccepted: () => {
        stopRing()
        if (currentCall.value) {
          currentCall.value = { ...currentCall.value, status: 'talking', answerTime: Date.now() }
        }
        notification.success({
          title: '通话已接通',
          content: `与 ${displayNumber.value} 通话中`,
          duration: 2000,
        })
      },
      onEnded: () => {
        stopRing()
        if (currentCall.value) {
          callHistory.value.unshift({ ...currentCall.value, status: 'hangup', endTime: Date.now() })
          if (callHistory.value.length > maxHistory) {
            callHistory.value.pop()
          }
          currentCall.value = { ...currentCall.value, status: 'hangup' }
          setTimeout(() => {
            if (currentCall.value?.status === 'hangup') currentCall.value = null
          }, 800)
        }
        mutedFlag.value = false
        notification.info({ title: '通话已结束', content: '正常挂断', duration: 2000 })
      },
      onKick: (reason: string) => {
        notification.warning({
          title: '座席已被踢下线',
          content: `原因：${reason || 'duplicate_login'}，请重新签入`,
          duration: 5000,
        })
        agentAuth.value = null
        workStatus.value = 'offline'
        connected.value = false
        bar = null
      },
      onSignin: () => {
        agentSignIn(extension).catch((e: any) => {
          notification.error({ title: '签入失败', content: e?.message || '未知错误' })
        })
      },
      onEvent: (msg: any) => {
        handleBusinessEvent(msg)
      },
      onSipRegistered: () => {
        sipRegistered.value = true
      },
      onSipUnregistered: () => {
        const was = sipRegistered.value
        sipRegistered.value = false
        if (was && connected.value) {
          warnSipOnce('SIP 已掉线', '软电话注册已失效，可能无法接听来电，请检查网络后重新签入')
        }
      },
      onSipRegistrationFailed: (cause: string) => {
        sipRegistered.value = false
        warnSipOnce(
          'SIP 注册失败',
          `软电话无法注册到语音服务器（${cause || 'unknown'}），来电将不能振铃；请检查 /sip-ws 代理与 FreeSWITCH WebSocket 端口`,
        )
      },
    }
  }

  /** 创建离线态电话条：不连接 WS/SIP，仅渲染"签入"入口（SIP 配置随实例就绪，签入时同实例 connect 无闪烁） */
  function ensureOfflineBar() {
    if (!barContainer) return
    // SDK 单例的 UI DOM 可能随容器卸载而脱离文档（组件重建/HMR）：
    // 此时销毁实例重建，否则电话条不可见
    if (bar) {
      const uiEl = (bar as any).ui?.el as HTMLElement | undefined
      if (uiEl && uiEl.isConnected) return
      destroyBar()
    }
    const userStore = useUserStore()
    const extension = userStore.userInfo?.username || ''
    const sdk = window.VocaraPhoneBar
    if (!userStore.token || !extension || !sdk) return

    const env = (import.meta as any).env
    const wsBase = String(env.VITE_PUSH_WS_URL || '/ws').replace(/\/$/, '')
    buildSipConfig(extension).then((sip) => {
      // 使用统一回调工厂：保证离线实例 connect 复用后也有完整通话回调
      bar = sdk.init({
        container: barContainer,
        wsUrl: `${wsBase}/agent`,
        token: userStore.token,
        extension,
        sip,
        ...buildPhoneBarHandlers(extension),
      })
    })
  }

  /** SDK 状态（大写，如 OFFLINE/CHECKED/READY/BUSY/WRAPUP）映射为本地工作状态 */
  function mapSdkStatus(s: string): AgentWorkStatus | null {
    switch ((s || '').toUpperCase()) {
      case 'READY':
        return 'ready'
      case 'BUSY':
        return 'busy'
      case 'WRAPUP':
        return 'wrapup'
      case 'OFFLINE':
        return 'offline'
      case 'CHECKED':
      case 'CHECKIN':
      case 'LOGIN':
        return 'checked'
      default:
        // RINGING 等瞬时不改变话务状态
        return null
    }
  }

  /** 销毁 SDK 实例并复位本地状态 */
  function destroyBar() {
    stopRing()
    if (bar) {
      try {
        bar.destroy()
      } catch (e) {
        // 忽略销毁异常，本地仍清理
      }
      bar = null
    }
    connected.value = false
    currentCall.value = null
    mutedFlag.value = false
    sipRegistered.value = false
  }

  // ===== 座席话务状态（SDK 连接即签入）=====

  /** 拉取当前账号绑定分机并缓存；未绑定或接口失败时返回 null */
  async function loadMyDn(): Promise<MyDnInfo | null> {
    try {
      const dn = await getMyDnApi()
      myDn.value = dn || null
    } catch {
      myDn.value = null
    }
    return myDn.value
  }

  /** 构建 SIP 软电话配置（WS 地址 + 注册凭据）；无 SIP 地址配置时返回 undefined */
  async function buildSipConfig(extension: string) {
    const env = (import.meta as any).env
    const sipRaw = String(env.VITE_SIP_WS_URL || '')
    if (!sipRaw) return undefined
    // jssip 要求绝对 ws(s):// URL：相对路径按当前页面协议补全为 wss/ws 同源地址
    const wsUri = sipRaw.startsWith('/')
      ? `${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}${sipRaw}`
      : sipRaw

    // 注册凭据：优先取后端"我的分机"（tb_dn，唯一密码源）；失败回落环境变量
    let sipExtension = extension
    let sipPassword = String(env.VITE_SIP_PASSWORD || '1234')
    const dn = await loadMyDn()
    if (dn?.dn_num) {
      sipExtension = dn.dn_num
      sipPassword = dn.password || sipPassword
    }
    return {
      wsUri,
      domain: env.VITE_SIP_DOMAIN || window.location.hostname,
      password: sipPassword,
      extension: sipExtension,
    }
  }

  /**
   * 签入上班：已有 SDK 实例时直接 connect（UI 不重建、无闪烁）；
   * 无实例（被踢/初始化缺失）才初始化新实例。
   * workStatus 由 push 下发的 agent.status_change 事件（经 SDK onStatusChange）权威驱动。
   */
  async function agentSignIn(extension: string) {
    if (!extension) throw new Error('缺少分机号')
    const userStore = useUserStore()
    if (!userStore.token) throw new Error('管理后台未登录')
    if (!window.VocaraPhoneBar) throw new Error('电话条 SDK 未加载')

    const env = (import.meta as any).env
    // push 座席 WS 端点：/ws/agent（开发环境经 vite 代理到 8084）
    const wsBase = String(env.VITE_PUSH_WS_URL || '/ws').replace(/\/$/, '')

    // 已有实例（离线态或在线）：
    if (bar) {
      if (!connected.value) {
        // 离线实例缺 SIP 配置时补齐（凭据不可用时为 undefined，仅业务 WS 签入）
        const sip = await buildSipConfig(extension)
        if (sip && !(bar as any).opts?.sip?.wsUri) {
          ;(bar as any).opts.sip = sip
        }
        // 同实例 connect：WS 连接即签入，UI 无感切换
        bar.connect()
        agentAuth.value = {
          agentId: extension,
          tenantId: String(userStore.userInfo?.tenant_id || '1'),
          extension,
        }
        workStatus.value = 'checked'
        autoReadyAfterSignIn = true
        return
      }
      // 已在线的重复签入：销毁重建
      destroyBar()
    }

    // SIP 软电话配置：注册号/密码来自后端 tb_dn（动态目录的唯一凭据源）
    const sip = await buildSipConfig(extension)

    bar = window.VocaraPhoneBar.init({
      container: barContainer || undefined,
      wsUrl: `${wsBase}/agent`,
      token: userStore.token,
      extension,
      // SIP 软电话配置：注册号/密码来自后端 tb_dn（动态目录的唯一凭据源）
      sip,
      ...buildPhoneBarHandlers(extension),
    })

    bar.connect()
    agentAuth.value = {
      agentId: extension,
      tenantId: String(userStore.userInfo?.tenant_id || '1'),
      extension,
    }
    // 乐观置 checked，等待 status_change 权威事件
    workStatus.value = 'checked'
    // 会话建立后自动示闲（handleSdkStatus 收到 CHECKED 时触发）
    autoReadyAfterSignIn = true
  }

  /** 示闲（ack 乐观更新，status_change 事件为权威） */
  async function agentSetReady() {
    if (!bar) throw new Error('座席未签入')
    bar.ready()
    workStatus.value = 'ready'
  }

  /** 示忙 */
  async function agentSetBusy(reason?: string) {
    if (!bar) throw new Error('座席未签入')
    bar.busy(reason)
    workStatus.value = 'busy'
  }

  /** 话后处理（复用示忙，ACD 不再分配来电） */
  async function agentSetWrapup() {
    if (!bar) throw new Error('座席未签入')
    bar.busy('wrapup')
    workStatus.value = 'wrapup'
  }

  /** 签出下班 */
  async function agentSignOut() {
    if (bar) {
      try {
        bar.disconnect()
      } catch (e) {
        // 忽略签出发送失败，本地仍清理
      }
      bar = null
    }
    stopRing()
    connected.value = false
    currentCall.value = null
    mutedFlag.value = false
    agentAuth.value = null
    workStatus.value = 'offline'
  }

  /** 销毁电话条（退出登录时由布局调用） */
  function destroyAdapter() {
    destroyBar()
    agentAuth.value = null
    workStatus.value = 'offline'
    myDn.value = null
  }

  // ===== 通话操作（委托 SDK 实例方法）=====

  function makeCall(called: string, _extras?: Record<string, any>) {
    if (!bar) {
      notification.warning({ title: '提示', content: '电话条未签入' })
      return false
    }
    bar.dial(called)
    currentCall.value = {
      callUuid: `sip-out-${Date.now()}`,
      agentId: agentAuth.value ? Number(agentAuth.value.agentId) || 0 : 0,
      tenantId: Number(agentAuth.value?.tenantId) || 0,
      queueId: 0,
      direction: 'outbound',
      callerNum: agentAuth.value?.extension || '',
      calledNum: called,
      ani: agentAuth.value?.extension || '',
      dnis: called,
      startTime: Date.now(),
      duration: 0,
      holdCount: 0,
      status: 'dialing',
      recording: false,
      muted: false,
      held: false,
    }
    return true
  }

  function answer() {
    if (!bar) return false
    stopRing()
    bar.answer()
    return true
  }

  function hangup() {
    if (!bar) return false
    stopRing()
    bar.hangup()
    return true
  }

  /** SDK 暂未提供的能力：保持接口签名，提示后返回 false */
  function unsupported(feature: string): boolean {
    notification.warning({ title: '提示', content: `当前电话条暂不支持${feature}` })
    return false
  }

  function hold() {
    return unsupported('呼叫保持')
  }

  function unhold() {
    return unsupported('呼叫保持')
  }

  function toggleHold() {
    return unsupported('呼叫保持')
  }

  function mute() {
    if (!bar) return false
    bar.mute(true)
    mutedFlag.value = true
    if (currentCall.value) currentCall.value = { ...currentCall.value, muted: true }
    return true
  }

  function unmute() {
    if (!bar) return false
    bar.mute(false)
    mutedFlag.value = false
    if (currentCall.value) currentCall.value = { ...currentCall.value, muted: false }
    return true
  }

  function toggleMute() {
    return mutedFlag.value ? unmute() : mute()
  }

  function sendDtmf(_digit: string) {
    return unsupported('DTMF 按键')
  }

  function transfer(_target: string) {
    return unsupported('转接')
  }

  function consultTransfer(_target: string) {
    return unsupported('咨询转接')
  }

  function conference(_target: string) {
    return unsupported('三方通话')
  }

  function startRecord() {
    return unsupported('录音控制')
  }

  function stopRecord() {
    return unsupported('录音控制')
  }

  function toggleRecord() {
    return unsupported('录音控制')
  }

  // ===== 桌面通知 =====

  function requestNotificationPermission() {
    if ('Notification' in window && Notification.permission === 'default') {
      Notification.requestPermission()
    }
  }

  return {
    // 状态
    connected,
    currentCall,
    callHistory,
    workStatus,
    agentAuth,
    myDn,
    sipRegistered,
    // 计算属性
    callStatus,
    isIdle,
    isRinging,
    isTalking,
    isHolding,
    isDialing,
    isInbound,
    isOutbound,
    displayNumber,
    displayName,
    formattedDuration,
    // SDK 挂载
    setPhoneBarContainer,
    destroyAdapter,
    // 我的分机
    loadMyDn,
    // 座席话务状态
    agentSignIn,
    agentSignOut,
    agentSetReady,
    agentSetBusy,
    agentSetWrapup,
    // 通话操作
    makeCall,
    answer,
    hangup,
    hold,
    unhold,
    toggleHold,
    mute,
    unmute,
    toggleMute,
    sendDtmf,
    transfer,
    consultTransfer,
    conference,
    startRecord,
    stopRecord,
    toggleRecord,
  }
})
