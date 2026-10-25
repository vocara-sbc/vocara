<template>
  <div class="form-template-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="模板名称/标识"
            clearable
            style="width: 220px"
          />
        </n-form-item>
        <n-form-item label="分类">
          <n-select
            v-model:value="queryForm.category"
            :options="categoryOptions"
            placeholder="全部"
            clearable
            style="width: 140px"
          />
        </n-form-item>
        <n-form-item label="状态">
          <n-select
            v-model:value="queryForm.status"
            :options="statusOptions"
            placeholder="全部"
            clearable
            style="width: 140px"
          />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" @click="handleSearch">查询</n-button>
          <n-button style="margin-left: 8px" @click="handleReset">重置</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 操作栏 + 表格 -->
    <n-card :bordered="false" class="table-card">
      <div class="table-header">
        <div class="table-title">表单模板</div>
        <div class="table-actions">
          <n-button type="primary" size="small" @click="handleAdd">
            新增模板
          </n-button>
        </div>
      </div>

      <n-data-table
        :columns="columns"
        :data="tableData"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: FormTemplateInfo) => row.id"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 历史版本抽屉 -->
    <n-drawer v-model:show="versionDrawerVisible" :width="560">
      <n-drawer-content title="历史版本" closable>
        <n-data-table
          :columns="versionColumns"
          :data="versionList"
          :row-key="(row: any) => row.id"
          size="small"
        />
        <!-- 只读查看某版本的字段构成 -->
        <div v-if="viewingVersion" class="version-view">
          <div class="version-view-head">
            <span>v{{ viewingVersion.version }} 字段构成（只读）</span>
            <n-button size="tiny" quaternary @click="viewingVersion = null">关闭查看</n-button>
          </div>
          <n-descriptions :column="1" bordered size="small">
            <n-descriptions-item
              v-for="f in versionFields(viewingVersion.schema)"
              :key="f.name"
              :label="f.label"
            >
              {{ f.type }} · {{ f.name }}
            </n-descriptions-item>
          </n-descriptions>
        </div>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from 'vue'
import { useRouter } from 'vue-router'
import { NButton, NTag } from 'naive-ui'
import { message, dialog } from '@/utils/naive'
import {
  getFormTemplatePage,
  updateFormTemplate,
  deleteFormTemplate,
  getFormCategories,
  getTemplateVersions,
  updateFormTemplateStatus,
} from '@/api/form'
import type { FormTemplateVersionInfo } from '@/api/form'

// 模板记录结构（与后端 tb_form_template 字段一致）
interface FormTemplateInfo {
  id: number
  template_name: string
  template_key: string
  category: string
  current_version_id: number
  status: number
  remark: string
  updated_at: string
}

const router = useRouter()
const loading = ref(false)

// ===== 查询 =====
const queryForm = reactive({
  keyword: '',
  category: undefined as string | undefined,
  status: undefined as number | undefined,
})
// 分类选项与名称映射从后端动态加载
const categoryOptions = ref<{ label: string; value: string }[]>([])
const categoryNameMap = reactive<Record<string, string>>({})

const statusOptions = [
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
]

// 加载分类：筛选项取启用分类，名称映射覆盖全部分类（含停用）
async function loadCategories() {
  const res: any = await getFormCategories()
  const list: any[] = res?.data || res || []
  categoryOptions.value = list
    .filter((c) => c.enabled)
    .map((c) => ({ label: c.name, value: c.code }))
  list.forEach((c) => { categoryNameMap[c.code] = c.name })
}

const tableData = ref<FormTemplateInfo[]>([])
const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
})

// 后端时间戳为 RFC3339，展示时截为「YYYY-MM-DD HH:mm:ss」
function formatTime(t: string) {
  return t ? t.replace('T', ' ').slice(0, 19) : '-'
}

// ===== 历史版本 =====
const versionDrawerVisible = ref(false)
const versionList = ref<FormTemplateVersionInfo[]>([])
const viewingVersion = ref<FormTemplateVersionInfo | null>(null)

// 打开版本抽屉：加载历史版本
async function openVersions(row: FormTemplateInfo) {
  viewingVersion.value = null
  const res: any = await getTemplateVersions(row.id)
  versionList.value = res?.data || res || []
  versionDrawerVisible.value = true
}

// versionFields 解析版本 schema，返回字段清单（仅数据字段）
function versionFields(schema: string): { name: string; label: string; type: string }[] {
  try {
    const parsed = JSON.parse(schema)
    return (parsed.fields || []).filter((f: any) => f.name)
  } catch {
    return []
  }
}

