<template>
  <div class="dashboard">
    <!-- 统计卡片 -->
    <n-row :gutter="16" class="stats-cards">
      <n-col :span="6" v-for="(card, index) in statCards" :key="index">
        <n-card class="stat-card" :bordered="false">
          <div class="stat-card-content">
            <div class="stat-info">
              <div class="stat-label">{{ card.label }}</div>
              <div class="stat-value" :style="{ color: card.color }">
                {{ card.value }}
              </div>
            </div>
            <div class="stat-icon" :style="{ background: card.bgColor }">
              <n-icon size="28" :style="{ color: card.iconColor }">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <path :d="card.iconPath" />
                </svg>
              </n-icon>
            </div>
          </div>
        </n-card>
      </n-col>
    </n-row>

    <!-- 图表区 -->
    <n-row :gutter="16" class="charts-row">
      <!-- 通话趋势 -->
      <n-col :span="16">
        <n-card title="通话趋势" :bordered="false" class="chart-card">
          <template #header-extra>
            <n-radio-group v-model:value="trendPeriod" size="small">
              <n-radio-button value="day">今日</n-radio-button>
              <n-radio-button value="week">本周</n-radio-button>
              <n-radio-button value="month">本月</n-radio-button>
            </n-radio-group>
          </template>
          <div ref="trendChartRef" class="chart-container"></div>
        </n-card>
      </n-col>

      <!-- 座席状态分布 -->
      <n-col :span="8">
        <n-card title="座席状态分布" :bordered="false" class="chart-card">
          <div ref="agentChartRef" class="chart-container chart-pie"></div>
        </n-card>
      </n-col>
    </n-row>

    <!-- 队列实时状态 -->
    <n-row :gutter="16" class="charts-row">
      <n-col :span="24">
        <n-card title="队列实时状态" :bordered="false" class="chart-card">
          <n-data-table
            :columns="queueColumns"
            :data="queueData"
            :pagination="false"
            size="small"
          />
        </n-card>
      </n-col>
    </n-row>

    <!-- 实时通话 -->
    <n-card title="实时通话" :bordered="false" class="chart-card">
      <template #header-extra>
        <n-tag type="success" size="small">
          <span class="live-dot"></span>
          实时更新
        </n-tag>
      </template>
      <n-data-table
        :columns="callColumns"
        :data="activeCalls"
        :pagination="false"
        size="small"
      />
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, h } from 'vue'
import * as echarts from 'echarts'
import { useUserStore } from '@/store/modules/user'
import { getRealtimeStats, getTrendStats, type TrendStatsData } from '@/api/stats'
import { getAllQueueStats } from '@/api/queue'
import { getActiveCalls, type ActiveCallInfo } from '@/api/callctrl'
import type { QueueInfo } from '@/types/queue'
import { formatDuration, formatDate } from '@/utils/format'

const userStore = useUserStore()

// 实时统计返回结构（与后端 model.RealtimeStats 的 JSON 字段一致）
interface RealtimeStatsData {
  total_agents: number
  online_agents: number
  ready_agents: number
  busy_agents: number
  rest_agents: number
  offline_agents: number
  wrap_up_agents: number
  total_queues: number
  active_queues: number
  waiting_calls: number
  talking_calls: number
  total_lines: number
  online_lines: number
  used_channels: number
  total_channels: number
  today_calls: number
  today_answered: number
  today_missed: number
}

// 统计卡片数据
const statCards = ref([
  {
    label: '今日通话量',
    value: '0',
    color: '#18a058',
    bgColor: 'rgba(24, 160, 88, 0.06)',
    iconColor: '#18a058',
    iconPath: 'M21 15a2 2 0 01-2 2H7l-4 4V5a2 2 0 012-2h14a2 2 0 012 2z',
  },
  {
    label: '今日接通率',
    value: '--',
    color: '#2080f0',
    bgColor: 'rgba(32, 128, 240, 0.06)',
    iconColor: '#2080f0',
    iconPath: 'M9 12l2 2 4-4M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10z',
  },
  {
    label: '排队中',
    value: '0',
    color: '#f0a020',
    bgColor: 'rgba(240, 160, 32, 0.06)',
    iconColor: '#f0a020',
    iconPath: 'M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8zM23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75',
  },
  {
    label: '座席在线',
    value: '0/0',
    color: '#7c3aed',
    bgColor: 'rgba(124, 58, 237, 0.06)',
    iconColor: '#7c3aed',
    iconPath: 'M17 21v-2a4 4 0 00-4-4H5a4 4 0 00-4 4v2M9 11a4 4 0 100-8 4 4 0 000 8zM23 21v-2a4 4 0 00-3-3.87M16 3.13a4 4 0 010 7.75',
  },
])

const trendPeriod = ref('day')

