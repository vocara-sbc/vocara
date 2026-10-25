<template>
  <div class="dn-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="分机号/名称"
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
        <n-form-item label="分机类型">
          <n-select
            v-model:value="queryForm.dn_type"
            :options="dnTypeOptions"
            placeholder="全部"
            clearable
            style="width: 140px"
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
        <div class="table-title">分机列表</div>
        <div class="table-actions">
          <!-- 后端 /dns/batch_import 按起始号段批量生成，前端号段录入界面未实现，按钮暂时隐藏 -->
          <n-button size="small" :loading="checking" @click="handleStatusCheck">
            注册状态检测
          </n-button>
          <n-button type="primary" size="small" style="margin-left: 8px" @click="handleAdd">
            新增分机
          </n-button>
        </div>
      </div>

      <n-data-table
        :columns="columns"
        :data="tableData"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: DnInfo) => row.id"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="showModal" preset="card" :title="isEdit ? '编辑分机' : '新增分机'" :style="{ width: '480px' }">
      <n-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-placement="left"
        label-width="90"
        size="medium"
      >
        <n-form-item label="分机号" path="dn_num">
          <n-input v-model:value="form.dn_num" placeholder="如：8001（创建后不可修改）" :disabled="isEdit" />
        </n-form-item>
        <n-form-item label="分机名称" path="dn_name">
          <n-input v-model:value="form.dn_name" placeholder="可选" />
        </n-form-item>
        <n-form-item label="注册密码" path="password">
          <n-input
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            :placeholder="isEdit ? '留空则不修改' : '留空则使用默认密码'"
          />
        </n-form-item>
        <n-form-item label="分机类型" path="dn_type">
          <n-select v-model:value="form.dn_type" :options="dnTypeOptions" style="width: 200px" />
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

    <!-- 绑定座席弹窗 -->
    <n-modal v-model:show="showBindModal" preset="card" title="绑定座席" :style="{ width: '420px' }">
      <n-form label-placement="left" label-width="90" size="medium">
        <n-form-item label="分机号">
          <span>{{ currentRow?.dn_num }}</span>
        </n-form-item>
        <n-form-item label="座席">
          <n-select
            v-model:value="bindAgentId"
            :options="agentOptions"
            placeholder="选择要绑定的座席"
            clearable
            filterable
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-button @click="showBindModal = false">取消</n-button>
        <n-button type="primary" :loading="saving" :disabled="!bindAgentId" @click="handleBindConfirm">确定</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, h } from 'vue'
import { NButton, NTag, NDropdown } from 'naive-ui'
import { message, dialog } from '@/utils/naive'
import {
  getDnList,
  createDn,
  updateDn,
  deleteDn,
  checkDnStatus,
  bindDnAgent,
  unbindDnAgent,
  getAgentAll,
} from '@/api/agent'
import { DnTypeOptions, type DnInfo, type AgentInfo } from '@/types/agent'

const loading = ref(false)
const saving = ref(false)
const checking = ref(false)

// ===== 查询 =====
const queryForm = reactive({
  keyword: '',
  status: undefined as number | undefined,
  dn_type: undefined as number | undefined,
})

const statusOptions = [
  { label: '启用', value: 1 },
  { label: '禁用', value: 0 },
]

const dnTypeOptions = DnTypeOptions

const tableData = ref<DnInfo[]>([])
const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
})

// ===== 表格列 =====
const columns = [
  { title: '分机号', key: 'dn_num', width: 100 },
  {
    title: '分机名称',
    key: 'dn_name',
    width: 120,
    render: (row: DnInfo) => row.dn_name || '-',
  },
  {
    title: '类型',
    key: 'dn_type',
    width: 100,
    render: (row: DnInfo) =>
      dnTypeOptions.find((t) => t.value === row.dn_type)?.label || '-',
  },
  {
    title: '绑定座席',
    key: 'agent_number',
    width: 110,
    render: (row: DnInfo) =>
      row.agent_id ? h(NTag, { size: 'small', type: 'info' }, () => row.agent_number || `#${row.agent_id}`) : '-',
  },
  {
    title: '注册状态',
    key: 'register_status',
    width: 100,
    render: (row: DnInfo) => {
      const registered = row.register_status === 1
      return h(
        'span',
        {
          style: {
            display: 'inline-flex',
            alignItems: 'center',
            gap: '6px',
            color: registered ? '#18a058' : '#9ca3af',
          },
        },
        [
          h('span', {
            style: {
              width: '8px',
              height: '8px',
              borderRadius: '50%',
              background: registered ? '#18a058' : '#9ca3af',
            },
          }),
          registered ? '已注册' : '未注册',
        ]
      )
    },
  },
  {
    title: '状态',
    key: 'status',
    width: 80,
    render: (row: DnInfo) =>
      h(
        NTag,
        { size: 'small', type: row.status === 1 ? 'success' : 'default' },
        () => (row.status === 1 ? '启用' : '禁用')
      ),
  },
  {
    title: '注册IP',
    key: 'register_ip',
    width: 130,
    render: (row: DnInfo) => row.register_ip || '-',
  },
  {
    title: '最后注册时间',
    key: 'last_register_at',
    width: 160,
    render: (row: DnInfo) => formatTime(row.last_register_at),
  },
  {
    title: '操作',
    key: 'actions',
    width: 140,
    fixed: 'right' as const,
    render: (row: DnInfo) =>
      h('div', { style: { display: 'flex', gap: '8px', alignItems: 'center' } }, [
        h(NButton, { size: 'small', onClick: () => handleEdit(row) }, () => '编辑'),
        h(
          NDropdown,
          {
            trigger: 'click',
            options: [
              { label: '注册状态检测', key: 'check' },
              row.agent_id
                ? { label: '解绑座席', key: 'unbind' }
                : { label: '绑定座席', key: 'bind' },
              { label: '删除', key: 'delete' },
            ],
            onSelect: (key: string) => handleAction(key, row),
          },
          () => h(NButton, { size: 'small', quaternary: true }, () => '更多 ▾')
        ),
      ]),
  },
]

