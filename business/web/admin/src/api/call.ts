import { request } from '@/utils/request'
import type { ApiResponse, PageData, PageQuery } from '@/types/common'
import type { CallRecord, CallQuery, ActiveCall } from '@/types/call'

// ========== 通话记录 ==========

// 呼入记录分页
export function getInboundRecords(params: CallQuery) {
  return request<PageData<CallRecord>>({
    url: '/v1/admin/records/inbound/page',
    method: 'get',
    params,
  })
}

// 呼出记录分页
export function getOutboundRecords(params: CallQuery) {
  return request<PageData<CallRecord>>({
    url: '/v1/admin/records/outbound/page',
    method: 'get',
    params,
  })
}

// 通话详情
export function getCallDetail(id: number) {
  return request<CallRecord>({
    url: `/v1/admin/records/${id}`,
    method: 'get',
  })
}

// 导出通话记录
export function exportRecords(params: CallQuery) {
  return request({
    url: '/v1/admin/records/export',
    method: 'get',
    params,
    responseType: 'blob',
  })
}

// ========== 活跃通话 ==========

// 今日通话列表
export function getActiveCalls(params?: { tenantId?: number }) {
  return request<ActiveCall[]>({
    url: '/v1/admin/records/today',
    method: 'get',
    params,
  })
}

// 实时统计统一从 @/api/stats 的 getRealtimeStats 获取

// ========== 录音 ==========

// 获取录音文件信息
export function getRecording(callId: string) {
  return request({
    url: `/v1/admin/records/${callId}/recording`,
    method: 'get',
  })
}

// 下载录音
export function downloadRecording(callId: string) {
  return request({
    url: `/v1/admin/records/${callId}/recording/download`,
    method: 'get',
    responseType: 'blob',
  })
}
