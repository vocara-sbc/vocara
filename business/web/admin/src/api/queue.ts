import { request } from '@/utils/request'
import type { PageData, PageQuery } from '@/types/common'
import type { QueueInfo, QueueCall } from '@/types/queue'

// 队列分页列表
export function getQueueList(params: PageQuery) {
  return request<PageData<QueueInfo>>({
    url: '/v1/admin/queues/page',
    method: 'get',
    params,
  })
}

// 队列列表（不分页）
export function getQueueAll() {
  return request<QueueInfo[]>({
    url: '/v1/admin/queues/list',
    method: 'get',
  })
}

// 队列详情
export function getQueueDetail(id: number) {
  return request<QueueInfo>({
    url: `/v1/admin/queues/${id}`,
    method: 'get',
  })
}

// 新增队列
export function createQueue(data: Partial<QueueInfo>) {
  return request({
    url: '/v1/admin/queues/save',
    method: 'post',
    data,
  })
}

// 更新队列
export function updateQueue(data: Partial<QueueInfo>) {
  return request({
    url: '/v1/admin/queues/update',
    method: 'post',
    data,
  })
}

// 删除队列
export function deleteQueue(id: number) {
  return request({
    url: `/v1/admin/queues/${id}`,
    method: 'delete',
  })
}

// 所有队列监控数据（返回 tb_queue 模型列表，含 wait_num/agent_ready_num/agent_busy_num 实时字段）
export function getAllQueueStats() {
  return request<QueueInfo[]>({
    url: '/v1/admin/queues/monitor/list',
    method: 'get',
  })
}

// 单队列实时监控（后端仅提供监控列表接口，按 ID 从列表中匹配）
export async function getQueueStats(queueId: number) {
  const list = await getAllQueueStats()
  return list.find((q) => q.id === queueId)
}

// 队列座席关联信息（后端返回队列 ID 与关联技能组 ID 列表）
export function getQueueAgents(id: number) {
  return request<{ queue_id: number; skill_ids: string }>({
    url: `/v1/admin/queues/${id}/agents`,
    method: 'get',
  })
}

// 队列等待通话列表
export function getQueueWaitingCalls(queueId: number) {
  return request<QueueCall[]>({
    url: `/v1/admin/queues/${queueId}/waiting`,
    method: 'get',
  })
}
