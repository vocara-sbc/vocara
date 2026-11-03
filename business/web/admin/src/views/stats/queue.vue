<template>
  <div class="stats-queue-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="队列名称/编码"
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
        <n-form-item label="队列">
          <n-select
            v-model:value="queryForm.queueId"
            :options="queueOptions"
            placeholder="全部队列"
            clearable
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
        <div class="stat-label">总来话数</div>
        <div class="stat-value" style="color: #f0a020">{{ summary.totalCalls }}</div>
        <div class="stat-sub">接听 {{ summary.answeredCalls }} / 放弃 {{ summary.abandonedCalls }}</div>
      </n-card>
      <n-card :bordered="false" class="stat-card">
        <div class="stat-label">接通率</div>
        <div class="stat-value" style="color: #18a058">{{ summary.answerRate }}</div>
        <div class="stat-sub">按各队列接听汇总计算</div>
      </n-card>
      <n-card :bordered="false" class="stat-card">
        <div class="stat-label">平均等待时长</div>
        <div class="stat-value" style="color: #2080f0">{{ summary.avgWaitTime }}</div>
        <div class="stat-sub">最大等待 {{ summary.maxWaitTime }}</div>
      </n-card>
      <n-card :bordered="false" class="stat-card">
        <div class="stat-label">放弃率</div>
        <div class="stat-value" style="color: #d03050">{{ summary.abandonRate }}</div>
        <div class="stat-sub">按各队列放弃汇总计算</div>
      </n-card>
    </div>

    <!-- 图表区 -->
    <n-row :gutter="16">
      <n-col :span="16">
        <n-card title="各队列等待时长对比" :bordered="false" class="chart-card">
          <div ref="waitChartRef" class="chart-container"></div>
        </n-card>
      </n-col>
      <n-col :span="8">
        <n-card title="各队列来话量占比" :bordered="false" class="chart-card">
          <div ref="pieChartRef" class="chart-container chart-pie"></div>
        </n-card>
      </n-col>
    </n-row>

    <!-- 明细表格 -->
    <n-card title="队列统计明细" :bordered="false" class="table-card">
      <n-data-table
        :columns="columns"
        :data="displayList"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: QueueStatsItem) => row.queue_id"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import * as echarts from 'echarts'
import { getQueueStats, type QueueStatsItem } from '@/api/stats'
import { getQueueAll } from '@/api/queue'

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
  queueId: undefined as number | undefined,
})

// ===== 队列下拉选项 =====
const queueOptions = ref<Array<{ label: string; value: number }>>([])

async function loadOptions() {
  try {
    const queues: any[] = await getQueueAll()
    queueOptions.value = (queues || []).map((q: any) => ({
      label: q.queue_name || `#${q.id}`,
      value: q.id,
    }))
  } catch (e) {
    // 选项加载失败不阻塞统计查询
  }
}

// ===== 统计卡汇总 =====
const summary = computed(() => {
  const list = displayList.value
  const totalCalls = list.reduce((s, r) => s + (r.total_calls || 0), 0)
  const answeredCalls = list.reduce((s, r) => s + (r.answered_calls || 0), 0)
  const abandonedCalls = list.reduce((s, r) => s + (r.abandoned_calls || 0), 0)
  const maxWaitTime = list.reduce((m, r) => Math.max(m, r.max_wait_time || 0), 0)
  // 平均等待按接听量加权
  const avgWaitTime =
    answeredCalls > 0
      ? Math.round(list.reduce((s, r) => s + (r.avg_wait_time || 0) * (r.answered_calls || 0), 0) / answeredCalls)
      : 0
  return {
    totalCalls: totalCalls.toLocaleString(),
    answeredCalls: answeredCalls.toLocaleString(),
    abandonedCalls: abandonedCalls.toLocaleString(),
    answerRate: totalCalls > 0 ? ((answeredCalls / totalCalls) * 100).toFixed(1) + '%' : '0.0%',
    abandonRate: totalCalls > 0 ? ((abandonedCalls / totalCalls) * 100).toFixed(1) + '%' : '0.0%',
    avgWaitTime: formatDuration(avgWaitTime),
    maxWaitTime: formatDuration(maxWaitTime),
  }
})

// 秒数格式化为可读时长
function formatDuration(seconds: number): string {
  if (!seconds || seconds <= 0) return '0秒'
  const m = Math.floor(seconds / 60)
  const s = seconds % 60
  if (m > 0) return `${m}分${s}秒`
  return `${s}秒`
}

// ===== 明细表格 =====
const columns = [
  { title: '队列名称', key: 'queue_name', width: 140 },
  { title: '队列编码', key: 'queue_code', width: 110 },
  { title: '来话总数', key: 'total_calls', width: 100, sorter: (a: QueueStatsItem, b: QueueStatsItem) => a.total_calls - b.total_calls },
  { title: '已接听', key: 'answered_calls', width: 90 },
  { title: '客户放弃', key: 'abandoned_calls', width: 90 },
  {
    title: '接通率',
    key: 'answer_rate',
    width: 90,
    render: (row: QueueStatsItem) => (row.answer_rate || 0).toFixed(1) + '%',
  },
  {
    title: '放弃率',
    key: 'abandon_rate',
    width: 90,
    render: (row: QueueStatsItem) => (row.abandon_rate || 0).toFixed(1) + '%',
  },
  {
    title: '平均等待',
    key: 'avg_wait_time',
    width: 100,
    render: (row: QueueStatsItem) => formatDuration(row.avg_wait_time),
  },
  {
    title: '最大等待',
    key: 'max_wait_time',
    width: 100,
    render: (row: QueueStatsItem) => formatDuration(row.max_wait_time),
  },
  {
    title: '平均通话',
    key: 'avg_talk_time',
    width: 100,
    render: (row: QueueStatsItem) => formatDuration(row.avg_talk_time),
  },
]

