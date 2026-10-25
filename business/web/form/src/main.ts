// VocaraForm SDK 入口：三个挂载入口各自创建独立 Vue 应用，
// 宿主只需提供容器与配置，卸载调用返回值的 unmount()
import { createApp, type App } from 'vue'
import { naivePlugin } from './naive'
import DesignerApp from './designer/DesignerApp.vue'
import RendererApp from './renderer/RendererApp.vue'
import RecordsApp from './records/RecordsApp.vue'

export interface MountOptions {
  apiBase: string
  getToken: () => string
  templateId?: number
  templateKey?: string
  readonly?: boolean
  data?: Record<string, any>
  schema?: string
  // 实例列表专用：首张卡片左上角直角，与宿主页签下缘拼接
  joined?: boolean
  onSave?: (payload: any) => void
  onSubmit?: (data: Record<string, any>) => Promise<void>
}

export interface MountHandle {
  unmount: () => void
  [key: string]: any
}

function mount(component: any, el: HTMLElement, props: Record<string,any>): { app: App; vm: any } {
  const app = createApp(component, props)
  // 每个挂载入口都是独立 Vue 应用，naive-ui 组件需要在各自应用内注册
  app.use(naivePlugin)
  const vm = app.mount(el)
  return { app, vm }
}

export function mountDesigner(el: HTMLElement, opts: MountOptions): MountHandle {
  const { app, vm } = mount(DesignerApp, el, opts)
  return {
    unmount: () => app.unmount(),
    getSchema: () => (vm as any).getSchema?.(),
  }
}

export function mountRenderer(el: HTMLElement, opts: MountOptions): MountHandle {
  const { app, vm } = mount(RendererApp, el, opts)
  return {
    unmount: () => app.unmount(),
    validate: () => (vm as any).validate?.(),
    getData: () => (vm as any).getData?.(),
  }
}

export function mountRecords(el: HTMLElement, opts: MountOptions): MountHandle {
  const { app } = mount(RecordsApp, el, opts)
  return { unmount: () => app.unmount() }
}

// 挂到全局，供 script 标签引入的宿主使用
;(window as any).VocaraForm = { mountDesigner, mountRenderer, mountRecords }
