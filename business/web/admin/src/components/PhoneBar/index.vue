<template>
  <!-- VocaraPhoneBar SDK 挂载容器：SDK 自带 UI 与样式（/phonebar/vocara-phonebar.css） -->
  <div ref="pbRef" class="phonebar-sdk-container"></div>
</template>

<script setup lang="ts">
// 电话条挂载壳：仅向 call store 注册容器，SDK init/connect 由 store 在签入时发起
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useCallStore } from '@/store/modules/call'

const callStore = useCallStore()
const pbRef = ref<HTMLElement | null>(null)

onMounted(() => {
  callStore.setPhoneBarContainer(pbRef.value)
})

onBeforeUnmount(() => {
  // 仅注销容器引用；SDK 实例为全局单例，签出/退出登录时由 store 统一销毁
  callStore.setPhoneBarContainer(null)
})
</script>

<style scoped>
.phonebar-sdk-container {
  width: 100%;
}
</style>
