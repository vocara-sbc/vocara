<template>
  <div class="tickets-page">
    <!-- 搜索 -->
    <n-card :bordered="false" class="search-card">
      <n-form inline label-placement="left" size="medium">
        <n-form-item label="标题">
          <n-input v-model:value="queryForm.keyword" placeholder="工单标题" clearable style="width: 180px" />
        </n-form-item>
        <n-form-item label="状态">
          <n-select v-model:value="queryForm.status" :options="statusOptions" placeholder="全部" clearable style="width: 130px" />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" @click="handleSearch">查询</n-button>
          <n-button style="margin-left: 8px" @click="handleReset">重置</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 工单表 -->
    <n-card :bordered="false" class="table-card">
      <div class="table-header">
        <div class="table-title">工单管理</div>
      </div>
      <n-data-table
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="pagination"
        remote
        :row-key="(row: TicketInfo) => row.id"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 详情抽屉：按字段快照展示 -->
    <n-drawer v-model:show="detailVisible" :width="460">
      <n-drawer-content title="工单详情" closable>
        <n-descriptions v-if="detailRecord" :column="1" bordered label-placement="left" size="small">
          <n-descriptions-item v-for="item in detailItems" :key="item.label" :label="item.label">
            {{ item.value }}
          </n-descriptions-item>
          <n-descriptions-item label="状态">{{ detailRecord.biz_status === 'open' ? '待处理' : '已处理' }}</n-descriptions-item>
          <n-descriptions-item label="受理座席">{{ detailRecord.agent_name || '-' }}</n-descriptions-item>
          <n-descriptions-item label="创建时间">{{ formatTime(detailRecord.created_at) }}</n-descriptions-item>
        </n-descriptions>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, h, reactive, ref, onMounted } from 'vue'
import { NButton, NTag, type DataTableColumns } from 'naive-ui'
import { message, dialog } from '@/utils/naive'
import {
  listTickets,
  getTicket,
  updateTicketStatus,
  type TicketInfo,
} from '@/api/form'

// 解析记录的表单数据 JSON
function parseData(row: TicketInfo): Record<string, any> {
  try { return JSON.parse(row.data) } catch { return {} }
}
// 解析字段快照
function parseSnapshot(row: TicketInfo): any[] {
  try { return JSON.parse((row as any).field_snapshot || '[]') } catch { return [] }
}

const ticketTypeMap: Record<string, string> = { consult: '咨询', complaint: '投诉', fault: '故障', other: '其他' }
const priorityMap: Record<string, string> = { normal: '普通', urgent: '紧急' }

const loading = ref(false)
const tableData = ref<TicketInfo[]>([])
const queryForm = reactive({ keyword: '', status: undefined as string | undefined })
const pagination = reactive({ page: 1, pageSize: 10, itemCount: 0, showSizePicker: true, pageSizes: [10, 20, 50] })

const statusOptions = [
  { label: '待处理', value: 'open' },
  { label: '已处理', value: 'closed' },
]

function formatTime(t: string) { return t ? t.replace('T', ' ').slice(0, 19) : '-' }

async function loadData() {
  loading.value = true
  try {
    const res: any = await listTickets({
      page: pagination.page, page_size: pagination.pageSize,
      keyword: queryForm.keyword, status: queryForm.status,
    })
    tableData.value = res?.list || []
    pagination.itemCount = res?.total || 0
  } finally { loading.value = false }
}
function handleSearch() { pagination.page = 1; loadData() }
function handleReset() {
  queryForm.keyword = ''; queryForm.status = undefined
  pagination.page = 1; loadData()
}
function handlePageChange(p: number) { pagination.page = p; loadData() }
function handlePageSizeChange(ps: number) { pagination.pageSize = ps; pagination.page = 1; loadData() }

// 详情
const detailVisible = ref(false)
const detailRecord = ref<TicketInfo | null>(null)
// 详情项：用快照翻译（选项值→label），按快照顺序
const detailItems = computed(() => {
  if (!detailRecord.value) return []
  const data = parseData(detailRecord.value)
  return parseSnapshot(detailRecord.value).map((s: any) => {
    let value = data[s.name]
    if (Array.isArray(value)) {
      value = value.map((v) => optionLabel(s, v)).join('、')
    } else if (value !== undefined && value !== null) {
      value = optionLabel(s, value)
    }
    return { label: s.label, value: value ?? '-' }
  })
})
function optionLabel(snap: any, v: any): string {
  const opt = (snap.options || []).find((o: any) => String(o.value) === String(v))
  return opt ? opt.label : String(v)
}
async function openDetail(row: TicketInfo) {
  const res: any = await getTicket(row.id)
  detailRecord.value = res || row
  detailVisible.value = true
}

// 处理工单
function handleClose(row: TicketInfo) {
  dialog.warning({
    title: '处理确认', content: '确认该工单处理完成？',
    onPositiveClick: () =>
      updateTicketStatus(row.id, 'closed').then(() => {
        message.success('工单已处理')
        loadData()
      }),
  })
}

const columns: DataTableColumns<TicketInfo> = [
  { title: '工单标题', key: 'title', width: 180, render: (row) => parseData(row).ticket_title || '-' },
  { title: '客户电话', key: 'phone', width: 130, render: (row) => parseData(row).phone || '-' },
  { title: '类型', key: 'type', width: 80, render: (row) => ticketTypeMap[parseData(row).ticket_type] || '-' },
  { title: '紧急', key: 'priority', width: 70, render: (row) => priorityMap[parseData(row).priority] || '-' },
  { title: '受理座席', key: 'agent', width: 100, render: (row) => row.agent_name || '-' },
  {
    title: '状态', key: 'status', width: 90,
    render: (row) => h(NTag, { size: 'small', type: row.biz_status === 'open' ? 'warning' : 'success' },
      { default: () => (row.biz_status === 'open' ? '待处理' : '已处理') }),
  },
  { title: '创建时间', key: 'created', width: 160, render: (row) => formatTime(row.created_at) },
  {
    title: '操作', key: 'actions', width: 120,
    render: (row) => [
      h(NButton, { size: 'small', quaternary: true, type: 'primary', style: 'margin-right:6px', onClick: () => openDetail(row) }, { default: () => '详情' }),
      row.biz_status === 'open'
        ? h(NButton, { size: 'small', quaternary: true, type: 'success', onClick: () => handleClose(row) }, { default: () => '处理' })
        : null,
    ],
  },
]

onMounted(loadData)
</script>

<style scoped>
.tickets-page { display: flex; flex-direction: column; gap: 16px; height: 100%; }

/* 数据卡填满搜索区之外的剩余高度，表格主体内部滚动、分页固定卡底右侧 */
.table-card { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.table-card :deep(.n-card-content) { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.table-card :deep(.n-data-table) { flex: 1; min-height: 0; }

.table-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.table-title { font-size: 14px; font-weight: 600; }
</style>
