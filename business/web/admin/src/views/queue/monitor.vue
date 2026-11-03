<template>
  <div class="queue-monitor-page">
    <!-- 实时数据卡片 -->
    <div class="stat-cards">
      <n-card :bordered="false" class="stat-card">
        <div class="stat-content">
          <div class="stat-label">当前等待人数</div>
          <div class="stat-value">{{ statData.waitingCount }}</div>
        </div>
        <div class="stat-icon" style="background: rgba(240, 160, 32, 0.1)">
          <svg viewBox="0 0 24 24" fill="none" stroke="#f0a020" stroke-width="2">
            <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
            <circle cx="9" cy="7" r="4"/>
            <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
            <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
          </svg>
        </div>
      </n-card>

      <n-card :bordered="false" class="stat-card">
        <div class="stat-content">
          <div class="stat-label">通话中座席</div>
          <div class="stat-value">{{ statData.talkingAgents }}</div>
        </div>
        <div class="stat-icon" style="background: rgba(24, 160, 88, 0.1)">
          <svg viewBox="0 0 24 24" fill="none" stroke="#18a058" stroke-width="2">
            <path d="M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z"/>
          </svg>
        </div>
      </n-card>

      <n-card :bordered="false" class="stat-card">
        <div class="stat-content">
          <div class="stat-label">空闲座席</div>
          <div class="stat-value">{{ statData.idleAgents }}</div>
        </div>
        <div class="stat-icon" style="background: rgba(32, 128, 240, 0.1)">
          <svg viewBox="0 0 24 24" fill="none" stroke="#2080f0" stroke-width="2">
            <path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/>
            <circle cx="12" cy="7" r="4"/>
          </svg>
        </div>
      </n-card>

      <n-card :bordered="false" class="stat-card">
        <div class="stat-content">
          <div class="stat-label">启用队列数</div>
          <div class="stat-value">{{ statData.activeQueues }}</div>
        </div>
        <div class="stat-icon" style="background: rgba(124, 58, 237, 0.1)">
          <svg viewBox="0 0 24 24" fill="none" stroke="#7c3aed" stroke-width="2">
            <circle cx="12" cy="12" r="10"/>
            <polyline points="12 6 12 12 16 14"/>
          </svg>
        </div>
      </n-card>
    </div>

    <!-- 队列选择 + 队列详情 -->
    <div class="monitor-content">
      <n-card :bordered="false" class="queue-sidebar">
        <div class="sidebar-title">队列列表</div>
        <n-spin :show="loading && queueList.length === 0">
          <div class="queue-list">
            <div
              v-for="q in queueList"
              :key="q.id"
              class="queue-item"
              :class="{ active: activeQueueId === q.id }"
              @click="selectQueue(q)"
            >
              <div class="queue-info">
                <span class="queue-name">{{ q.queue_name }}</span>
                <span class="queue-badge">{{ q.wait_num }}</span>
              </div>
              <div class="queue-status">
                <span
                  class="status-dot"
                  :style="{ background: q.agent_ready_num > 0 ? '#18a058' : '#9ca3af' }"
                ></span>
                <span>就绪 {{ q.agent_ready_num }} · 忙碌 {{ q.agent_busy_num }}</span>
              </div>
            </div>
            <n-empty v-if="!loading && queueList.length === 0" description="暂无启用队列" size="small" />
          </div>
        </n-spin>
      </n-card>

      <n-card :bordered="false" class="detail-card">
        <div class="card-header">
          <div class="card-title">队列实时详情 - {{ currentQueue?.queue_name || '未选择' }}</div>
          <n-button type="primary" size="small" @click="handleRefresh">
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

        <n-empty v-if="!currentQueue" description="请选择左侧队列" />
        <template v-else>
          <div class="detail-stats">
            <div class="detail-stat">
              <div class="ds-value" style="color: #f0a020">{{ currentQueue.wait_num }}</div>
              <div class="ds-label">等待中</div>
            </div>
            <div class="detail-stat">
              <div class="ds-value" style="color: #18a058">{{ currentQueue.agent_ready_num }}</div>
              <div class="ds-label">就绪座席</div>
            </div>
            <div class="detail-stat">
              <div class="ds-value" style="color: #2080f0">{{ currentQueue.agent_busy_num }}</div>
              <div class="ds-label">忙碌座席</div>
            </div>
          </div>

          <n-descriptions :column="2" label-placement="left" class="queue-desc" bordered>
            <n-descriptions-item label="队列编码">{{ currentQueue.queue_code }}</n-descriptions-item>
            <n-descriptions-item label="队列类型">
              {{ currentQueue.queue_type === 1 ? '呼入' : '呼出' }}
            </n-descriptions-item>
            <n-descriptions-item label="分配策略">{{ strategyText }}</n-descriptions-item>
            <n-descriptions-item label="最大等待时间">{{ currentQueue.max_wait_time }} 秒</n-descriptions-item>
            <n-descriptions-item label="最大等待人数">
              {{ currentQueue.max_wait_num === 0 ? '不限' : currentQueue.max_wait_num }}
            </n-descriptions-item>
            <n-descriptions-item label="会后处理时间">{{ currentQueue.wrap_up_time }} 秒</n-descriptions-item>
          </n-descriptions>
        </template>
      </n-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { getAllQueueStats } from '@/api/queue'