// 队列表格列（字段与 monitor/list 返回的 tb_queue 模型一致）
const queueColumns = [
  { title: '队列名称', key: 'queue_name' },
  { title: '等待中', key: 'wait_num', width: 90 },
  { title: '就绪座席', key: 'agent_ready_num', width: 100 },
  { title: '忙碌座席', key: 'agent_busy_num', width: 100 },
  {
    title: '最大等待时间',
    key: 'max_wait_time',
    width: 120,
    render: (row: QueueInfo) => `${row.max_wait_time} 秒`,
  },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row: QueueInfo) => {
      const enabled = row.status === 1
      return h(
        'span',
        { style: { color: enabled ? '#18a058' : '#9ca3af' } },
        enabled ? '启用' : '禁用'
      )
    },
  },
]

const queueData = ref<QueueInfo[]>([])

// 实时通话列（字段与 callctrl model.CallInfo 一致）
const callColumns = [
  { title: '通话ID', key: 'callId', ellipsis: { tooltip: true } },
  {
    title: '类型',
    key: 'callType',
    width: 70,
    render: (row: ActiveCallInfo) => {
      const texts: Record<string, string> = { inbound: '呼入', outbound: '呼出', internal: '内部' }
      return texts[row.callType] || row.callType
    },
  },
  { title: '主叫', key: 'caller', width: 120 },
  { title: '被叫', key: 'called', width: 120 },
  {
    title: '座席',
    key: 'agentName',
    width: 90,
    render: (row: ActiveCallInfo) => row.agentName || row.extension || '-',
  },
  {
    title: '队列',
    key: 'queueId',
    width: 100,
    render: (row: ActiveCallInfo) => row.queueId || '-',
  },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row: ActiveCallInfo) => {
      const texts: Record<string, string> = {
        ringing: '振铃中',
        answered: '已接听',
        bridged: '通话中',
        held: '保持中',
      }
      return texts[row.status] || row.status
    },
  },
  {
    title: '通话时长',
    key: 'duration',
    width: 90,
    render: (row: ActiveCallInfo) => {
      const base = row.answerTime || row.startTime
      return base ? formatDuration(Math.floor(Date.now() / 1000) - base) : '-'
    },
  },
]

const activeCalls = ref<ActiveCallInfo[]>([])

// 图表引用
const trendChartRef = ref<HTMLElement>()
const agentChartRef = ref<HTMLElement>()

let trendChart: echarts.ECharts | null = null
let agentChart: echarts.ECharts | null = null

// 渲染通话趋势图；数据为空时展示空态文案
function renderTrendChart(data: TrendStatsData | null) {
  if (!trendChart) return
  const empty = !data || data.dates.length === 0
  trendChart.setOption({
    tooltip: {
      trigger: 'axis',
      backgroundColor: '#ffffff',
      borderColor: '#e8ecf1',
      textStyle: { color: '#1f2937' },
    },
    legend: {
      data: ['呼入', '呼出'],
      textStyle: { color: '#6b7280' },
      right: 10,
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '3%',
      containLabel: true,
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: data?.dates || [],
      axisLine: { lineStyle: { color: '#e8ecf1' } },
      axisLabel: { color: '#9ca3af' },
    },
    yAxis: {
      type: 'value',
      minInterval: 1,
      axisLine: { lineStyle: { color: '#e8ecf1' } },
      axisLabel: { color: '#9ca3af' },
      splitLine: { lineStyle: { color: '#f3f4f6' } },
    },
    series: [
      {
        name: '呼入',
        type: 'line',
        smooth: true,
        data: data?.inbound || [],
        itemStyle: { color: '#18a058' },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(24, 160, 88, 0.25)' },
            { offset: 1, color: 'rgba(0, 184, 148, 0.02)' },
          ]),
        },
      },
      {
        name: '呼出',
        type: 'line',
        smooth: true,
        data: data?.outbound || [],
        itemStyle: { color: '#2080f0' },
        areaStyle: {
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(32, 128, 240, 0.25)' },
            { offset: 1, color: 'rgba(9, 132, 227, 0.02)' },
          ]),
        },
      },
    ],
    graphic: empty
      ? [
          {
            type: 'text',
            left: 'center',
            top: 'middle',
            style: { text: '暂无数据', fill: '#9ca3af', fontSize: 14 },
          },
        ]
      : [],
  })
}

