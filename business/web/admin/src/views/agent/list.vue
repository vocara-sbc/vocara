<template>
  <div class="agent-list-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="工号/姓名/手机号"
            clearable
            style="width: 200px"
            @keyup.enter="handleSearch"
          />
        </n-form-item>
        <n-form-item label="状态">
          <n-select
            v-model:value="queryForm.status"
            :options="statusOptions"
            placeholder="全部"
            clearable
            style="width: 120px"
          />
        </n-form-item>
        <n-form-item label="组织机构">
          <n-select
            v-model:value="queryForm.org_id"
            :options="orgOptions"
            placeholder="全部"
            clearable
            filterable
            style="width: 150px"
          />
        </n-form-item>
        <n-form-item label="技能组">
          <n-select
            v-model:value="queryForm.skill_id"
            :options="skillOptions"
            placeholder="全部"
            clearable
            filterable
            style="width: 150px"
          />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" @click="handleSearch">查询</n-button>
          <n-button style="margin-left: 8px" @click="handleReset">重置</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 操作栏 + 表格 -->
    <n-card :bordered="false" class="table-card">
      <div class="table-header">
        <div class="table-title">座席列表</div>
        <div class="table-actions">
          <!-- 后端 /agents/import 接收 JSON 数组而非文件上传，前端导入向导未实现，按钮暂时隐藏 -->
          <n-button size="small" :loading="exporting" @click="handleExport">
            导出
          </n-button>
          <n-button type="primary" size="small" style="margin-left: 8px" @click="handleAdd">
            新增座席
          </n-button>
        </div>
      </div>

      <n-data-table
        :columns="columns"
        :data="tableData"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: AgentInfo) => row.id"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="showModal" preset="card" :title="isEdit ? '编辑座席' : '新增座席'" :style="{ width: '560px' }">
      <n-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-placement="left"
        label-width="90"
        size="medium"
      >
        <n-form-item label="座席工号" path="agent_number">
          <n-input v-model:value="form.agent_number" placeholder="登录工号（创建后不可修改）" :disabled="isEdit" />
        </n-form-item>
        <n-form-item label="座席姓名" path="agent_name">
          <n-input v-model:value="form.agent_name" placeholder="座席姓名" />
        </n-form-item>
        <n-form-item v-if="!isEdit" label="登录密码" path="password">
          <n-input
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            placeholder="留空则使用默认密码"
          />
        </n-form-item>
        <n-form-item label="角色" path="role">
          <n-select v-model:value="form.role" :options="roleOptions" style="width: 200px" />
        </n-form-item>
        <n-form-item label="组织机构" path="org_id">
          <n-select
            v-model:value="form.org_id"
            :options="orgOptions"
            placeholder="可选"
            clearable
            filterable
          />
        </n-form-item>
        <n-form-item label="技能组" path="skillGroupIds">
          <n-select
            v-model:value="skillGroupIds"
            :options="skillOptions"
            placeholder="可选，提交后按所选覆盖该座席的技能配置"
            multiple
            clearable
            filterable
          />
        </n-form-item>
        <n-form-item label="绑定分机" path="dn">
          <n-select
            v-model:value="form.dn"
            :options="dnOptions"
            placeholder="可选，选择可用分机号"
            clearable
            filterable
          />
        </n-form-item>
        <n-form-item label="手机号" path="mobile">
          <n-input v-model:value="form.mobile" placeholder="可选" />
        </n-form-item>
        <n-form-item label="邮箱" path="email">
          <n-input v-model:value="form.email" placeholder="可选" />
        </n-form-item>
        <n-form-item label="状态" path="status">
          <n-switch v-model:value="statusSwitch" />
          <span style="margin-left: 8px; color: #6b7280">{{ statusSwitch ? '启用' : '禁用' }}</span>
        </n-form-item>
        <n-form-item label="备注" path="remark">
          <n-input v-model:value="form.remark" type="textarea" :rows="2" placeholder="可选" />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-button @click="showModal = false">取消</n-button>
        <n-button type="primary" :loading="saving" @click="handleSave">保存</n-button>
      </template>
    </n-modal>

    <!-- 重置密码弹窗 -->
    <n-modal v-model:show="showPwdModal" preset="card" title="重置密码" :style="{ width: '420px' }">
      <n-form label-placement="left" label-width="90" size="medium">
        <n-form-item label="座席">
          <span>{{ currentRow?.agent_name }}（{{ currentRow?.agent_number }}）</span>
        </n-form-item>
        <n-form-item label="新密码">
          <n-input
            v-model:value="newPassword"
            type="password"
            show-password-on="click"
            placeholder="请输入新密码"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-button @click="showPwdModal = false">取消</n-button>
        <n-button type="primary" :loading="saving" @click="handleResetPasswordConfirm">确定</n-button>
      </template>
    </n-modal>

    <!-- 绑定分机弹窗 -->
    <n-modal v-model:show="showDnModal" preset="card" title="绑定分机" :style="{ width: '420px' }">
      <n-form label-placement="left" label-width="90" size="medium">
        <n-form-item label="座席">
          <span>{{ currentRow?.agent_name }}（{{ currentRow?.agent_number }}）</span>
        </n-form-item>
        <n-form-item label="分机号">
          <n-select
            v-model:value="bindDn"
            :options="bindDnOptions"
            placeholder="选择可用分机，清空则解除绑定"
            clearable
            filterable
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-button @click="showDnModal = false">取消</n-button>
        <n-button type="primary" :loading="saving" @click="handleBindDnConfirm">确定</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, h } from 'vue'
