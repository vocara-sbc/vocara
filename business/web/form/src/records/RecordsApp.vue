<template>
  <div class="vocara-form-records">
    <!-- 模板选择 + 搜索区（一张卡） -->
    <n-card :bordered="false" :class="['records-card', { 'records-card--joined': joined }]">
      <div class="search-bar">
        <n-select
          v-model:value="currentKey"
          :options="templateOptions"
          placeholder="选择表单模板"
          style="width: 200px"
          @update:value="handleTemplateChange"
        />
        <template v-for="field in searchFields" :key="field.name">
          <n-input
            v-if="['input', 'textarea', 'number'].includes(field.type)"
            v-model:value="searchValues[field.name]"
            :placeholder="field.label"
            clearable
            style="width: 180px"
            @keyup.enter="handleSearch"
          />
          <n-select
            v-else-if="['radio', 'checkbox', 'select'].includes(field.type)"
            v-model:value="searchValues[field.name]"
            :options="field.props?.options || []"
            :placeholder="field.label"
            clearable
            style="width: 160px"
            @update:value="handleSearch"
          />
          <n-date-picker
            v-else-if="field.type === 'date'"
            v-model:formatted-value="searchValues[field.name]"
            type="date"
            value-format="yyyy-MM-dd"
            :placeholder="field.label"
            clearable
            style="width: 160px"
            @update:value="handleSearch"
          />
        </template>
        <n-button type="primary" @click="handleSearch">查询</n-button>
        <n-button @click="handleReset">重置</n-button>
      </div>
    </n-card>

    <!-- 数据表（一张卡） -->
    <n-card :bordered="false" class="records-card table-card">
      <div class="table-header">
        <div class="table-title">{{ currentTemplate?.template_name || '表单数据' }}</div>
        <div class="table-actions">
          <n-button size="small" :loading="exporting" @click="handleExport">导出</n-button>
          <n-button size="small" type="primary" :disabled="!currentKey" @click="openEditor()">新增</n-button>
        </div>
      </div>
      <n-data-table
        :columns="tableColumns"
        :data="tableData"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: FormRecord) => row.id"
        remote
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 填报/编辑弹窗：复用渲染器 -->
    <n-modal v-model:show="editorVisible" preset="card" :title="editingRecord ? '编辑' : '新增'" class="record-edit-modal" style="width: 720px">
      <RendererApp
        v-if="editorVisible && currentTemplate"
        ref="rendererRef"
        :api-base="apiBase"
        :get-token="getToken"
        :schema="editorSchema"
        :data="editingData"
      />
      <div class="editor-extra">
        <n-input v-model:value="editingRecordKey" placeholder="业务主键（选填，如客户手机号）" style="max-width: 280px" />
      </div>
      <template #footer>
        <n-button @click="editorVisible = false">取消</n-button>
        <n-button type="primary" :loading="saving" @click="handleSaveRecord">保存</n-button>
      </template>
    </n-modal>

    <!-- 详情抽屉：快照只读展示 -->
    <n-drawer v-model:show="detailVisible" :width="420">
      <n-drawer-content title="详情" closable>
        <n-descriptions :column="1" label-placement="left" bordered size="small">
          <n-descriptions-item v-for="item in detailItems" :key="item.label" :label="item.label">{{ item.value }}</n-descriptions-item>
          <n-descriptions-item label="创建时间">{{ formatTime(detailRecord?.created_at) }}</n-descriptions-item>
        </n-descriptions>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, onMounted } from 'vue'
import { NButton, type DataTableColumns } from 'naive-ui'
import { message, dialog } from '../deps'
import { createFormApi } from '../api'
import {
  LAYOUT_FIELD_TYPES,
  parseSchema,
  type FieldSnapshotItem,
  type FormRecord,
  type FormSchema,
  type FormTemplate,
  type RecordCondition,
} from '../schema/types'
import { collectFields } from '../schema/walk'
import RendererApp from '../renderer/RendererApp.vue'

interface Props {
  apiBase: string
  getToken: () => string
  templateKey?: string
  // 嵌入管理后台布局时启用：首张卡片左上角直角，与页签下缘拼接
  joined?: boolean
}

const props = defineProps<Props>()

const api = createFormApi({ apiBase: props.apiBase, getToken: props.getToken })

// ---------- 模板选择与搜索 ----------

const templates = ref<FormTemplate[]>([])
const currentKey = ref('')
const searchValues = reactive<Record<string, any>>({})

