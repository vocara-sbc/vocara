<template>
  <div class="workbench-page">
    <!-- ============ 通话中：中间左右两栏（左工单 / 右客户） ============ -->
    <template v-if="currentCallInfo">
      <div class="call-grid">
        <!-- 左：工单表单 -->
        <n-card :bordered="false" class="call-form-card">
          <template #header>
            <div class="card-header">
              <span>通话工单</span>
              <n-tag size="small" type="warning">待提交</n-tag>
            </div>
          </template>
          <FormRenderer
            v-if="ticketSchema"
            ref="ticketFormRef"
            :schema="ticketSchema"
            :data="ticketInitData"
          />
          <div class="form-action-bar">
            <n-button v-if="ticketSaved" type="success" disabled>已提交</n-button>
            <n-button v-else type="primary" :loading="savingTicket" @click="saveTicketForm">提交工单</n-button>
          </div>
        </n-card>

        <!-- 右：客户表单 -->
        <n-card :bordered="false" class="call-form-card">
          <template #header>
            <div class="card-header">
              <span>客户信息</span>
              <n-tag size="small" type="success">
                <span class="live-dot"></span>通话中
              </n-tag>
            </div>
          </template>
          <FormRenderer
            v-if="customerSchema"
            ref="customerFormRef"
            :schema="customerSchema"
            :data="customerInitData"
          />
          <div class="form-action-bar">
            <n-button v-if="customerSaved" type="success" disabled>已保存</n-button>
            <n-button v-else type="primary" :loading="savingCustomer" @click="saveCustomerForm">保存客户信息</n-button>
          </div>
        </n-card>
      </div>

      <!-- 通话中右侧：最近通话 -->
      <div class="side-area call-side">
        <n-card :bordered="false" class="recent-card">
          <template #header>
            <div class="card-header"><span>最近通话</span></div>
          </template>
          <div class="recent-list">
            <div v-for="(call, idx) in recentCalls.slice(0, 8)" :key="idx" class="recent-item">
              <div class="recent-icon" :class="call.type">
                {{ call.type === 'inbound' ? '↓' : '↑' }}
              </div>
              <div class="recent-info">
                <div class="recent-phone">{{ call.caller }}</div>
                <div class="recent-time">{{ call.time }}</div>
              </div>
              <div class="recent-duration">{{ call.duration }}</div>
            </div>
            <div v-if="recentCalls.length === 0" class="empty-recent">暂无通话记录</div>
          </div>
        </n-card>
      </div>
    </template>

    <!-- ============ 空闲态：历史通话 + 最近通话 ============ -->
    <template v-else>
      <div class="main-area">
        <!-- 历史通话（占满中间，避免空白） -->
        <n-card :bordered="false" class="history-card">
          <template #header>
            <div class="card-header">
              <span>历史通话</span>
              <n-text depth="3" style="font-size: 12px">共 {{ historyList.length }} 条</n-text>
            </div>
          </template>
          <div class="history-list">
            <div v-for="item in historyList" :key="item.id" class="history-item">
              <div class="history-icon" :class="item.type">
                {{ item.type === 'inbound' ? '↓' : '↑' }}
              </div>
              <div class="history-content">
                <div class="history-top">
                  <span class="history-phone">{{ item.phone }}</span>
                  <span class="history-duration">{{ item.duration }}</span>
                </div>
                <div class="history-bottom">
                  <span class="history-time">{{ item.time }}</span>
                  <span class="history-result">{{ item.result }}</span>
                </div>
              </div>
            </div>
            <div v-if="historyList.length === 0" class="empty-history">暂无历史通话</div>
          </div>
        </n-card>
      </div>

      <!-- 空闲态右侧：最近通话 -->
      <div class="side-area">
        <!-- 最近通话 -->
        <n-card :bordered="false" class="recent-card">
          <template #header>
            <div class="card-header">
              <span>最近通话</span>
              <n-text depth="3" style="font-size: 12px; cursor: pointer" @click="$router.push('/records/inbound')">查看全部</n-text>
            </div>
          </template>
          <div class="recent-list">
            <div v-for="(call, idx) in recentCalls.slice(0, 5)" :key="idx" class="recent-item">
              <div class="recent-icon" :class="call.type">
                {{ call.type === 'inbound' ? '↓' : '↑' }}
              </div>
              <div class="recent-info">
                <div class="recent-phone">{{ call.caller }}</div>
                <div class="recent-time">{{ call.time }}</div>
              </div>
              <div class="recent-duration">{{ call.duration }}</div>
            </div>
            <div v-if="recentCalls.length === 0" class="empty-recent">暂无通话记录</div>
          </div>
        </n-card>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useCallStore } from '@/store/modules/call'