const versionColumns = [
  { title: '版本', key: 'version', width: 70, render: (row: any) => `v${row.version}` },
  { title: '发布时间', key: 'created_at', render: (row: any) => formatTime(row.created_at) },
  {
    title: '操作',
    key: 'op',
    width: 80,
    render: (row: FormTemplateVersionInfo) =>
      h(NButton, { size: 'tiny', quaternary: true, type: 'primary', onClick: () => (viewingVersion.value = row) }, () => '查看'),
  },
]

// ===== 表格列 =====
const columns = [
  { title: '模板名称', key: 'template_name', width: 180 },
  {
    title: '模板标识',
    key: 'template_key',
    width: 160,
    render: (row: FormTemplateInfo) => h(NTag, { size: 'small' }, () => row.template_key),
  },
  {
    title: '分类',
    key: 'category',
    width: 90,
    render: (row: FormTemplateInfo) => categoryNameMap[row.category] || row.category || '-',
  },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row: FormTemplateInfo) =>
      h(
        NTag,
        { size: 'small', type: row.status === 1 ? 'success' : 'default' },
        () => (row.status === 1 ? '启用' : '停用')
      ),
  },
  { title: '备注', key: 'remark', ellipsis: { tooltip: true } },
  {
    title: '更新时间',
    key: 'updated_at',
    width: 170,
    render: (row: FormTemplateInfo) => formatTime(row.updated_at),
  },
  {
    title: '操作',
    key: 'actions',
    width: 290,
    render: (row: FormTemplateInfo) => [
      h(
        NButton,
        { size: 'small', type: 'primary', quaternary: true, style: 'margin-right: 8px', onClick: () => handleDesign(row) },
        () => '设计'
      ),
      h(
        NButton,
        { size: 'small', quaternary: true, style: 'margin-right: 8px', onClick: () => openVersions(row) },
        () => '版本'
      ),
      h(
        NButton,
        { size: 'small', quaternary: true, style: 'margin-right: 8px', onClick: () => handleToggleStatus(row) },
        () => (row.status === 1 ? '停用' : '启用')
      ),
      h(
        NButton,
        { size: 'small', type: 'error', quaternary: true, onClick: () => handleDelete(row) },
        () => '删除'
      ),
    ],
  },
]

// ===== 数据加载 =====
async function loadData() {
  loading.value = true
  try {
    const res: any = await getFormTemplatePage({
      page: pagination.page,
      page_size: pagination.pageSize,
      keyword: queryForm.keyword || undefined,
      category: queryForm.category || undefined,
      status: queryForm.status,
    })
    tableData.value = res?.list || res?.data?.list || []
    pagination.itemCount = res?.total ?? res?.data?.total ?? 0
  } catch (e: any) {
    message.error(e?.message || '加载失败')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  pagination.page = 1
  loadData()
}

function handleReset() {
  queryForm.keyword = ''
  queryForm.category = undefined
  queryForm.status = undefined
  pagination.page = 1
  loadData()
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

// ===== 新增/设计/启停/删除 =====
// 新增模板直接进入设计器：模板名称与字段在设计器内填写，
// 保存时由设计页走创建接口（空 schema 无法通过后端 fields 非空校验）
function handleAdd() {
  router.push('/system/form-template/design')
}

function handleDesign(row: FormTemplateInfo) {
  router.push(`/system/form-template/design?id=${row.id}`)
}

// 启停走轻量状态接口，不产生新版本
async function handleToggleStatus(row: FormTemplateInfo) {
  const target = row.status === 1 ? 0 : 1
  try {
    await updateFormTemplateStatus(row.id, target)
    message.success(target === 1 ? '已启用' : '已停用')
    loadData()
  } catch (e: any) {
    message.error(e?.message || '操作失败')
  }
}

function handleDelete(row: FormTemplateInfo) {
  dialog.warning({
    title: '删除确认',
    content: `确定删除表单模板「${row.template_name}」（${row.template_key}）吗？删除后基于该模板的表单将无法新增数据。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteFormTemplate(row.id)
        message.success('删除成功')
        loadData()
      } catch (e: any) {
        message.error(e?.message || '删除失败')
      }
    },
  })
}

onMounted(async () => {
  await loadCategories()
  loadData()
})
</script>

<style scoped>
.form-template-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
}
.search-card :deep(.n-card-content) {
  padding-bottom: 8px;
}

/* 数据卡填满搜索区之外的剩余高度，表格主体内部滚动、分页固定卡底右侧 */
.table-card {
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
.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}
.table-title {
  font-weight: 600;
  font-size: 14px;
}
</style>
