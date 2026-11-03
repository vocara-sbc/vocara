<template>
  <div class="active-page">
    <!-- 统计卡片 -->
    <div class="stat-cards">
      <n-card :bordered="false" class="stat-card">
        <div class="stat-content">
          <div class="stat-label">活跃通话</div>
          <div class="stat-value">{{ statData.activeCount }}</div>
        </div>
        <div class="stat-icon" style="background: rgba(24, 160, 88, 0.1)">
          <svg viewBox="0 0 24 24" fill="none" stroke="#18a058" stroke-width="2">
            <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/>
          </svg>
        </div>
      </n-card>

      <n-card :bordered="false" class="stat-card">
        <div class="stat-content">
          <div class="stat-label">通话中</div>
          <div class="stat-value">{{ statData.bridgedCount }}</div>
        </div>
        <div class="stat-icon" style="background: rgba(32, 128, 240, 0.1)">
          <svg viewBox="0 0 24 24" fill="none" stroke="#2080f0" stroke-width="2">
            <line x1="19" y1="12" x2="5" y2="12"/>
            <polyline points="12 19 5 12 12 5"/>
          </svg>
        </div>
      </n-card>

      <n-card :bordered="false" class="stat-card">
        <div class="stat-content">
          <div class="stat-label">振铃中</div>
          <div class="stat-value">{{ statData.ringingCount }}</div>
        </div>
        <div class="stat-icon" style="background: rgba(124, 58, 237, 0.1)">
          <svg viewBox="0 0 24 24" fill="none" stroke="#7c3aed" stroke-width="2">
            <line x1="5" y1="12" x2="19" y2="12"/>
            <polyline points="12 5 19 12 12 19"/>
          </svg>
        </div>
      </n-card>

      <n-card :bordered="false" class="stat-card">
        <div class="stat-content">
          <div class="stat-label">保持中</div>
          <div class="stat-value">{{ statData.heldCount }}</div>
        </div>
        <div class="stat-icon" style="background: rgba(240, 160, 32, 0.1)">
          <svg viewBox="0 0 24 24" fill="none" stroke="#f0a020" stroke-width="2">
            <path d="M23 6l-9.5 9.5-5-5L1 18"/>
            <path d="M17 6h6v6"/>
          </svg>
        </div>
      </n-card>
    </div>

    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="主叫/被叫/座席"
            clearable
            style="width: 220px"
          />
        </n-form-item>
        <n-form-item label="通话类型">
          <n-select
            v-model:value="queryForm.callType"
            :options="callTypeOptions"
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

    <!-- 表格 -->
    <n-card :bordered="false" class="table-card">
      <div class="table-header">
        <div class="table-title">实时通话列表</div>
        <n-button size="small" @click="handleRefresh">
          <template #icon>
            <n-icon size="14">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="23 4 23 10 17 10"/>
                <polyline points="1 20 1 14 7 14"/>
                <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
              </svg>
            </n-icon>
          </template>
          刷新
        </n-button>
      </div>

      <n-data-table
        :columns="columns"
        :data="tableData"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: ActiveCallInfo) => row.callId"
        flex-height
      />
    </n-card>

    <!-- 监督操作分机确认弹窗 -->
    <n-modal
      v-model:show="superviseVisible"
      preset="card"
      :title="superviseTitle"
      style="width: 420px"
      :bordered="false"
    >
      <n-form label-placement="left" label-width="80">
        <n-form-item label="通话">
          <span>{{ superviseTarget?.caller }} → {{ superviseTarget?.called }}</span>
        </n-form-item>
        <n-form-item label="班长分机">
          <n-input
            v-model:value="supervisorExtension"
            placeholder="请输入执行监督操作的分机号"
            @keyup.enter="handleSuperviseConfirm"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-button @click="superviseVisible = false">取消</n-button>
        <n-button type="primary" :loading="superviseSubmitting" @click="handleSuperviseConfirm">
          确认{{ superviseTitle }}
        </n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, h } from 'vue'
import { message, dialog } from '@/utils/naive'
import { useUserStore } from '@/store/modules/user'
import {
  getActiveCalls,
  getCallStats,
  hangupCall,
  eavesdropCall,
  whisperCall,
  bargeCall,
  type ActiveCallInfo,
} from '@/api/callctrl'
import { formatDuration, formatUnixTime } from '@/utils/format'

const userStore = useUserStore()

const loading = ref(false)

// 通话统计（callctrl /stats 接口字段）
const statData = reactive({
  activeCount: 0,
  bridgedCount: 0,
  ringingCount: 0,
  heldCount: 0,
})

