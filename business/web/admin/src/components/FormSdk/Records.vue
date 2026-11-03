<template>
  <div ref="el" class="form-sdk-mount"></div>
</template>

<script setup lang="ts">
// VocaraForm SDK 实例管理挂载壳：SDK 为独立构建的自包含应用（web/form），
// 这里只负责注册容器、透传 apiBase/token、卸载时 unmount
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useUserStore } from '@/store/modules/user'

interface Props {
  templateKey?: string
}

const props = defineProps<Props>()

const userStore = useUserStore()
const el = ref<HTMLElement | null>(null)
let handle: { unmount: () => void } | null = null

onMounted(() => {
  handle = (window as any).VocaraForm.mountRecords(el.value, {
    apiBase: '/api/v1/admin/forms',
    getToken: () => userStore.token,
    templateKey: props.templateKey,
    joined: true,
  })
})

onBeforeUnmount(() => {
  handle?.unmount()
  handle = null
})
</script>

<style scoped>
.form-sdk-mount { height: 100%; }
</style>
