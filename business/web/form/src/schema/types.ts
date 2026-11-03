// 表单 schema 类型（与后端 service/form_schema.go 同构，改动需两侧同步）

export interface FormFieldRules {
  required?: boolean
  pattern?: string
  patternHint?: string
  maxLength?: number
}

export interface FormOption {
  label: string
  value: string | number
}

export interface FormField {
  id: string
  type: string
  name: string
  label: string
  span: number
  props: Record<string, any>
  rules: FormFieldRules
  defaultValue: any
  showInTable: boolean
  tableWidth: number
  searchable: boolean
  sort: number
}

// GridColumn 栅格列：span 为列宽，fields 为列内纵向堆叠的字段节点
export interface GridColumn {
  span: number
  fields: FormNode[]
}

// GridContainer 栅格节点：columns 为列定义（2–4 列）
export interface GridContainer {
  id: string
  type: typeof GRID_NODE_TYPE
  columns: GridColumn[]
}

// 字段节点或栅格节点的判别联合
export type FormNode = FormField | GridContainer

export const GRID_NODE_TYPE = 'grid'

// isGridNode 判别节点是否为栅格节点
export function isGridNode(node: FormNode): node is GridContainer {
  return node.type === GRID_NODE_TYPE
}

export interface FormSchema {
  version: number
  formConfig: {
    labelWidth?: number
    labelPlacement?: 'left' | 'top'
    size?: 'small' | 'medium' | 'large'
    [key: string]: any
  }
  fields: FormNode[]
}

export interface FieldSnapshotItem {
  name: string
  label: string
  type: string
  options?: FormOption[]
}

export interface FormTemplate {
  id: number
  tenant_id: number
  template_key: string
  template_name: string
  category: string
  schema: string
  status: number
  remark: string
  created_at: string
  updated_at: string
}

export interface FormRecord {
  id: number
  tenant_id: number
  template_id: number
  template_version_id: number
  template_key: string
  biz_type: string
  biz_status: string
  customer_id: number
  call_id: string
  direction: string
  agent_id: string
  agent_name: string
  // 业务主键选填，未填时后端返回 null
  record_key: string | null
  data: string
  field_snapshot: string
  created_at: string
  updated_at: string
}

export interface RecordCondition {
  field: string
  op: 'eq' | 'like'
  value: string
}

export interface PageData<T> {
  list: T[]
  total: number
  page: number
  page_size: number
  pages: number
}

// 布局类字段（不产生数据、不参与校验与快照），与后端 layoutFieldTypes 一致
export const LAYOUT_FIELD_TYPES = ['divider', 'static-text']

export function parseSchema(raw: string): FormSchema {
  const schema = JSON.parse(raw) as FormSchema
  if (!schema.fields) schema.fields = []
  return schema
}