import { NButton, NTag, NDropdown } from 'naive-ui'
import { message, dialog } from '@/utils/naive'
import {
  getAgentList,
  createAgent,
  updateAgent,
  deleteAgent,
  toggleAgentStatus,
  resetAgentPassword,
  bindAgentDn,
  updateAgentSkill,
  exportAgents,
  getOrganizationAll,
  getSkillGroupAll,
  getDnAvailable,
} from '@/api/agent'
import {
  AgentWorkStatusMap,
  AgentRoleOptions,
  type AgentInfo,
  type Organization,
  type SkillGroup,
  type DnInfo,
} from '@/types/agent'

const loading = ref(false)
const saving = ref(false)
const exporting = ref(false)

// ===== 查询 =====
const queryForm = reactive({
  keyword: '',
  status: undefined as number | undefined,
  org_id: undefined as number | undefined,
  skill_id: undefined as number | undefined,
})

const statusOptions = [
  { label: '启用', value: 1 },
  { label: '禁用', value: 0 },
]

const roleOptions = AgentRoleOptions

const tableData = ref<AgentInfo[]>([])
const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
})

// ===== 下拉选项 =====
const orgOptions = ref<Array<{ label: string; value: number }>>([])
const skillOptions = ref<Array<{ label: string; value: number }>>([])
const dnOptions = ref<Array<{ label: string; value: string }>>([])

// 组织机构 id → 名称映射（表格展示用）
const orgNameMap = computed(() => {
  const m = new Map<number, string>()
  orgOptions.value.forEach((o) => m.set(o.value, o.label))
  return m
})

