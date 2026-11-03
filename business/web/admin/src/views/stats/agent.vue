<template>
  <div class="stats-agent-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="座席号/姓名"
            clearable
            style="width: 200px"
            @keyup.enter="handleSearch"
          />
        </n-form-item>
        <n-form-item label="日期范围">
          <n-date-picker
            v-model:value="queryForm.dateRange"
            type="daterange"
            clearable
            style="width: 260px"
          />
        </n-form-item>
        <n-form-item label="组织机构">
          <n-select
            v-model:value="queryForm.orgId"
            :options="orgOptions"
            placeholder="全部"
            clearable
            style="width: 160px"
            @update:value="handleOrgChange"
          />
        </n-form-item>
        <n-form-item label="座席">
          <n-select
            v-model:value="queryForm.agentId"
            :options="agentOptions"
            placeholder="全部"
            clearable
            filterable
            style="width: 160px"
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

    <!-- 统计卡片：汇总当前结果集 -->
    <div class="stat-cards">
      <n-card :bordered="false" class="stat-card">
        <div class="stat-label">总通话量</div>
        <div class="stat-value" style="color: #18a058">{{ summary.totalCalls }}</div>
        <div class="stat-sub">呼入 {{ summary.inboundCalls }} / 呼出 {{ summary.outboundCalls }}</div>
      </n-card>
      <n-card :bordered="false" class="stat-card">
        <div class="stat-label">接通率</div>
        <div class="stat-value" style="color: #7c3aed">{{ summary.answerRate }}</div>
        <div class="stat-sub">接听 {{ summary.answeredCalls }} / 未接 {{ summary.missedCalls }}</div>
      </n-card>
      <n-card :bordered="false" class="stat-card">
        <div class="stat-label">平均通话时长</div>
        <div class="stat-value" style="color: #2080f0">{{ summary.avgDuration }}</div>
        <div class="stat-sub">总通话时长 {{ summary.totalTalk }}</div>
      </n-card>
      <n-card :bordered="false" class="stat-card">
        <div class="stat-label">平均满意度</div>
        <div class="stat-value" style="color: #f0a020">{{ summary.satisfaction }}</div>
        <div class="stat-sub">转接 {{ summary.transferCount }} 次 / 保持 {{ summary.holdCount }} 次</div>
      </n-card>
    </div>

    <!-- 图表区 -->
    <n-card title="座席通话量排行" :bordered="false" class="chart-card">
      <div ref="rankChartRef" class="chart-container"></div>
    </n-card>

    <!-- 明细表格 -->
    <n-card title="座席统计明细" :bordered="false" class="table-card">
      <n-data-table
        :columns="columns"
        :data="displayList"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: AgentStatsItem) => row.agent_id"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onUnmounted, h } from 'vue'
import { NTag } from 'naive-ui'
import * as echarts from 'echarts'
import { getAgentStats, type AgentStatsItem } from '@/api/stats'
import { getOrganizationAll, getAgentAll } from '@/api/agent'

const loading = ref(false)

// 默认查询近 7 天
function defaultRange(): [number, number] {
  const end = new Date()
  const start = new Date()
  start.setDate(start.getDate() - 6)
  start.setHours(0, 0, 0, 0)
  return [start.getTime(), end.getTime()]
}

const queryForm = reactive({
  keyword: '',
  dateRange: defaultRange() as [number, number] | null,
  orgId: undefined as number | undefined,
  agentId: undefined as number | undefined,
})

// ===== 下拉选项 =====
const orgOptions = ref<Array<{ label: string; value: number }>>([])
// 座席全量列表，选中组织机构后按 org_id 过滤
const allAgents = ref<Array<{ id: number; name: string; number: string; orgId: number }>>([])

const agentOptions = computed(() =>
  allAgents.value
    .filter((a) => !queryForm.orgId || a.orgId === queryForm.orgId)
    .map((a) => ({ label: `${a.name} (${a.number})`, value: a.id }))
)

// 组织机构 id → 名称映射（后端统计项的 org_name 未填充时用映射兜底）
const orgNameMap = computed(() => {
  const m = new Map<number, string>()
  orgOptions.value.forEach((o) => m.set(o.value, o.label))
  return m
})

async function loadOptions() {
  try {
    const [groups, agents]: any[] = await Promise.all([getOrganizationAll(), getAgentAll()])
    orgOptions.value = (groups || []).map((g: any) => ({
      label: g.org_name || g.org_name || `#${g.id}`,
      value: g.id,
    }))
    allAgents.value = (agents || []).map((a: any) => ({
      id: a.id,
      name: a.agent_name || a.agentName || '',
      number: a.agent_number || a.agentNo || '',
      orgId: a.org_id ?? a.orgId ?? 0,
    }))
  } catch (e) {
    // 选项加载失败不阻塞统计查询
  }
}

