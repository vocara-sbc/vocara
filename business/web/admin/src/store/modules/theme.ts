import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const collapsed = ref<boolean>(false) // 侧边栏折叠状态
  const primaryColor = ref<string>('#18a058') // 主题色（绿色）

  function toggleCollapsed() {
    collapsed.value = !collapsed.value
  }

  function restoreSettings() {
    const color = localStorage.getItem('vocara_primary_color')
    if (color) {
      primaryColor.value = color
    }
  }

  return {
    collapsed,
    primaryColor,
    toggleCollapsed,
    restoreSettings,
  }
})
