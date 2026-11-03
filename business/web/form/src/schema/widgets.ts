// 字段类型注册表：新增字段类型只需在这里加一条定义
import { GRID_NODE_TYPE, type FormField, type FormFieldRules, type FormNode, type GridContainer } from './types'

export interface WidgetDef {
  type: string
  title: string
  icon: string // emoji 图标，保持 SDK 零图标依赖
  group: 'basic' | 'choice' | 'advanced' | 'layout'
  isLayout?: boolean
  defaultProps: () => Record<string, any>
}

export const WIDGET_GROUPS = [
  { key: 'basic', title: '基础字段' },
  { key: 'choice', title: '选择字段' },
  { key: 'advanced', title: '高级字段' },
  { key: 'layout', title: '布局元素' },
]

export const WIDGET_DEFS: WidgetDef[] = [
  { type: 'input', title: '单行文本', icon: '📝', group: 'basic',
    defaultProps: () => ({ placeholder: '请输入', maxlength: 50 }) },
  { type: 'textarea', title: '多行文本', icon: '📄', group: 'basic',
    defaultProps: () => ({ placeholder: '请输入', rows: 3, maxlength: 500 }) },
  { type: 'number', title: '数字', icon: '🔢', group: 'basic',
    defaultProps: () => ({ min: undefined, max: undefined, precision: 0 }) },
  { type: 'date', title: '日期', icon: '📅', group: 'basic',
    defaultProps: () => ({ format: 'yyyy-MM-dd' }) },
  { type: 'switch', title: '开关', icon: '🔘', group: 'basic', defaultProps: () => ({}) },
  { type: 'rate', title: '评分', icon: '⭐', group: 'basic', defaultProps: () => ({ count: 5 }) },
  { type: 'radio', title: '单选框组', icon: '🔵', group: 'choice',
    defaultProps: () => ({ options: [{ label: '选项一', value: '1' }, { label: '选项二', value: '2' }] }) },
  { type: 'checkbox', title: '多选框组', icon: '☑️', group: 'choice',
    defaultProps: () => ({ options: [{ label: '选项一', value: '1' }, { label: '选项二', value: '2' }] }) },
  { type: 'select', title: '下拉选择', icon: '🔽', group: 'choice',
    defaultProps: () => ({ options: [{ label: '选项一', value: '1' }, { label: '选项二', value: '2' }], multiple: false, filterable: false, placeholder: '请选择' }) },
  { type: GRID_NODE_TYPE, title: '栅格', icon: '▦', group: 'layout', isLayout: true,
    defaultProps: () => ({}) },
  { type: 'divider', title: '分割线', icon: '➖', group: 'layout', isLayout: true,
    defaultProps: () => ({ title: '' }) },
  { type: 'static-text', title: '静态文本', icon: '💬', group: 'layout', isLayout: true,
    defaultProps: () => ({ content: '说明文字' }) },
]

export const WIDGET_MAP: Record<string, WidgetDef> = Object.fromEntries(WIDGET_DEFS.map((w) => [w.type, w]))

let fieldSeq = 1
let gridSeq = 1

// createGrid 生成默认两列、span 各 12 的栅格节点
export function createGrid(): GridContainer {
  const seq = gridSeq++
  return {
    id: `g_${Date.now().toString(36)}_${seq}`,
    type: GRID_NODE_TYPE,
    columns: [
      { span: 12, fields: [] },
      { span: 12, fields: [] },
    ],
  }
}

// createField 按类型生成节点：栅格返回栅格节点，其余返回字段节点
// （id/name 唯一；字段默认占满一行，数据字段默认进列表不搜索）
export function createField(type: string): FormNode {
  if (type === GRID_NODE_TYPE) return createGrid()
  const def = WIDGET_MAP[type]
  const seq = fieldSeq++
  const name = `${type}_${seq}`
  const rules: FormFieldRules = { required: false, pattern: '', patternHint: '', maxLength: 0 }
  return {
    id: `f_${Date.now().toString(36)}_${seq}`,
    type,
    name,
    label: def?.title || type,
    span: 24,
    props: def?.defaultProps() || {},
    rules,
    defaultValue: def?.isLayout ? undefined : (type === 'checkbox' ? [] : ''),
    showInTable: def?.isLayout ? false : true,
    tableWidth: 0,
    searchable: false,
    sort: seq,
  }
}
