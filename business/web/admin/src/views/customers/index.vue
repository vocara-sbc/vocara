<template>
  <div class="customers-page">
    <!-- 搜索 -->
    <n-card :bordered="false" class="search-card">
      <n-form inline label-placement="left" size="medium">
        <n-form-item label="关键字">
          <n-input v-model:value="keyword" placeholder="姓名/手机号" clearable style="width: 200px" @keyup.enter="handleSearch" />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" @click="handleSearch">查询</n-button>
          <n-button style="margin-left: 8px" @click="handleReset">重置</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 客户表 -->
    <n-card :bordered="false" class="table-card">
      <div class="table-header">
        <div class="table-title">客户管理</div>
      </div>
      <n-data-table
        :columns="columns"
        :data="tableData"
        :loading="loading"
        :pagination="pagination"
        remote
        :row-key="(row: CustomerInfo) => row.id"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 详情抽屉 -->
    <n-drawer v-model:show="detailVisible" :width="500">
      <n-drawer-content title="客户详情" closable>
        <template v-if="detail">
          <n-descriptions :column="1" bordered label-placement="left" size="small">
            <n-descriptions-item label="客户姓名">{{ detail.customer.name || '-' }}</n-descriptions-item>
            <n-descriptions-item label="手机号">{{ detail.customer.phone }}</n-descriptions-item>
            <n-descriptions-item label="客户标签">{{ customerTags }}</n-descriptions-item>
            <n-descriptions-item label="累计来电">{{ detail.customer.call_count }}</n-descriptions-item>
            <n-descriptions-item label="首次出现">{{ formatTime(detail.customer.first_seen_at) }}</n-descriptions-item>
            <n-descriptions-item label="最近出现">{{ formatTime(detail.customer.last_seen_at) }}</n-descriptions-item>
          </n-descriptions>

            <div class="record-section">关联工单/记录（{{ detail.records.length }}）</div>
            <n-data-table
              :columns="recordColumns"
              :data="detail.records"
              :row-key="(row: any) => row.id"
              size="small"
            />
          </template>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { computed, h, ref, reactive, onMounted } from 'vue'
import { NButton, NTag, type DataTableColumns } from 'naive-ui'
import { listCustomers, getCustomer, type CustomerInfo } from '@/api/form'

const loading = ref(false)
const tableData = ref<CustomerInfo[]>([])
const keyword = ref('')
const pagination = reactive({ page: 1, pageSize: 10, itemCount: 0, showSizePicker: true, pageSizes: [10, 20, 50] })

function formatTime(t: string) { return t ? t.replace('T', ' ').slice(0, 19) : '-' }
function parseData(c: CustomerInfo): Record<string, any> {
  try { return JSON.parse(c.data) } catch { return {} }
}

async function loadData() {
  loading.value = true
  try {
    const res: any = await listCustomers({
      page: pagination.page, page_size: pagination.pageSize, keyword: keyword.value,
    })
    tableData.value = res?.list || []
    pagination.itemCount = res?.total || 0
  } finally { loading.value = false }
}
function handleSearch() { pagination.page = 1; loadData() }
function handleReset() { keyword.value = ''; pagination.page = 1; loadData() }
function handlePageChange(p: number) { pagination.page = p; loadData() }
function handlePageSizeChange(ps: number) { pagination.pageSize = ps; pagination.page = 1; loadData() }

// 详情
const detailVisible = ref(false)
const detail = ref<{ customer: CustomerInfo; records: any[] } | null>(null)
const customerTags = computed(() => {
  if (!detail.value) return '-'
  try {
    const d = JSON.parse(detail.value.customer.data)
    return (d.customer_tag || []).join('、') || '-'
  } catch { return '-' }
})

async function openDetail(row: CustomerInfo) {
  const res: any = await getCustomer(row.id)
  detail.value = res || { customer: row, records: [] }
  detailVisible.value = true
}

const columns: DataTableColumns<CustomerInfo> = [
  { title: '客户姓名', key: 'name', render: (row) => row.name || '-' },
  { title: '手机号', key: 'phone', render: (row) => row.phone },
  {
    title: '客户标签', key: 'tags',
    render: (row) => (parseData(row).customer_tag || []).map((t: string) =>
      h(NTag, { size: 'small', type: 'info' }, { default: () => t }),
    ),
  },
  { title: '累计来电', key: 'count', width: 90, render: (row) => row.call_count },
  { title: '最近来电', key: 'last', width: 160, render: (row) => formatTime(row.last_seen_at) },
  {
    title: '操作', key: 'actions', width: 90,
    render: (row) => h(NButton, { size: 'small', quaternary: true, type: 'primary', onClick: () => openDetail(row) }, { default: () => '详情' }),
  },
]

// 详情中的关联记录表
const recordColumns: DataTableColumns<any> = [
  { title: '类型', key: 'biz_type', width: 80, render: (row) => row.biz_type },
  { title: '状态', key: 'biz_status', width: 80, render: (row) => row.biz_status || '-' },
  { title: '时间', key: 'created_at', render: (row) => formatTime(row.created_at) },
]

onMounted(loadData)
</script>

<style scoped>
.customers-page { display: flex; flex-direction: column; gap: 16px; height: 100%; }

/* 数据卡填满搜索区之外的剩余高度，表格主体内部滚动、分页固定卡底右侧 */
.table-card { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.table-card :deep(.n-card-content) { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.table-card :deep(.n-data-table) { flex: 1; min-height: 0; }

.table-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.table-title { font-size: 14px; font-weight: 600; }
.record-section { margin: 14px 0 8px; font-size: 14px; font-weight: 600; color: #1f2937; }
</style>
