<template>
  <div class="vocara-form-designer">
    <!-- 顶部工具栏 -->
    <div class="designer-toolbar">
      <div class="toolbar-left">
        <n-input v-model:value="meta.templateName" placeholder="模板名称" style="width: 200px" size="small" />
        <n-select v-model:value="meta.category" :options="categoryOptions" style="width: 120px" size="small" />
      </div>
      <div class="toolbar-right">
        <n-button size="small" @click="previewVisible = true" :disabled="!fields.length">预览</n-button>
        <n-button size="small" type="primary" :loading="saving" @click="handleSave">保存</n-button>
      </div>
    </div>

    <div class="designer-body">
      <WidgetPanel class="designer-left" @add="handleAdd" />
      <CanvasPanel
        class="designer-canvas"
        v-model:fields="fields"
        v-model:selectedId="selectedId"
      />
      <PropsPanel
        class="designer-right"
        :node="selectedNode"
        :in-column="selectedNode ? inColumn(selectedNode) : false"
        :form-config="formConfig"
      />
    </div>

    <!-- 预览弹窗：直接复用渲染器 -->
    <n-modal v-model:show="previewVisible" preset="card" title="表单预览" style="width: 720px">
      <RendererApp
        v-if="previewVisible"
        :api-base="apiBase"
        :get-token="getToken"
        :schema="JSON.stringify({ version: 1, formConfig, fields })"
      />
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, onMounted } from 'vue'
import { message } from '../deps'
import { createFormApi } from '../api'
import { parseSchema, isGridNode, type FormNode } from '../schema/types'
import { createField } from '../schema/widgets'
import WidgetPanel from './WidgetPanel.vue'
import CanvasPanel from './CanvasPanel.vue'
import PropsPanel from './PropsPanel.vue'
import RendererApp from '../renderer/RendererApp.vue'

interface Props {
  apiBase: string
  getToken: () => string
  templateId?: number
  onSave?: (payload: Record<string, any>) => void
}

const props = defineProps<Props>()

const fields = ref<FormNode[]>([])
const selectedId = ref('')
const formConfig = reactive({ labelWidth: 90, labelPlacement: 'left' as const, size: 'medium' as const })
const meta = reactive({ templateKey: '', templateName: '未命名表单', category: 'general', status: 1, remark: '' })
// 分类选项从后端动态加载（仅启用项）
const categoryOptions = ref<{ label: string; value: string }[]>([])
const previewVisible = ref(false)
const saving = ref(false)

const selectedNode = computed<FormNode | null>(() => findNode(fields.value, selectedId.value))

// findNode 在节点树（含栅格各列）中按 id 查找节点
function findNode(nodes: FormNode[], id: string): FormNode | null {
  for (const n of nodes) {
    if (n.id === id) return n
    if (isGridNode(n)) {
      for (const col of n.columns) {
        const hit = findNode(col.fields, id)
        if (hit) return hit
      }
    }
  }
  return null
}

// inColumn 判断节点是否位于某个栅格列内（顶层找到即非列内）
function inColumn(node: FormNode): boolean {
  return !fields.value.some((f) => f.id === node.id)
}

const api = createFormApi({ apiBase: props.apiBase, getToken: props.getToken })

onMounted(async () => {
  // 分类始终加载（仅启用项）
  try {
    const cats = await api.categoryList()
    categoryOptions.value = cats
      .filter((c) => c.enabled)
      .map((c) => ({ label: c.name, value: c.code }))
  } catch {
    categoryOptions.value = []
  }

  if (!props.templateId) return
  try {
    const tpl = await api.templateGet(props.templateId)
    meta.templateKey = tpl.template_key
    meta.templateName = tpl.template_name
    meta.category = tpl.category
    meta.status = tpl.status
    meta.remark = tpl.remark
    // 停用分类仍需回显当前值
    if (!categoryOptions.value.some((o) => o.value === tpl.category)) {
      categoryOptions.value.push({ label: tpl.category, value: tpl.category })
    }
    // schema 在版本表：取该模板当前版本
    const version = await api.templateSchemaByKey(tpl.template_key)
    const schema = parseSchema(version.schema)
    Object.assign(formConfig, schema.formConfig || {})
    fields.value = schema.fields
  } catch (e: any) {
    message.error(e?.message || '加载模板失败')
  }
})

// 组件面板双击添加（拖拽落点由 CanvasPanel 处理）
function handleAdd(type: string) {
  const field = createField(type)
  fields.value.push(field)
  selectedId.value = field.id
}

function getSchema(): string {
  return JSON.stringify({ version: 1, formConfig, fields: fields.value })
}

async function handleSave() {
  if (!meta.templateName.trim()) {
    message.warning('请填写模板名称')
    return
  }
  if (!fields.value.some((f) => !['divider', 'static-text'].includes(f.type))) {
    message.warning('表单至少需要一个数据字段')
    return
  }
  const payload = {
    id: props.templateId,
    template_key: meta.templateKey || `form_${Date.now().toString(36)}`,
    template_name: meta.templateName,
    category: meta.category,
    schema: getSchema(),
    status: meta.status,
    remark: meta.remark,
  }
  // 宿主接管保存（模板列表页自己调 API 并跳回）；未接管时 SDK 直连 apiBase
  if (props.onSave) {
    props.onSave(payload)
    return
  }
  saving.value = true
  try {
    if (props.templateId) await api.templateUpdate(payload)
    else await api.templateSave(payload)
    message.success('保存成功')
  } catch (e: any) {
    message.error(e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

defineExpose({ getSchema })
</script>

<style scoped>
.vocara-form-designer { height: 100%; display: flex; flex-direction: column; background: #ecf3ef; }
.designer-toolbar {
  height: 48px; flex-shrink: 0; display: flex; align-items: center; justify-content: space-between;
  padding: 0 12px; background: #fff; border-radius: 10px; margin: 12px 12px 0;
  box-shadow: 0 1px 4px rgba(15, 40, 25, 0.06);
}
.toolbar-left { display: flex; gap: 8px; }
.toolbar-right { display: flex; gap: 8px; }
.designer-body { flex: 1; min-height: 0; display: flex; gap: 12px; padding: 12px; }
.designer-left { width: 220px; flex-shrink: 0; }
.designer-canvas { flex: 1; min-width: 0; }
.designer-right { width: 300px; flex-shrink: 0; }
</style>
