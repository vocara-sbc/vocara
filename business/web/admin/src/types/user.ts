// 用户相关类型

export interface UserInfo {
  user_id: number
  username: string
  nickname: string
  avatar?: string
  mobile?: string
  email?: string
  gender?: number
  role: 'admin' | 'supervisor' | 'agent'
  roles: string[]
  permissions: string[]
  tenant_id: number
  tenant_name: string
  tenant_code: string
  // 登录后跳转的首页路径
  home_path: string
}

// 个人资料更新请求（字段白名单，与后端一致）
export interface ProfileUpdateForm {
  agent_name: string
  mobile?: string
  email?: string
  gender: number
}

export interface LoginForm {
  username: string
  password: string
  tenant_code?: string
  remember?: boolean
}

export interface LoginResult {
  token: string
  user_info: UserInfo
}
