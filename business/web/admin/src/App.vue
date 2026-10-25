<template>
  <n-config-provider
    :theme-overrides="themeOverrides"
    :locale="zhCN"
    :date-locale="dateZhCN"
  >
    <n-loading-bar-provider>
      <n-dialog-provider>
        <n-notification-provider>
          <n-message-provider>
            <!-- 顶层 router-view 不加 key：布局组件跨页面切换必须保持挂载，
                 否则电话条容器随布局重建、SDK 连接被销毁导致座席被签出；
                 页面级刷新由布局内部 router-view 的 key 负责 -->
            <router-view v-slot="{ Component }">
              <component :is="Component" />
            </router-view>
          </n-message-provider>
        </n-notification-provider>
      </n-dialog-provider>
    </n-loading-bar-provider>
  </n-config-provider>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  zhCN,
  dateZhCN,
  NConfigProvider,
  NLoadingBarProvider,
  NDialogProvider,
  NNotificationProvider,
  NMessageProvider,
} from 'naive-ui'
import { useThemeStore } from '@/store/modules/theme'

const themeStore = useThemeStore()

// 只覆盖主色和几个关键变量，其他用 Naive UI 浅色主题默认值
const themeOverrides = computed(() => ({
  common: {
    primaryColor: '#18a058',
    primaryColorHover: '#36ad6a',
    primaryColorPressed: '#0c7a43',
    primaryColorSuppl: '#36ad6a',
    borderRadius: '8px',
  },
}))
</script>

<style>


</style>
