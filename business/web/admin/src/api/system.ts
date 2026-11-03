import { request } from '@/utils/request'
import type { PageData, PageQuery } from '@/types/common'

// ========== 参数配置 ==========

// 参数配置项（字段与后端 tb_param_config 模型一致）
export interface SystemConfigItem {
  id: number
  tenant_id: number
  config_key: string
  config_value: string
  // 配置类型：string/int/bool/json
  config_type: string
  config_desc?: string
  is_system: number
  created_at?: string
  updated_at?: string
}

// 配置保存参数（与后端 ConfigSaveRequest 一致）
export interface ConfigSaveParams {
  config_key: string
  config_value: string
  config_type?: string
  config_desc?: string
}

export function getConfigList() {
  return request<SystemConfigItem[]>({
    url: '/v1/admin/system/param/list',
    method: 'get',
  })
}

export function saveConfig(data: ConfigSaveParams) {
  return request({
    url: '/v1/admin/system/param/save',
    method: 'post',
    data,
  })
}

export function deleteConfig(id: number) {
  return request({
    url: `/v1/admin/system/param/${id}`,
    method: 'delete',
  })
}

// ========== 操作日志 ==========

export function getOperationLogList(params: PageQuery) {
  return request<PageData<any>>({
    url: '/v1/admin/system/operation_log/page',
    method: 'get',
    params,
  })
}

// ========== FS 实例管理 ==========

export function getFsInstanceList(params: PageQuery) {
  return request<PageData<any>>({
    url: '/v1/admin/system/fs/page',
    method: 'get',
    params,
  })
}

export function getFsInstanceAll() {
  return request({
    url: '/v1/admin/system/fs/list',
    method: 'get',
  })
}

export function createFsInstance(data: any) {
  return request({
    url: '/v1/admin/system/fs/save',
    method: 'post',
    data,
  })
}

export function updateFsInstance(data: any) {
  return request({
    url: '/v1/admin/system/fs/update',
    method: 'post',
    data,
  })
}

export function deleteFsInstance(id: number) {
  return request({
    url: `/v1/admin/system/fs/${id}`,
    method: 'delete',
  })
}

// ========== 租户管理 ==========

// 租户信息（字段与后端 tb_tenant 模型一致）
export interface TenantInfo {
  id: number
  tenant_name: string
  tenant_code: string
  contact_name?: string
  contact_phone?: string
  contact_email?: string
  address?: string
  status: number
  expire_time?: string
  max_agents: number
  max_lines: number
  remark?: string
  created_at?: string
  updated_at?: string
}

export function getTenantList(params: PageQuery) {
  return request<PageData<TenantInfo>>({
    url: '/v1/admin/system/tenant/page',
    method: 'get',
    params,
  })
}

export function getTenantAll() {
  return request<TenantInfo[]>({
    url: '/v1/admin/system/tenant/list',
    method: 'get',
  })
}

export function createTenant(data: Partial<TenantInfo>) {
  return request({
    url: '/v1/admin/system/tenant/save',
    method: 'post',
    data,
  })
}

export function updateTenant(data: Partial<TenantInfo>) {
  return request({
    url: '/v1/admin/system/tenant/update',
    method: 'post',
    data,
  })
}

export function deleteTenant(id: number) {
  return request({
    url: `/v1/admin/system/tenant/${id}`,
    method: 'delete',
  })
}