// 查询表单（前端过滤已拉取的真实数据）
const queryForm = reactive({
  keyword: '',
  callType: undefined as string | undefined,
})

const callTypeOptions = [
  { label: '呼入', value: 'inbound' },
  { label: '呼出', value: 'outbound' },
  { label: '内部', value: 'internal' },
]

const callTypeText: Record<string, string> = {
  inbound: '呼入',
  outbound: '呼出',
  internal: '内部',
}

const callTypeColor: Record<string, string> = {
  inbound: '#2080f0',
  outbound: '#7c3aed',
  internal: '#f0a020',
}

// 通话状态文案与颜色（callctrl model.CallStatus 取值）
const statusText: Record<string, string> = {
  created: '已创建',
  ringing: '振铃中',
  early_media: '回铃中',
  agent_answered: '座席已接',
  answered: '已接听',
  bridged: '通话中',
  held: '保持中',
  transferring: '转接中',
  supervising: '被监听',
  three_way: '三方通话',
  failed: '失败',
  cancelled: '已取消',
  ended: '已结束',
}

const statusColor: Record<string, string> = {
  ringing: '#2080f0',
  early_media: '#2080f0',
  agent_answered: '#2080f0',
  answered: '#2080f0',
  bridged: '#18a058',
  held: '#f0a020',
  transferring: '#7c3aed',
  supervising: '#7c3aed',
  three_way: '#7c3aed',
  failed: '#d03050',
  cancelled: '#9ca3af',
  ended: '#9ca3af',
}

// 表格列
const columns = [
  { title: '通话ID', key: 'callId', width: 200, ellipsis: { tooltip: true } },
  {
    title: '类型',
    key: 'callType',
    width: 80,
    render: (row: ActiveCallInfo) => {
      return h(
        'span',
        { style: { color: callTypeColor[row.callType] || '#6b7280', fontSize: '13px' } },
        callTypeText[row.callType] || row.callType
      )
    },
  },
  { title: '主叫号码', key: 'caller', width: 130 },
  { title: '被叫号码', key: 'called', width: 130 },
  {
    title: '座席',
    key: 'agentName',
    width: 100,
    render: (row: ActiveCallInfo) => row.agentName || row.extension || '-',
  },
  {
    title: '所属队列',
    key: 'queueId',
    width: 120,
    render: (row: ActiveCallInfo) => row.queueId || '-',
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row: ActiveCallInfo) => {
      return h(
        'span',
        { style: { color: statusColor[row.status] || '#6b7280', fontSize: '13px' } },
        statusText[row.status] || row.status
      )
    },
  },
  {
    title: '通话时长',
    key: 'talkTime',
    width: 100,
    render: (row: ActiveCallInfo) => {
      // 以接听时间（无则开始时间）为起点，随秒级计时器实时计算
      const base = row.answerTime || row.startTime
      const text = base ? formatDuration(nowTs.value - base) : '-'
      return h('span', { style: { color: '#18a058', fontWeight: 500 } }, text)
    },
  },
  {
    title: '开始时间',
    key: 'startTime',
    width: 170,
    render: (row: ActiveCallInfo) => formatUnixTime(row.startTime),
  },
  {
    title: '操作',
    key: 'actions',
    width: 220,
    fixed: 'right' as const,
    render: (row: ActiveCallInfo) => {
      return h('div', { style: { display: 'flex', gap: '8px' } }, [
        h(
          'a',
          {
            style: { color: '#18a058', cursor: 'pointer', fontSize: '13px' },
            onClick: () => openSupervise('eavesdrop', row),
          },
          '监听'
        ),
        h(
          'a',
          {
            style: { color: '#f0a020', cursor: 'pointer', fontSize: '13px' },
            onClick: () => openSupervise('whisper', row),
          },
          '耳语'
        ),
        h(
          'a',
          {
            style: { color: '#2080f0', cursor: 'pointer', fontSize: '13px' },
            onClick: () => openSupervise('barge', row),
          },
          '强插'
        ),
        h(
          'a',
          {
            style: { color: '#d03050', cursor: 'pointer', fontSize: '13px' },
            onClick: () => handleHangup(row),
          },
          '强拆'
        ),
      ])
    },
  },
]

// 活跃通话全量数据（一次拉取一页大容量，前端过滤与分页）
const allCalls = ref<ActiveCallInfo[]>([])

// 秒级计时器基准时间，驱动通话时长列刷新
const nowTs = ref(Math.floor(Date.now() / 1000))

// 过滤后的表格数据
const tableData = computed(() => {
  const kw = queryForm.keyword.trim()
  return allCalls.value.filter((row) => {
    if (queryForm.callType && row.callType !== queryForm.callType) return false
    if (!kw) return true
    return [row.caller, row.called, row.agentName, row.extension]
      .filter(Boolean)
      .some((v) => String(v).includes(kw))
  })
})

