import { request } from '@/utils/request'
import type { LoginForm, ProfileUpdateForm, UserInfo } from '@/types/user'
import type { ApiResponse } from '@/types/common'

// 登录
export function loginApi(data: LoginForm) {
  return request<ApiResponse>({
    url: '/v1/admin/auth/login',
    method: 'post',
    data,
  })
}

// 登出
export function logoutApi() {
  return request({
    url: '/v1/admin/auth/logout',
    method: 'post',
  })
}

// 获取用户信息
export function getUserInfoApi() {
  return request<UserInfo>({
    url: '/v1/admin/auth/userinfo',
    method: 'get',
  })
}

// 修改密码
export function changePasswordApi(data: { old_password: string; new_password: string }) {
  return request({
    url: '/v1/admin/auth/change_password',
    method: 'post',
    data,
  })
}

// 更新个人资料（字段白名单：姓名、手机、邮箱、性别）
export function updateProfileApi(data: ProfileUpdateForm) {
  return request({
    url: '/v1/admin/auth/profile',
    method: 'put',
    data,
  })
}

// 当前登录座席的绑定分机信息（SIP 注册凭据，电话条签入用）
export interface MyDnInfo {
  dn_num: string
  dn_name: string
  password: string
}

// 获取"我的分机"（含 SIP 注册密码；未绑定分机时后端返回业务错误）
export function getMyDnApi() {
  return request<MyDnInfo>({
    url: '/v1/admin/dns/my',
    method: 'get',
  })
}
