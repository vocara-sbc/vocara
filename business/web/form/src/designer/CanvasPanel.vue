<template>
  <div class="canvas-panel">
    <VueDraggable
      :model-value="fields"
      @update:model-value="emit('update:fields', $event)"
      :group="{ name: 'vocara-form-designer', pull: true, put: true }"
      :animation="200"
      handle=".drag-handle"
      item-key="id"
      class="canvas-list"
      @add="onAdd"
    >
      <FieldNode
        v-for="node in fields"
        :key="node.id"
        :node="node"
        :selected-id="selectedId"
        :preview-data="previewData"
        placement="flow"
        @select="(id) => emit('update:selectedId', id)"
        @remove="handleRemoveNode"
        @duplicate="handleDuplicateNode"
      />
    </VueDraggable>
    <n-empty v-if="!fields.length" description="从左侧拖入字段，或双击字段添加" class="canvas-empty" />
  </div>
</template>

<script setup lang="ts">
import { nextTick, reactive, watch } from 'vue'
import { VueDraggable } from 'vue-draggable-plus'
import {
  isGridNode,
  type FormField,
  type FormNode,
} from '../schema/types'
import { collectFields } from '../schema/walk'
import FieldNode from './FieldNode.vue'

const props = defineProps<{ fields: FormNode[]; selectedId: string }>()
const emit = defineEmits<{
  'update:fields': [fields: FormNode[]]
  'update:selectedId': [id: string]
}>()

// 画布预览数据：只服务设计器内控件显示与交互，不写入模板 schema
const previewData = reactive<Record<string, any>>({})

// 新字段按类型铺初始值，与渲染器数据初始化规则一致
function initPreviewValue(field: FormField) {
  if (field.defaultValue !== undefined && field.defaultValue !== null && field.defaultValue !== '') {
    previewData[field.name] = field.defaultValue
  } else if (field.type === 'checkbox' || (field.type === 'select' && field.props?.multiple)) {
    previewData[field.name] = []
  } else if (field.type === 'number' || field.type === 'rate' || field.type === 'date') {
    previewData[field.name] = null
  } else if (field.type === 'switch') {
    previewData[field.name] = false
  } else {
    previewData[field.name] = ''
  }
}

// 拍平字段名集合变化（新增/复制/删列）时补齐预览值
watch(
  () => collectFields(props.fields).map((f) => f.name).join(','),
  () => collectFields(props.fields).forEach((f) => {
    if (!(f.name in previewData)) initPreviewValue(f)
  }),
  { immediate: true },
)

// 删除顶层节点（字段或整个栅格）
function handleRemoveNode(node: FormNode) {
  emit('update:fields', props.fields.filter((f) => f.id !== node.id))
}

let copySeq = 1

// 复制顶层字段节点（栅格节点无复制入口）
function handleDuplicateNode(node: FormNode) {
  if (isGridNode(node)) return
  const source: FormField = node
  const copy: FormField = JSON.parse(JSON.stringify(source))
  copy.id = `f_${Date.now().toString(36)}_copy${copySeq++}`
  copy.name = `${source.name}_copy`
  copy.label = `${source.label} 副本`
  const index = props.fields.findIndex((f) => f.id === node.id)
  const next = [...props.fields]
  next.splice(index + 1, 0, copy)
  emit('update:fields', next)
}

// 外部拖入（来自面板 clone）：onAdd 同步触发时 props.fields 尚未更新，
// 新节点 id 从事件携带的克隆数据（clonedData）读取；缺失时 nextTick 后按落点读取
function onAdd(evt: any) {
  const clonedId = evt?.clonedData?.id
  if (clonedId) {
    emit('update:selectedId', clonedId)
    return
  }
  nextTick(() => {
    const node = props.fields[evt.newIndex]
    if (node) emit('update:selectedId', node.id)
  })
}
</script>

<style scoped>
.canvas-panel { height: 100%; overflow-y: auto; background: #fff; border-radius: 10px; padding: 12px; box-shadow: 0 1px 4px rgba(15, 40, 25, 0.06); position: relative; }
.canvas-list { display: flex; flex-wrap: wrap; gap: 8px; min-height: 200px; align-content: flex-start; }
.canvas-empty { position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; pointer-events: none; }
</style>
