<template>
  <div class="widget-panel">
    <div v-for="group in WIDGET_GROUPS" :key="group.key" class="widget-group">
      <div class="group-title">{{ group.title }}</div>
      <VueDraggable
        :model-value="widgetsOf(group.key)"
        :group="{ name: 'vocara-form-designer', pull: 'clone', put: false }"
        :sort="false"
        :clone="cloneWidget"
        item-key="type"
        class="widget-list"
      >
        <div v-for="widget in widgetsOf(group.key)" :key="widget.type" class="widget-item" @dblclick="emit('add', widget.type)">
          <span class="widget-icon">{{ widget.icon }}</span>
          <span class="widget-title">{{ widget.title }}</span>
        </div>
      </VueDraggable>
    </div>
  </div>
</template>

<script setup lang="ts">
import { VueDraggable } from 'vue-draggable-plus'
import { WIDGET_DEFS, WIDGET_GROUPS, createField } from '../schema/widgets'

// 双击添加由 CanvasPanel 监听（组件面板与画布分离，双击事件经 emit 上抛）
const emit = defineEmits<{ add: [type: string] }>()

const widgetsOf = (group: string) => WIDGET_DEFS.filter((w) => w.group === group)

// clone 钩子在拖拽落点生成全新字段（拖出的是字段实例，不是类型定义）
const cloneWidget = (widget: any) => createField(widget.type)
</script>

<style scoped>
.widget-panel { height: 100%; overflow-y: auto; background: #fff; border-radius: 10px; padding: 12px; box-shadow: 0 1px 4px rgba(15, 40, 25, 0.06); }
.group-title { font-size: 12px; color: #9ca3af; margin: 8px 0 6px; }
.widget-list { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; min-height: 40px; }
.widget-item {
  /* 上下两行：图标在上、文字在下，避免横排时长文字被挤压换行 */
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 4px; padding: 8px 4px; color: #374151; text-align: center;
  background: #f9fafb; border: 1px solid #e5e7eb; border-radius: 6px; cursor: grab; user-select: none;
}
.widget-item:hover { border-color: #18a058; color: #18a058; }
.widget-icon { font-size: 20px; line-height: 1; }
.widget-title {
  font-size: 12px; line-height: 1.2; max-width: 100%;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
</style>
