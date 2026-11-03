<template>
  <!-- 修改密码全局弹窗：Header 下拉与个人中心共用，可见性由 ui store 统一管理 -->
  <n-modal
    :show="uiStore.passwordModalVisible"
    preset="card"
    title="修改密码"
    style="width: 420px"
    @update:show="handleVisibleChange"
  >
    <n-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-placement="left"
      label-width="88"
      size="medium"
    >
      <n-form-item label="旧密码" path="old_password">
        <n-input
          v-model:value="form.old_password"
          type="password"
          show-password-on="click"
          placeholder="请输入旧密码"
        />
      </n-form-item>
      <n-form-item label="新密码" path="new_password">
        <n-input
          v-model:value="form.new_password"
          type="password"
          show-password-on="click"
          placeholder="8–20 位，须含字母和数字"
        />
      </n-form-item>
      <n-form-item label="确认新密码" path="confirm_password">
        <n-input
          v-model:value="form.confirm_password"
          type="password"
          show-password-on="click"
          placeholder="请再次输入新密码"
          @keyup.enter="handleSubmit"
        />
      </n-form-item>
    </n-form>
    <template #footer>
      <n-button @click="uiStore.closePasswordModal()">取消</n-button>
      <n-button type="primary" :loading="submitting" @click="handleSubmit">保存</n-button>
    </template>
  </n-modal>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import type { FormInst, FormRules } from 'naive-ui'
import { message } from '@/utils/naive'
import { changePasswordApi } from '@/api/auth'
import { useUiStore } from '@/store/modules/ui'
import { useUserStore } from '@/store/modules/user'
import { useCallStore } from '@/store/modules/call'

const uiStore = useUiStore()
const userStore = useUserStore()
const callStore = useCallStore()
const router = useRouter()

const formRef = ref<FormInst>()
const submitting = ref(false)

const form = reactive({
  old_password: '',
  new_password: '',
  confirm_password: '',
})

// 新密码规则与后端一致：8–20 位，须同时包含字母和数字
function isStrongPassword(v: string): boolean {
  if (v.length < 8 || v.length > 20) return false
  return /[A-Za-z]/.test(v) && /[0-9]/.test(v)
}

const rules: FormRules = {
  old_password: {
    required: true,
    trigger: ['blur', 'input'],
    message: '请输入旧密码',
  },
  new_password: [
    {
      required: true,
      trigger: ['blur', 'input'],
      message: '请输入新密码',
    },
    {
      validator: (_rule, value: string) => isStrongPassword(value),
      trigger: ['blur', 'input'],
      message: '密码长度为 8–20 位，须同时包含字母和数字',
    },
    {
      validator: (_rule, value: string) => value !== form.old_password,
      trigger: ['blur', 'input'],
      message: '新密码不能与旧密码相同',
    },
  ],
  confirm_password: [
    {
      required: true,
      trigger: ['blur', 'input'],
      message: '请再次输入新密码',
    },
    {
      validator: (_rule, value: string) => value === form.new_password,
      trigger: ['blur', 'input'],
      message: '两次输入的新密码不一致',
    },
  ],
}

function resetForm() {
  form.old_password = ''
  form.new_password = ''
  form.confirm_password = ''
  formRef.value?.restoreValidation()
}

function handleVisibleChange(visible: boolean) {
  if (visible) {
    uiStore.passwordModalVisible = true
  } else {
    uiStore.closePasswordModal()
    resetForm()
  }
}

async function handleSubmit() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  submitting.value = true
  try {
    await changePasswordApi({
      old_password: form.old_password,
      new_password: form.new_password,
    })
    message.success('密码修改成功，请重新登录')
    uiStore.closePasswordModal()
    // JWT 无状态，密码变更后重新登录建立新会话
    callStore.destroyAdapter()
    userStore.logout()
    router.push('/login')
  } catch {
    // 错误提示由请求拦截器统一处理
  } finally {
    submitting.value = false
  }
}

// 每次弹窗打开时清空上一次输入
watch(
  () => uiStore.passwordModalVisible,
  (visible) => {
    if (visible) resetForm()
  },
)
</script>
