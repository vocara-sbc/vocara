import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import NProgress from 'nprogress'
import { useUserStore } from '@/store/modules/user'
import { message } from '@/utils/naive'

// 呼叫控制服务（callctrl 8083）专用实例：不走 /api baseURL，直接命中 vite 的 /callapi 代理
const callService: AxiosInstance = axios.create({
  baseURL: '/callapi',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json;charset=utf-8',
  },
})

// 请求拦截器：携带与主实例一致的 Bearer 令牌
callService.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    NProgress.start()
    const userStore = useUserStore()
    if (userStore.token) {
      config.headers.Authorization = `Bearer ${userStore.token}`
    }
    return config
  },
  (error) => {
    NProgress.done()
    return Promise.reject(error)
  }
)

// 响应拦截器：业务成功（code 为 0/200）直接解包 data，其余统一提示并拒绝
callService.interceptors.response.use(
  (resp: AxiosResponse) => {
    NProgress.done()
    const res = resp.data
    if (res && typeof res === 'object' && 'code' in res) {
      if (res.code === 0 || res.code === 200) return res.data
      message.error(res.message || '请求失败')
      return Promise.reject(new Error(res.message || '请求失败'))
    }
    return res
  },
  (error) => {
    NProgress.done()
    if (error.response) {
      message.error(error.response.data?.message || `请求失败 (${error.response.status})`)
    } else if (error.request) {
      message.error('网络错误，请检查网络连接')
    } else {
      message.error(error.message || '请求失败')
    }
    return Promise.reject(error)
  }
)

function callRequest<T = any>(config: AxiosRequestConfig): Promise<T> {
  // 响应拦截器已统一解包为业务 data，这里按拦截器后的真实形态断言
  return callService.request<any, T>(config) as Promise<T>
}

// ========== 类型定义（字段与 callctrl 服务 model.CallInfo / service.CallStats 一致） ==========

// 活跃通话信息
export interface ActiveCallInfo {
  callId: string
  // 呼叫类型：inbound-呼入 outbound-呼出 internal-内部
  callType: string
  tenantId: string
  caller: string
  callerName?: string
  called: string
  calledName?: string
  agentId?: string
  agentName?: string
  extension?: string
  queueId?: string
  // 通话状态：created/ringing/early_media/agent_answered/answered/bridged/held/transferring/supervising/three_way 等
  status: string
  // 通话开始时间（Unix 秒）
  startTime?: number
  // 接听时间（Unix 秒）
  answerTime?: number
  // 通话时长（秒，从接听开始算）
  duration?: number
  isHeld?: boolean
  superviseMode?: string
}

// 活跃通话分页数据（后端 PageData 为 snake_case）
export interface ActiveCallPage {
  list: ActiveCallInfo[]
  total: number
  page: number
  page_size: number
  pages: number
}

// 通话统计
export interface CallStatsInfo {
  tenantId: string
  activeCount: number
  bridgedCount: number
  ringingCount: number
  heldCount: number
  supervisingCount: number
}

// 监督操作（监听/耳语/强插）请求参数
export interface SuperviseParams {
  callId: string
  supervisorId: string
  supervisorName?: string
  extension?: string
  tenantId?: string
  timeout?: number
}

// ========== 监控接口 ==========

// 活跃通话列表
export function getActiveCalls(params: { tenantId: string; page?: number; pageSize?: number }) {
  return callRequest<ActiveCallPage>({
    url: '/active',
    method: 'get',
    params,
  })
}

// 通话统计
export function getCallStats(tenantId: string) {
  return callRequest<CallStatsInfo>({
    url: '/stats',
    method: 'get',
    params: { tenantId },
  })
}

// 挂断通话（强拆）
export function hangupCall(data: { callId: string; cause?: string }) {
  return callRequest<null>({
    url: '/hangup',
    method: 'post',
    data,
  })
}

// ========== 监督接口 ==========

// 监听（只能听不能说）
export function eavesdropCall(data: SuperviseParams) {
  return callRequest<unknown>({
    url: '/eavesdrop',
    method: 'post',
    data,
  })
}

// 耳语（班长对座席说话，客户听不到）
export function whisperCall(data: SuperviseParams) {
  return callRequest<null>({
    url: '/whisper',
    method: 'post',
    data,
  })
}

// 强插（三方都能说话）
export function bargeCall(data: SuperviseParams) {
  return callRequest<null>({
    url: '/barge',
    method: 'post',
    data,
  })
}

// 停止监听/耳语/强插
export function stopSupervise(callId: string) {
  return callRequest<null>({
    url: '/stop-supervise',
    method: 'post',
    data: { callId },
  })
}

// 切换监听模式：eavesdrop / whisper / barge
export function switchSuperviseMode(callId: string, mode: 'eavesdrop' | 'whisper' | 'barge') {
  return callRequest<null>({
    url: '/supervise-mode',
    method: 'post',
    data: { callId, mode },
  })
}
