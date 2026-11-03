<template>
  <div ref="el" class="form-sdk-mount"></div>
</template>

<script setup lang="ts">
// VocaraForm SDK 设计器挂载壳：SDK 为独立构建的自包含应用（web/form），
// 这里只负责注册容器、透传 apiBase/token、卸载时 unmount
import { ref, onMounted, onBeforeUnmount } from 'vue'
import { useUserStore } from '@/store/modules/user'

interface Props {
  templateId?: number
  onSave?: (payload: any) => void
}

const props = defineProps<Props>()
const emit = defineEmits<{ save: [payload: any] }>()

const userStore = useUserStore()
const el = ref<HTMLElement | null>(null)
let handle: { unmount: () => void } | null = null

onMounted(() => {
  handle = (window as any).VocaraForm.mountDesigner(el.value, {
    apiBase: '/api/v1/admin/forms',
    getToken: () => userStore.token,
    templateId: props.templateId,
    onSave: (payload: any) => emit('save', payload),
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