// ===== 表格列 =====
const columns = [
  { title: '座席工号', key: 'agent_number', width: 100 },
  { title: '座席姓名', key: 'agent_name', width: 110 },
  {
    title: '分机号',
    key: 'dn',
    width: 90,
    render: (row: AgentInfo) => row.dn || '-',
  },
  {
    title: '组织机构',
    key: 'org_id',
    width: 120,
    render: (row: AgentInfo) =>
      row.org_id ? orgNameMap.value.get(row.org_id) || `#${row.org_id}` : '-',
  },
  {
    title: '角色',
    key: 'role',
    width: 90,
    render: (row: AgentInfo) =>
      roleOptions.find((r) => r.value === row.role)?.label || row.role || '-',
  },
  {
    title: '工作状态',
    key: 'work_status',
    width: 100,
    render: (row: AgentInfo) => {
      const s = AgentWorkStatusMap[row.work_status] || AgentWorkStatusMap[0]
      return h(
        'span',
        { style: { display: 'inline-flex', alignItems: 'center', gap: '6px', color: s.color } },
        [
          h('span', {
            style: { width: '8px', height: '8px', borderRadius: '50%', background: s.color },
          }),
          s.text,
        ]
      )
    },
  },
  {
    title: '状态',
    key: 'status',
    width: 80,
    render: (row: AgentInfo) =>
      h(
        NTag,
        { size: 'small', type: row.status === 1 ? 'success' : 'default' },
        () => (row.status === 1 ? '启用' : '禁用')
      ),
  },
  {
    title: '最后登录',
    key: 'last_login_at',
    width: 160,
    render: (row: AgentInfo) => formatTime(row.last_login_at),
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    fixed: 'right' as const,
    render: (row: AgentInfo) =>
      h('div', { style: { display: 'flex', gap: '8px', alignItems: 'center' } }, [
        h(NButton, { size: 'small', onClick: () => handleEdit(row) }, () => '编辑'),
        h(
          NDropdown,
          {
            trigger: 'click',
            options: [
              { label: '绑定分机', key: 'bindDn' },
              { label: '重置密码', key: 'resetPwd' },
              { label: row.status === 1 ? '禁用' : '启用', key: 'toggle' },
              { label: '删除', key: 'delete' },
            ],
            onSelect: (key: string) => handleAction(key, row),
          },
          () => h(NButton, { size: 'small', quaternary: true }, () => '更多 ▾')
        ),
      ]),
  },
]

// 时间展示：后端零值时间（0001-01-01）视为无记录
function formatTime(t?: string) {
  if (!t || t.startsWith('0001-01-01')) return '-'
  return t.replace('T', ' ').slice(0, 19)
}

// ===== 新增/编辑表单 =====
const showModal = ref(false)
const isEdit = ref(false)
const formRef = ref()
const skillGroupIds = ref<number[]>([])

const emptyForm = () => ({
  id: 0,
  agent_number: '',
  agent_name: '',
  password: '',
  mobile: '',
  email: '',
  dn: undefined as string | undefined,
  role: 'agent',
  status: 1,
  org_id: undefined as number | undefined,
  remark: '',
})
const form = reactive(emptyForm())

const statusSwitch = computed({
  get: () => form.status === 1,
  set: (v: boolean) => {
    form.status = v ? 1 : 0
  },
})

const formRules = {
  agent_number: { required: true, message: '请输入座席工号', trigger: 'blur' },
  agent_name: { required: true, message: '请输入座席姓名', trigger: 'blur' },
}

// ===== 重置密码 / 绑定分机 =====
const currentRow = ref<AgentInfo | null>(null)
const showPwdModal = ref(false)
const newPassword = ref('')
const showDnModal = ref(false)
const bindDn = ref<string | undefined>(undefined)
// 绑定弹窗的分机选项：可用分机 + 当前已绑定分机
const bindDnOptions = ref<Array<{ label: string; value: string }>>([])

// ===== 数据加载 =====
async function loadOptions() {
  try {
    const [groups, skills, dns]: [Organization[], SkillGroup[], DnInfo[]] = await Promise.all([
      getOrganizationAll(),
      getSkillGroupAll(),
      getDnAvailable(),
    ])
    orgOptions.value = (groups || []).map((g) => ({ label: g.org_name, value: g.id }))
    skillOptions.value = (skills || []).map((s) => ({ label: s.skill_name, value: s.id }))
    dnOptions.value = (dns || []).map((d) => ({ label: d.dn_num, value: d.dn_num }))
  } catch (e) {
    // 选项加载失败不阻塞表格，错误提示由拦截器统一处理
  }
}

