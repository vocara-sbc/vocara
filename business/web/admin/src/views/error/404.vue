<template>
  <div class="not-found-page">
    <div class="error-content">
      <div class="error-code">404</div>
      <div class="error-title">页面未找到</div>
      <div class="error-desc">抱歉，您访问的页面不存在或已被移除。</div>
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
    <div class="error-decoration">
      <div class="glow-circle glow-1"></div>
      <div class="glow-circle glow-2"></div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

function goHome() {
  router.push('/dashboard')
}

function goBack() {
  router.back()
}
</script>

<style scoped lang="css">
.not-found-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
  position: relative;
  overflow: hidden;
}

.error-content {
  text-align: center;
  position: relative;
  z-index: 1;
}

.error-code {
  font-size: 160px;
  font-weight: 800;
  line-height: 1;
  background: linear-gradient(135deg, #18a058, #36ad6a);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  letter-spacing: 8px;
  margin-bottom: 16px;
}

.error-title {
  font-size: 28px;
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

.error-decoration {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.glow-circle {
  position: absolute;
  border-radius: 50%;
  opacity: 0.3;
}

.glow-1 {
  width: 500px;
  height: 500px;
  background: #18a058;
  top: 10%;
  left: 10%;
  filter: blur(100px);
  opacity: 0.08;
}

.glow-2 {
  width: 400px;
  height: 400px;
  background: #36ad6a;
  bottom: 10%;
  right: 10%;
  filter: blur(80px);
  opacity: 0.06;
}
</style>