import { useUserStore } from '@/store/modules/user'
import { message } from '@/utils/naive'
import { getInboundRecords, getOutboundRecords } from '@/api/call'
import {
  createFormRecord,
  getTemplateSchemaByKey,
  listCustomers,
} from '@/api/form'
import FormRenderer from '@/components/FormSdk/Renderer.vue'

const callStore = useCallStore()
const userStore = useUserStore()

onMounted(() => {
  loadHistory()
})

// ===== 当前通话（客户信息卡数据源：callStore 真实通话 + 同号码历史统计）=====
const currentCallInfo = computed(() => {
  const c = callStore.currentCall
  if (!c) return null
  const phone = callStore.displayNumber || c.callerNum || c.calledNum || ''
  const history = historyList.value.filter((h) => h.phone === phone)
  return {
    phone,
    name: callStore.displayName || '',
    direction: c.direction === 'inbound' ? '呼入' : '呼出',
    queueName: c.queueId ? `队列 #${c.queueId}` : '直接呼叫',
    duration: callStore.formattedDuration,
    historyCount: history.length,
    lastTime: history.length ? history[0].time : '-',
  }
})

// ===== 通话表单（客户信息 + 通话工单）=====
const customerSchema = ref('')
const ticketSchema = ref('')
const customerInitData = ref<Record<string, any>>({})
const ticketInitData = ref<Record<string, any>>({})
const customerFormRef = ref()
const ticketFormRef = ref()
const customerSaved = ref(false)
const ticketSaved = ref(false)
const savingCustomer = ref(false)
const savingTicket = ref(false)

// resolveCustomerInit 按号码查客户主档：命中带出历史信息，未命中仅预填电话
async function resolveCustomerInit(phone: string) {
  if (!phone) return {}
  try {
    const res: any = await listCustomers({ page: 1, page_size: 20, keyword: phone })
    const exact = (res?.list || []).find((c: any) => c.phone === phone)
    if (exact) {
      try { return JSON.parse(exact.data) } catch { return { phone } }
    }
  } catch {
    // 查询失败按新客户处理
  }
  return { phone }
}

// callContext 通话上下文（记录关联通话与座席）
function callContext() {
  const c = callStore.currentCall
  if (!c) return {}
  return {
    call_id: c.callUuid,
    direction: c.direction,
    agent_id: String(c.agentId ?? ''),
    agent_name: userStore.userInfo?.nickname || userStore.userInfo?.username || '',
  }
}

// setupCallForms 通话建立：先取预填数据，再一次性设置 schema + data，
// 保证 RendererApp 挂载时 props.data 已就绪（onMounted 的 initFormData 只执行一次）
async function setupCallForms() {
  const phone = callStore.displayNumber
  customerSaved.value = false
  ticketSaved.value = false
  try {
    // 先并行取两个模板 schema 与客户预填数据
    const [cv, tv, custData]: any[] = await Promise.all([
      getTemplateSchemaByKey('customer'),
      getTemplateSchemaByKey('ticket'),
      resolveCustomerInit(phone),
    ])
    // 同一同步块赋值：RendererApp 在下一帧挂载时即可拿到 schema 与 data
    customerInitData.value = custData || { phone }
    ticketInitData.value = { phone }
    customerSchema.value = cv?.schema || ''
    ticketSchema.value = tv?.schema || ''
  } catch (e: any) {
    message.error('表单加载失败：' + (e?.message || ''))
  }
}

