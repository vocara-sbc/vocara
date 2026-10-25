<template>
  <div ref="el" class="form-sdk-mount"></div>
</template>

<script setup lang="ts">
// VocaraForm SDK 渲染器挂载壳：SDK 为独立构建的自包含应用（web/form），
// 透传 schema/data 与只读标记，向父组件暴露 validate/getData
import { ref, onMounted, onBeforeUnmount } from 'vue'

interface Props {
  apiBase?: string
  getToken?: () => string
  schema?: string
  data?: Record<string, any>
  readonly?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  apiBase: '',
  getToken: () => '',
  schema: '',
  readonly: false,
})

const el = ref<HTMLElement | null>(null)
let handle: {
  unmount: () => void
  validate?: () => boolean | Promise<boolean>
  getData?: () => Record<string, any>
} | null = null

onMounted(() => {
  handle = (window as any).VocaraForm.mountRenderer(el.value, {
    apiBase: props.apiBase,
    getToken: props.getToken,
    schema: props.schema,
    data: props.data,
    readonly: props.readonly,
  })
})

onBeforeUnmount(() => {
  handle?.unmount()
  handle = null
})

// validate 校验全表，返回 false 时字段下方展示校验信息
function validate() {
  return handle?.validate?.()
}

// getData 返回全量字段值
function getData() {
  return handle?.getData?.()
}

defineExpose({ validate, getData })
</script>

<style scoped>
.form-sdk-mount { height: 100%; }
</style>
