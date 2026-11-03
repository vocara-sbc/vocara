// 座席域类型定义
// 字段名与 vocara-admin 后端模型 JSON 输出保持一致（snake_case）

// 座席（对应 tb_agent）
export interface AgentInfo {
  id: number
  tenant_id: number
  agent_number: string
  agent_name: string
  mobile: string
  email: string
  avatar: string
  gender: number
  dn: string
  role: string
  status: number
  work_status: number
  org_id: number
  skill_levels: string
  last_login_at: string
  last_login_ip: string
  remark: string
  created_at: string
  updated_at: string
}

// 座席保存/更新表单
export interface AgentForm {
  id?: number
  agent_number: string
  agent_name: string
  password?: string
  mobile?: string
  email?: string
  gender?: number
  dn?: string
  role?: string
  status?: number
  org_id?: number
  remark?: string
}

// 座席工作状态（对应 tb_agent.work_status）
export const AgentWorkStatusMap: Record<number, { text: string; color: string }> = {
  0: { text: '离线', color: '#9ca3af' },
  1: { text: '空闲', color: '#18a058' },
  2: { text: '忙碌', color: '#f0a020' },
  3: { text: '通话中', color: '#2080f0' },
  4: { text: '休息', color: '#7c3aed' },
  5: { text: '会后处理', color: '#0e7490' },
}

// 座席角色（对应 tb_agent.role）
export const AgentRoleOptions = [
  { label: '座席', value: 'agent' },
  { label: '班长', value: 'supervisor' },
  { label: '管理员', value: 'admin' },
]

// 组织机构（对应 tb_organization）
export interface Organization {
  id: number
  tenant_id: number
  org_name: string
  org_code: string
  parent_id: number
  level: number
  path: string
  leader_id: number
  leader_name: string
  sort: number
  status: number
  remark: string
  created_at: string
  updated_at: string
}

// 技能组（对应 tb_skill_group）
export interface SkillGroup {
  id: number
  tenant_id: number
  skill_name: string
  skill_code: string
  skill_type: number
  status: number
  priority: number
  agent_count: number
  remark: string
  created_at: string
  updated_at: string
}

// 技能组座席（对应 tb_agent_skill）
export interface SkillGroupAgent {
  id: number
  skill_group_id: number
  agent_id: number
  agent_number: string
  agent_name: string
  skill_level: number
  created_at: string
}

// 技能类型（对应 tb_skill_group.skill_type）
export const SkillTypeOptions = [
  { label: '呼入', value: 1 },
  { label: '呼出', value: 2 },
  { label: '混合型', value: 3 },
]

// 分机（对应 tb_dn）
export interface DnInfo {
  id: number
  tenant_id: number
  dn_num: string
  dn_name: string
  dn_type: number
  agent_id: number
  agent_number: string
  status: number
  register_status: number
  register_ip: string
  register_ua: string
  last_register_at: string
  fs_instance_id: number
  remark: string
  created_at: string
  updated_at: string
}

// 分机类型（对应 tb_dn.dn_type）
export const DnTypeOptions = [
  { label: '座席分机', value: 1 },
  { label: '通用分机', value: 2 },
]
