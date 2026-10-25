<template>
  <div class="grid-container" @click.stop>
    <div class="grid-head">
      <span class="grid-title">▦ 栅格（{{ node.columns.length }} 列）</span>
      <button class="action-btn danger" title="删除整个栅格" @click.stop="emit('remove')">✕</button>
    </div>

    <div class="grid-columns">
      <div v-for="(col, i) in node.columns" :key="i" class="grid-col">
        <div class="col-head">
          <span class="col-title">列{{ i + 1 }}</span>
          <n-slider
            :value="col.span"
            @update:value="(v: number) => (col.span = v)"
            :min="4" :max="24" :step="2" style="width: 80px"
          />
          <button
            class="action-btn danger" title="删列"
            :disabled="node.columns.length <= 2"
            @click.stop="removeColumn(i)"
          >✕</button>
        </div>

        <VueDraggable
          :model-value="col.fields"
          @update:model-value="col.fields = $event"
          :group="{ name: 'vocara-form-designer' }"
          item-key="id"
          class="col-drop"
          :put="putOnlyField"
        >
          <FieldNode
            v-for="child in col.fields"
            :key="child.id"
            :node="child"
            :selected-id="selectedId"
            :preview-data="previewData"
            placement="column"
            @select="(id) => emit('select', id)"
            @remove="(childNode) => removeChild(childNode, i)"
            @duplicate="(childNode) => duplicateChild(childNode, i)"
          />
          <div v-if="!col.fields.length" class="col-hint">放入字段</div>
        </VueDraggable>
      </div>
    </div>

    <button
      class="add-col"
      :disabled="node.columns.length >= 4"
      @click.stop="addColumn"
    >＋加列</button>
  </div>
</template>

<script setup lang="ts">
// GridContainer 栅格容器（设计态）
import { VueDraggable } from 'vue-draggable-plus'
import {
  isGridNode,
  type FormField,
  type FormNode,
  type GridContainer as GridNode,
} from '../schema/types'
import FieldNode from './FieldNode.vue'

const props = defineProps<{
  node: GridNode
  selectedId: string
  previewData: Record<string, any>
}>()
const emit = defineEmits<{
  select: [id: string]
  remove: [] // 删除整个栅格
}>()

let copySeq = 1

const addColumn = () => props.node.columns.push({ span: 8, fields: [] })

// 删列：列数下限 2；列内字段并入前一列，避免丢失
const removeColumn = (i: number) => {
  if (props.node.columns.length <= 2) return
  const orphan = props.node.columns[i].fields
  props.node.columns.splice(i, 1)
  const target = props.node.columns[Math.max(0, i - 1)]
  target.fields.push(...orphan)
}

// 删除列内字段
const removeChild = (childNode: FormNode, colIndex: number) => {
  const col = props.node.columns[colIndex]
  col.fields = col.fields.filter((f) => f.id !== childNode.id)
}

// 复制列内字段：深拷贝并赋新 id/name/label，落在同列
const duplicateChild = (childNode: FormNode, colIndex: number) => {
  if (isGridNode(childNode)) return
  const source: FormField = childNode
  const copy: FormField = JSON.parse(JSON.stringify(source))
  copy.id = `f_${Date.now().toString(36)}_copy${copySeq++}`
  copy.name = `${source.name}_copy`
  copy.label = `${source.label} 副本`
  props.node.columns[colIndex].fields.push(copy)
}

// 列放置区拒绝栅格节点（禁止栅格嵌套）
const putOnlyField = (evt: any) => {
  const dragged = evt?.draggedContext?.element as FormNode
  return !!dragged && !isGridNode(dragged)
}
</script>

<style scoped>
.grid-container {
  border: 1px solid #b7e4c7; border-radius: 8px; background: #f4fbf6; padding: 8px;
}
.grid-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.grid-title { font-size: 13px; font-weight: 600; color: #18a058; }
.grid-columns { display: flex; gap: 8px; }
.grid-col {
  flex: 1; min-width: 0; border: 1px dashed #cfe8d8; border-radius: 6px; padding: 6px; background: #fff;
}
.col-head { display: flex; align-items: center; gap: 6px; margin-bottom: 6px; }
.col-title { font-size: 12px; color: #6b7280; }
.action-btn { border: none; background: none; cursor: pointer; color: #6b7280; font-size: 12px; padding: 2px 4px; }
.action-btn:disabled { color: #d1d5db; cursor: not-allowed; }
.action-btn.danger:hover { color: #d03050; }
.col-drop { min-height: 80px; display: flex; flex-direction: column; gap: 6px; }
.col-hint { color: #9ca3af; font-size: 12px; text-align: center; padding: 20px 0; }
.add-col {
  margin-top: 8px; width: 100%; border: 1px dashed #18a058; background: none;
  color: #18a058; border-radius: 6px; padding: 4px; cursor: pointer; font-size: 12px;
}
.add-col:disabled { color: #d1d5db; border-color: #d1d5db; cursor: not-allowed; }
</style>