// 是否在通话中（仅 true/false 两态）。
// 注意：不能直接 watch currentCallInfo——通话计时每秒更新 currentCall，
// currentCallInfo 每秒返回新对象会导致每秒重载表单（闪烁+输入丢失）。
const inCall = computed(() => !!callStore.currentCall)
let formsLoaded = false
watch(
  inCall,
  (onCall) => {
    if (onCall && !formsLoaded) {
      formsLoaded = true
      setupCallForms()
    } else if (!onCall && formsLoaded) {
      formsLoaded = false
      customerSchema.value = ''
      ticketSchema.value = ''
      customerSaved.value = false
      ticketSaved.value = false
    }
  },
  { immediate: true },
)

// saveCustomerForm 保存客户信息（提交即 upsert 客户主档）
async function saveCustomerForm() {
  try {
    savingCustomer.value = true
    const ok = await customerFormRef.value?.validate()
    if (!ok) return
    const data = customerFormRef.value?.getData()
    await createFormRecord({ template_key: 'customer', data, ...callContext() })
    customerSaved.value = true
    message.success('客户信息已保存')
  } catch (e: any) {
    if (e?.message) message.error(e.message)
  } finally {
    savingCustomer.value = false
  }
}

// saveTicketForm 保存工单（提交即生成工单，出现在工单管理）
async function saveTicketForm() {
  try {
    savingTicket.value = true
    const ok = await ticketFormRef.value?.validate()
    if (!ok) return
    const data = ticketFormRef.value?.getData()
    await createFormRecord({ template_key: 'ticket', data, ...callContext() })
    ticketSaved.value = true
    message.success('工单已提交')
  } catch (e: any) {
    if (e?.message) message.error(e.message)
  } finally {
    savingTicket.value = false
  }
}

// ===== 历史通话（真实接口：呼入/呼出记录合并取最近 10 条）=====
const historyList = ref<any[]>([])

async function loadHistory() {
  try {
    const [inRes, outRes]: any[] = await Promise.all([
      getInboundRecords({ page: 1, page_size: 10 }),
      getOutboundRecords({ page: 1, page_size: 10 }),
    ])
    const inList = (inRes?.list || []).map((r: any) => mapRecord(r, 'inbound'))
    const outList = (outRes?.list || []).map((r: any) => mapRecord(r, 'outbound'))
    historyList.value = [...inList, ...outList]
      .sort((a: any, b: any) => b.ts - a.ts)
      .slice(0, 10)
  } catch (e) {
    // 历史记录加载失败保持空态
  }
}

function mapRecord(r: any, type: string) {
  const ts = new Date(r.start_time || r.startTime || 0).getTime()
  return {
    id: r.id || r.call_id || r.callId,
    phone: type === 'inbound' ? r.caller_number || r.caller || r.callerNum : r.called_number || r.called || r.calledNum,
    type,
    ts,
    time: formatTime(ts),
    duration: formatDuration(r.talk_duration ?? r.talkDuration ?? r.duration ?? 0),
    result: r.hangup_cause || r.hangupCause || '-',
  }
}

// ===== 最近通话（callStore 真实会话历史，空态展示）=====
const recentCalls = computed(() => {
  return callStore.callHistory.slice(0, 5).map((call, i) => ({
    id: i,
    caller: call.direction === 'inbound' ? call.callerNum : call.calledNum,
    type: call.direction,
    time: new Date(call.startTime || 0).toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }),
    duration: formatDuration(call.duration),
  }))
})

function formatDuration(sec: number): string {
  const m = Math.floor(sec / 60).toString().padStart(2, '0')
  const s = (sec % 60).toString().padStart(2, '0')
  return `${m}:${s}`
}