// 渲染座席状态分布饼图（取实时统计的座席工作状态字段）
function renderAgentChart(stats: RealtimeStatsData | null) {
  if (!agentChart) return
  const items = stats
    ? [
        { value: stats.ready_agents, name: '就绪', itemStyle: { color: '#18a058' } },
        { value: stats.busy_agents, name: '忙碌', itemStyle: { color: '#2080f0' } },
        { value: stats.rest_agents, name: '休息', itemStyle: { color: '#f0a020' } },
        { value: stats.wrap_up_agents, name: '话后处理', itemStyle: { color: '#7c3aed' } },
        { value: stats.offline_agents, name: '离线', itemStyle: { color: '#9ca3af' } },
      ]
    : []
  const total = items.reduce((sum, it) => sum + it.value, 0)
  agentChart.setOption({
    tooltip: {
      trigger: 'item',
      backgroundColor: '#ffffff',
      borderColor: '#e8ecf1',
      textStyle: { color: '#1f2937' },
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
        avoidLabelOverlap: false,
        label: { show: false },
        emphasis: {
          label: { show: true, fontSize: 14, fontWeight: 'bold' },
        },
        labelLine: { show: false },
        data: items,
      },
    ],
    graphic:
      total === 0
        ? [
            {
              type: 'text',
              left: 'center',
              top: 'middle',
              style: { text: '暂无数据', fill: '#9ca3af', fontSize: 14 },
            },
          ]
        : [],
  })
}

// 统计卡片取实时统计字段
function applyRealtimeStats(stats: RealtimeStatsData) {
  statCards.value[0].value = String(stats.today_calls)
  statCards.value[1].value =
    stats.today_calls > 0
      ? `${((stats.today_answered / stats.today_calls) * 100).toFixed(1)}%`
      : '--'
  statCards.value[2].value = String(stats.waiting_calls)
  statCards.value[3].value = `${stats.online_agents}/${stats.total_agents}`
  renderAgentChart(stats)
}

// 趋势时间范围：今日/本周（周一至今）/本月（1 号至今）
function trendRange(): { start_date: string; end_date: string } {
  const now = new Date()
  const end = formatDate(now)
  let start = end
  if (trendPeriod.value === 'week') {
    const monday = new Date(now)
    const day = now.getDay() || 7
    monday.setDate(now.getDate() - day + 1)
    start = formatDate(monday)
  } else if (trendPeriod.value === 'month') {
    start = formatDate(new Date(now.getFullYear(), now.getMonth(), 1))
  }
  return { start_date: start, end_date: end }
}

// 拉取趋势数据并渲染
async function loadTrend() {
  try {
    const data = await getTrendStats(trendRange())
    renderTrendChart(data)
  } catch {
    // 错误提示由拦截器统一处理，图表保留上一帧
  }
}

// 拉取全部实时数据；单项失败互不影响，页面保留上一帧
async function loadDashboard() {
  const tenantId = String(userStore.userInfo?.tenant_id ?? 0)
  const [statsRes, queuesRes, callsRes] = await Promise.allSettled([
    getRealtimeStats(),
    getAllQueueStats(),
    getActiveCalls({ tenantId, page: 1, pageSize: 5 }),
  ])
  if (statsRes.status === 'fulfilled') {
    applyRealtimeStats(statsRes.value as unknown as RealtimeStatsData)
  }
  if (queuesRes.status === 'fulfilled') {
    queueData.value = queuesRes.value || []
  }
  if (callsRes.status === 'fulfilled') {
    activeCalls.value = callsRes.value.list || []
  }
}

function handleResize() {
  trendChart?.resize()
  agentChart?.resize()
}

let refreshTimer: number | undefined

watch(trendPeriod, () => {
  loadTrend()
})

onMounted(() => {
  if (trendChartRef.value) trendChart = echarts.init(trendChartRef.value)
  if (agentChartRef.value) agentChart = echarts.init(agentChartRef.value)
  renderTrendChart(null)
  renderAgentChart(null)
  loadDashboard()
  loadTrend()
  // 30 秒定时刷新真实取数
  refreshTimer = window.setInterval(() => {
    loadDashboard()
    loadTrend()
  }, 30000)
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  if (refreshTimer) clearInterval(refreshTimer)
  trendChart?.dispose()
  agentChart?.dispose()
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped lang="css">
/* 各块垂直间距统一由 gap 提供：n-row 根元素带内联 margin 缩写，margin-bottom 会被其覆盖 */
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.stat-card-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.stat-info {
  flex: 1;
}

.stat-label {
  font-size: 13px;
  color: #9ca3af;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  line-height: 1.2;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

/* n-col 经 flex 拉伸等高；gutter 会在卡外包一层 div，贯通该层让卡片填满列高，并排卡片底边齐平 */
.dashboard :deep(.n-col) {
  display: flex;
  flex-direction: column;
}

.dashboard :deep(.n-col > div) {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.dashboard :deep(.n-col .n-card) {
  flex: 1;
}

.chart-container {
  width: 100%;
  height: 300px;
}

.live-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  background: #18a058;
  border-radius: 50%;
  margin-right: 6px;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}
</style>
