<template>
  <div class="license-page">
    <!-- 授权状态概览 -->
    <n-card :bordered="false">
      <div class="status-overview">
        <div class="status-badge" :class="`status-badge--${info.state}`">
          <n-icon size="30">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
            </svg>
          </n-icon>
        </div>
        <div class="status-main">
          <div class="status-line">
            <n-tag :type="statusTagType" size="large" round>{{ statusText }}</n-tag>
            <span class="customer-name">{{ info.customer || '未授权客户' }}</span>
          </div>
          <div class="status-message">{{ info.message || '系统尚未加载授权信息' }}</div>
          <div class="status-sub">{{ validityText }}　·　{{ daysText }}</div>
        </div>
        <div class="status-actions">
          <n-button type="primary" @click="openImport">更新授权</n-button>
          <n-button :loading="reloading" @click="handleReload">重新加载</n-button>
        </div>
      </div>
    </n-card>

    <!-- 授权详情 -->
    <n-card :bordered="false" title="授权详情">
      <n-descriptions bordered :column="3" label-placement="left" size="medium">
        <n-descriptions-item label="授权编号">{{ info.license_id || '-' }}</n-descriptions-item>
        <n-descriptions-item label="客户名称">{{ info.customer || '-' }}</n-descriptions-item>
        <n-descriptions-item label="产品版本">{{ editionText }}</n-descriptions-item>
        <n-descriptions-item label="授权类型">{{ licenseTypeText }}</n-descriptions-item>
        <n-descriptions-item label="签发时间">{{ fmtDateTime(info.issued_at) }}</n-descriptions-item>
        <n-descriptions-item label="宽限天数">{{ info.grace_days ?? '-' }} 天</n-descriptions-item>
        <n-descriptions-item label="生效时间">{{ fmtDateTime(info.not_before) }}</n-descriptions-item>
        <n-descriptions-item label="到期时间">
          {{ info.perpetual ? '永久' : fmtDateTime(info.expire_at) }}
        </n-descriptions-item>
        <n-descriptions-item label="绑定模式">{{ bindingModeText }}</n-descriptions-item>
      </n-descriptions>
    </n-card>

    <!-- 功能模块与容量配额 -->
    <n-grid cols="1 m:2" responsive="screen" :x-gap="16" :y-gap="16">
      <n-gi>
        <n-card :bordered="false" title="功能模块" class="grid-card">
          <div class="module-list">
            <div v-for="m in moduleMeta" :key="m.key" class="module-item">
              <span class="module-name">{{ m.name }}</span>
              <n-tag :type="moduleEnabled(m.key) ? 'success' : 'default'" size="small" round>
                {{ moduleEnabled(m.key) ? '已授权' : '未授权' }}
              </n-tag>
            </div>
          </div>
        </n-card>
      </n-gi>
      <n-gi>
        <n-card :bordered="false" title="容量配额" class="grid-card">
          <n-grid :cols="2" :x-gap="12">
            <n-gi v-for="l in limitMeta" :key="l.key">
              <div class="limit-item">
                <div class="limit-value">{{ limitDisplay(l.key) }}</div>
                <div class="limit-label">{{ l.name }}</div>
              </div>
            </n-gi>
          </n-grid>
        </n-card>
      </n-gi>
    </n-grid>

    <!-- 设备绑定 -->
    <n-card :bordered="false" title="设备绑定">
      <div class="bind-row">
        <div class="bind-info">
          <div class="bind-label">安装标识</div>
          <div class="bind-value monospace">{{ info.install_id || '-' }}</div>
        </div>
        <n-button size="small" @click="copyText(info.install_id)">复制</n-button>
      </div>
      <div class="bind-row">
        <div class="bind-info">
          <div class="bind-label">本机机器码</div>
          <div class="bind-value monospace">{{ info.local_machine || '-' }}</div>
        </div>
        <n-button size="small" @click="copyText(info.local_machine)">复制</n-button>
      </div>
      <div class="bind-row">
        <div class="bind-info">
          <div class="bind-label">授权绑定指纹</div>
          <div class="bind-value monospace">
            {{ (info.fingerprint?.values && info.fingerprint.values.join('，')) || '-' }}
          </div>
        </div>
        <n-tag :type="bindingMatch ? 'success' : 'error'" size="small">
          {{ bindingMatch ? '匹配' : '不匹配' }}
        </n-tag>
      </div>
    </n-card>

    <!-- 更新授权弹窗 -->
    <n-modal v-model:show="importVisible">
      <n-card class="import-card" :bordered="false" title="更新授权" role="dialog">
        <n-space vertical :size="12">
          <n-upload
            :show-file-list="false"
            accept=".lic,license,.txt"
            :custom-request="handleFilePick"
          >
            <n-button size="small" secondary>选择 License 文件</n-button>
          </n-upload>
          <n-input
            v-model:value="licenseText"
            type="textarea"
            :rows="8"
            placeholder="将 License 文本粘贴到此处，或选择 .lic 文件自动填充"
            class="license-input"
          />
        </n-space>
        <template #footer>
          <div class="import-footer">
            <n-button @click="importVisible = false">取消</n-button>
            <n-button type="primary" :loading="importing" @click="handleImport">导入并生效</n-button>
          </div>
        </template>
      </n-card>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { message } from '@/utils/naive'
