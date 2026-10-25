// 离散 message/dialog：SDK 可同时挂载多个实例，不依赖宿主的全局 provider。
// discrete api 延迟到首次调用时创建：脚本可在 <head> 中加载，
// 此时 document.body 尚未就绪，createDiscreteApi 挂载独立应用需要 body 节点
import { createDiscreteApi, type MessageApi, type DialogApi } from 'naive-ui'

type DiscreteApi = { message: MessageApi; dialog: DialogApi }

let discrete: DiscreteApi | null = null

function getDiscrete(): DiscreteApi {
  if (!discrete) {
    discrete = createDiscreteApi(['message', 'dialog'])
  }
  return discrete
}

// 以同名接口代理转发，首次方法调用触发 discrete api 创建；
// 方法绑定真实 api 为 this，调用方代码与直接使用 createDiscreteApi 一致
function lazyApi<T extends object>(pick: (api: DiscreteApi) => T): T {
  return new Proxy({}, {
    get(_target, prop) {
      const target = pick(getDiscrete()) as Record<PropertyKey, unknown>
      const value = target[prop]
      return typeof value === 'function' ? value.bind(target) : value
    },
  }) as T
}

export const message = lazyApi((api) => api.message)
export const dialog = lazyApi((api) => api.dialog)
