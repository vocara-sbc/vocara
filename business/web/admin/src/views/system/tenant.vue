<template>
  <div class="system-tenant-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="租户名称/编码"
            clearable
            style="width: 220px"
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
        <div class="table-title">租户列表</div>
        <div class="table-actions">
          <n-button type="primary" size="small" @click="handleAdd">
            <template #icon>
              <n-icon size="14">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
              </n-icon>
            </template>
            新增租户
          </n-button>
        </div>
      </div>

      <n-data-table
        :columns="columns"
        :data="tableData"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: TenantInfo) => row.id"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="showModal" preset="card" :title="isEdit ? '编辑租户' : '新增租户'" :style="{ width: '640px' }">
      <n-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-placement="left"
        label-width="110"
        size="medium"
      >
        <n-form-item label="租户名称" path="tenant_name">
          <n-input v-model:value="form.tenant_name" placeholder="如：华创科技有限公司" />
        </n-form-item>
        <n-form-item label="租户编码" path="tenant_code">
          <n-input v-model:value="form.tenant_code" placeholder="如：T0002（创建后不可修改）" :disabled="isEdit" />
        </n-form-item>
        <n-form-item label="联系人" path="contact_name">
          <n-input v-model:value="form.contact_name" placeholder="可选" />
        </n-form-item>
        <n-form-item label="联系电话" path="contact_phone">
          <n-input v-model:value="form.contact_phone" placeholder="可选" />
        </n-form-item>
        <n-form-item label="联系邮箱" path="contact_email">
          <n-input v-model:value="form.contact_email" placeholder="可选" />
        </n-form-item>
        <n-form-item label="地址" path="address">
          <n-input v-model:value="form.address" placeholder="可选" />
        </n-form-item>
        <n-form-item label="座席配额" path="max_agents">
          <n-input-number v-model:value="form.max_agents" :min="0" style="width: 180px" />
          <span style="margin-left: 8px; color: #6b7280">0 表示不限制</span>
        </n-form-item>
        <n-form-item label="线路配额" path="max_lines">
          <n-input-number v-model:value="form.max_lines" :min="0" style="width: 180px" />
          <span style="margin-left: 8px; color: #6b7280">0 表示不限制</span>
        </n-form-item>
        <n-form-item label="过期时间" path="expire_time">
          <n-date-picker
            v-model:formatted-value="form.expire_time"
            type="date"
            value-format="yyyy-MM-dd"
            placeholder="不选择表示永久有效"
            clearable
            style="width: 240px"
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
  getTenantList,
  createTenant,
  updateTenant,
  deleteTenant,
  type TenantInfo,
} from '@/api/system'

const loading = ref(false)
const saving = ref(false)

// ===== 查询 =====
const queryForm = reactive({
  keyword: '',
  status: undefined as number | undefined,
})

const statusOptions = [
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
]

const tableData = ref<TenantInfo[]>([])
const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
})

// 过期时间后端返回 RFC3339，零值时间表示永久有效
function formatExpireTime(raw?: string): string {
  if (!raw || raw.startsWith('0001-01-01')) return '永久'
  return raw.slice(0, 10)
}

// ===== 表格列 =====
const columns = [
  { title: '租户编码', key: 'tenant_code', width: 110 },
  { title: '租户名称', key: 'tenant_name', width: 180 },
  {
    title: '联系人',
    key: 'contact_name',
    width: 100,
    render: (row: TenantInfo) => row.contact_name || '-',
  },
  {
    title: '联系电话',
    key: 'contact_phone',
    width: 130,
    render: (row: TenantInfo) => row.contact_phone || '-',
  },
  {
    title: '座席配额',
    key: 'max_agents',
    width: 100,
    render: (row: TenantInfo) => (row.max_agents > 0 ? row.max_agents : '不限'),
  },
  {
    title: '线路配额',
    key: 'max_lines',
    width: 100,
    render: (row: TenantInfo) => (row.max_lines > 0 ? row.max_lines : '不限'),
  },
  {
    title: '过期时间',
    key: 'expire_time',
    width: 120,
    render: (row: TenantInfo) => formatExpireTime(row.expire_time),
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row: TenantInfo) => {
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
    title: '创建时间',
    key: 'created_at',
    width: 170,
    render: (row: TenantInfo) => (row.created_at ? row.created_at.replace('T', ' ').slice(0, 19) : '-'),
  },
  {
    title: '操作',
    key: 'actions',
    width: 130,
    fixed: 'right' as const,
    render: (row: TenantInfo) => {
      // 租户独立配置与登录态切换暂无后端接口，「配置/切换」入口暂不展示
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
  tenant_name: '',
  tenant_code: '',
  contact_name: '',
  contact_phone: '',
  contact_email: '',
  address: '',
  max_agents: 0,
  max_lines: 0,
  expire_time: null as string | null,
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
  tenant_name: { required: true, message: '请输入租户名称', trigger: 'blur' },
  tenant_code: { required: true, message: '请输入租户编码', trigger: 'blur' },
}

// ===== 数据加载 =====
async function loadData() {
  loading.value = true
  try {
    const res = await getTenantList({
      page: pagination.page,
      page_size: pagination.pageSize,
      keyword: queryForm.keyword || undefined,
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

// ===== 新增/编辑/删除 =====
function handleAdd() {
  Object.assign(form, emptyForm())
  isEdit.value = false
  showModal.value = true
}

function handleEdit(row: TenantInfo) {
  const expireText = formatExpireTime(row.expire_time)
  Object.assign(form, emptyForm(), {
    ...row,
    expire_time: expireText === '永久' ? null : expireText,
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
  // 后端按 2006-01-02 解析过期时间，空串表示永久有效
  const payload = {
    id: form.id,
    tenant_name: form.tenant_name,
    tenant_code: form.tenant_code,
    contact_name: form.contact_name,
    contact_phone: form.contact_phone,
    contact_email: form.contact_email,
    address: form.address,
    max_agents: form.max_agents,
    max_lines: form.max_lines,
    expire_time: form.expire_time || '',
    status: form.status,
    remark: form.remark,
  }
  try {
    if (isEdit.value) {
      await updateTenant(payload)
      message.success('更新成功')
    } else {
      await createTenant(payload)
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

function handleDelete(row: TenantInfo) {
  dialog.warning({
    title: '删除确认',
    content: `确定要删除租户「${row.tenant_name}」吗？该操作不可恢复。`,
    positiveText: '确定删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteTenant(row.id)
        message.success('删除成功')
        loadData()
      } catch {
        // 错误提示由请求拦截器统一处理
      }
    },
  })
}

onMounted(() => {
  loadData()
})
</script>

<style scoped lang="css">
.system-tenant-page {
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