// 未固定单模板时，选择器提供「全部模板」
const templateOptions = computed(() => {
  const options = templates.value.map((t) => ({ label: t.template_name, value: t.template_key }))
  if (!props.templateKey) {
    options.unshift({ label: '全部模板', value: '' })
  }
  return options
})
const currentTemplate = computed(() => templates.value.find((t) => t.template_key === currentKey.value))
const schemaObj = computed<FormSchema>(() =>
  currentTemplate.value ? parseSchema(currentTemplate.value.schema) : { version: 1, formConfig: {}, fields: [] },
)
// 全部模板模式下的可搜索字段并集（按字段名去重，属性以首个模板为准）
const allSearchFields = computed(() => {
  const seen = new Set<string>()
  const list: ReturnType<typeof collectFields> = []
  templates.value.forEach((t) => {
    const schema = parseSchema(t.schema)
    collectFields(schema.fields).forEach((f) => {
      if (f.searchable && !LAYOUT_FIELD_TYPES.includes(f.type) && !seen.has(f.name)) {
        seen.add(f.name)
        list.push(f)
      }
    })
  })
  return list
})
// 可搜索字段：单模板取该模板 schema，全部模板取并集
const searchFields = computed(() => {
  if (currentKey.value) {
    return collectFields(schemaObj.value.fields).filter(
      (f) => f.searchable && !LAYOUT_FIELD_TYPES.includes(f.type),
    )
  }
  return allSearchFields.value
})

function clearSearchValues() {
  Object.keys(searchValues).forEach((k) => delete searchValues[k])
}

// 文本类模糊匹配，选择/日期类精确匹配；空值不生成条件
function buildConditions(): RecordCondition[] {
  const conditions: RecordCondition[] = []
  searchFields.value.forEach((f) => {
    const v = searchValues[f.name]
    if (v === undefined || v === null || v === '') return
    const op = ['input', 'textarea', 'number'].includes(f.type) ? 'like' : 'eq'
    conditions.push({ field: f.name, op, value: String(v) })
  })
  return conditions
}

function handleTemplateChange() {
  clearSearchValues()
  pagination.page = 1
  loadData()
}

function handleSearch() {
  pagination.page = 1
  loadData()
}

function handleReset() {
  clearSearchValues()
  pagination.page = 1
  loadData()
}

// ---------- 快照翻译（value → label） ----------

// data/field_snapshot 均为 JSON 字符串，解析失败按空处理
function parseJsonSafe<T>(raw: string, fallback: T): T {
  try {
    return JSON.parse(raw) as T
  } catch {
    return fallback
  }
}

function snapshotOf(row: FormRecord): FieldSnapshotItem[] {
  return parseJsonSafe<FieldSnapshotItem[]>(row.field_snapshot, [])
}

// 选择类字段把存储值翻译成选项文案；多选顿号拼接
function translateValue(value: any, snap?: FieldSnapshotItem): string {
  if (value === undefined || value === null || value === '') return '-'
  const toLabel = (v: any): string => {
    if (!snap?.options?.length) return String(v)
    const opt = snap.options.find((o) => String(o.value) === String(v))
    return opt ? opt.label : String(v)
  }
  if (Array.isArray(value)) return value.length ? value.map(toLabel).join('、') : '-'
  return toLabel(value)
}

function isOptionType(type: string): boolean {
  return ['select', 'radio', 'checkbox'].includes(type)
}

function translateCell(row: FormRecord, field: { name: string; type: string }): string {
  const data = parseJsonSafe<Record<string, any>>(row.data, {})
  const value = data[field.name]
  if (!isOptionType(field.type)) {
    return value === undefined || value === null || value === '' ? '-' : String(value)
  }
  const snap = snapshotOf(row).find((s) => s.name === field.name)
  return translateValue(value, snap)
}

function formatTime(raw?: string): string {
  return raw ? raw.replace('T', ' ').slice(0, 19) : '-'
}

// ---------- 数据表 ----------

const tableData = ref<FormRecord[]>([])
const loading = ref(false)
const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
})

// 模板标识 → 模板名称映射，跨模板列表展示所属模板
const templateNameMap = computed(() => {
  const m = new Map<string, string>()
  templates.value.forEach((t) => m.set(t.template_key, t.template_name))
  return m
})