async function loadData() {
  loading.value = true
  try {
    const res = await getAgentList({
      page: pagination.page,
      page_size: pagination.pageSize,
      keyword: queryForm.keyword || undefined,
      status: queryForm.status,
      org_id: queryForm.org_id,
      // 后端分页查询当前未实现按技能组过滤，该参数透传预留
      skill_id: queryForm.skill_id,
    })
    tableData.value = res?.list || []
    pagination.itemCount = res?.total ?? 0
  } catch (e) {
    tableData.value = []
    pagination.itemCount = 0
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  pagination.page = 1
  loadData()
}

function handleReset() {
  queryForm.keyword = ''
  queryForm.status = undefined
  queryForm.org_id = undefined
  queryForm.skill_id = undefined
  pagination.page = 1
  loadData()
}

function handlePageChange(page: number) {
  pagination.page = page
  loadData()
}

function handlePageSizeChange(pageSize: number) {
  pagination.pageSize = pageSize
  pagination.page = 1
  loadData()
}

// ===== 新增/编辑/删除 =====
function handleAdd() {
  Object.assign(form, emptyForm())
  skillGroupIds.value = []
  isEdit.value = false
  showModal.value = true
}

function handleEdit(row: AgentInfo) {
  Object.assign(form, emptyForm(), {
    id: row.id,
    agent_number: row.agent_number,
    agent_name: row.agent_name,
    mobile: row.mobile,
    email: row.email,
    dn: row.dn || undefined,
    role: row.role || 'agent',
    status: row.status,
    org_id: row.org_id || undefined,
    remark: row.remark,
  })
  // 后端未提供座席技能查询接口，编辑时技能组留空；留空提交不会改动既有技能配置
  skillGroupIds.value = []
  isEdit.value = true
  showModal.value = true
}

async function handleSave() {
  try {
    await formRef.value?.validate()
  } catch {
    return
  }
  saving.value = true
  try {
    let agentId = form.id
    if (isEdit.value) {
      await updateAgent({
        id: form.id,
        agent_number: form.agent_number,
        agent_name: form.agent_name,
        mobile: form.mobile,
        email: form.email,
        dn: form.dn || '',
        role: form.role,
        status: form.status,
        org_id: form.org_id || 0,
        remark: form.remark,
      })
      message.success('更新成功')
    } else {
      const created = await createAgent({
        agent_number: form.agent_number,
        agent_name: form.agent_name,
        password: form.password,
        mobile: form.mobile,
        email: form.email,
        dn: form.dn || '',
        role: form.role,
        status: form.status,
        org_id: form.org_id || 0,
        remark: form.remark,
      })
      agentId = created?.id || 0
      message.success('创建成功')
    }
    // 选择了技能组时才覆盖技能配置；留空表示不改动
    if (agentId && skillGroupIds.value.length > 0) {
      await updateAgentSkill(
        agentId,
        skillGroupIds.value.map((id) => ({ skill_id: id, skill_level: 5 }))
      )
    }
    showModal.value = false
    loadData()
    loadOptions()
  } catch (e) {
    // 错误提示由拦截器统一处理
  } finally {
    saving.value = false
  }
}

function handleAction(key: string, row: AgentInfo) {
  if (key === 'bindDn') {
    openBindDn(row)
  } else if (key === 'resetPwd') {
    currentRow.value = row
    newPassword.value = ''
    showPwdModal.value = true
  } else if (key === 'toggle') {
    handleToggle(row)
  } else if (key === 'delete') {
    handleDelete(row)
  }
}

function handleToggle(row: AgentInfo) {
  const enable = row.status !== 1
  dialog.warning({
    title: enable ? '启用确认' : '禁用确认',
    content: `确定${enable ? '启用' : '禁用'}座席「${row.agent_name}」吗？`,
    positiveText: '确定',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await toggleAgentStatus(row.id, enable)
        message.success(enable ? '已启用' : '已禁用')
        loadData()
      } catch (e) {
        // 错误提示由拦截器统一处理
      }
    },
  })
}

