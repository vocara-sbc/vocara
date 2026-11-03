import { request } from '@/utils/request'
import type { PageData, PageQuery } from '@/types/common'
import type {
  AgentInfo,
  AgentForm,
  Organization,
  SkillGroup,
  SkillGroupAgent,
  DnInfo,
} from '@/types/agent'

// ========== 座席管理 ==========

// 座席分页列表
export function getAgentList(params: PageQuery) {
  return request<PageData<AgentInfo>>({
    url: '/v1/admin/agents/page',
    method: 'get',
    params,
  })
}

// 座席列表（不分页）
export function getAgentAll(params?: { keyword?: string }) {
  return request<AgentInfo[]>({
    url: '/v1/admin/agents/list',
    method: 'get',
    params,
  })
}

// 座席详情
export function getAgentDetail(id: number) {
  return request<AgentInfo>({
    url: `/v1/admin/agents/${id}`,
    method: 'get',
  })
}

// 新增座席
export function createAgent(data: AgentForm) {
  return request<AgentInfo>({
    url: '/v1/admin/agents/save',
    method: 'post',
    data,
  })
}

// 更新座席
export function updateAgent(data: AgentForm) {
  return request({
    url: '/v1/admin/agents/update',
    method: 'post',
    data,
  })
}

// 删除座席
export function deleteAgent(id: number) {
  return request({
    url: `/v1/admin/agents/${id}`,
    method: 'delete',
  })
}

// 启用/禁用座席
export function toggleAgentStatus(id: number, enabled: boolean) {
  return request({
    url: `/v1/admin/agents/${id}/${enabled ? 'enable' : 'disable'}`,
    method: 'post',
  })
}

// 重置座席密码（后端字段为 new_password）
export function resetAgentPassword(id: number, newPassword: string) {
  return request({
    url: `/v1/admin/agents/${id}/reset_password`,
    method: 'post',
    data: { new_password: newPassword },
  })
}

// 绑定分机（后端字段为 dn，即分机号字符串；传空串表示解除绑定）
export function bindAgentDn(id: number, dn: string) {
  return request({
    url: `/v1/admin/agents/${id}/bind_dn`,
    method: 'post',
    data: { dn },
  })
}

// 配置座席技能（覆盖式：后端先清空再按 skill_items 重建）
export function updateAgentSkill(id: number, skillItems: Array<{ skill_id: number; skill_level: number }>) {
  return request({
    url: `/v1/admin/agents/${id}/skill`,
    method: 'post',
    data: { skill_items: skillItems },
  })
}

// 批量导入座席（后端接收 JSON 数组，逐条创建）
export function importAgents(items: AgentForm[]) {
  return request<{ success: number; failed: number }>({
    url: '/v1/admin/agents/import',
    method: 'post',
    data: items,
  })
}

// 导出座席（后端返回全量 JSON 数据，由前端生成下载文件）
export function exportAgents(params: PageQuery) {
  return request<{ list: AgentInfo[]; total: number }>({
    url: '/v1/admin/agents/export',
    method: 'get',
    params,
  })
}

// ========== 组织机构管理 ==========

export function getOrganizationAll() {
  return request<Organization[]>({
    url: '/v1/admin/organizations/list',
    method: 'get',
  })
}

export function createOrganization(data: Partial<Organization>) {
  return request({
    url: '/v1/admin/organizations/save',
    method: 'post',
    data,
  })
}

export function updateOrganization(data: Partial<Organization>) {
  return request({
    url: '/v1/admin/organizations/update',
    method: 'post',
    data,
  })
}

export function deleteOrganization(id: number) {
  return request({
    url: `/v1/admin/organizations/${id}`,
    method: 'delete',
  })
}

// ========== 技能组管理 ==========

export function getSkillGroupList(params: PageQuery) {
  return request<PageData<SkillGroup>>({
    url: '/v1/admin/skill_groups/page',
    method: 'get',
    params,
  })
}

export function getSkillGroupAll() {
  return request<SkillGroup[]>({
    url: '/v1/admin/skill_groups/list',
    method: 'get',
  })
}

export function createSkillGroup(data: Partial<SkillGroup>) {
  return request({
    url: '/v1/admin/skill_groups/save',
    method: 'post',
    data,
  })
}

export function updateSkillGroup(data: Partial<SkillGroup>) {
  return request({
    url: '/v1/admin/skill_groups/update',
    method: 'post',
    data,
  })
}

export function deleteSkillGroup(id: number) {
  return request({
    url: `/v1/admin/skill_groups/${id}`,
    method: 'delete',
  })
}

// 技能组座席列表
export function getSkillGroupAgents(id: number) {
  return request<SkillGroupAgent[]>({
    url: `/v1/admin/skill_groups/${id}/agents`,
    method: 'get',
  })
}

// 保存技能组座席配置（agent_levels 为 座席ID → 技能等级 的映射，增量覆盖语义）
export function setSkillGroupAgents(id: number, agentLevels: Record<number, number>) {
  return request({
    url: `/v1/admin/skill_groups/${id}/agents`,
    method: 'post',
    data: { agent_levels: agentLevels },
  })
}

// ========== 分机管理 ==========

export function getDnList(params: PageQuery) {
  return request<PageData<DnInfo>>({
    url: '/v1/admin/dns/page',
    method: 'get',
    params,
  })
}

export function getDnAll() {
  return request<DnInfo[]>({
    url: '/v1/admin/dns/list',
    method: 'get',
  })
}

// 可用（未绑定座席的）分机列表
export function getDnAvailable() {
  return request<DnInfo[]>({
    url: '/v1/admin/dns/available',
    method: 'get',
  })
}

export function createDn(data: Partial<DnInfo> & { password?: string }) {
  return request({
    url: '/v1/admin/dns/save',
    method: 'post',
    data,
  })
}

export function updateDn(data: Partial<DnInfo> & { password?: string }) {
  return request({
    url: '/v1/admin/dns/update',
    method: 'post',
    data,
  })
}

export function deleteDn(id: number) {
  return request({
    url: `/v1/admin/dns/${id}`,
    method: 'delete',
  })
}

// 分机注册状态检测（后端返回全量分机及其最新注册状态）
export function checkDnStatus() {
  return request<DnInfo[]>({
    url: '/v1/admin/dns/status_check',
    method: 'post',
  })
}

// 分机绑定座席
export function bindDnAgent(id: number, agentId: number) {
  return request({
    url: `/v1/admin/dns/${id}/bind_agent`,
    method: 'post',
    data: { agent_id: agentId },
  })
}

// 分机解绑座席
export function unbindDnAgent(id: number) {
  return request({
    url: `/v1/admin/dns/${id}/unbind_agent`,
    method: 'post',
  })
}

// 座席状态数量统计（后端按启用/禁用维度返回，键为 enabled/disabled）
export function getAgentStatusCount() {
  return request<Record<string, number>>({
    url: '/v1/admin/agents/status/count',
    method: 'get',
  })
}