// 前端分页
const pagination = reactive({
  pageSize: 10,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
})

// 当前租户 ID（callctrl 接口要求以字符串传递）
function tenantId(): string {
  return String(userStore.userInfo?.tenant_id ?? 0)
}

// 拉取活跃通话与统计；请求失败保留上一帧数据（拦截器已统一提示）
async function loadData(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    const [callsRes, statsRes] = await Promise.allSettled([
      getActiveCalls({ tenantId: tenantId(), page: 1, pageSize: 100 }),
      getCallStats(tenantId()),
    ])
    if (callsRes.status === 'fulfilled') {
      allCalls.value = callsRes.value.list || []
    }
    if (statsRes.status === 'fulfilled') {
      statData.activeCount = statsRes.value.activeCount
      statData.bridgedCount = statsRes.value.bridgedCount
      statData.ringingCount = statsRes.value.ringingCount
      statData.heldCount = statsRes.value.heldCount
    }
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  loadData()
}

function handleReset() {
  queryForm.keyword = ''
  queryForm.callType = undefined
}

function handleRefresh() {
  loadData()
}

// ========== 监督操作 ==========

const SUPERVISOR_EXT_KEY = 'vocara_supervisor_ext'

const superviseVisible = ref(false)
const superviseSubmitting = ref(false)
const superviseAction = ref<'eavesdrop' | 'whisper' | 'barge'>('eavesdrop')
const superviseTarget = ref<ActiveCallInfo | null>(null)
// 管理端用户信息不含分机号，监督操作的分机由使用者输入并本地记忆
const supervisorExtension = ref(localStorage.getItem(SUPERVISOR_EXT_KEY) || '')

const superviseActionText: Record<string, string> = {
  eavesdrop: '监听',
  whisper: '耳语',
  barge: '强插',
}

const superviseTitle = computed(() => superviseActionText[superviseAction.value])

function openSupervise(action: 'eavesdrop' | 'whisper' | 'barge', row: ActiveCallInfo) {
  superviseAction.value = action
  superviseTarget.value = row
  superviseVisible.value = true
}

async function handleSuperviseConfirm() {
  const ext = supervisorExtension.value.trim()
  if (!ext) {
    message.warning('请输入班长分机号')
    return
  }
  const target = superviseTarget.value
  if (!target) return

  const params = {
    callId: target.callId,
    supervisorId: String(userStore.userInfo?.user_id ?? ''),
    supervisorName: userStore.userInfo?.nickname || userStore.userInfo?.username || '',
    extension: ext,
    tenantId: tenantId(),
  }

  superviseSubmitting.value = true
  try {
    if (superviseAction.value === 'eavesdrop') {
      await eavesdropCall(params)
    } else if (superviseAction.value === 'whisper') {
      await whisperCall(params)
    } else {
      await bargeCall(params)
    }
    localStorage.setItem(SUPERVISOR_EXT_KEY, ext)
    message.success(`已发起${superviseActionText[superviseAction.value]}，请接听分机 ${ext}`)
    superviseVisible.value = false
  } catch {
    // 错误提示由拦截器统一处理
  } finally {
    superviseSubmitting.value = false
  }
}

// 强拆通话
function handleHangup(row: ActiveCallInfo) {
  dialog.warning({
    title: '强拆确认',
    content: `确定要强拆通话 ${row.caller} → ${row.called} 吗？`,
    positiveText: '强拆',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await hangupCall({ callId: row.callId })
        message.success('强拆指令已下发')
        loadData(false)
      } catch {
        // 错误提示由拦截器统一处理
      }
    },
  })
}

let pollTimer: number | undefined
let tickTimer: number | undefined

onMounted(() => {
  loadData()
  // 8 秒轮询真实重取
  pollTimer = window.setInterval(() => loadData(false), 8000)
  // 秒级计时驱动通话时长列
  tickTimer = window.setInterval(() => {
    nowTs.value = Math.floor(Date.now() / 1000)
  }, 1000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
  if (tickTimer) clearInterval(tickTimer)
})
</script>

<style scoped lang="css">
.active-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
}

.stat-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  flex-shrink: 0;
}

.stat-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.stat-content {
  flex: 1;
}

.stat-label {
  font-size: 13px;
  color: #6b7280;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #1f2937;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.stat-icon svg {
  width: 24px;
  height: 24px;
}

/* 搜索卡固定高度，数据卡填满剩余高度：表格主体内部滚动、分页固定卡底右侧 */
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
</style>
