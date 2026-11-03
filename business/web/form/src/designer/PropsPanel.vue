<template>
  <div class="props-panel">
    <!-- ===== 栅格节点属性 ===== -->
    <template v-if="node && isGridNode(node)">
      <div class="panel-title">栅格属性</div>
      <n-form label-placement="top" size="small">
        <n-form-item label="列数（加列/删列在画布操作）">
          <span>{{ node.columns.length }} 列</span>
        </n-form-item>
        <n-form-item v-for="(col, i) in node.columns" :key="i" :label="`列${i + 1} 列宽`">
          <n-slider v-model:value="col.span" :min="4" :max="24" :step="2" />
        </n-form-item>
      </n-form>
    </template>

    <!-- ===== 字段节点属性 ===== -->
    <template v-else-if="field">
      <div class="panel-title">字段属性</div>
      <n-form label-placement="top" size="small">
        <n-form-item label="字段标题">
          <n-input v-model:value="field.label" />
        </n-form-item>
        <n-form-item v-if="!isLayout" label="字段名（数据键）">
          <n-input v-model:value="field.name" placeholder="英文标识" @blur="normalizeName" />
        </n-form-item>
        <!-- 布局宽度仅顶层流式字段可配；列内字段占满列宽 -->
        <n-form-item v-if="!isLayout && !inColumn" label="布局宽度">
          <n-slider v-model:value="field.span" :min="4" :max="24" :step="1" :marks="{ 8: '1/3', 12: '1/2', 24: '整行' }" />
        </n-form-item>

        <!-- 选择类字段：选项编辑器 -->
        <n-form-item v-if="hasOptions" label="选项">
          <div class="options-editor">
            <div v-for="(opt, i) in field.props.options" :key="i" class="option-row">
              <n-input v-model:value="opt.label" placeholder="显示文本" size="small" />
              <n-input v-model:value="opt.value" placeholder="值" size="small" />
              <n-button size="tiny" quaternary type="error" @click="field.props.options.splice(i, 1)">✕</n-button>
            </div>
            <n-button size="small" dashed block @click="field.props.options.push({ label: `选项${field.props.options.length + 1}`, value: String(field.props.options.length + 1) })">添加选项</n-button>
          </div>
        </n-form-item>
        <n-form-item v-if="field.type === 'select'" label="多选">
          <n-switch v-model:value="field.props.multiple" />
        </n-form-item>
        <n-form-item v-if="field.type === 'select'" label="可搜索">
          <n-switch v-model:value="field.props.filterable" />
        </n-form-item>

        <!-- 类型特有 -->
        <n-form-item v-if="['input', 'textarea', 'select'].includes(field.type)" label="占位提示">
          <n-input v-model:value="field.props.placeholder" />
        </n-form-item>
        <n-form-item v-if="field.type === 'textarea'" label="行数">
          <n-input-number v-model:value="field.props.rows" :min="2" :max="10" style="width: 100%" />
        </n-form-item>
        <n-form-item v-if="field.type === 'number'" label="最小值 / 最大值">
          <div style="display: flex; gap: 8px">
            <n-input-number v-model:value="field.props.min" placeholder="最小" style="flex: 1" />
            <n-input-number v-model:value="field.props.max" placeholder="最大" style="flex: 1" />
          </div>
        </n-form-item>
        <n-form-item v-if="field.type === 'date'" label="日期格式">
          <n-select v-model:value="field.props.format" :options="[
            { label: 'yyyy-MM-dd', value: 'yyyy-MM-dd' },
            { label: 'yyyy-MM-dd HH:mm:ss', value: 'yyyy-MM-dd HH:mm:ss' },
          ]" />
        </n-form-item>
        <n-form-item v-if="field.type === 'divider'" label="分割线标题">
          <n-input v-model:value="field.props.title" />
        </n-form-item>
        <n-form-item v-if="field.type === 'static-text'" label="文本内容">
          <n-input v-model:value="field.props.content" type="textarea" :rows="2" />
        </n-form-item>

        <!-- 校验（布局字段无） -->
        <template v-if="!isLayout">
          <n-divider style="margin: 8px 0">校验</n-divider>
          <n-form-item label="必填">
            <n-switch v-model:value="field.rules.required" />
          </n-form-item>
          <n-form-item v-if="['input', 'textarea'].includes(field.type)" label="最大长度（0 不限）">
            <n-input-number v-model:value="field.rules.maxLength" :min="0" style="width: 100%" />
          </n-form-item>
          <n-form-item v-if="['input', 'textarea'].includes(field.type)" label="正则校验">
            <n-input v-model:value="field.rules.pattern" placeholder="如 ^1[3-9]\d{9}$" />
          </n-form-item>
          <n-form-item v-if="field.rules.pattern" label="校验失败提示">
            <n-input v-model:value="field.rules.patternHint" />
          </n-form-item>

          <n-divider style="margin: 8px 0">列表与搜索</n-divider>
          <n-form-item label="列表中显示">
            <n-switch v-model:value="field.showInTable" />
          </n-form-item>
          <n-form-item v-if="field.showInTable" label="列宽（0 自适应）">
            <n-input-number v-model:value="field.tableWidth" :min="0" style="width: 100%" />
          </n-form-item>
          <n-form-item label="可作为搜索条件">
            <n-switch v-model:value="field.searchable" />
          </n-form-item>
        </template>
      </n-form>
    </template>

    <!-- ===== 无选中节点：表单全局配置 ===== -->
    <template v-else>
      <div class="panel-title">表单配置</div>
      <n-form label-placement="top" size="small">
        <n-form-item label="标签宽度">
          <n-input-number v-model:value="formConfig.labelWidth" :min="0" :max="200" style="width: 100%" />
        </n-form-item>
        <n-form-item label="标签位置">
          <n-radio-group v-model:value="formConfig.labelPlacement">
            <n-radio-button value="left">左侧</n-radio-button>
            <n-radio-button value="top">顶部</n-radio-button>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="组件尺寸">
          <n-radio-group v-model:value="formConfig.size">
            <n-radio-button value="small">小</n-radio-button>
            <n-radio-button value="medium">中</n-radio-button>
            <n-radio-button value="large">大</n-radio-button>
          </n-radio-group>
        </n-form-item>
      </n-form>
      <n-empty description="点击画布中的字段可配置其属性" style="margin-top: 32px" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  isGridNode,
  LAYOUT_FIELD_TYPES,
  type FormField,
  type FormNode,
} from '../schema/types'

