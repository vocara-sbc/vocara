<template>
  <!-- 栅格节点：交给栅格容器，顶层放置时占满整行 -->
  <GridContainer
    v-if="isGrid(node)"
    :node="node"
    :selected-id="selectedId"
    :preview-data="previewData"
    style="width: calc(100% - 4px)"
    @select="(id) => emit('select', id)"
    @remove="() => emit('remove', node)"
  />

  <!-- 普通字段卡：宽度按放置上下文（顶层流式取 span，列内占满） -->
  <div
    v-else
    class="field-node"
    :class="{ active: node.id === selectedId }"
    :style="{ width: `calc(${widthPct}% - 4px)` }"
    @click.stop="emit('select', node.id)"
  >
    <div class="item-head">
      <span class="drag-handle" title="拖拽排序">⠿</span>
      <span class="item-label">{{ node.label }}</span>
      <span class="item-type">{{ typeTitle(node.type) }}</span>
      <span class="item-name">{{ node.name }}</span>
      <span class="item-actions">
        <button class="action-btn" title="复制" @click.stop="emit('duplicate', node)">⧉</button>
        <button class="action-btn danger" title="删除" @click.stop="emit('remove', node)">✕</button>
      </span>
    </div>
    <!-- 控件预览：与渲染器同一组件 -->
    <div class="item-preview">
      <FieldWidget :field="node" v-model="previewData[node.name]" />
    </div>
  </div>
</template>

<script setup lang="ts">
// FieldNode 设计器递归节点
import { computed } from 'vue'
import { isGridNode, type FormNode } from '../schema/types'
import { WIDGET_MAP } from '../schema/widgets'
import FieldWidget from '../renderer/FieldWidget.vue'
import GridContainer from './GridContainer.vue'

// placement：flow-顶层流式放置（字段宽度取 span）；column-栅格列内（占满列宽）
const props = defineProps<{
  node: FormNode
  selectedId: string
  previewData: Record<string, any>
  placement?: 'flow' | 'column'
}>()
const emit = defineEmits<{
  select: [id: string]
  remove: [node: FormNode]
  duplicate: [node: FormNode]
}>()

const isGrid = isGridNode
const typeTitle = (type: string) => WIDGET_MAP[type]?.title || type

// 字段宽度百分比：列内占满；顶层流式按字段 span（缺省整行）
const widthPct = computed(() => {
  if (props.placement === 'column') return 100
  const span = props.node.type === 'grid' ? 24 : ((props.node as any).span || 24)
  return (Math.min(24, span) / 24 * 100).toFixed(2)
})
</script>

<style scoped>
.field-node {
  padding: 8px 10px; font-size: 13px;
  background: #f9fafb; border: 1px dashed #d1d5db; border-radius: 8px; cursor: pointer;
}
.field-node.active { border: 1px solid #18a058; background: #f6fef9; }
.item-head { display: flex; align-items: center; gap: 8px; }
.drag-handle { cursor: grab; color: #9ca3af; }
.item-label { font-weight: 500; color: #1f2937; }
.item-type { color: #18a058; font-size: 12px; }
.item-name { color: #9ca3af; font-size: 12px; font-family: monospace; }
.item-actions { margin-left: auto; display: flex; gap: 4px; }
.action-btn { border: none; background: none; cursor: pointer; color: #6b7280; font-size: 13px; padding: 2px 4px; }
.action-btn:hover { color: #18a058; }
.action-btn.danger:hover { color: #d03050; }
.item-preview { margin-top: 8px; cursor: default; }
</style>
