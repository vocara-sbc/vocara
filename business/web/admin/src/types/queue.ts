// 队列相关类型（字段与后端 tb_queue 模型一致，snake_case）

export interface QueueInfo {
  id: number
  tenant_id: number
  queue_name: string
  queue_code: string
  // 队列类型：1-呼入 2-呼出
  queue_type: number
  // 分配策略：round_robin-轮询 longest_idle-最长空闲 skill_level-技能等级
  strategy: string
  max_wait_time: number
  max_wait_num: number
  // 溢出处理：0-挂断 1-转队列 2-转IVR 3-转座席
  overflow_type: number
  overflow_value?: string
  // 无座席处理：0-等待 1-溢出 2-挂断
  no_agent_type: number
  wrap_up_time: number
  skill_ids?: string
  status: number
  wait_num: number
  agent_ready_num: number
  agent_busy_num: number
  sound_file?: string
  remark?: string
  created_at?: string
  updated_at?: string
}

// 队列分页查询参数
export interface QueuePageQuery {
  page?: number
  page_size?: number
  keyword?: string
  queue_type?: number
  status?: number
}

export interface QueueStats {
  queueId: number
  queueName: string
  waitingCount: number
  pausedCount: number
  servingCount: number
  totalToday: number
  answeredToday: number
  abandonedToday: number
  timeoutToday: number
  avgWaitTime: number
  maxWaitTime: number
  longestCallId?: string
  sla: number
  slaLevel: 'good' | 'normal' | 'bad'
  updateTime: string
}

export interface QueueCall {
  callId: string
  callerNum: string
  calledNum: string
  priority: number
  position: number
  waitTime: number
  eqTime: string
  tenantId: number
}