const props = defineProps<{
  node: FormNode | null
  inColumn: boolean
  formConfig: { labelWidth: number; labelPlacement: 'left' | 'top'; size: 'small' | 'medium' | 'large' }
}>()

// field 字段节点视图（栅格节点时为 null）
const field = computed<FormField | null>(() =>
  props.node && !isGridNode(props.node) ? props.node : null,
)

const isLayout = computed(() => !!field.value && LAYOUT_FIELD_TYPES.includes(field.value.type))
const hasOptions = computed(() => !!field.value && ['radio', 'checkbox', 'select'].includes(field.value.type))

// 字段名规整：只允许字母/数字/下划线，且不以数字开头（与后端动态查询字段名防线一致）
function normalizeName() {
  if (!field.value) return
  field.value.name = field.value.name.replace(/[^A-Za-z0-9_]/g, '_').replace(/^(\d)/, '_$1')
}
</script>

<style scoped>
.props-panel { height: 100%; overflow-y: auto; background: #fff; border-radius: 10px; padding: 12px; box-shadow: 0 1px 4px rgba(15, 40, 25, 0.06); }
.panel-title { font-size: 14px; font-weight: 600; color: #1f2937; margin-bottom: 12px; }
.options-editor { width: 100%; display: flex; flex-direction: column; gap: 6px; }
.option-row { display: grid; grid-template-columns: 1fr 1fr 24px; gap: 6px; align-items: center; }
</style>