function handleDelete(row: AgentInfo) {
  dialog.warning({
    title: '删除确认',
    content: `确定删除座席「${row.agent_name}」（工号 ${row.agent_number}）吗？删除后其技能配置与分机绑定将一并解除。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteAgent(row.id)
        message.success('删除成功')
        loadData()
      } catch (e) {
        // 错误提示由拦截器统一处理
      }
    },
  })
}

async function handleResetPasswordConfirm() {
  if (!newPassword.value) {
    message.warning('请输入新密码')
    return
  }
  saving.value = true
  try {
    await resetAgentPassword(currentRow.value!.id, newPassword.value)
    message.success('密码已重置')
    showPwdModal.value = false
  } catch (e) {
    // 错误提示由拦截器统一处理
  } finally {
    saving.value = false
  }
}

async function openBindDn(row: AgentInfo) {
  currentRow.value = row
  bindDn.value = row.dn || undefined
  try {
    const dns = await getDnAvailable()
    const opts = (dns || []).map((d) => ({ label: d.dn_num, value: d.dn_num }))
    // 当前已绑定的分机不在可用列表中，补充进来以便展示
    if (row.dn && !opts.some((o) => o.value === row.dn)) {
      opts.unshift({ label: `${row.dn}（当前绑定）`, value: row.dn })
    }
    bindDnOptions.value = opts
    showDnModal.value = true
  } catch (e) {
    // 错误提示由拦截器统一处理
  }
}

async function handleBindDnConfirm() {
  saving.value = true
  try {
    await bindAgentDn(currentRow.value!.id, bindDn.value || '')
    message.success(bindDn.value ? '绑定成功' : '已解除绑定')
    showDnModal.value = false
    loadData()
    loadOptions()
  } catch (e) {
    // 错误提示由拦截器统一处理
  } finally {
    saving.value = false
  }
}

// ===== 导出 =====
async function handleExport() {
  exporting.value = true
  try {
    const res = await exportAgents({
      keyword: queryForm.keyword || undefined,
      status: queryForm.status,
      org_id: queryForm.org_id,
    })
    const list = res?.list || []
    if (list.length === 0) {
      message.warning('没有可导出的数据')
      return
    }
    const header = ['座席工号', '座席姓名', '手机号', '邮箱', '分机号', '角色', '组织机构', '状态']
    const rows = list.map((a) => [
      a.agent_number,
      a.agent_name,
      a.mobile || '',
      a.email || '',
      a.dn || '',
      roleOptions.find((r) => r.value === a.role)?.label || a.role || '',
      a.org_id ? orgNameMap.value.get(a.org_id) || '' : '',
      a.status === 1 ? '启用' : '禁用',
    ])
    // 单元格含逗号/引号/换行时按 CSV 规范转义；加 BOM 保证 Excel 中文不乱码
    const escapeCell = (v: string) => (/[",\n]/.test(v) ? `"${v.replace(/"/g, '""')}"` : v)
    const csv = '\uFEFF' + [header, ...rows].map((r) => r.map(escapeCell).join(',')).join('\n')
    const blob = new Blob([csv], { type: 'text/csv;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `座席列表_${new Date().toISOString().slice(0, 10)}.csv`
    a.click()
    URL.revokeObjectURL(url)
    message.success(`已导出 ${list.length} 条数据`)
  } catch (e) {
    // 错误提示由拦截器统一处理
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  loadOptions()
  loadData()
})
</script>

<style scoped lang="css">
.agent-list-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
}

/* 数据卡填满搜索区之外的剩余高度，表格主体内部滚动、分页固定卡底右侧 */
.table-card {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.table-card :deep(.n-card-content) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.table-card :deep(.n-data-table) {
  flex: 1;
  min-height: 0;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.table-title {
  font-size: 14px;
  font-weight: 600;
}

.table-actions {
  display: flex;
  align-items: center;
}
</style>
