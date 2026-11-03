<template>
  <div class="stats-trend-page">
    <!-- 日期选择 -->
    <n-card :bordered="false" class="filter-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="日期范围">
          <n-date-picker
            v-model:value="queryForm.dateRange"
            type="daterange"
            clearable
            style="width: 260px"
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

    <!-- 主图表：按天粒度的呼入/呼出量 -->
    <n-card title="呼叫量趋势分析" :bordered="false" class="chart-card">
      <n-spin :show="loading">
        <div ref="mainChartRef" class="chart-container"></div>
      </n-spin>
    </n-card>

    <!-- 峰值分析：由趋势数据计算得出 -->
    <n-card title="峰值与区间概览" :bordered="false" class="table-card">
      <n-row :gutter="16">
        <n-col :span="8" v-for="(item, index) in peakData" :key="index">
          <div class="peak-card">
            <div class="peak-label">{{ item.label }}</div>
            <div class="peak-value">{{ item.value }}</div>
            <div class="peak-time">{{ item.time }}</div>
          </div>
        </n-col>
      </n-row>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted } from 'vue'
import * as echarts from 'echarts'
import { getTrendStats } from '@/api/stats'

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
  dateRange: defaultRange() as [number, number] | null,
})

// 峰值与区间概览卡片，由接口返回的日期序列计算
const peakData = ref([
  { label: '峰值日呼叫量', value: '0 次', time: '峰值日期：-' },
  { label: '区间总呼叫量', value: '0 次', time: '呼入 0 / 呼出 0' },
  { label: '日均呼叫量', value: '0 次', time: '统计区间：-' },
])

const mainChartRef = ref<HTMLElement>()
let mainChart: echarts.ECharts | null = null

function renderMainChart(dates: string[], inbound: number[], outbound: number[]) {
  if (!mainChartRef.value) return
  if (!mainChart) mainChart = echarts.init(mainChartRef.value)
  if (dates.length === 0) {
    mainChart.setOption(
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
  const total = dates.map((_, i) => (inbound[i] || 0) + (outbound[i] || 0))
  mainChart.setOption(
    {
      tooltip: {
        trigger: 'axis',
        backgroundColor: '#ffffff',
        borderColor: '#e8ecf1',
        textStyle: { color: '#1f2937' },
      },
      legend: {
        data: ['呼入量', '呼出量', '总呼叫量'],
        textStyle: { color: '#6b7280' },
        right: 10,
      },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: {
        type: 'category',
        boundaryGap: false,
        data: dates,
        axisLine: { lineStyle: { color: '#e8ecf1' } },
        axisLabel: { color: '#9ca3af' },
      },
      yAxis: {
        type: 'value',
        axisLine: { lineStyle: { color: '#e8ecf1' } },
        axisLabel: { color: '#9ca3af' },
        splitLine: { lineStyle: { color: '#f3f4f6' } },
      },
      series: [
        {
          name: '呼入量',
          type: 'line',
          smooth: true,
          data: inbound,
          itemStyle: { color: '#18a058' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(24, 160, 88, 0.25)' },
              { offset: 1, color: 'rgba(0, 184, 148, 0.02)' },
            ]),
          },
        },
        {
          name: '呼出量',
          type: 'line',
          smooth: true,
          data: outbound,
          itemStyle: { color: '#2080f0' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(32, 128, 240, 0.25)' },
              { offset: 1, color: 'rgba(9, 132, 227, 0.02)' },
            ]),
          },
        },
        {
          name: '总呼叫量',
          type: 'line',
          smooth: true,
          data: total,
          itemStyle: { color: '#7c3aed' },
          lineStyle: { type: 'dashed' },
        },
      ],
    },
    true
  )
}

function renderPeakCards(dates: string[], inbound: number[], outbound: number[], startDate: string, endDate: string) {
  if (dates.length === 0) {
    peakData.value = [
      { label: '峰值日呼叫量', value: '0 次', time: '峰值日期：-' },
      { label: '区间总呼叫量', value: '0 次', time: '呼入 0 / 呼出 0' },
      { label: '日均呼叫量', value: '0 次', time: '统计区间：-' },
    ]
    return
  }
  const totalSeries = dates.map((_, i) => (inbound[i] || 0) + (outbound[i] || 0))
  const totalCalls = totalSeries.reduce((s, v) => s + v, 0)
  const inboundSum = inbound.reduce((s, v) => s + (v || 0), 0)
  const outboundSum = outbound.reduce((s, v) => s + (v || 0), 0)
  let peakIdx = 0
  totalSeries.forEach((v, i) => {
    if (v > totalSeries[peakIdx]) peakIdx = i
  })
  peakData.value = [
    {
      label: '峰值日呼叫量',
      value: `${totalSeries[peakIdx].toLocaleString()} 次`,
      time: `峰值日期：${dates[peakIdx]}`,
    },
    {
      label: '区间总呼叫量',
      value: `${totalCalls.toLocaleString()} 次`,
      time: `呼入 ${inboundSum.toLocaleString()} / 呼出 ${outboundSum.toLocaleString()}`,
    },
    {
      label: '日均呼叫量',
      value: `${Math.round(totalCalls / dates.length).toLocaleString()} 次`,
      time: `统计区间：${startDate} ~ ${endDate}`,
    },
  ]
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
    const res = await getTrendStats({
      start_date: queryForm.dateRange ? formatDate(queryForm.dateRange[0]) : undefined,
      end_date: queryForm.dateRange ? formatDate(queryForm.dateRange[1]) : undefined,
    })
    const dates = res?.dates || []
    const inbound = res?.inbound || []
    const outbound = res?.outbound || []
    renderMainChart(dates, inbound, outbound)
    renderPeakCards(dates, inbound, outbound, res?.start_date || '-', res?.end_date || '-')
  } catch (e) {
    // 报错由请求拦截器统一提示，页面保持空态
    renderMainChart([], [], [])
    renderPeakCards([], [], [], '-', '-')
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  loadData()
}

function handleReset() {
  queryForm.dateRange = defaultRange()
  loadData()
}

function handleResize() {
  mainChart?.resize()
}

onMounted(() => {
  loadData()
  window.addEventListener('resize', handleResize)
})

onUnmounted(() => {
  mainChart?.dispose()
  mainChart = null
  window.removeEventListener('resize', handleResize)
})
</script>

<style scoped lang="css">
.stats-trend-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
}

/* 筛选卡与峰值卡固定高度，主图表卡填满剩余高度 */
.filter-card {
  flex-shrink: 0;
}

.table-card {
  flex-shrink: 0;
}

.chart-card {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.chart-card :deep(.n-card-content) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.chart-card :deep(.n-spin-container),
.chart-card :deep(.n-spin-content) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.chart-container {
  width: 100%;
  flex: 1;
  min-height: 320px;
}

.peak-card {
  padding: 20px;
  background: #f9fafb;
  border-radius: 10px;
  text-align: center;
  border: 1px solid #e8ecf1;
}

.peak-label {
  font-size: 13px;
  color: #6b7280;
  margin-bottom: 10px;
}

.peak-value {
  font-size: 24px;
  font-weight: 700;
  color: #18a058;
  margin-bottom: 8px;
}

.peak-time {
  font-size: 12px;
  color: #9ca3af;
}
</style>
