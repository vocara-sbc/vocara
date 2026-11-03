// 通话相关类型

import type { PageQuery } from './common'

export interface CallRecord {
  id: number
  callId: string
  callType: 'inbound' | 'outbound' | 'internal'
  callerNum: string
  calledNum: string
  agentId?: number
  agentName?: string
  extension?: string
  queueId?: number
  queueName?: string
  startTime: string
  answerTime?: string
  endTime?: string
  duration: number
  talkDuration: number
  waitDuration: number
  hangupCause: string
  hangupSide: 'caller' | 'callee' | 'system'
  recordingFile?: string
  recordingUrl?: string
  tenantId: number
}

export interface CallQuery extends PageQuery {
  callType?: string
  agentId?: number
  queueId?: number
  startTime?: string
  endTime?: string
  hangupCause?: string
}

// 活跃通话
export interface ActiveCall {
  callId: string
  callType: 'inbound' | 'outbound'
  callerNum: string
  calledNum: string
  agentId: number
  agentName: string
  extension: string
  queueId?: number
  queueName?: string
  status: 'ringing' | 'talking' | 'hold' | 'transferring'
  startTime: string
  answerTime?: string
  duration: number
  tenantId: number
}

// 实时统计
export interface RealtimeStats {
  totalCalls: number
  activeCalls: number
  waitingCalls: number
  answeredCalls: number
  abandonedCalls: number
  avgWaitTime: number
  maxWaitTime: number
  avgTalkTime: number
  serviceLevel: number
}