// 操作列：详情/编辑/删除
function buildActionColumn(): DataTableColumns<FormRecord>[number] {
  return {
    title: '操作',
    key: 'actions',
    width: 170,
    render: (row: FormRecord) =>
      [
        { label: '详情', onClick: () => openDetail(row) },
        { label: '编辑', onClick: () => openEditor(row) },
        { label: '删除', danger: true, onClick: () => handleDelete(row) },
      ].map((btn) =>
        h(
          NButton,
          {
            size: 'small',
            text: true,
            type: btn.danger ? 'error' : 'primary',
            style: 'margin-right: 8px',
            onClick: btn.onClick,
          },
          { default: () => btn.label },
        ),
      ),
  }
}

const tableColumns = computed<DataTableColumns<FormRecord>>(() => {
  // 全部模板模式：所属模板 + 通用列；单模板模式：模板表格字段 + 通用列
  const baseColumns: DataTableColumns<FormRecord> = []
  if (!currentKey.value) {
    baseColumns.push({
      title: '所属模板',
      key: 'template_key',
      width: 140,
      render: (row: FormRecord) => templateNameMap.value.get(row.template_key) || row.template_key || '-',
    })
  } else {
    collectFields(schemaObj.value.fields)
      .filter((f) => f.showInTable && !LAYOUT_FIELD_TYPES.includes(f.type))
      .forEach((f) => {
        baseColumns.push({
          title: f.label,
          key: f.name,
          width: f.tableWidth || undefined,
          render: (row: FormRecord) => translateCell(row, f),
        })
      })
  }
  baseColumns.push(
    { title: '业务主键', key: 'record_key', width: 140, render: (row: FormRecord) => row.record_key || '-' },
    { title: '创建时间', key: 'created_at', width: 170, render: (row: FormRecord) => formatTime(row.created_at) },
    buildActionColumn(),
  )
  return baseColumns
})

