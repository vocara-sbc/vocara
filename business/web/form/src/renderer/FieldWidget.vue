<template>
  <!-- 单个字段的控件渲染：渲染器填报与设计器画布预览共用同一套控件 -->
  <n-divider v-if="field.type === 'divider'" style="margin: 8px 0">
    {{ field.props?.title || '' }}
  </n-divider>
  <div v-else-if="field.type === 'static-text'" class="static-text">
    {{ field.props?.content || field.label }}
  </div>

  <n-input
    v-else-if="field.type === 'input'"
    :value="modelValue"
    @update:value="emit('update:modelValue', $event)"
    :placeholder="field.props?.placeholder"
    :maxlength="field.props?.maxlength"
    :disabled="disabled"
    clearable
  />
  <n-input
    v-else-if="field.type === 'textarea'"
    :value="modelValue"
    @update:value="emit('update:modelValue', $event)"
    type="textarea"
    :rows="field.props?.rows ?? 3"
    :placeholder="field.props?.placeholder"
    :maxlength="field.props?.maxlength"
    :disabled="disabled"
  />
  <n-input-number
    v-else-if="field.type === 'number'"
    :value="modelValue"
    @update:value="emit('update:modelValue', $event)"
    :min="field.props?.min"
    :max="field.props?.max"
    :precision="field.props?.precision"
    :disabled="disabled"
    style="width: 100%"
  />
  <n-radio-group
    v-else-if="field.type === 'radio'"
    :value="modelValue"
    @update:value="emit('update:modelValue', $event)"
    :disabled="disabled"
  >
    <n-radio v-for="opt in fieldOptions" :key="String(opt.value)" :value="opt.value">{{ opt.label }}</n-radio>
  </n-radio-group>
  <n-checkbox-group
    v-else-if="field.type === 'checkbox'"
    :value="modelValue"
    @update:value="emit('update:modelValue', $event)"
    :disabled="disabled"
  >
    <n-checkbox v-for="opt in fieldOptions" :key="String(opt.value)" :value="opt.value">{{ opt.label }}</n-checkbox>
  </n-checkbox-group>
  <n-select
    v-else-if="field.type === 'select'"
    :value="modelValue"
    @update:value="emit('update:modelValue', $event)"
    :options="fieldOptions"
    :multiple="!!field.props?.multiple"
    :filterable="!!field.props?.filterable"
    :placeholder="field.props?.placeholder"
    :disabled="disabled"
    clearable
  />
  <n-date-picker
    v-else-if="field.type === 'date'"
    :formatted-value="modelValue"
    @update:formatted-value="emit('update:modelValue', $event)"
    :type="field.props?.format === 'yyyy-MM-dd HH:mm:ss' ? 'datetime' : 'date'"
    :format="field.props?.format || 'yyyy-MM-dd'"
    :value-format="field.props?.format || 'yyyy-MM-dd'"
    :disabled="disabled"
    style="width: 100%"
    clearable
  />
  <n-switch
    v-else-if="field.type === 'switch'"
    :value="modelValue"
    @update:value="emit('update:modelValue', $event)"
    :disabled="disabled"
  />
  <n-rate
    v-else-if="field.type === 'rate'"
    :value="modelValue"
    @update:value="emit('update:modelValue', $event)"
    :count="field.props?.count ?? 5"
    :disabled="disabled"
  />
  <div v-else class="unknown-field">不支持的字段类型：{{ field.type }}</div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { FormField, FormOption } from '../schema/types'

// FieldWidget 字段控件
const props = defineProps<{
  field: FormField
  modelValue?: any
  disabled?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: any]
}>()

const fieldOptions = computed<FormOption[]>(() => props.field.props?.options || [])
</script>

<style scoped>
.static-text { color: #6b7280; font-size: 14px; line-height: 1.6; }
.unknown-field { color: #d03050; font-size: 13px; }
</style>