function formatTime(ts: number): string {
  if (!ts) return '-'
  const d = new Date(ts)
  const today = new Date()
  const sameDay = d.toDateString() === today.toDateString()
  const hm = d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  if (sameDay) return `今天 ${hm}`
  return `${d.getMonth() + 1}月${d.getDate()}日 ${hm}`
}
</script>

<style scoped lang="css">
.workbench-page {
  display: flex;
  gap: 16px;
  height: calc(100vh - 60px - 40px - 32px - 40px); /* header + tabs + padding + footer */
  min-height: 600px;
}

/* ===== 左侧主区 ===== */
.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

/* ===== 通话中：中间左右两栏 ===== */
.call-grid {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
  min-width: 0;
}
.call-form-card {
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
.call-form-card :deep(.n-card-content) {
  overflow-y: auto;
}
.call-side {
  width: 300px;
  flex-shrink: 0;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 14px;
}

.live-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  background: currentColor;
  border-radius: 50%;
  margin-right: 4px;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

.form-action-bar {
  margin-top: 12px;
  display: flex;
  justify-content: flex-end;
}

/* 历史通话 */
.history-card {
  flex: 1;
  border-radius: 12px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.history-card :deep(.n-card-content) {
  flex: 1;
  overflow-y: auto;
}

.history-list {
  display: flex;
  flex-direction: column;
}

.history-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-radius: 8px;
}

.history-icon {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 14px;
  flex-shrink: 0;
}

.history-icon.inbound {
  background: #f0fdf4;
  color: #10b981;
}

.history-icon.outbound {
  background: #eff6ff;
  color: #3b82f6;
}

.history-content {
  flex: 1;
  min-width: 0;
}

.history-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.history-phone {
  font-size: 14px;
  font-weight: 500;
  color: #1f2937;
}

.history-duration {
  font-size: 13px;
  color: #6b7280;
  font-family: 'Courier New', monospace;
}

.history-bottom {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: #9ca3af;
}

.history-result {
  font-size: 12px;
  color: #6b7280;
}

.empty-history {
  text-align: center;
  padding: 32px 0;
  color: #9ca3af;
  font-size: 13px;
}

/* ===== 右侧边栏 ===== */
.side-area {
  width: 340px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-y: auto;
  padding-right: 2px;
}

/* 最近通话 */
.recent-card {
  border-radius: 12px;
}

.recent-list {
  display: flex;
  flex-direction: column;
}

.recent-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px solid #f3f4f6;
}

.recent-item:last-child {
  border-bottom: none;
}

.recent-icon {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 600;
  flex-shrink: 0;
}

.recent-icon.inbound {
  background: #f0fdf4;
  color: #10b981;
}

.recent-icon.outbound {
  background: #eff6ff;
  color: #3b82f6;
}

.recent-info {
  flex: 1;
  min-width: 0;
}

.recent-phone {
  font-size: 13px;
  color: #1f2937;
  margin-bottom: 2px;
}

.recent-time {
  font-size: 11px;
  color: #9ca3af;
}

.recent-duration {
  font-size: 12px;
  color: #6b7280;
  font-family: 'Courier New', monospace;
}

.empty-recent {
  text-align: center;
  padding: 20px 0;
  color: #9ca3af;
  font-size: 12px;
}

/* 滚动条美化 */
.side-area::-webkit-scrollbar,
.history-card :deep(.n-card-content)::-webkit-scrollbar {
  width: 4px;
}

.side-area::-webkit-scrollbar-thumb,
.history-card :deep(.n-card-content)::-webkit-scrollbar-thumb {
  background: #d1d5db;
  border-radius: 2px;
}

.side-area::-webkit-scrollbar-track,
.history-card :deep(.n-card-content)::-webkit-scrollbar-track {
  background: transparent;
}

/* 响应式：窄屏收窄右侧栏宽度 */
@media (max-width: 1200px) {
  .side-area {
    width: 300px;
  }
}
</style>
