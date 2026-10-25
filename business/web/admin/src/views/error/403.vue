<template>
  <div class="forbidden-page">
    <div class="error-content">
      <div class="error-code">403</div>
      <div class="error-title">无访问权限</div>
      <div class="error-desc">抱歉，您没有权限访问此页面，请联系管理员。</div>
      <div class="error-actions">
        <n-button type="primary" @click="goHome">
          <template #icon>
            <n-icon size="16">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
                <polyline points="9 22 9 12 15 12 15 22"/>
              </svg>
            </n-icon>
          </template>
          返回首页
        </n-button>
        <n-button style="margin-left: 12px" @click="goBack">
          <template #icon>
            <n-icon size="16">
              <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <polyline points="15 18 9 12 15 6"/>
              </svg>
            </n-icon>
          </template>
          返回上一页
        </n-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useUserStore } from '@/store/modules/user'

const router = useRouter()
const userStore = useUserStore()

function goHome() {
  router.push(userStore.homePath)
}

function goBack() {
  router.back()
}
</script>

<style scoped lang="css">
.forbidden-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
}

.error-content {
  text-align: center;
}

.error-code {
  font-size: 140px;
  font-weight: 800;
  line-height: 1;
  background: linear-gradient(135deg, #d03050, #e5484d);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  letter-spacing: 6px;
  margin-bottom: 16px;
}

.error-title {
  font-size: 24px;
  font-weight: 600;
  color: #1f2937;
  margin-bottom: 12px;
}

.error-desc {
  font-size: 14px;
  color: #6b7280;
  margin-bottom: 32px;
}

.error-actions {
  display: flex;
  justify-content: center;
}
</style>
