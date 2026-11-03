import { createApp } from 'vue'
import { createPinia } from 'pinia'
import naive from 'naive-ui'
import App from './App.vue'
import router from './router'
import { setupNaiveDiscreteApi } from './utils/naive'

import 'nprogress/nprogress.css'
import './styles/global.css'

const app = createApp(App)

app.use(createPinia())
app.use(router)

// 全局注册 Naive UI 组件（各页面直接使用 n-* 标签，无需逐页 import）
app.use(naive)

// 初始化 Naive UI 全局 API（Message/Dialog/Notification/LoadingBar）
setupNaiveDiscreteApi()

app.mount('#app')