async function loadData() {
  loading.value = true
  try {
    const res = await api.recordPage(currentKey.value, buildConditions(), pagination.page, pagination.pageSize)
    tableData.value = res.list || []
    pagination.itemCount = res.total || 0
  } catch (e: any) {
    message.error(e?.message || '加载数据失败')
  } finally {
    loading.value = false
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  loadData()
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize
  pagination.page = 1
  loadData()
}

// ---------- 新增/编辑弹窗 ----------

const editorVisible = ref(false)
const saving = ref(false)
const editingRecord = ref<FormRecord | null>(null)
const editingData = ref<Record<string, any> | undefined>(undefined)
const editingRecordKey = ref('')
const rendererRef = ref<{ validate: () => Promise<boolean>; getData: () => Record<string, any> }>()

// 编辑弹窗使用的 schema：编辑老记录用其绑定版本，新增用模板当前版本
const editorSchema = ref('')

async function openEditor(record?: FormRecord) {
  editingRecord.value = record || null
  editingData.value = undefined
  editingRecordKey.value = record?.record_key || ''
  editorSchema.value = ''
  if (record) {
    try {
      const detail = await api.recordGet(record.id)
      editingRecord.value = detail
      editingData.value = parseJsonSafe<Record<string, any>>(detail.data, {})
      editingRecordKey.value = detail.record_key || ''
      // 按记录绑定版本取 schema；旧记录无版本时回退当前版本
      if (detail.template_version_id) {
        const ver = await api.templateVersionGet(detail.template_version_id)
        editorSchema.value = ver.schema
      } else {
        editorSchema.value = currentTemplate.value?.schema || ''
      }
    } catch (e: any) {
      message.error(e?.message || '加载数据失败')
      return
    }
  } else {
    editorSchema.value = currentTemplate.value?.schema || ''
  }
  editorVisible.value = true
}

async function handleSaveRecord() {
  const renderer = rendererRef.value
  if (!renderer) return
  if (!(await renderer.validate())) return
  saving.value = true
  try {
    const data = renderer.getData()
    if (editingRecord.value) {
      await api.recordUpdate({ id: editingRecord.value.id, record_key: editingRecordKey.value || undefined, data })
    } else {
      await api.recordSave({ template_key: currentKey.value, record_key: editingRecordKey.value || undefined, data })
    }
    message.success('保存成功')
    editorVisible.value = false
    loadData()
  } catch (e: any) {
    message.error(e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function handleDelete(row: FormRecord) {
  dialog.warning({
    title: '删除确认',
    content: '确定删除该条数据吗？删除后不可恢复。',
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await api.recordDelete(row.id)
        message.success('删除成功')
        loadData()
      } catch (e: any) {
        message.error(e?.message || '删除失败')
      }
    },
  })
}

// ---------- 详情抽屉 ----------

const detailVisible = ref(false)
const detailRecord = ref<FormRecord | null>(null)
const detailItems = ref<{ label: string; value: string }[]>([])

async function openDetail(row: FormRecord) {
  try {
    const record = await api.recordGet(row.id)
    detailRecord.value = record
    const data = parseJsonSafe<Record<string, any>>(record.data, {})
    const items: { label: string; value: string }[] = []
    if (record.record_key) items.push({ label: '业务主键', value: record.record_key })
    snapshotOf(record)
      .filter((s) => !LAYOUT_FIELD_TYPES.includes(s.type))
      .forEach((s) => {
        const value = data[s.name]
        items.push({
          label: s.label,
          value: isOptionType(s.type) ? translateValue(value, s) : value === undefined || value === null || value === '' ? '-' : String(value),
        })
      })
    detailItems.value = items
    detailVisible.value = true
  } catch (e: any) {
    message.error(e?.message || '加载详情失败')
  }
}

// ---------- 导出 ----------

const exporting = ref(false)

// 从 Content-Disposition 头解析文件名，兼容 filename*=UTF-8'' 与 filename= 两种写法
function parseRespFileName(resp: Response): string {
  const disposition = resp.headers.get('content-disposition') || ''
  const utf8Match = disposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8Match) return decodeURIComponent(utf8Match[1].trim().replace(/"/g, ''))
  const plainMatch = disposition.match(/filename=([^;]+)/i)
  if (plainMatch) return plainMatch[1].trim().replace(/"/g, '')
  return ''
}

// fetch Response 转浏览器下载；响应头未携带文件名时使用 fallbackName
async function downloadBlobResp(resp: Response, fallbackName: string) {
  const fileName = parseRespFileName(resp) || fallbackName
  const blob = await resp.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = fileName
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

async function handleExport() {
  exporting.value = true
  try {
    const resp = await api.recordExport(currentKey.value, buildConditions())
    await downloadBlobResp(resp, `${currentTemplate.value?.template_name || '全部模板'}.csv`)
  } catch (e: any) {
    message.error(e?.message || '导出失败')
  } finally {
    exporting.value = false
  }
}

// ---------- 初始化 ----------

onMounted(async () => {
  try {
    const list = await api.templateList()
    // 版本化后主表不含 schema：按 key 补取各模板当前版本 schema（失败则该模板按空处理）
    await Promise.all(
      list.map(async (t: any) => {
        try {
          const ver = await api.templateSchemaByKey(t.template_key)
          t.schema = ver.schema
        } catch {
          // 单个模板版本读取失败不阻断列表
        }
      }),
    )
    templates.value = list
  } catch (e: any) {
    message.error(e?.message || '加载模板列表失败')
    return
  }
  if (props.templateKey) {
    // 挂载参数固定单模板时，只展示该模板
    const target = templates.value.find((t) => t.template_key === props.templateKey)
    if (target) {
      currentKey.value = target.template_key
      loadData()
    }
    return
  }
  // 默认展示全部模板的数据
  loadData()
})
</script>

<style scoped>
.vocara-form-records {
  height: 100%;
  padding: 0;
  box-sizing: border-box;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #ecf3ef;
}
.records-card {
  border-radius: 10px;
  box-shadow: 0 1px 4px rgba(15, 40, 25, 0.06);
}
.records-card--joined {
  border-radius: 0 10px 10px 10px;
}
/* 搜索卡固定高度，数据卡填满剩余高度：表格主体内部滚动、分页固定卡底右侧 */
.vocara-form-records > .records-card:first-child {
  flex-shrink: 0;
}
.table-card {
  margin-top: 16px;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.table-card :deep(.n-card-content) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.table-card :deep(.n-data-table) {
  flex: 1;
  min-height: 0;
}
.search-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.table-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.table-title {
  font-size: 14px;
  font-weight: 600;
}
.table-actions {
  display: flex;
  gap: 8px;
}
.editor-extra {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px dashed #e2e8f0;
}
</style>

<!-- 非 scoped：弹窗经 teleport 挂到 body，scoped 规则无法命中；
     以弹窗专属 class 限定作用域，不影响宿主其他卡片 -->
<style>
.record-edit-modal .n-card__footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
