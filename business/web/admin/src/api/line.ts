import { request } from '@/utils/request'
import type { PageData } from '@/types/common'
import type { LineInfo, LinePageQuery } from '@/types/line'

// 线路分页列表
export function getLinePage(params: LinePageQuery) {
  return request<PageData<LineInfo>>({
    url: '/v1/admin/lines/page',
    method: 'get',
    params,
  })
}

// 线路列表（不分页）
export function getLineAll() {
  return request<LineInfo[]>({
    url: '/v1/admin/lines/list',
    method: 'get',
  })
}

// 线路详情
export function getLineDetail(id: number) {
  return request<LineInfo>({
    url: `/v1/admin/lines/${id}`,
    method: 'get',
  })
}

// 新增线路
export function createLine(data: Partial<LineInfo> & { password?: string }) {
  return request({
    url: '/v1/admin/lines/save',
    method: 'post',
    data,
  })
}

// 更新线路
export function updateLine(data: Partial<LineInfo> & { password?: string }) {
  return request({
    url: '/v1/admin/lines/update',
    method: 'post',
    data,
  })
}

// 删除线路
export function deleteLine(id: number) {
  return request({
    url: `/v1/admin/lines/${id}`,
    method: 'delete',
  })
}