import type { QueueInfo } from '@/types/queue'

const loading = ref(false)
const activeQueueId = ref<number>(0)

// 统计数据（由队列监控列表实时字段聚合）
const statData = reactive({
  waitingCount: 0,
  talkingAgents: 0,
  idleAgents: 0,
  activeQueues: 0,
})

// 队列列表（monitor/list 返回 tb_queue 模型，含实时字段）
const queueList = ref<QueueInfo[]>([])

// 当前选中队列
const currentQueue = computed(() => {
  return queueList.value.find((q) => q.id === activeQueueId.value)
})

const strategyTextMap: Record<string, string> = {
  round_robin: '轮询',
  longest_idle: '最长空闲',
  skill_level: '技能等级',
}

const strategyText = computed(() => {
  if (!currentQueue.value) return '-'
  return strategyTextMap[currentQueue.value.strategy] || currentQueue.value.strategy
})

// 拉取队列监控列表；请求失败保留上一帧数据（拦截器已统一提示）
async function loadData(showLoading = true) {
  if (showLoading) loading.value = true
  try {
    const list = await getAllQueueStats()
    queueList.value = list || []
    statData.activeQueues = queueList.value.length
    statData.waitingCount = queueList.value.reduce((sum, q) => sum + (q.wait_num || 0), 0)
    statData.talkingAgents = queueList.value.reduce((sum, q) => sum + (q.agent_busy_num || 0), 0)
    statData.idleAgents = queueList.value.reduce((sum, q) => sum + (q.agent_ready_num || 0), 0)
    // 保持选中态；选中项不存在时回退到第一个队列
    if (!queueList.value.some((q) => q.id === activeQueueId.value)) {
      activeQueueId.value = queueList.value[0]?.id ?? 0
    }
  } catch {
    // 错误提示由拦截器统一处理
  } finally {
    loading.value = false
  }
}

function selectQueue(q: QueueInfo) {
  activeQueueId.value = q.id
}

function handleRefresh() {
  loadData()
}

let pollTimer: number | undefined

onMounted(() => {
  loadData()
  // 8 秒轮询真实重取
  pollTimer = window.setInterval(() => loadData(false), 8000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})
</script>

<!-- 排队等待通话的指定座席、提升优先级、移出队列操作依赖 ACD 服务的排队监控接口，
     当前管理端网关未暴露该接口，相关入口待接口开放后再提供 -->

<style scoped lang="css">
.queue-monitor-page {
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
  border-radius: 12px;
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

/* 队列侧栏与详情卡填满统计区之下的剩余高度 */
.monitor-content {
  display: flex;
  gap: 16px;
  flex: 1;
  min-height: 0;
}

.queue-sidebar {
  border-radius: 12px;
  width: 240px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
}

.queue-sidebar :deep(.n-card-content) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.sidebar-title {
  font-size: 15px;
  font-weight: 600;
  color: #1f2937;
  margin-bottom: 12px;
}

.queue-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.queue-item {
  padding: 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  border: 1px solid #e8ecf1;
  background: #f9fafb;
}

.queue-item:hover {
  background: rgba(24, 160, 88, 0.06);
  border-color: rgba(24, 160, 88, 0.25);
}

.queue-item.active {
  background: rgba(24, 160, 88, 0.1);
  border-color: #18a058;
}

.queue-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 6px;
}

.queue-name {
  font-size: 14px;
  font-weight: 500;
  color: #1f2937;
}

.queue-badge {
  background: #f0a020;
  color: #ffffff;
  font-size: 12px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 10px;
  min-width: 24px;
  text-align: center;
}

.queue-status {
  font-size: 12px;
  color: #6b7280;
  display: flex;
  align-items: center;
  gap: 6px;
}

.status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.detail-card {
  border-radius: 12px;
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}

.detail-card :deep(.n-card-content) {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.detail-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

.detail-stat {
  background: #f9fafb;
  border: 1px solid #e8ecf1;
  border-radius: 10px;
  padding: 16px;
  text-align: center;
}

.ds-value {
  font-size: 26px;
  font-weight: 700;
  margin-bottom: 6px;
}

.ds-label {
  font-size: 13px;
  color: #6b7280;
}

.queue-desc {
  margin-top: 4px;
}
</style>