import {
  getLicenseInfo,
  importLicense,
  reloadLicense,
  type LicenseInfo,
} from '@/api/license'

const info = ref<LicenseInfo>({} as LicenseInfo)

// ===== 状态展示 =====
const statusTextMap: Record<string, string> = {
  valid: '授权有效',
  expiring: '即将到期',
  grace: '宽限期',
  expired: '已过期',
  invalid: '授权无效',
}

const statusTagTypeMap: Record<string, 'success' | 'warning' | 'error'> = {
  valid: 'success',
  expiring: 'warning',
  grace: 'error',
  expired: 'error',
  invalid: 'error',
}

const statusText = computed(() => statusTextMap[info.value.state] || '未授权')
const statusTagType = computed(
  () => statusTagTypeMap[info.value.state] || 'error',
)

const editionMap: Record<string, string> = {
  enterprise: '企业版',
  standard: '标准版',
}
const editionText = computed(() => editionMap[info.value.edition || ''] || info.value.edition || '-')

const licenseTypeMap: Record<string, string> = {
  formal: '正式授权',
  trial: '试用授权',
}
const licenseTypeText = computed(
  () => licenseTypeMap[info.value.license_type || ''] || info.value.license_type || '-',
)

const bindingModeMap: Record<string, string> = {
  host: '主机绑定',
  install: '集群绑定',
  none: '不绑定',
}
const bindingModeText = computed(
  () => bindingModeMap[info.value.fingerprint?.mode || ''] || info.value.fingerprint?.mode || '-',
)

const validityText = computed(() => {
  if (info.value.perpetual) return '永久授权'
  if (!info.value.not_before && !info.value.expire_at) return '暂无有效期'
  return `${fmtDate(info.value.not_before)} 至 ${fmtDate(info.value.expire_at)}`
})

const daysText = computed(() => {
  if (info.value.perpetual) return '永久有效'
  if (info.value.state === 'expired') return '已超出宽限期'
  if (info.value.state === 'invalid') return '授权不可用'
  return `剩余 ${info.value.days_left ?? 0} 天`
})

// 绑定指纹与本机是否一致；不绑定模式恒为匹配
const bindingMatch = computed(() => {
  const mode = info.value.fingerprint?.mode
  if (!mode || mode === 'none') return true
  const local = info.value.local_machine
  return !!local && (info.value.fingerprint?.values || []).includes(local)
})

// ===== 功能模块与容量 =====
const moduleMeta = [
  { key: 'record', name: '通话录音' },
  { key: 'supervise', name: '监听耳语' },
  { key: 'predictive_dialer', name: '预测式外呼' },
  { key: 'report', name: '统计报表' },
  { key: 'open_api', name: '开放接口' },
  { key: 'crm_form', name: '客户工单' },
]

const limitMeta = [
  { key: 'agents', name: '座席数' },
  { key: 'concurrent_calls', name: '并发通话' },
  { key: 'lines', name: '线路数' },
  { key: 'tenants', name: '租户数' },
]

