<template>
  <div class="lines-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="线路名称/编码"
            clearable
            style="width: 200px"
            @keyup.enter="handleSearch"
          />
        </n-form-item>
        <n-form-item label="线路类型">
          <n-select
            v-model:value="queryForm.line_type"
            :options="lineTypeOptions"
            placeholder="全部"
            clearable
            style="width: 140px"
          />
        </n-form-item>
        <n-form-item label="状态">
          <n-select
            v-model:value="queryForm.status"
            :options="statusOptions"
            placeholder="全部"
            clearable
            style="width: 140px"
          />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" @click="handleSearch">
            <template #icon>
              <n-icon>
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/>
                </svg>
              </n-icon>
            </template>
            查询
          </n-button>
          <n-button style="margin-left: 8px" @click="handleReset">重置</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 操作栏 + 表格 -->
    <n-card :bordered="false" class="table-card">
      <div class="table-header">
        <div class="table-title">线路列表</div>
        <div class="table-actions">
          <n-button size="small" :loading="refreshing" @click="handleRefresh">
            <template #icon>
              <n-icon size="14">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <polyline points="23 4 23 10 17 10"/>
                  <polyline points="1 20 1 14 7 14"/>
                  <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                </svg>
              </n-icon>
            </template>
            刷新
          </n-button>
          <n-button type="primary" size="small" style="margin-left: 8px" @click="handleAdd">
            <template #icon>
              <n-icon size="14">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
              </n-icon>
            </template>
            新增线路
          </n-button>
        </div>
      </div>

      <n-data-table
        :columns="columns"
        :data="tableData"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: LineInfo) => row.id"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="showModal" preset="card" :title="isEdit ? '编辑线路' : '新增线路'" :style="{ width: '640px' }">
      <n-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-placement="left"
        label-width="110"
        size="medium"
      >
        <n-form-item label="线路名称" path="line_name">
          <n-input v-model:value="form.line_name" placeholder="如：主用 SIP 中继" />
        </n-form-item>
        <n-form-item label="线路编码" path="line_code">
          <n-input v-model:value="form.line_code" placeholder="如：LINE-SIP-01（创建后不可修改）" :disabled="isEdit" />
        </n-form-item>
        <n-form-item label="线路类型" path="line_type">
          <n-radio-group v-model:value="form.line_type">
            <n-radio :value="1">SIP 中继</n-radio>
            <n-radio :value="2">PRI 中继</n-radio>
            <n-radio :value="3">模拟线</n-radio>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="运营商" path="provider">
          <n-input v-model:value="form.provider" placeholder="可选，如：电信/联通" />
        </n-form-item>
        <n-form-item label="网关地址" path="gateway">
          <n-input v-model:value="form.gateway" placeholder="如：192.168.1.100" />
        </n-form-item>
        <n-form-item label="网关端口" path="port">
          <n-input-number v-model:value="form.port" :min="1" :max="65535" style="width: 180px" />
        </n-form-item>
        <n-form-item label="认证用户名" path="username">
          <n-input v-model:value="form.username" placeholder="可选" />
        </n-form-item>
        <n-form-item label="认证密码" path="password">
          <n-input
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            :placeholder="isEdit ? '留空则不修改' : '可选'"
          />
        </n-form-item>
        <n-form-item label="主叫号码" path="caller_id">
          <n-input v-model:value="form.caller_id" placeholder="外呼时透传的主叫号码" />
        </n-form-item>
        <n-form-item label="最大通道数" path="max_channels">
          <n-input-number v-model:value="form.max_channels" :min="1" style="width: 180px" />
        </n-form-item>
        <n-form-item label="所属 FS 实例" path="fs_instance_id">
          <n-select
            v-model:value="form.fs_instance_id"
            :options="fsInstanceOptions"
            placeholder="可选"
            clearable
            filterable
          />
        </n-form-item>
        <n-form-item label="状态" path="status">
          <n-switch v-model:value="statusSwitch" />
          <span style="margin-left: 8px; color: #6b7280">{{ statusSwitch ? '启用' : '停用' }}</span>
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
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, h } from 'vue'
import { message, dialog } from '@/utils/naive'
import {
  getLinePage,
  createLine,
  updateLine,
  deleteLine,
} from '@/api/line'
import type { LineInfo } from '@/types/line'
import { getFsInstanceAll } from '@/api/system'

const loading = ref(false)
const saving = ref(false)
const refreshing = ref(false)

// ===== 查询 =====
const queryForm = reactive({
  keyword: '',
  line_type: undefined as number | undefined,
  status: undefined as number | undefined,
})

const lineTypeOptions = [
  { label: 'SIP 中继', value: 1 },
  { label: 'PRI 中继', value: 2 },
  { label: '模拟线', value: 3 },
]

const statusOptions = [
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
]

const lineTypeTextMap: Record<number, string> = {
  1: 'SIP 中继',
  2: 'PRI 中继',
  3: '模拟线',
}

// ===== FS 实例下拉 =====
const fsInstanceOptions = ref<Array<{ label: string; value: number }>>([])

const tableData = ref<LineInfo[]>([])
const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
})