function formatTime(t?: string) {
  if (!t || t.startsWith('0001-01-01')) return '-'
  return t.replace('T', ' ').slice(0, 19)
}

// ===== 新增/编辑表单 =====
const showModal = ref(false)
const isEdit = ref(false)
const formRef = ref()

const emptyForm = () => ({
  id: 0,
  dn_num: '',
  dn_name: '',
  password: '',
  dn_type: 1,
  status: 1,
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
  dn_num: {
    required: true,
    pattern: /^\d{3,10}$/,
    message: '分机号须为 3~10 位数字',
    trigger: 'blur',
  },
}

// ===== 绑定座席 =====
const currentRow = ref<DnInfo | null>(null)
const showBindModal = ref(false)
const bindAgentId = ref<number | undefined>(undefined)
const agentOptions = ref<Array<{ label: string; value: number }>>([])

// ===== 数据加载 =====
async function loadData() {
  loading.value = true
  try {
    const res = await getDnList({
      page: pagination.page,
      page_size: pagination.pageSize,
      keyword: queryForm.keyword || undefined,
      status: queryForm.status,
      dn_type: queryForm.dn_type,
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
  queryForm.dn_type = undefined
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
  isEdit.value = false
  showModal.value = true
}

function handleEdit(row: DnInfo) {
  Object.assign(form, emptyForm(), {
    id: row.id,
    dn_num: row.dn_num,
    dn_name: row.dn_name,
    dn_type: row.dn_type || 1,
    status: row.status,
    remark: row.remark,
  })
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
    if (isEdit.value) {
      await updateDn({
        id: form.id,
        dn_name: form.dn_name,
        password: form.password || undefined,
        dn_type: form.dn_type,
        status: form.status,
        remark: form.remark,
      })
      message.success('更新成功')
    } else {
      await createDn({
        dn_num: form.dn_num,
        dn_name: form.dn_name,
        password: form.password || undefined,
        dn_type: form.dn_type,
        status: form.status,
        remark: form.remark,
      })
      message.success('创建成功')
    }
    showModal.value = false
    loadData()
  } catch (e) {
    // 错误提示由拦截器统一处理
  } finally {
    saving.value = false
  }
}

function handleDelete(row: DnInfo) {
  dialog.warning({
    title: '删除确认',
    content: `确定删除分机「${row.dn_num}」吗？其与座席的绑定关系将一并解除。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteDn(row.id)
        message.success('删除成功')
        loadData()
      } catch (e) {
        // 错误提示由拦截器统一处理
      }
    },
  })
}

// ===== 操作 =====
function handleAction(key: string, row: DnInfo) {
  if (key === 'check') {
    handleStatusCheck()
  } else if (key === 'bind') {
    openBindModal(row)
  } else if (key === 'unbind') {
    handleUnbind(row)
  } else if (key === 'delete') {
    handleDelete(row)
  }
}

// 注册状态检测：后端返回全量分机的最新注册状态，前端据此刷新列表
async function handleStatusCheck() {
  checking.value = true
  try {
    await checkDnStatus()
    message.success('状态检测完成')
    loadData()
  } catch (e) {
    // 错误提示由拦截器统一处理
  } finally {
    checking.value = false
  }
}

async function openBindModal(row: DnInfo) {
  currentRow.value = row
  bindAgentId.value = undefined
  try {
    const agents = await getAgentAll()
    agentOptions.value = (agents || []).map((a) => ({
      label: `${a.agent_name}（${a.agent_number}）`,
      value: a.id,
    }))
    showBindModal.value = true
  } catch (e) {
    // 错误提示由拦截器统一处理
  }
}

async function handleBindConfirm() {
  if (!currentRow.value || !bindAgentId.value) return
  saving.value = true
  try {
    await bindDnAgent(currentRow.value.id, bindAgentId.value)
    message.success('绑定成功')
    showBindModal.value = false
    loadData()
  } catch (e) {
    // 错误提示由拦截器统一处理
  } finally {
    saving.value = false
  }
}

function handleUnbind(row: DnInfo) {
  dialog.warning({
    title: '解绑确认',
    content: `确定解除分机「${row.dn_num}」与座席「${row.agent_number}」的绑定吗？`,
    positiveText: '解绑',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await unbindDnAgent(row.id)
        message.success('已解绑')
        loadData()
      } catch (e) {
        // 错误提示由拦截器统一处理
      }
    },
  })
}

onMounted(() => {
  loadData()
})
</script>

<style scoped lang="css">
.dn-page {
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