// 切换组织机构时，若已选座席不在该组内则清空
function handleOrgChange() {
  if (queryForm.agentId && !agentOptions.value.some((o) => o.value === queryForm.agentId)) {
    queryForm.agentId = undefined
  }
}

// ===== 统计卡汇总（基于当前展示结果集） =====
const summary = computed(() => {
  const list = displayList.value
  const totalCalls = list.reduce((s, r) => s + (r.total_calls || 0), 0)
  const inboundCalls = list.reduce((s, r) => s + (r.inbound_calls || 0), 0)
  const outboundCalls = list.reduce((s, r) => s + (r.outbound_calls || 0), 0)
  const answeredCalls = list.reduce((s, r) => s + (r.answered_calls || 0), 0)
  const missedCalls = list.reduce((s, r) => s + (r.missed_calls || 0), 0)
  const talkDuration = list.reduce((s, r) => s + (r.talk_duration || 0), 0)
  const transferCount = list.reduce((s, r) => s + (r.transfer_count || 0), 0)
  const holdCount = list.reduce((s, r) => s + (r.hold_count || 0), 0)
  const satList = list.filter((r) => r.satisfaction > 0)
  const satisfaction = satList.length
    ? (satList.reduce((s, r) => s + r.satisfaction, 0) / satList.length).toFixed(2)
    : '0.00'
  return {
    totalCalls: totalCalls.toLocaleString(),
    inboundCalls: inboundCalls.toLocaleString(),
    outboundCalls: outboundCalls.toLocaleString(),
    answeredCalls: answeredCalls.toLocaleString(),
    missedCalls: missedCalls.toLocaleString(),
    answerRate: totalCalls > 0 ? ((answeredCalls / totalCalls) * 100).toFixed(1) + '%' : '0.0%',
    avgDuration: answeredCalls > 0 ? formatDuration(Math.round(talkDuration / answeredCalls)) : '0秒',
    totalTalk: formatDuration(talkDuration),
    satisfaction,
    transferCount: transferCount.toLocaleString(),
    holdCount: holdCount.toLocaleString(),
  }
})

// 秒数格式化为可读时长
function formatDuration(seconds: number): string {
  if (!seconds || seconds <= 0) return '0秒'
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  const s = seconds % 60
  if (h > 0) return `${h}小时${m}分`
  if (m > 0) return `${m}分${s}秒`
  return `${s}秒`
}

// ===== 明细表格 =====
const columns = [
  { title: '座席号', key: 'agent_number', width: 90 },
  { title: '姓名', key: 'agent_name', width: 100 },
  {
    title: '组织机构',
    key: 'org_id',
    width: 120,
    render: (row: AgentStatsItem) =>
      row.org_name || orgNameMap.value.get(row.org_id) || (row.org_id ? `#${row.org_id}` : '-'),
  },
  { title: '通话数', key: 'total_calls', width: 90, sorter: (a: AgentStatsItem, b: AgentStatsItem) => a.total_calls - b.total_calls },
  { title: '呼入', key: 'inbound_calls', width: 80 },
  { title: '呼出', key: 'outbound_calls', width: 80 },
  { title: '接听', key: 'answered_calls', width: 80 },
  { title: '未接', key: 'missed_calls', width: 80 },
  {
    title: '接通率',
    key: 'answer_rate',
    width: 90,
    render: (row: AgentStatsItem) =>
      row.total_calls > 0 ? ((row.answered_calls / row.total_calls) * 100).toFixed(1) + '%' : '-',
  },
  {
    title: '通话时长',
    key: 'talk_duration',
    width: 110,
    sorter: (a: AgentStatsItem, b: AgentStatsItem) => a.talk_duration - b.talk_duration,
    render: (row: AgentStatsItem) => formatDuration(row.talk_duration),
  },
  {
    title: '平均通话',
    key: 'avg_duration',
    width: 100,
    render: (row: AgentStatsItem) => formatDuration(row.avg_duration),
  },
  { title: '转接', key: 'transfer_count', width: 70 },
  { title: '保持', key: 'hold_count', width: 70 },
  {
    title: '满意度',
    key: 'satisfaction',
    width: 90,
    render: (row: AgentStatsItem) =>
      row.satisfaction > 0
        ? h(NTag, { size: 'small', type: 'success' }, () => row.satisfaction.toFixed(2))
        : '-',
  },
]