// ===== 表格列 =====
const columns = [
  { title: '线路编码', key: 'line_code', width: 130 },
  { title: '线路名称', key: 'line_name', width: 150 },
  {
    title: '线路类型',
    key: 'line_type',
    width: 100,
    render: (row: LineInfo) => lineTypeTextMap[row.line_type] || '-',
  },
  {
    title: '运营商',
    key: 'provider',
    width: 100,
    render: (row: LineInfo) => row.provider || '-',
  },
  {
    title: '网关地址',
    key: 'gateway',
    width: 170,
    render: (row: LineInfo) => (row.gateway ? `${row.gateway}:${row.port || 5060}` : '-'),
  },
  {
    title: '主叫号码',
    key: 'caller_id',
    width: 130,
    render: (row: LineInfo) => row.caller_id || '-',
  },
  {
    title: '通道(已用/最大)',
    key: 'used_channels',
    width: 130,
    render: (row: LineInfo) => `${row.used_channels} / ${row.max_channels}`,
  },
  {
    title: '注册状态',
    key: 'register_status',
    width: 100,
    render: (row: LineInfo) => {
      const registered = row.register_status === 1
      return h(
        'span',
        {
          style: {
            display: 'inline-flex',
            alignItems: 'center',
            gap: '6px',
            color: registered ? '#2080f0' : '#9ca3af',
          },
        },
        [
          h('span', {
            style: {
              width: '8px',
              height: '8px',
              borderRadius: '50%',
              background: registered ? '#2080f0' : '#9ca3af',
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
    width: 100,
    render: (row: LineInfo) => {
      return h(
        'span',
        {
          style: {
            display: 'inline-flex',
            alignItems: 'center',
            gap: '6px',
            color: row.status === 1 ? '#18a058' : '#9ca3af',
          },
        },
        [
          h('span', {
            style: {
              width: '8px',
              height: '8px',
              borderRadius: '50%',
              background: row.status === 1 ? '#18a058' : '#9ca3af',
            },
          }),
          row.status === 1 ? '启用' : '停用',
        ]
      )
    },
  },
  {
    title: '操作',
    key: 'actions',
    width: 130,
    fixed: 'right' as const,
    render: (row: LineInfo) => {
      // 线路连通性测试暂无后端接口，「测试」入口暂不展示
      return h('div', { style: { display: 'flex', gap: '8px' } }, [
        h(
          'a',
          {
            style: { color: '#18a058', cursor: 'pointer', fontSize: '13px' },
            onClick: () => handleEdit(row),
          },
          '编辑'
        ),
        h(
          'a',
          {
            style: { color: '#d03050', cursor: 'pointer', fontSize: '13px' },
            onClick: () => handleDelete(row),
          },
          '删除'
        ),
      ])
    },
  },
]

// ===== 表单 =====
const showModal = ref(false)
const isEdit = ref(false)
const formRef = ref()
const emptyForm = () => ({
  id: 0,
  line_name: '',
  line_code: '',
  line_type: 1,
  provider: '',
  gateway: '',
  port: 5060,
  username: '',
  password: '',
  caller_id: '',
  max_channels: 30,
  fs_instance_id: undefined as number | undefined,
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
  line_name: { required: true, message: '请输入线路名称', trigger: 'blur' },
  line_code: { required: true, message: '请输入线路编码', trigger: 'blur' },
  gateway: { required: true, message: '请输入网关地址', trigger: 'blur' },
}

// ===== 数据加载 =====
async function loadFsInstances() {
  try {
    const list: any[] = await getFsInstanceAll()
    fsInstanceOptions.value = (list || []).map((f: any) => ({
      label: `${f.instance_code || f.instance_name || f.host}（${f.host}）`,
      value: f.id,
    }))
  } catch {
    // 选项加载失败不阻塞表格
  }
}

async function loadData() {
  loading.value = true
  try {
    const res = await getLinePage({
      page: pagination.page,
      page_size: pagination.pageSize,
      keyword: queryForm.keyword || undefined,
      line_type: queryForm.line_type,
      status: queryForm.status,
    })
    tableData.value = res?.list || []
    pagination.itemCount = res?.total ?? 0
  } catch {
    // 错误提示由请求拦截器统一处理，列表保持空态
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
  queryForm.line_type = undefined
  queryForm.status = undefined
  pagination.page = 1
  loadData()
}

function handlePageChange(page: number) {
  pagination.page = page
  loadData()
}

function handlePageSizeChange(size: number) {
  pagination.pageSize = size
  pagination.page = 1
  loadData()
}

// 刷新当前页列表
async function handleRefresh() {
  refreshing.value = true
  try {
    await loadData()
    message.success('已刷新')
  } finally {
    refreshing.value = false
  }
}

// ===== 新增/编辑/删除 =====
function handleAdd() {
  Object.assign(form, emptyForm())
  isEdit.value = false
  showModal.value = true
}

function handleEdit(row: LineInfo) {
  Object.assign(form, emptyForm(), {
    ...row,
    password: '',
    fs_instance_id: row.fs_instance_id || undefined,
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
  const payload = {
    id: form.id,
    line_name: form.line_name,
    line_code: form.line_code,
    line_type: form.line_type,
    provider: form.provider,
    gateway: form.gateway,
    port: form.port,
    username: form.username,
    password: form.password,
    caller_id: form.caller_id,
    max_channels: form.max_channels,
    fs_instance_id: form.fs_instance_id || 0,
    status: form.status,
    remark: form.remark,
  }
  try {
    if (isEdit.value) {
      await updateLine(payload)
      message.success('更新成功')
    } else {
      await createLine(payload)
      message.success('创建成功')
    }
    showModal.value = false
    loadData()
  } catch {
    // 错误提示由请求拦截器统一处理
  } finally {
    saving.value = false
  }
}

function handleDelete(row: LineInfo) {
  dialog.warning({
    title: '删除确认',
    content: `确定要删除线路「${row.line_name}」吗？删除后相关呼叫将无法使用该线路。`,
    positiveText: '确定删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteLine(row.id)
        message.success('删除成功')
        loadData()
      } catch {
        // 错误提示由请求拦截器统一处理
      }
    },
  })
}

onMounted(() => {
  loadFsInstances()
  loadData()
})
</script>

<style scoped lang="css">
.lines-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
}

.search-card :deep(.n-card-content) {
  padding-bottom: 8px;
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
