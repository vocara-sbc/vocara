<template>
  <div v-if="showBanner" class="license-banner" :class="`license-banner--${state}`">
    <n-icon size="16" class="banner-icon">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
        <line x1="12" y1="8" x2="12" y2="13" />
        <circle cx="12" cy="16" r="0.6" fill="currentColor" />
      </svg>
    </n-icon>
    <span class="banner-text">{{ bannerText }}</span>
    <span class="banner-link" @click="goLicense">查看授权</span>
    <span v-if="closable" class="banner-close" @click="closed = true">
      <n-icon size="14">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <line x1="18" y1="6" x2="6" y2="18" />
          <line x1="6" y1="6" x2="18" y2="18" />
        </svg>
      </n-icon>
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { dialog } from '@/utils/naive'
import { getLicenseInfo } from '@/api/license'

const REFRESH_INTERVAL = 3600 * 1000

const router = useRouter()
const state = ref('')
const daysLeft = ref(0)
const closed = ref(false)

// 紧急状态（宽限/过期/无效）登录后仅主动弹窗一次，避免反复打断
const urgentDialogShown = ref(false)
let lastFetch = 0
let timer: number | undefined

const showBanner = computed(() => {
  if (state.value === 'valid' || !state.value) return false
  // 即将到期允许手动收起；宽限与失效状态持续强提示
  if (state.value === 'expiring') return !closed.value
  return true
})

const closable = computed(() => state.value === 'expiring')

const bannerText = computed(() => {
  switch (state.value) {
    case 'expiring':
      return `授权将于 ${daysLeft.value} 天后到期，为避免影响业务请及时续期。`
    case 'grace':
      return '授权已到期，正处于宽限期，请立即续期；宽限期结束后将限制使用。'
    case 'expired':
      return '授权已超出宽限期，外呼与写操作已被限制，请更新授权。'
    case 'invalid':
      return '授权信息无效，请在「授权许可」页导入有效的 License 文件。'
    default:
      return ''
  }
})

function goLicense() {
  router.push('/system/license')
}

async function refresh(force = false) {
  const now = Date.now()
  if (!force && now - lastFetch < 60000) return
  lastFetch = now
  try {
    const data = await getLicenseInfo()
    state.value = data.state
    daysLeft.value = data.days_left
    maybeShowUrgentDialog(data.state)
  } catch {
    // 单次刷新失败保留上一次状态，错误提示由请求拦截器处理
  }
}

function maybeShowUrgentDialog(current: string) {
  if (urgentDialogShown.value) return
  if (current !== 'grace' && current !== 'expired' && current !== 'invalid') return
  urgentDialogShown.value = true

  const isInvalid = current === 'invalid'
  dialog.error({
    title: '授权异常',
    content: isInvalid
      ? '当前系统授权信息无效，请导入有效的 License 文件。'
      : '系统授权已到期，请及时更新授权，到期后外呼与写操作将被限制。',
    positiveText: '前往授权许可',
    negativeText: '稍后处理',
    onPositiveClick: () => router.push('/system/license'),
  })
}

function handleVisibility() {
  if (!document.hidden) refresh()
}

onMounted(() => {
  refresh(true)
  timer = window.setInterval(() => refresh(true), REFRESH_INTERVAL)
  document.addEventListener('visibilitychange', handleVisibility)
})

onBeforeUnmount(() => {
  if (timer) window.clearInterval(timer)
  document.removeEventListener('visibilitychange', handleVisibility)
})
</script>

<style scoped>
.license-banner {
  position: relative;
  flex-shrink: 0;
  width: 100%;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  font-size: 13px;
}

.banner-icon {
  flex-shrink: 0;
}

.banner-text {
  line-height: 1;
}

.banner-link {
  cursor: pointer;
  font-weight: 600;
  text-decoration: underline;
}

.banner-close {
  position: absolute;
  right: 16px;
  display: flex;
  align-items: center;
  cursor: pointer;
  opacity: 0.8;
}

.banner-close:hover {
  opacity: 1;
}

/* 即将到期：琥珀色 */
.license-banner--expiring {
  background: #fdf3dc;
  color: #9a6b0a;
}

.license-banner--expiring .banner-link {
  color: #9a6b0a;
}

/* 宽限 / 过期 / 无效：红色 */
.license-banner--grace,
.license-banner--expired,
.license-banner--invalid {
  background: #fbe3e0;
  color: #b02a21;
}

.license-banner--grace .banner-link,
.license-banner--expired .banner-link,
.license-banner--invalid .banner-link {
  color: #b02a21;
}
</style>
