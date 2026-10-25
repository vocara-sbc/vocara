import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse, InternalAxiosRequestConfig } from 'axios'
import { useUserStore } from '@/store/modules/user'
import { message, dialog } from '@/utils/naive'
import NProgress from 'nprogress'
import router from '@/router'

const service: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json;charset=utf-8',
  },
})

// 请求拦截器
service.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    NProgress.start()
    const userStore = useUserStore()
    if (userStore.token) {
      config.headers.Authorization = `Bearer ${userStore.token}`
    }
    return config
  },
  (error) => {
    NProgress.done()
    return Promise.reject(error)
  }
)

// 响应拦截器
service.interceptors.response.use(
  (response: AxiosResponse) => {
    NProgress.done()
    const res = response.data

    // 如果是二进制数据（下载文件等），直接返回
    if (response.config.responseType === 'blob') {
      return response
    }

    // 业务错误
    if (res.code !== 0 && res.code !== 200) {
      // token 过期
      if (res.code === 40100 || res.code === 40101 || res.code === 40102) {
        const userStore = useUserStore()
        userStore.logout()
        dialog.warning({
          title: '登录已过期',
          content: '您的登录状态已过期，请重新登录',
          positiveText: '重新登录',
          onPositiveClick: () => {
            router.push('/login')
          },
        })
      } else if (res.code >= 70001 && res.code <= 70099) {
        // License 授权错误：到期/失效的持续提示由全局 LicenseGuard 横幅承接，此处透传后端文案
        message.error(res.message || '授权异常')
      } else {
        message.error(res.message || '请求失败')
      }
      return Promise.reject(new Error(res.message || '请求失败'))
    }

    return res.data
  },
  (error) => {
    NProgress.done()

    if (error.response) {
      const status = error.response.status
      switch (status) {
        case 401:
          {
            const userStore = useUserStore()
            userStore.logout()
            message.error('登录已过期，请重新登录')
            router.push('/login')
          }
          break
        case 403:
          message.error('没有访问权限')
          break
        case 404:
          message.error('请求的资源不存在')
          break
        case 500:
          message.error('服务器内部错误')
          break
        default:
          message.error(error.response.data?.message || `请求失败 (${status})`)
      }
    } else if (error.request) {
      message.error('网络错误，请检查网络连接')
    } else {
      message.error(error.message || '请求失败')
    }

    return Promise.reject(error)
  }
)

// 通用请求方法
export function request<T = any>(config: AxiosRequestConfig): Promise<T> {
  // axios 1.20 的 request 返回条件类型 AxiosResponseResult，与 Promise<T> 不直接相容；
  // 响应拦截器已统一解包为业务 data，这里按拦截器后的真实形态断言
  return service.request<any, T>(config) as Promise<T>
}

// 座席中心专用实例（agentcenter 8081）：不走 /api baseURL，直接命中 vite 的 /agentapi 代理
const agentService: AxiosInstance = axios.create({
  baseURL: '/agentapi',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json;charset=utf-8',
  },
})

agentService.interceptors.response.use(
  (resp: AxiosResponse) => {
    NProgress.done()
    const res = resp.data
    // 业务成功直接返回 data，与主实例行为一致
    if (res && typeof res === 'object' && 'code' in res) {
      if (res.code === 0) return res.data
      if (res.code === 40101 || res.code === 40100 || res.code === 401) {
        return Promise.reject(new Error(res.message || '座席鉴权失败'))
      }
      return Promise.reject(new Error(res.message || '请求失败'))
    }
    return res
  },
  (error) => Promise.reject(error),
)

export { agentService }
export default service
