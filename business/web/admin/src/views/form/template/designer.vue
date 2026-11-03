<template>
  <div class="form-designer-page">
    <FormSdkDesigner :template-id="templateId" @save="handleSave" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from '@/utils/naive'
import FormSdkDesigner from '@/components/FormSdk/Designer.vue'
import { createFormTemplate, updateFormTemplate } from '@/api/form'

const route = useRoute()
const router = useRouter()
const templateId = computed(() => Number(route.query.id) || undefined)

// SDK 抛出保存意图，admin 统一走自己的请求封装（错误提示/loading 语义与全站一致）
async function handleSave(payload: any) {
  try {
    if (templateId.value) await updateFormTemplate({ ...payload, id: templateId.value })
    else await createFormTemplate(payload)
    message.success('保存成功')
    router.push('/system/form-template')
  } catch {
    // 错误提示由请求拦截器统一弹出
  }
}
</script>

<style scoped>
.form-designer-page { height: 100%; }
</style>
