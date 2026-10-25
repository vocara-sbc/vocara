import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { UserInfo, LoginForm } from '@/types/user'
import { loginApi, logoutApi, getUserInfoApi } from '@/api/auth'
import { useTabStore } from '@/store/modules/tab'

const TOKEN_KEY = 'vocara_admin_token'

// 后端 UserInfo 为 snake_case 且以 user_info 包裹，统一归一化为前端使用的 camelCase
function normalizeUserInfo(data: any): UserInfo {
  return {
    user_id: data?.user_id ?? data?.id ?? 0,
    username: data?.username ?? '',
    nickname: data?.nickname ?? '',
    avatar: data?.avatar || '',
    mobile: data?.mobile || '',
    email: data?.email || '',
    gender: data?.gender ?? 0,
    role: (data?.role ?? 'agent') as UserInfo['role'],
    roles: data?.roles || (data?.role ? [data.role] : []),
    permissions: data?.permissions || [],
    tenant_id: data?.tenant_id ?? data?.tenantId ?? 0,
    tenant_name: data?.tenant_name ?? data?.tenantName ?? '',
    tenant_code: data?.tenant_code ?? data?.tenantCode ?? '',
    home_path: data?.home_path ?? data?.homePath ?? '/dashboard',
  }
}

export const useUserStore = defineStore('user', () => {
  const token = ref<string>(localStorage.getItem(TOKEN_KEY) || '')
  const userInfo = ref<UserInfo | null>(null)

  const isLogin = computed(() => !!token.value)
  const role = computed(() => userInfo.value?.role || '')
  const roles = computed(() => userInfo.value?.roles || [])
  const permissions = computed(() => userInfo.value?.permissions || [])
  const homePath = computed(() => userInfo.value?.home_path || '/dashboard')

  // 判断是否拥有指定权限
  function hasPermission(perm: string): boolean {
    return permissions.value.includes(perm)
  }

  // 判断是否拥有任意一个权限
  function hasAnyPermission(perms: string[]): boolean {
    return perms.some(p => permissions.value.includes(p))
  }

  // 是否是管理员角色（admin 或 supervisor）
  function isAdmin(): boolean {
    const r = role.value
    return r === 'admin' || r === 'supervisor'
  }

  // 是否是普通座席
  function isAgent(): boolean {
    return role.value === 'agent'
  }

  // 登录
  async function login(form: LoginForm) {
    const res = await loginApi(form)
    const data = res as unknown as { token: string; user_info?: unknown }
    token.value = data.token
    userInfo.value = normalizeUserInfo(data.user_info)
    localStorage.setItem(TOKEN_KEY, data.token)
    return data
  }

  // 登出
  function logout() {
    try {
      logoutApi()
    } catch (e) {
      // 忽略登出接口错误
    }
    token.value = ''
    userInfo.value = null
    localStorage.removeItem(TOKEN_KEY)
    // 清空跨账号残留的标签页状态（退出登录即结束本会话的标签记录）
    useTabStore().reset()
  }

  // 获取用户信息
  async function fetchUserInfo() {
    const res = await getUserInfoApi()
    userInfo.value = normalizeUserInfo(res)
    return userInfo.value
  }

  // 设置 token（用于从其他地方恢复登录状态）
  function setToken(t: string) {
    token.value = t
    localStorage.setItem(TOKEN_KEY, t)
  }

  // 设置用户信息（mock 用）
  function setUserInfo(info: UserInfo) {
    userInfo.value = info
  }

  return {
    token,
    userInfo,
    isLogin,
    role,
    roles,
    permissions,
    homePath,
    hasPermission,
    hasAnyPermission,
    isAdmin,
    isAgent,
    login,
    logout,
    fetchUserInfo,
    setToken,
    setUserInfo,
  }
})
