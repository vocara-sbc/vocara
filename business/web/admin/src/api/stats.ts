import { request } from '@/utils/request'

// ===== 统计报表查询参数 =====
// 后端 StatsQuery 按 snake_case 绑定：start_date / end_date / org_id / agent_id / queue_id / line_id
export interface StatsQueryParams {
  start_date?: string
  end_date?: string
  org_id?: number
  agent_id?: number
  queue_id?: number
  line_id?: number
}

// ===== 统计报表返回结构（与后端 model 的 JSON 字段一致）=====

// 座席话务统计
export interface AgentStatsItem {
  agent_id: number
  agent_number: string
  agent_name: string
  org_id: number
  org_name: string
  stat_date: string
  total_calls: number
  inbound_calls: number
  outbound_calls: number
  answered_calls: number
  missed_calls: number
  total_duration: number
  talk_duration: number
  avg_duration: number
  max_duration: number
  work_duration: number
  ready_duration: number
  busy_duration: number
  rest_duration: number
  wrap_up_duration: number
  offline_duration: number
  satisfaction: number
  transfer_count: number
  hold_count: number
  first_call_resolution: number
}

// 队列统计
export interface QueueStatsItem {
  queue_id: number
  queue_name: string
  queue_code: string
  stat_date: string
  total_calls: number
  answered_calls: number
  abandoned_calls: number
  overflow_calls: number
  answer_rate: number
  abandon_rate: number
  avg_wait_time: number
  max_wait_time: number
  avg_talk_time: number
  service_level_20: number
  service_level_30: number
  max_wait_num: number
  avg_agent_ready: number
}

// 线路统计
export interface LineStatsItem {
  line_id: number
  line_name: string
  line_code: string
  stat_date: string
  total_calls: number
  inbound_calls: number
  outbound_calls: number
  answered_calls: number
  failed_calls: number
  success_rate: number
  avg_duration: number
  total_duration: number
  peak_channels: number
  avg_channels: number
}

// 趋势统计：按天粒度的呼入/呼出序列
export interface TrendStatsData {
  dates: string[]
  inbound: number[]
  outbound: number[]
  start_date: string
  end_date: string
}

// 线路选项（线路统计页的筛选下拉用）
export interface LineOption {
  id: number
  line_name: string
  line_code: string
}

// 座席话务统计：返回全量数组，前端本地分页
export function getAgentStats(params: StatsQueryParams) {
  return request<AgentStatsItem[]>({
    url: '/v1/admin/stats/agent',
    method: 'get',
    params,
  })
}

// 队列统计：返回全量数组，前端本地分页
export function getQueueStats(params: StatsQueryParams) {
  return request<QueueStatsItem[]>({
    url: '/v1/admin/stats/queue',
    method: 'get',
    params,
  })
}

// 线路统计：返回全量数组，前端本地分页
export function getLineStats(params: StatsQueryParams) {
  return request<LineStatsItem[]>({
    url: '/v1/admin/stats/line',
    method: 'get',
    params,
  })
}

// 趋势统计：后端固定按天聚合，无粒度/指标参数
export function getTrendStats(params: Pick<StatsQueryParams, 'start_date' | 'end_date'>) {
  return request<TrendStatsData>({
    url: '/v1/admin/stats/trend',
    method: 'get',
    params,
  })
}

// 实时统计
export function getRealtimeStats(tenantId?: number) {
  return request({
    url: '/v1/admin/stats/realtime',
    method: 'get',
    params: { tenantId },
  })
}

// 线路列表（不分页，用于统计筛选下拉）
export function getLineAll() {
  return request<LineOption[]>({
    url: '/v1/admin/lines/list',
    method: 'get',
  })
}