const tableData = ref<AgentStatsItem[]>([])

// 关键字经"查询"按钮应用，过滤在接口结果集上本地完成
const appliedKeyword = ref('')

// 展示结果集：按座席号/姓名过滤，汇总卡、排行图、明细表共用同一结果
const displayList = computed(() => {
  const kw = appliedKeyword.value.trim()
  if (!kw) return tableData.value
  return tableData.value.filter(
    (r) =>
      (r.agent_number || '').includes(kw) ||
      (r.agent_name || '').includes(kw),
  )
})

watch(displayList, (list) => renderRankChart(list))

// 接口返回全量数组，表格前端本地分页
const pagination = reactive({
  pageSize: 10,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
})

// ===== 排行图 =====
const rankChartRef = ref<HTMLElement>()
let rankChart: echarts.ECharts | null = null

function renderRankChart(list: AgentStatsItem[]) {
  if (!rankChartRef.value) return
  if (!rankChart) {
    rankChart = echarts.init(rankChartRef.value)
  }
  // 按通话量排序取前 10，横向条形图需要逆序展示
  const top = [...list].sort((a, b) => b.total_calls - a.total_calls).slice(0, 10).reverse()

  if (top.length === 0) {
    rankChart.setOption(
      {
        title: { text: '暂无数据', left: 'center', top: 'center', textStyle: { color: '#9ca3af', fontSize: 14, fontWeight: 'normal' } },
        xAxis: { show: false },
        yAxis: { show: false },
        series: [],
      },
      true
    )
    return
  }

  rankChart.setOption(
    {
      tooltip: {
        trigger: 'axis',
        backgroundColor: '#ffffff',
        borderColor: '#e8ecf1',
        textStyle: { color: '#1f2937' },
        axisPointer: { type: 'shadow' },
      },
      legend: { data: ['呼入', '呼出'], textStyle: { color: '#6b7280' }, right: 10 },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'value',
        axisLine: { lineStyle: { color: '#e8ecf1' } },
        axisLabel: { color: '#9ca3af' },
        splitLine: { lineStyle: { color: '#f3f4f6' } },
      },
      yAxis: {
        type: 'category',
        data: top.map((r) => r.agent_name || r.agent_number),
        axisLine: { lineStyle: { color: '#e8ecf1' } },
        axisLabel: { color: '#6b7280' },
      },
      series: [
        {
          name: '呼入',
          type: 'bar',
          stack: 'total',
          data: top.map((r) => r.inbound_calls),
          itemStyle: { color: '#18a058' },
        },
        {
          name: '呼出',
          type: 'bar',
          stack: 'total',
          data: top.map((r) => r.outbound_calls),
          itemStyle: { color: '#2080f0', borderRadius: [0, 4, 4, 0] },
        },
      ],
    },
    true
  )
}

// ===== 查询 =====
function formatDate(ts: number): string {
  const d = new Date(ts)
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

async function loadData() {
  loading.value = true
  try {
    const res = await getAgentStats({
      start_date: queryForm.dateRange ? formatDate(queryForm.dateRange[0]) : undefined,
      end_date: queryForm.dateRange ? formatDate(queryForm.dateRange[1]) : undefined,
      org_id: queryForm.orgId,
    })
    let list = res || []
    // 座席筛选在结果集上按 agent_id 过滤，保证与所选座席一致
    if (queryForm.agentId) {
      list = list.filter((r) => r.agent_id === queryForm.agentId)
    }
    tableData.value = list
  } catch (e) {
    // 报错由请求拦截器统一提示，页面保持空态
    tableData.value = []
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  appliedKeyword.value = queryForm.keyword
  loadData()
}

function handleReset() {
  queryForm.keyword = ''
  appliedKeyword.value = ''
  queryForm.dateRange = defaultRange()
  queryForm.orgId = undefined
  queryForm.agentId = undefined
  loadData()
}

function handleResize() {
  rankChart?.resize()
}

onMounted(() => {
  loadOptions()
  loadData()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  rankChart?.dispose()
  rankChart = null
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped lang="css">
/* 统计页内容块多：自然流排列，页面在内容区整体滚动，明细表按数据自然展开 */
.stats-agent-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.search-card :deep(.n-card-content) {
  padding-bottom: 8px;
}

.stat-cards {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.stat-label {
  font-size: 13px;
  color: #6b7280;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 26px;
  font-weight: 700;
  margin-bottom: 6px;
}

.stat-sub {
  font-size: 12px;
  color: #9ca3af;
}

.chart-container {
  width: 100%;
  height: 340px;
}
</style>
