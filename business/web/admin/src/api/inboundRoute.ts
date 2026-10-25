import { request } from '@/utils/request'
import type { PageData, PageQuery } from '@/types/common'

// 呼入接待配置（DID 路由）
export interface InboundRouteInfo {
  id: number
  route_name: string
  route_code: string
  did?: string
  welcome_sound?: string
  wait_sound?: string
  queue_id?: number
  queue_name?: string
  skill_group_id?: number
  work_time_type?: number
  work_time_config?: string
  holiday_ids?: string
  off_work_type?: number
  off_work_value?: string
  off_work_sound?: string
  status: number
  sort?: number
  remark?: string
  created_at?: string
  updated_at?: string
}

export interface InboundRoutePageQuery extends PageQuery {
  route_name?: string
  route_code?: string
  did?: string
  queue_id?: number
  status?: number
  keyword?: string
}

// 呼入接待分页列表
export function getInboundRoutePage(params: InboundRoutePageQuery) {
  return request<PageData<InboundRouteInfo>>({
    url: '/v1/admin/inbound-routes/page',
    method: 'get',
    params,
  })
}

// 呼入接待列表（不分页）
export function getInboundRouteAll() {
  return request<InboundRouteInfo[]>({
    url: '/v1/admin/inbound-routes/list',
    method: 'get',
  })
}

// 呼入接待详情
export function getInboundRouteDetail(id: number) {
  return request<InboundRouteInfo>({
    url: `/v1/admin/inbound-routes/${id}`,
    method: 'get',
  })
}

// 新增呼入接待
export function createInboundRoute(data: Partial<InboundRouteInfo>) {
  return request({
    url: '/v1/admin/inbound-routes/save',
    method: 'post',
    data,
  })
}

// 更新呼入接待
export function updateInboundRoute(data: Partial<InboundRouteInfo>) {
  return request({
    url: '/v1/admin/inbound-routes/update',
    method: 'post',
    data,
  })
}

// 删除呼入接待
export function deleteInboundRoute(id: number) {
  return request({
    url: `/v1/admin/inbound-routes/${id}`,
    method: 'delete',
  })
}
