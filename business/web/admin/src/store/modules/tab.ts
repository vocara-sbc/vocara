import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { RouteLocationNormalized } from 'vue-router'

export interface TabItem {
  path: string
  title: string
  affix?: boolean
  icon?: string
}

export const useTabStore = defineStore('tab', () => {
  // 初始为空：标签页全部由实际访问产生，不预置固定页
  const tabs = ref<TabItem[]>([])
  const activeTab = ref<string>('')

  const tabList = computed(() => tabs.value)

  function addTab(route: RouteLocationNormalized) {
    if (tabs.value.every((t) => t.path !== route.path)) {
      const title = (route.meta?.title as string) || route.name as string
      tabs.value.push({
        path: route.path,
        title,
      })
    }
    activeTab.value = route.path
  }

  function removeTab(path: string) {
    const idx = tabs.value.findIndex((t) => t.path === path)
    if (idx > -1) {
      if (tabs.value[idx].affix) return
      tabs.value.splice(idx, 1)
      if (activeTab.value === path) {
        // 跳转到前一个或后一个标签
        const nextTab = tabs.value[idx] || tabs.value[idx - 1]
        if (nextTab) {
          activeTab.value = nextTab.path
        }
      }
    }
  }

  function setActiveTab(path: string) {
    activeTab.value = path
  }

  function closeOtherTabs(path: string) {
    tabs.value = tabs.value.filter((t) => t.path === path || t.affix)
    if (!tabs.value.some((t) => t.path === activeTab.value)) {
      activeTab.value = path
    }
  }

  function closeAllTabs() {
    tabs.value = tabs.value.filter((t) => t.affix)
    if (tabs.value.length > 0) {
      activeTab.value = tabs.value[0].path
    }
  }

  // 退出登录时清空标签页：标签属会话级状态，不得在跨账号间残留
  function reset() {
    tabs.value = []
    activeTab.value = ''
  }

  return {
    tabs,
    activeTab,
    tabList,
    addTab,
    removeTab,
    setActiveTab,
    closeOtherTabs,
    closeAllTabs,
    reset,
  }
})