const tableData = ref<QueueStatsItem[]>([])

// 关键字经"查询"按钮应用，过滤在接口结果集上本地完成
const appliedKeyword = ref('')

// 展示结果集：按队列名称/编码过滤，汇总卡、图表、明细表共用同一结果
const displayList = computed(() => {
  const kw = appliedKeyword.value.trim()
  if (!kw) return tableData.value
  return tableData.value.filter(
    (r) =>
      (r.queue_name || '').includes(kw) ||
      (r.queue_code || '').includes(kw),
  )
})

watch(displayList, (list) => {
  renderWaitChart(list)
  renderPieChart(list)
})

// 接口返回全量数组，表格前端本地分页
const pagination = reactive({
  pageSize: 10,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
})

// ===== 图表 =====
const waitChartRef = ref<HTMLElement>()
const pieChartRef = ref<HTMLElement>()
let waitChart: echarts.ECharts | null = null
let pieChart: echarts.ECharts | null = null

const PIE_COLORS = ['#18a058', '#2080f0', '#7c3aed', '#f0a020', '#d03050', '#0ea5e9']

function emptyOption(text = '暂无数据') {
  return {
    title: { text, left: 'center', top: 'center', textStyle: { color: '#9ca3af', fontSize: 14, fontWeight: 'normal' as const } },
    xAxis: { show: false },
    yAxis: { show: false },
    series: [],
  }
}

function renderWaitChart(list: QueueStatsItem[]) {
  if (!waitChartRef.value) return
  if (!waitChart) waitChart = echarts.init(waitChartRef.value)
  if (list.length === 0) {
    waitChart.setOption(emptyOption(), true)
    return
  }
  waitChart.setOption(
    {
      tooltip: {
        trigger: 'axis',
        backgroundColor: '#ffffff',
        borderColor: '#e8ecf1',
        textStyle: { color: '#1f2937' },
        axisPointer: { type: 'shadow' },
        valueFormatter: (v: number) => formatDuration(v),
      },
      legend: { data: ['平均等待', '最大等待'], textStyle: { color: '#6b7280' }, right: 10 },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'category',
        data: list.map((r) => r.queue_name),
        axisLine: { lineStyle: { color: '#e8ecf1' } },
        axisLabel: { color: '#6b7280' },
      },
      yAxis: {
        type: 'value',
        name: '秒',
        axisLine: { lineStyle: { color: '#e8ecf1' } },
        axisLabel: { color: '#9ca3af' },
        splitLine: { lineStyle: { color: '#f3f4f6' } },
        nameTextStyle: { color: '#9ca3af' },
      },
      series: [
        {
          name: '平均等待',
          type: 'bar',
          data: list.map((r) => r.avg_wait_time),
          itemStyle: { color: '#18a058', borderRadius: [4, 4, 0, 0] },
        },
        {
          name: '最大等待',
          type: 'bar',
          data: list.map((r) => r.max_wait_time),
          itemStyle: { color: '#f0a020', borderRadius: [4, 4, 0, 0] },
        },
      ],
    },
    true
  )
}

function renderPieChart(list: QueueStatsItem[]) {
  if (!pieChartRef.value) return
  if (!pieChart) pieChart = echarts.init(pieChartRef.value)
  const data = list
    .filter((r) => r.total_calls > 0)
    .map((r, i) => ({ value: r.total_calls, name: r.queue_name, itemStyle: { color: PIE_COLORS[i % PIE_COLORS.length] } }))
  if (data.length === 0) {
    pieChart.setOption(emptyOption(), true)
    return
  }
  pieChart.setOption(
    {
      tooltip: {
        trigger: 'item',
        backgroundColor: '#ffffff',
        borderColor: '#e8ecf1',
        textStyle: { color: '#1f2937' },
        formatter: '{b}: {c} ({d}%)',
      },
      legend: {
        orient: 'vertical',
        left: 'left',
        textStyle: { color: '#6b7280' },
        top: 'center',
      },
      series: [
        {
          type: 'pie',
          radius: ['50%', '70%'],
          center: ['65%', '50%'],
          label: { show: false },
          data,
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
    const res = await getQueueStats({
      start_date: queryForm.dateRange ? formatDate(queryForm.dateRange[0]) : undefined,
      end_date: queryForm.dateRange ? formatDate(queryForm.dateRange[1]) : undefined,
      queue_id: queryForm.queueId,
    })
    tableData.value = res || []
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
  queryForm.queueId = undefined
  loadData()
}

function handleResize() {
  waitChart?.resize()
  pieChart?.resize()
}

onMounted(() => {
  loadOptions()
  loadData()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  waitChart?.dispose()
  pieChart?.dispose()
  waitChart = null
  pieChart = null
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped lang="css">
/* 统计页内容块多：自然流排列，页面在内容区整体滚动，明细表按数据自然展开 */
.stats-queue-page {
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

.chart-pie {
  height: 340px;
}
</style>
