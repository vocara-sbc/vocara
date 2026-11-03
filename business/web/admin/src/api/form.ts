import { request } from '@/utils/request'
import type { PageData, PageQuery } from '@/types/common'

// 表单模板分页
export function getFormTemplatePage(params: PageQuery) {
  return request<PageData<any>>({ url: '/v1/admin/forms/templates/page', method: 'get', params })
}

// 表单模板详情
export function getFormTemplate(id: number) {
  return request<any>({ url: `/v1/admin/forms/templates/${id}`, method: 'get' })
}

// 新建模板
export function createFormTemplate(data: any) {
  return request({ url: '/v1/admin/forms/templates/save', method: 'post', data })
}

// 更新模板
export function updateFormTemplate(data: any) {
  return request({ url: '/v1/admin/forms/templates/update', method: 'post', data })
}

// 删除模板
export function deleteFormTemplate(id: number) {
  return request({ url: `/v1/admin/forms/templates/${id}`, method: 'delete' })
}

// 表单分类
export interface FormCategoryInfo {
  code: string
  name: string
  sort: number
  enabled: boolean
  builtin: boolean
}

// 分类列表
export function getFormCategories() {
  return request<FormCategoryInfo[]>({ url: '/v1/admin/forms/categories', method: 'get' })
}

// 保存分类（全量）
export function saveFormCategories(list: FormCategoryInfo[]) {
  return request({ url: '/v1/admin/forms/categories', method: 'post', data: list })
}

// 模板版本
export interface FormTemplateVersionInfo {
  id: number
  template_id: number
  version: number
  schema: string
  created_at: string
}

// 更新模板启停（不产生版本）
export function updateFormTemplateStatus(id: number, status: number) {
  return request({ url: `/v1/admin/forms/templates/${id}/status?status=${status}`, method: 'post' })
}

// 模板历史版本列表
export function getTemplateVersions(id: number) {
  return request<FormTemplateVersionInfo[]>({ url: `/v1/admin/forms/templates/${id}/versions`, method: 'get' })
}

// 单个版本完整内容
export function getTemplateVersion(versionId: number) {
  return request<FormTemplateVersionInfo>({ url: `/v1/admin/forms/templates/version/${versionId}`, method: 'get' })
}

// ===== 客户管理 =====
export interface CustomerInfo {
  id: number
  phone: string
  name: string
  data: string
  call_count: number
  first_seen_at: string
  last_seen_at: string
}
// 客户分页
export function listCustomers(params: PageQuery) {
  return request<PageData<CustomerInfo>>({ url: '/v1/admin/forms/customers', method: 'get', params })
}
// 客户详情（含关联记录）
export function getCustomer(id: number) {
  return request<{ customer: CustomerInfo; records: any[] }>({ url: `/v1/admin/forms/customers/${id}`, method: 'get' })
}

// ===== 工单管理 =====
export interface TicketInfo {
  id: number
  template_id: number
  template_version_id: number
  template_key: string
  biz_type: string
  biz_status: string
  customer_id: number
  call_id: string
  direction: string
  agent_id: string
  agent_name: string
  data: string
  created_at: string
  updated_at: string
}
// 工单分页（status/keyword）
export function listTickets(params: PageQuery) {
  return request<PageData<TicketInfo>>({ url: '/v1/admin/forms/tickets', method: 'get', params })
}
// 工单详情
export function getTicket(id: number) {
  return request<TicketInfo>({ url: `/v1/admin/forms/tickets/${id}`, method: 'get' })
}
// 工单状态流转
export function updateTicketStatus(id: number, status: string) {
  return request({ url: `/v1/admin/forms/tickets/${id}/status?status=${status}`, method: 'post' })
}

// 按模板标识取当前版本 schema
export function getTemplateSchemaByKey(key: string) {
  return request<FormTemplateVersionInfo>({ url: `/v1/admin/forms/templates/key/${key}`, method: 'get' })
}

// 表单记录保存（工作台填报，带通话上下文）
export function createFormRecord(data: {
  template_key: string
  data: Record<string, any>
  record_key?: string
  call_id?: string
  direction?: string
  agent_id?: string
  agent_name?: string
}) {
  return request({ url: '/v1/admin/forms/records/save', method: 'post', data })
}
