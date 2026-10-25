import { defineStore } from 'pinia'
import { ref } from 'vue'

// 界面级 UI 状态：承载不属于具体页面的全局弹窗
export const useUiStore = defineStore('ui', () => {
  // 修改密码弹窗：Header 下拉菜单与个人中心页共用同一实例
  const passwordModalVisible = ref(false)

  function openPasswordModal() {
    passwordModalVisible.value = true
  }

  function closePasswordModal() {
    passwordModalVisible.value = false
  }

  return {
    passwordModalVisible,
    openPasswordModal,
    closePasswordModal,
  }
})
