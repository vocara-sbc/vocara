<template>
  <div class="system-log-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="操作人">
          <n-input
            v-model:value="queryForm.username"
            placeholder="操作人用户名"
            clearable
            style="width: 160px"
          />
        </n-form-item>
        <n-form-item label="操作模块">
          <n-input
            v-model:value="queryForm.module"
            placeholder="模块名称"
            clearable
            style="width: 140px"
          />
        </n-form-item>
        <n-form-item label="操作类型">
          <n-select
            v-model:value="queryForm.operationType"
            :options="typeOptions"
            placeholder="全部"
            clearable
            style="width: 120px"
          />
        </n-form-item>
        <n-form-item label="操作时间">
          <n-date-picker
            v-model:value="queryForm.dateRange"
            type="daterange"
            clearable
            style="width: 260px"
          />
        </n-form-item>
        <n-form-item label="操作结果">
          <n-select
            v-model:value="queryForm.status"
            :options="resultOptions"
            placeholder="全部"
            clearable
            style="width: 100px"
          />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" @click="handleSearch">
            <template #icon>
              <n-icon>
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
                </svg>
              </n-icon>
            </template>
            查询
          </n-button>
          <n-button style="margin-left: 8px" @click="handleReset">重置</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 表格 -->
    <n-card :bordered="false" class="table-card">
      <div class="table-header">
        <div class="table-title">操作日志</div>
      </div>

      <n-data-table
        :columns="columns"
        :data="tableData"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: OperationLogItem) => row.id"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 详情弹窗 -->
    <n-modal v-model:show="showDetail" preset="card" title="日志详情" :style="{ width: '720px' }">
      <n-descriptions v-if="detail" :column="2" label-placement="left" bordered size="small">
        <n-descriptions-item label="日志ID">{{ detail.id }}</n-descriptions-item>
        <n-descriptions-item label="操作时间">{{ formatTime(detail.created_at) }}</n-descriptions-item>
        <n-descriptions-item label="操作人">{{ detail.username || '-' }}</n-descriptions-item>
        <n-descriptions-item label="操作类型">{{ typeLabel(detail.operation_type) }}</n-descriptions-item>
        <n-descriptions-item label="操作模块">{{ detail.module || '-' }}</n-descriptions-item>
        <n-descriptions-item label="请求方法">{{ detail.method || '-' }}</n-descriptions-item>
        <n-descriptions-item label="操作结果">
          <n-tag size="small" :type="detail.status === 1 ? 'success' : 'error'">
            {{ detail.status === 1 ? '成功' : '失败' }}
          </n-tag>
        </n-descriptions-item>
        <n-descriptions-item label="耗时">{{ detail.cost_time != null ? detail.cost_time + ' ms' : '-' }}</n-descriptions-item>
        <n-descriptions-item label="IP 地址">{{ detail.ip || '-' }}</n-descriptions-item>
        <n-descriptions-item label="追踪ID">{{ detail.trace_id || '-' }}</n-descriptions-item>
        <n-descriptions-item label="操作内容" :span="2">{{ detail.operation || '-' }}</n-descriptions-item>
        <n-descriptions-item label="请求URL" :span="2">{{ detail.url || '-' }}</n-descriptions-item>
        <n-descriptions-item label="错误信息" :span="2">{{ detail.error_msg || '-' }}</n-descriptions-item>
        <n-descriptions-item label="用户代理" :span="2">{{ detail.user_agent || '-' }}</n-descriptions-item>
        <n-descriptions-item label="请求参数" :span="2">
          <pre class="data-block">{{ detail.request_data || '-' }}</pre>
        </n-descriptions-item>
        <n-descriptions-item label="响应数据" :span="2">
          <pre class="data-block">{{ detail.response_data || '-' }}</pre>
        </n-descriptions-item>
      </n-descriptions>
      <template #footer>
        <n-button @click="showDetail = false">关闭</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from 'vue'
import { NButton, NTag } from 'naive-ui'
import { getOperationLogList } from '@/api/system'

// 操作日志记录结构（与后端 model 的 JSON 字段一致）
interface OperationLogItem {
  id: number
  tenant_id: number
  operation_type: string
  module: string
  operation: string
  method: string
  url: string
  request_data: string
  response_data: string
  user_id: number
  username: string
  ip: string
  user_agent: string
  status: number
  error_msg: string
  cost_time: number
  trace_id: string
  created_at: string
}

const loading = ref(false)

