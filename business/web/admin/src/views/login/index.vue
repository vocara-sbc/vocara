<template>
  <div class="login-container">
    <!-- 背景装饰 -->
    <div class="bg-decoration">
      <div class="bg-circle bg-circle-1"></div>
      <div class="bg-circle bg-circle-2"></div>
    </div>

    <!-- 登录卡片 -->
    <div class="login-card">
      <!-- 左侧品牌 -->
      <div class="login-left">
        <div class="brand">
          <div class="brand-logo">V</div>
          <h1 class="brand-title">VOCARA</h1>
          <p class="brand-subtitle">专业呼叫中心管理系统</p>
        </div>
        <div class="features">
          <div class="feature-item">
            <span class="feature-icon">📞</span>
            <span>全渠道通话统一管理</span>
          </div>
          <div class="feature-item">
            <span class="feature-icon">👥</span>
            <span>智能分配，来电不等待</span>
          </div>
          <div class="feature-item">
            <span class="feature-icon">📈</span>
            <span>座席绩效一目了然</span>
          </div>
          <div class="feature-item">
            <span class="feature-icon">🎧</span>
            <span>实时监听与质检</span>
          </div>
        </div>
      </div>

      <!-- 右侧表单 -->
      <div class="login-right">
        <h2 class="login-title">欢迎登录</h2>
        <p class="login-subtitle">请输入您的账号和密码</p>

        <n-form
          ref="formRef"
          :model="form"
          :rules="rules"
          label-placement="left"
          label-width="0"
          size="large"
          class="login-form"
        >
          <n-form-item path="username" class="form-item">
            <n-input
              v-model:value="form.username"
              placeholder="请输入用户名"
              size="large"
              clearable
            >
              <template #prefix>
                <n-icon size="18" class="prefix-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M20 21v-2a4 4 0 00-4-4H8a4 4 0 00-4 4v2"/>
                    <circle cx="12" cy="7" r="4"/>
                  </svg>
                </n-icon>
              </template>
            </n-input>
          </n-form-item>

          <n-form-item path="password" class="form-item">
            <n-input
              v-model:value="form.password"
              type="password"
              placeholder="请输入密码"
              size="large"
              show-password-on="click"
              @keyup.enter="handleLogin"
            >
              <template #prefix>
                <n-icon size="18" class="prefix-icon">
                  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0110 0v4"/>
                  </svg>
                </n-icon>
              </template>
            </n-input>
          </n-form-item>

          <div class="login-options">
            <n-checkbox v-model:checked="form.remember">记住我</n-checkbox>
            <a class="forgot-link" href="#" @click.prevent>忘记密码？</a>
          </div>

          <n-button
            type="primary"
            block
            size="large"
            :loading="loading"
            class="login-btn"
            @click="handleLogin"
          >
            {{ loading ? '登录中...' : '登 录' }}
          </n-button>
        </n-form>

        <div class="login-footer">
          <p>© 2025 Vocara Team. All rights reserved.</p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useUserStore } from '@/store/modules/user'
import { message } from '@/utils/naive'
import type { FormInst, FormRules } from 'naive-ui'

const router = useRouter()
const route = useRoute()
const userStore = useUserStore()

const formRef = ref<FormInst>()
const loading = ref(false)

const form = reactive({
  username: '',
  password: '',
  remember: true,
})

const rules: FormRules = {
  username: {
    required: true,
    message: '请输入用户名',
    trigger: 'blur',
  },
  password: {
    required: true,
    message: '请输入密码',
    trigger: 'blur',
  },
}

async function handleLogin() {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch (e) {
    return
  }

  loading.value = true
  try {
    await userStore.login({
      username: form.username,
      password: form.password,
      remember: form.remember,
    })

    message.success('登录成功')
    let redirect = (route.query.redirect as string) || userStore.homePath
    // 座席角色只允许工作台与通话记录路径（含后端未下发 home_path 时的 /dashboard 兜底）
    if (userStore.userInfo?.role === 'agent' && !(redirect.startsWith('/workbench') || redirect.startsWith('/records'))) {
      redirect = '/workbench'
    }
    router.push(redirect)
  } catch {
    // 错误提示已由 axios 响应拦截器统一弹出，这里只需恢复按钮状态
  } finally {
    loading.value = false
  }
}
</script>

<style scoped lang="css">
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f5f7fa;
  position: relative;
  overflow: hidden;
  padding: 20px;
}

/* 背景装饰 */
.bg-decoration {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
}

.bg-circle {
  position: absolute;
  border-radius: 50%;
}

.bg-circle-1 {
  width: 500px;
  height: 500px;
  background: #dcfce7;
  top: -150px;
  left: -150px;
  filter: blur(80px);
  opacity: 0.6;
}

.bg-circle-2 {
  width: 400px;
  height: 400px;
  background: #dbeafe;
  bottom: -100px;
  right: -100px;
  filter: blur(80px);
  opacity: 0.6;
}

/* 登录卡片 */
.login-card {
  display: flex;
  width: 880px;
  max-width: 100%;
  background: #ffffff;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.08), 0 0 0 1px rgba(0, 0, 0, 0.04);
  position: relative;
  z-index: 1;
}

/* 左侧 */
.login-left {
  width: 45%;
  padding: 48px 40px;
  background: linear-gradient(135deg, #f0fdf4 0%, #dcfce7 100%);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
}

.brand {
  margin-bottom: 40px;
}

.brand-logo {
  width: 56px;
  height: 56px;
  background: linear-gradient(135deg, #18a058, #0c7a43);
  border-radius: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #ffffff;
  font-size: 28px;
  font-weight: 700;
  margin-bottom: 20px;
  box-shadow: 0 8px 20px rgba(24, 160, 88, 0.25);
}

.brand-title {
  font-size: 32px;
  font-weight: 700;
  margin: 0 0 8px 0;
  color: #166534;
  letter-spacing: 4px;
}

.brand-subtitle {
  font-size: 14px;
  color: #4b5563;
  margin: 0;
}

.features {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.feature-item {
  display: flex;
  align-items: center;
  gap: 12px;
  color: #1f2937;
  font-size: 14px;
}

.feature-icon {
  width: 36px;
  height: 36px;
  background: #ffffff;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(24, 160, 88, 0.08);
}

/* 右侧 */
.login-right {
  flex: 1;
  padding: 48px 50px;
  display: flex;
  flex-direction: column;
}

.login-title {
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 8px 0;
  color: #1f2937;
}

.login-subtitle {
  font-size: 14px;
  color: #6b7280;
  margin: 0 0 32px 0;
}

.login-form {
  flex: 1;
}

.form-item {
  margin-bottom: 20px;
}

.prefix-icon {
  color: #9ca3af;
}

.login-options {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
  font-size: 13px;
}

.forgot-link {
  color: #18a058;
  text-decoration: none;
  font-size: 13px;
}

.forgot-link:hover {
  color: #0c7a43;
}

.login-btn {
  height: 44px;
  font-size: 15px;
  font-weight: 500;
  letter-spacing: 2px;
  border-radius: 8px;
  margin-bottom: 20px;
}

.login-footer {
  margin-top: auto;
  text-align: center;
  font-size: 12px;
  color: #9ca3af;
}

.login-footer p {
  margin: 0;
}

/* 响应式 */
@media (max-width: 768px) {
  .login-card {
    flex-direction: column;
    width: 100%;
    max-width: 400px;
  }

  .login-left {
    width: 100%;
    padding: 32px 24px;
  }

  .features {
    display: none;
  }

  .brand-title {
    font-size: 24px;
  }

  .login-right {
    padding: 32px 24px;
  }
}
</style>
