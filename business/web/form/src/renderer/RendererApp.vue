<template>
  <div class="vocara-form-renderer">
    <n-spin :show="loading">
      <n-form
        ref="formRef"
        :model="formData"
        :label-width="schemaObj.formConfig.labelWidth ?? 90"
        :label-placement="schemaObj.formConfig.labelPlacement ?? 'left'"
        :size="schemaObj.formConfig.size ?? 'medium'"
        :disabled="readonly"
      >
        <n-grid :cols="24" :x-gap="16">
          <!-- NGrid 直接子节点必须是 form-item-gi，不能包中间组件 -->
          <template v-for="node in schemaObj.fields" :key="node.id">
            <!-- 普通字段 -->
            <n-form-item-gi
              v-if="!isGridNode(node)"
              :span="node.span || 24"
              :label="node.label"
              :path="node.name"
              :rule="buildNaiveRules(node)"
            >
              <FieldWidget :field="node" v-model="formData[node.name]" />
            </n-form-item-gi>

            <!-- 栅格节点：整行容器内嵌独立网格 -->
            <n-form-item-gi v-else :span="24" :show-label="false">
              <n-grid :cols="24" :x-gap="12">
                <n-gi v-for="(col, i) in node.columns" :key="i" :span="col.span">
                  <template v-for="child in col.fields" :key="child.id">
                    <n-form-item
                      v-if="!isGridNode(child)"
                      :label="child.label"
                      :path="child.name"
                      :rule="buildNaiveRules(child)"
                    >
                      <FieldWidget :field="child" v-model="formData[child.name]" />
                    </n-form-item>
                  </template>
                </n-gi>
              </n-grid>
            </n-form-item-gi>
          </template>
        </n-grid>
      </n-form>

      <!-- 提交按钮（传了 onSubmit 才显示） -->
      <div v-if="!readonly && onSubmit" class="renderer-actions">
        <n-button type="primary" :loading="submitting" @click="handleSubmit">提交</n-button>
      </div>
    </n-spin>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue'
import type { FormInst } from 'naive-ui'
import { message } from '../deps'
import { createFormApi } from '../api'
import { LAYOUT_FIELD_TYPES, isGridNode, parseSchema, type FormSchema } from '../schema/types'
import { collectFields } from '../schema/walk'
import { buildNaiveRules } from '../schema/rules'
import FieldWidget from './FieldWidget.vue'

interface Props {
  apiBase: string
  getToken: () => string
  schema?: string
  templateId?: number
  readonly?: boolean
  data?: Record<string, any>
  onSubmit?: (data: Record<string, any>) => Promise<void>
}

const props = withDefaults(defineProps<Props>(), { readonly: false })

const loading = ref(false)
const submitting = ref(false)
const formRef = ref<FormInst>()
// 同步 schema 在 setup 顶层即解析为初始值，首帧直接渲染字段
const schemaObj = ref<FormSchema>(
  props.schema
    ? (() => { try { return parseSchema(props.schema) } catch { return { version: 1, formConfig: {}, fields: [] } } })()
    : { version: 1, formConfig: {}, fields: [] },
)
// 数据模型：schema 加载后按字段初始化（checkbox/多选 select 默认数组）
const formData = reactive<Record<string, any>>({})

const isLayout = (type: string) => LAYOUT_FIELD_TYPES.includes(type)

// 初始化/回填：按拍平字段铺默认值，再覆盖外部 data
function initFormData(data?: Record<string, any>) {
  collectFields(schemaObj.value.fields).forEach((f) => {
    if (isLayout(f.type)) return
    if (data && data[f.name] !== undefined) {
      formData[f.name] = data[f.name]
    } else if (f.defaultValue !== undefined && f.defaultValue !== null && f.defaultValue !== '') {
      formData[f.name] = f.defaultValue
    } else if (f.type === 'checkbox' || (f.type === 'select' && f.props?.multiple)) {
      formData[f.name] = []
    } else if (f.type === 'number' || f.type === 'rate' || f.type === 'date') {
      formData[f.name] = null
    } else if (f.type === 'switch') {
      formData[f.name] = false
    } else {
      formData[f.name] = ''
    }
  })
}

onMounted(async () => {
  if (props.schema) {
    try {
      schemaObj.value = parseSchema(props.schema)
    } catch (e: any) {
      // schema 损坏时保持空 schema，页面呈现空表单空态，不阻断挂载
      message.error(e?.message || '表单 schema 解析失败')
    }
  } else if (props.templateId) {
    loading.value = true
    try {
      const api = createFormApi({ apiBase: props.apiBase, getToken: props.getToken })
      const tpl = await api.templateGet(props.templateId)
      schemaObj.value = parseSchema(tpl.schema)
    } catch (e: any) {
      message.error(e?.message || '加载表单模板失败')
    } finally {
      loading.value = false
    }
  }
  initFormData(props.data)
})

async function validate(): Promise<boolean> {
  try {
    await formRef.value?.validate()
    return true
  } catch {
    return false
  }
}

function getData(): Record<string, any> {
  // 只回传 schema 中声明的数据字段（布局字段与外部多塞的键都不带出）
  const data: Record<string, any> = {}
  collectFields(schemaObj.value.fields).forEach((f) => {
    if (!isLayout(f.type)) data[f.name] = formData[f.name]
  })
  return data
}

async function handleSubmit() {
  if (!(await validate())) return
  submitting.value = true
  try {
    await props.onSubmit?.(getData())
  } finally {
    submitting.value = false
  }
}

defineExpose({ validate, getData })
</script>

<style scoped>
.vocara-form-renderer { padding: 8px; }
.renderer-actions { margin-top: 16px; text-align: center; }
</style>
