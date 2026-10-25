<template>
  <div class="stats-line-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="线路编码/名称"
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
        <n-form-item label="线路">
          <n-select
            v-model:value="queryForm.lineId"
            :options="lineOptions"
            placeholder="全部线路"
            clearable
            filterable
            style="width: 200px"
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
        <div class="stat-label">总呼叫量</div>
        <div class="stat-value" style="color: #18a058">{{ summary.totalCalls }}</div>
        <div class="stat-sub">覆盖线路 {{ summary.lineCount }} 条</div>
      </n-card>
      <n-card :bordered="false" class="stat-card">
        <div class="stat-label">接通数</div>
        <div class="stat-value" style="color: #2080f0">{{ summary.answeredCalls }}</div>
        <div class="stat-sub">各线路接通汇总</div>
      </n-card>
      <n-card :bordered="false" class="stat-card">
        <div class="stat-label">失败数</div>
        <div class="stat-value" style="color: #d03050">{{ summary.failedCalls }}</div>
        <div class="stat-sub">各线路失败汇总</div>
      </n-card>
      <n-card :bordered="false" class="stat-card">
        <div class="stat-label">成功率</div>
        <div class="stat-value" style="color: #7c3aed">{{ summary.successRate }}</div>
        <div class="stat-sub">接通 / 总呼叫</div>
      </n-card>
    </div>

    <!-- 图表区 -->
    <n-card title="各线路呼叫量对比" :bordered="false" class="chart-card">
      <div ref="barChartRef" class="chart-container"></div>
    </n-card>

    <!-- 明细表格 -->
    <n-card title="线路统计明细" :bordered="false" class="table-card">
      <n-data-table
        :columns="columns"
        :data="displayList"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: LineStatsItem) => row.line_id"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted, onUnmounted } from 'vue'
import * as echarts from 'echarts'
import { getLineStats, getLineAll, type LineStatsItem } from '@/api/stats'

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
  lineId: undefined as number | undefined,
})

// ===== 线路下拉选项 =====
const lineOptions = ref<Array<{ label: string; value: number }>>([])

async function loadOptions() {
  try {
    const lines: any[] = await getLineAll()
    lineOptions.value = (lines || []).map((l: any) => ({
      label: l.line_name ? `${l.line_name} (${l.line_code || l.id})` : `#${l.id}`,
      value: l.id,
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
  const failedCalls = list.reduce((s, r) => s + (r.failed_calls || 0), 0)
  return {
    totalCalls: totalCalls.toLocaleString(),
    answeredCalls: answeredCalls.toLocaleString(),
    failedCalls: failedCalls.toLocaleString(),
    successRate: totalCalls > 0 ? ((answeredCalls / totalCalls) * 100).toFixed(1) + '%' : '0.0%',
    lineCount: list.length,
  }
})

// ===== 明细表格 =====
const columns = [
  { title: '线路编码', key: 'line_code', width: 120 },
  { title: '线路名称', key: 'line_name', width: 160 },
  { title: '总呼叫', key: 'total_calls', width: 100, sorter: (a: LineStatsItem, b: LineStatsItem) => a.total_calls - b.total_calls },
  { title: '接通数', key: 'answered_calls', width: 100 },
  { title: '失败数', key: 'failed_calls', width: 100 },
  {
    title: '成功率',
    key: 'success_rate',
    width: 100,
    render: (row: LineStatsItem) => (row.success_rate || 0).toFixed(1) + '%',
  },
]

const tableData = ref<LineStatsItem[]>([])

// 关键字经"查询"按钮应用，过滤在接口结果集上本地完成
const appliedKeyword = ref('')

// 展示结果集：按线路编码/名称过滤，汇总卡、对比图、明细表共用同一结果
const displayList = computed(() => {
  const kw = appliedKeyword.value.trim()
  if (!kw) return tableData.value
  return tableData.value.filter(
    (r) =>
      (r.line_code || '').includes(kw) ||
      (r.line_name || '').includes(kw),
  )
})

watch(displayList, (list) => renderBarChart(list))

// 接口返回全量数组，表格前端本地分页
const pagination = reactive({
  pageSize: 10,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
})

// ===== 对比图 =====
const barChartRef = ref<HTMLElement>()
let barChart: echarts.ECharts | null = null

function renderBarChart(list: LineStatsItem[]) {
  if (!barChartRef.value) return
  if (!barChart) barChart = echarts.init(barChartRef.value)
  if (list.length === 0) {
    barChart.setOption(
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
  barChart.setOption(
    {
      tooltip: {
        trigger: 'axis',
        backgroundColor: '#ffffff',
        borderColor: '#e8ecf1',
        textStyle: { color: '#1f2937' },
        axisPointer: { type: 'shadow' },
      },
      legend: { data: ['总呼叫', '接通'], textStyle: { color: '#6b7280' }, right: 10 },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'category',
        data: list.map((r) => r.line_name || r.line_code),
        axisLine: { lineStyle: { color: '#e8ecf1' } },
        axisLabel: { color: '#6b7280', rotate: list.length > 6 ? 15 : 0 },
      },
      yAxis: {
        type: 'value',
        axisLine: { lineStyle: { color: '#e8ecf1' } },
        axisLabel: { color: '#9ca3af' },
        splitLine: { lineStyle: { color: '#f3f4f6' } },
      },
      series: [
        {
          name: '总呼叫',
          type: 'bar',
          data: list.map((r) => r.total_calls),
          itemStyle: { color: '#18a058', borderRadius: [4, 4, 0, 0] },
        },
        {
          name: '接通',
          type: 'bar',
          data: list.map((r) => r.answered_calls),
          itemStyle: { color: '#2080f0', borderRadius: [4, 4, 0, 0] },
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
    const res = await getLineStats({
      start_date: queryForm.dateRange ? formatDate(queryForm.dateRange[0]) : undefined,
      end_date: queryForm.dateRange ? formatDate(queryForm.dateRange[1]) : undefined,
      line_id: queryForm.lineId,
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
  queryForm.lineId = undefined
  loadData()
}

function handleResize() {
  barChart?.resize()
}

onMounted(() => {
  loadOptions()
  loadData()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  barChart?.dispose()
  barChart = null
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped lang="css">
/* 统计页内容块多：自然流排列，页面在内容区整体滚动，明细表按数据自然展开 */
.stats-line-page {
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
  height: 360px;
}
</style>