function moduleEnabled(key: string) {
  return info.value.modules?.[key] === true
}

function limitDisplay(key: string) {
  const value = (info.value.limits as Record<string, number> | undefined)?.[key]
  if (value === undefined || value === 0) return '不限'
  return String(value)
}

// ===== 导入与重载 =====
const importVisible = ref(false)
const licenseText = ref('')
const importing = ref(false)
const reloading = ref(false)

function openImport() {
  licenseText.value = ''
  importVisible.value = true
}

// 选取文件后读取文本填充输入框，由导入动作统一提交
async function handleFilePick(options: any) {
  const file = options?.file?.file as File | undefined
  if (!file) return
  licenseText.value = await file.text()
}

async function handleImport() {
  const content = licenseText.value.trim()
  if (!content) {
    message.warning('请粘贴 License 文本或选择 License 文件')
    return
  }
  importing.value = true
  try {
    info.value = await importLicense(content)
    message.success('授权已更新并生效')
    importVisible.value = false
  } catch {
    // 错误提示由请求拦截器统一处理
  } finally {
    importing.value = false
  }
}

async function handleReload() {
  reloading.value = true
  try {
    info.value = await reloadLicense()
    message.success('已从磁盘重新加载授权')
  } catch {
    // 错误提示由请求拦截器统一处理
  } finally {
    reloading.value = false
  }
}

async function copyText(text?: string) {
  if (!text) return
  try {
    await navigator.clipboard.writeText(text)
    message.success('已复制到剪贴板')
  } catch {
    message.error('复制失败，请手动选择文本复制')
  }
}

// ===== 时间格式化 =====
function pad(n: number) {
  return String(n).padStart(2, '0')
}

function fmtDateTime(s?: string) {
  if (!s) return '-'
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

function fmtDate(s?: string) {
  if (!s) return '-'
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

onMounted(async () => {
  try {
    info.value = await getLicenseInfo()
  } catch {
    // 错误提示由请求拦截器统一处理
  }
})
</script>

<style scoped>
.license-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* ===== 状态概览 ===== */
.status-overview {
  display: flex;
  align-items: center;
  gap: 20px;
}

.status-badge {
  width: 64px;
  height: 64px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: #fff;
}

.status-badge--valid {
  background: linear-gradient(135deg, #36ad6a, #18a058);
}
.status-badge--expiring {
  background: linear-gradient(135deg, #f7b500, #e8930c);
}
.status-badge--grace,
.status-badge--expired,
.status-badge--invalid {
  background: linear-gradient(135deg, #ef6a5c, #d94335);
}

.status-main {
  flex: 1;
  min-width: 0;
}

.status-line {
  display: flex;
  align-items: center;
  gap: 12px;
}

.customer-name {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.status-message {
  margin-top: 6px;
  font-size: 13px;
  color: #4b5563;
}

.status-sub {
  margin-top: 4px;
  font-size: 12px;
  color: #8a919b;
}

.status-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.grid-card {
  height: 100%;
}

/* ===== 功能模块 ===== */
.module-list {
  display: flex;
  flex-direction: column;
}

.module-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 9px 0;
  border-bottom: 1px solid #f0f2f4;
}

.module-item:last-child {
  border-bottom: none;
}

.module-name {
  font-size: 14px;
  color: #374151;
}

/* ===== 容量配额 ===== */
.limit-item {
  padding: 14px 16px;
  background: #f6faf7;
  border-radius: 8px;
  text-align: center;
}

.limit-value {
  font-size: 22px;
  font-weight: 700;
  color: #18a058;
}

.limit-label {
  margin-top: 4px;
  font-size: 12px;
  color: #6b7280;
}

/* ===== 设备绑定 ===== */
.bind-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 10px 0;
}

.bind-row + .bind-row {
  border-top: 1px solid #f0f2f4;
}

.bind-label {
  font-size: 12px;
  color: #8a919b;
}

.bind-value {
  margin-top: 2px;
  font-size: 14px;
  color: #1f2937;
}

.monospace {
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
  word-break: break-all;
}

/* ===== 导入弹窗 ===== */
.import-card {
  width: 520px;
  max-width: calc(100vw - 32px);
}

.license-input {
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
}

.import-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
