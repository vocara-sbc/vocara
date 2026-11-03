<template>
  <div class="profile-page">
    <!-- 基本资料：本人可维护姓名、手机、邮箱、性别，保存后同步全局用户态 -->
    <n-card :bordered="false" title="基本资料">
      <div class="profile-header">
        <n-avatar round :size="56">
          {{ form.agent_name?.charAt(0) || 'A' }}
        </n-avatar>
        <div class="profile-meta">
          <div class="profile-name">{{ form.agent_name }}</div>
          <div class="profile-sub">工号 {{ userStore.userInfo?.username }}</div>
        </div>
      </div>

      <n-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-placement="left"
        label-width="72"
        style="max-width: 560px"
      >
        <n-form-item label="姓名" path="agent_name">
          <n-input v-model:value="form.agent_name" placeholder="请输入姓名" />
        </n-form-item>
        <n-form-item label="手机号" path="mobile">
          <n-input v-model:value="form.mobile" placeholder="选填，11 位手机号" clearable />
        </n-form-item>
        <n-form-item label="邮箱" path="email">
          <n-input v-model:value="form.email" placeholder="选填" clearable />
        </n-form-item>
        <n-form-item label="性别" path="gender">
          <n-radio-group v-model:value="form.gender">
            <n-radio :value="1">男</n-radio>
            <n-radio :value="2">女</n-radio>
            <n-radio :value="0">保密</n-radio>
          </n-radio-group>
        </n-form-item>
      </n-form>

      <template #footer>
        <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
      </template>
    </n-card>

    <!-- 账号信息：只读展示身份与归属 -->
    <n-card :bordered="false" title="账号信息">
      <n-descriptions bordered :column="2" label-placement="left" size="medium">
        <n-descriptions-item label="工号">
          {{ userStore.userInfo?.username }}
        </n-descriptions-item>
        <n-descriptions-item label="角色">{{ roleText }}</n-descriptions-item>
        <n-descriptions-item label="所属租户">
          {{ userStore.userInfo?.tenant_name }}
        </n-descriptions-item>
        <n-descriptions-item label="租户编码">
          {{ userStore.userInfo?.tenant_code }}
        </n-descriptions-item>
      </n-descriptions>
    </n-card>

    <!-- 安全设置 -->
    <n-card :bordered="false" title="安全设置">
      <div class="security-row">
        <span class="security-label">登录密码</span>
        <n-button size="small" @click="uiStore.openPasswordModal()">修改密码</n-button>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, onMounted } from 'vue'
import type { FormInst, FormRules } from 'naive-ui'
import { message } from '@/utils/naive'
import { updateProfileApi } from '@/api/auth'
import { useUserStore } from '@/store/modules/user'
import { useUiStore } from '@/store/modules/ui'

const userStore = useUserStore()
const uiStore = useUiStore()

const formRef = ref<FormInst>()
const saving = ref(false)

const form = reactive({
  agent_name: '',
  mobile: '',
  email: '',
  gender: 0,
})

const roleTextMap: Record<string, string> = {
  admin: '管理员',
  supervisor: '班长',
  agent: '座席',
}
const roleText = computed(() => roleTextMap[userStore.userInfo?.role || ''] || '-')

const rules: FormRules = {
  agent_name: {
    required: true,
    trigger: ['blur', 'input'],
    message: '请输入姓名',
  },
  mobile: {
    validator: (_rule, value: string) => !value || /^1[3-9]\d{9}$/.test(value),
    trigger: ['blur', 'input'],
    message: '手机号格式不正确',
  },
  email: {
    validator: (_rule, value: string) =>
      !value || /^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}$/.test(value),
    trigger: ['blur', 'input'],
    message: '邮箱格式不正确',
  },
}

// 用当前用户态预填表单
function fillFromUserInfo() {
  const info = userStore.userInfo
  if (!info) return
  form.agent_name = info.nickname
  form.mobile = info.mobile || ''
  form.email = info.email || ''
  form.gender = info.gender ?? 0
}

async function handleSave() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    await updateProfileApi({
      agent_name: form.agent_name,
      mobile: form.mobile || undefined,
      email: form.email || undefined,
      gender: form.gender,
    })
    message.success('资料保存成功')
    // 重新拉取用户信息，Header 昵称等全局展示与后端保持一致
    await userStore.fetchUserInfo()
  } catch {
    // 错误提示由请求拦截器统一处理
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  if (!userStore.userInfo) {
    await userStore.fetchUserInfo()
  }
  fillFromUserInfo()
})
</script>

<style scoped>
.profile-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.profile-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}

.profile-name {
  font-size: 18px;
  font-weight: 600;
}

.profile-sub {
  margin-top: 4px;
  font-size: 13px;
  color: #6b7280;
}

.security-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  max-width: 560px;
}

.security-label {
  font-size: 14px;
}
</style>