const queryForm = reactive({
  username: '',
  module: '',
  operationType: undefined as string | undefined,
  dateRange: null as [number, number] | null,
  status: undefined as number | undefined,
})

// 操作类型与后端落库值一致：login/logout/add/update/delete/query
const typeOptions = [
  { label: '登录', value: 'login' },
  { label: '登出', value: 'logout' },
  { label: '新增', value: 'add' },
  { label: '修改', value: 'update' },
  { label: '删除', value: 'delete' },
  { label: '查询', value: 'query' },
]

// 操作结果：1-成功 0-失败
const resultOptions = [
  { label: '成功', value: 1 },
  { label: '失败', value: 0 },
]

const typeLabelMap: Record<string, string> = {
  login: '登录',
  logout: '登出',
  add: '新增',
  update: '修改',
  delete: '删除',
  query: '查询',
}

function typeLabel(type: string): string {
  return typeLabelMap[type] || type || '-'
}

// 后端时间为 RFC3339 格式，转为本地可读时间
function formatTime(t: string): string {
  if (!t) return '-'
  const d = new Date(t)
  if (isNaN(d.getTime())) return t
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const columns = [
  { title: '日志ID', key: 'id', width: 80 },
  {
    title: '操作时间',
    key: 'created_at',
    width: 170,
    render: (row: OperationLogItem) => formatTime(row.created_at),
  },
  { title: '操作人', key: 'username', width: 110 },
  {
    title: '操作类型',
    key: 'operation_type',
    width: 100,
    render: (row: OperationLogItem) => typeLabel(row.operation_type),
  },
  { title: '操作模块', key: 'module', width: 110 },
  {
    title: '操作内容',
    key: 'operation',
    width: 260,
    ellipsis: { tooltip: true },
  },
  { title: 'IP 地址', key: 'ip', width: 130 },
  {
    title: '耗时',
    key: 'cost_time',
    width: 90,
    render: (row: OperationLogItem) => (row.cost_time != null ? `${row.cost_time} ms` : '-'),
  },
  {
    title: '操作结果',
    key: 'status',
    width: 90,
    render: (row: OperationLogItem) =>
      h(
        NTag,
        { size: 'small', type: row.status === 1 ? 'success' : 'error' },
        () => (row.status === 1 ? '成功' : '失败')
      ),
  },
  {
    title: '操作',
    key: 'actions',
    width: 80,
    fixed: 'right' as const,
    render: (row: OperationLogItem) =>
      h(
        NButton,
        { size: 'small', quaternary: true, type: 'primary', onClick: () => handleDetail(row) },
        () => '详情'
      ),
  },
]

const tableData = ref<OperationLogItem[]>([])

const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
})

// ===== 详情弹窗 =====
const showDetail = ref(false)
const detail = ref<OperationLogItem | null>(null)

function handleDetail(row: OperationLogItem) {
  detail.value = row
  showDetail.value = true
}

// ===== 数据加载 =====
function formatDate(ts: number): string {
  const d = new Date(ts)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

async function loadData() {
  loading.value = true
  try {
    const res: any = await getOperationLogList({
      page: pagination.page,
      page_size: pagination.pageSize,
      username: queryForm.username || undefined,
      module: queryForm.module || undefined,
      operation_type: queryForm.operationType,
      status: queryForm.status,
      start_time: queryForm.dateRange ? `${formatDate(queryForm.dateRange[0])} 00:00:00` : undefined,
      end_time: queryForm.dateRange ? `${formatDate(queryForm.dateRange[1])} 23:59:59` : undefined,
    })
    tableData.value = res?.list || []
    pagination.itemCount = res?.total ?? 0
  } catch (e) {
    // 报错由请求拦截器统一提示，页面保持空态
    tableData.value = []
    pagination.itemCount = 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  pagination.page = 1
  loadData()
}

function handleReset() {
  queryForm.username = ''
  queryForm.module = ''
  queryForm.operationType = undefined
  queryForm.dateRange = null
  queryForm.status = undefined
  pagination.page = 1
  loadData()
}

function handlePageChange(page: number) {
  pagination.page = page
  loadData()
}

function handlePageSizeChange(size: number) {
  pagination.pageSize = size
  pagination.page = 1
  loadData()
}

onMounted(() => {
  loadData()
})
</script>

<style scoped lang="css">
.system-log-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
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
  font-size: 14px;
  font-weight: 600;
}

.data-block {
  margin: 0;
  max-height: 160px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
  font-size: 12px;
  color: #4b5563;
}
</style>
