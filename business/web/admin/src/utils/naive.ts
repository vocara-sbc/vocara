import {
  createDiscreteApi,
  lightTheme,
} from 'naive-ui'
import { ref } from 'vue'

// Naive UI 全局 API 实例
// 以便在非组件环境（如 axios 拦截器、store）中使用 message/dialog 等

let message: any = null
let dialog: any = null
let notification: any = null
let loadingBar: any = null

export function setupNaiveDiscreteApi() {
  const configProviderProps = ref({
    theme: lightTheme,
    themeOverrides: {
      common: {
        primaryColor: '#18a058',
        primaryColorHover: '#36ad6a',
        primaryColorPressed: '#0c7a43',
      },
    },
  })

  const {
    message: msg,
    dialog: dlg,
    notification: notif,
    loadingBar: lb,
  } = createDiscreteApi(['message', 'dialog', 'notification', 'loadingBar'], {
    configProviderProps,
  })

  message = msg
  dialog = dlg
  notification = notif
  loadingBar = lb
}

export { message, dialog, notification, loadingBar }
