<template>
  <div class="outbound-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="主叫号码/被叫号码/座席"
            clearable
            style="width: 220px"
          />
        </n-form-item>
        <n-form-item label="通话日期">
          <n-date-picker
            v-model:value="queryForm.dateRange"
            type="daterange"
            clearable
            style="width: 260px"
          />
        </n-form-item>
        <n-form-item label="通话结果">
          <n-select
            v-model:value="queryForm.result"
            :options="resultOptions"
            placeholder="全部"
            clearable
            style="width: 140px"
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

    <!-- 操作栏 + 表格 -->
    <n-card :bordered="false" class="table-card">
      <div class="table-header">
        <div class="table-title">呼出记录</div>
        <div class="table-actions">
          <n-button size="small" :loading="exporting" @click="handleExport">
            <template #icon>
              <n-icon size="14">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path d="M21 15v4a2 2 0 01-2 2H5a2 2 0 01-2-2v-4"/>
                  <polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
              </n-icon>
            </template>
            导出记录
          </n-button>
        </div>
      </div>

      <n-data-table
        :columns="columns"
        :data="tableData"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: any) => row.id"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 通话详情抽屉 -->
    <n-drawer v-model:show="detailVisible" :width="420">
      <n-drawer-content title="通话详情" closable>
        <n-descriptions :column="1" label-placement="left" bordered size="small">
          <n-descriptions-item label="通话ID">{{ detail.call_id || '-' }}</n-descriptions-item>
          <n-descriptions-item label="主叫号码">{{ detail.caller || '-' }}</n-descriptions-item>
          <n-descriptions-item label="被叫号码">{{ detail.called || '-' }}</n-descriptions-item>
          <n-descriptions-item label="呼叫座席">{{ detail.agent_name || '-' }}</n-descriptions-item>
          <n-descriptions-item label="所属队列">{{ detail.queue_name || (detail.queue_id ? `#${detail.queue_id}` : '-') }}</n-descriptions-item>
          <n-descriptions-item label="技能组">{{ detail.skill_group_name || '-' }}</n-descriptions-item>
          <n-descriptions-item label="通话时长">{{ fmtDuration(detail.talk_duration) }}</n-descriptions-item>
          <n-descriptions-item label="接听状态">{{ detail.answer_status === 1 ? '已接听' : '未接听' }}</n-descriptions-item>
          <n-descriptions-item label="挂机原因">{{ detail.hangup_cause || '-' }}</n-descriptions-item>
          <n-descriptions-item label="开始时间">{{ fmtTime(detail.start_time) }}</n-descriptions-item>
          <n-descriptions-item label="结束时间">{{ fmtTime(detail.end_time) }}</n-descriptions-item>
          <n-descriptions-item label="录音文件">{{ detail.record_file ? '有' : '无' }}</n-descriptions-item>
        </n-descriptions>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, h } from 'vue'
import { message } from '@/utils/naive'
import { getOutboundRecords, getCallDetail, exportRecords } from '@/api/call'
import { downloadBlob } from '@/utils/download'

const loading = ref(false)
const exporting = ref(false)

// 详情抽屉
const detailVisible = ref(false)
const detail = ref<any>({})

// 查询表单
const queryForm = reactive({
  keyword: '',
  dateRange: null as [number, number] | null,
  result: undefined as string | undefined,
})

const resultOptions = [
  { label: '已接通', value: 'answered' },
  { label: '未接听', value: 'noanswer' },
]

// 表格列
const columns = [
  { title: '通话ID', key: 'callId', width: 180 },
  { title: '主叫号码', key: 'caller', width: 140 },
  { title: '被叫号码', key: 'called', width: 140 },
  { title: '座席', key: 'agentName', width: 100 },
  { title: '座席分机', key: 'dnNumber', width: 100 },
  { title: '通话时长', key: 'talkTime', width: 110 },
  {
    title: '通话结果',
    key: 'result',
    width: 110,
    render: (row: any) => {
      const resultMap: Record<string, { color: string; text: string }> = {
        answered: { color: '#18a058', text: '已接通' },
        noanswer: { color: '#f0a020', text: '未接听' },
      }
      const r = resultMap[row.result] || resultMap.answered
      return h('span', { style: { color: r.color } }, r.text)
    },
  },
  { title: '呼叫开始时间', key: 'startTime', width: 180 },
  {
    title: '操作',
    key: 'actions',
    width: 160,
    fixed: 'right',
    render: (row: any) => {
      return h('div', { style: { display: 'flex', gap: '8px' } }, [
        h(
          'a',
          {
            style: { color: '#18a058', cursor: 'pointer', fontSize: '13px' },
            onClick: () => handleDetail(row),
          },
          '详情'
        ),
        row.hasRecord
          ? h(
              'a',
              {
                style: { color: '#2080f0', cursor: 'pointer', fontSize: '13px' },
                onClick: () => handlePlay(row),
              },
              '播放录音'
            )
          : null,
      ])
    },
  },
]

// 表格数据（真实接口：tb_call_record 统一话单）
const tableData = ref<any[]>([])

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
})

function fmtDuration(sec: number): string {
  const m = Math.floor((sec || 0) / 60).toString().padStart(2, '0')
  const s = ((sec || 0) % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

function fmtTime(t: string): string {
  if (!t) return '-'
  return t.replace('T', ' ').slice(0, 19)
}

function mapRow(r: any) {
  return {
    id: r.id,
    callId: r.call_id,
    caller: r.caller || r.dn_number || '-',
    called: r.called || '-',
    agentName: r.agent_name || '-',
    dnNumber: r.dn_number || '-',
    talkTime: fmtDuration(r.talk_duration),
    result: r.answer_status === 1 ? 'answered' : 'noanswer',
    startTime: fmtTime(r.start_time),
    hasRecord: !!r.record_file,
  }
}

async function loadData() {
  loading.value = true
  try {
    const params: any = {
      page: pagination.page,
      page_size: pagination.pageSize,
      keyword: queryForm.keyword || undefined,
    }
    if (queryForm.result === 'answered') params.answer_status = 1
    else if (queryForm.result === 'noanswer') params.answer_status = 0
    if (queryForm.dateRange) {
      params.start_time = new Date(queryForm.dateRange[0]).toISOString()
      params.end_time = new Date(queryForm.dateRange[1] + 86400000 - 1).toISOString()
    }
    const res: any = await getOutboundRecords(params)
    tableData.value = (res?.list || []).map(mapRow)
    pagination.itemCount = res?.total ?? 0
  } catch (e: any) {
    message.error(e?.message || '加载呼出记录失败')
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
  queryForm.dateRange = null
  queryForm.result = undefined
  handleSearch()
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

async function handleDetail(row: any) {
  try {
    detail.value = await getCallDetail(row.id)
    detailVisible.value = true
  } catch (e: any) {
    message.error(e?.message || '获取详情失败')
  }
}

function handlePlay(row: any) {
  window.open(`/api/v1/admin/records/${row.callId}/recording`, '_blank')
}

// 导出当前查询条件的记录（响应为文件流，统一走 downloadBlob 落盘）
async function handleExport() {
  exporting.value = true
  try {
    const params: any = { call_type: 2, keyword: queryForm.keyword || undefined }
    if (queryForm.result === 'answered') params.answer_status = 1
    else if (queryForm.result === 'noanswer') params.answer_status = 0
    if (queryForm.dateRange) {
      params.start_time = new Date(queryForm.dateRange[0]).toISOString()
      params.end_time = new Date(queryForm.dateRange[1] + 86400000 - 1).toISOString()
    }
    const resp = await exportRecords(params)
    downloadBlob(resp as any, `呼出记录_${new Date().toISOString().slice(0, 10)}.csv`)
  } catch (e: any) {
    message.error(e?.message || '导出失败')
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped lang="css">
.outbound-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
}

.search-card {
  flex-shrink: 0;
}

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

.table-actions {
  display: flex;
  align-items: center;
}
</style>
