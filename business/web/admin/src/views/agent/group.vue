<template>
  <div class="org-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="机构名称/编码/负责人"
            clearable
            style="width: 220px"
            @keyup.enter="handleSearch"
          />
        </n-form-item>
        <n-form-item>
          <n-button type="primary" @click="handleSearch">查询</n-button>
          <n-button style="margin-left: 8px" @click="handleReset">重置</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 操作栏 -->
    <n-card :bordered="false" class="table-card">
      <div class="table-header">
        <div class="table-title">组织机构</div>
        <n-button type="primary" size="small" @click="handleAdd(null)">新增顶级机构</n-button>
      </div>

      <!-- 机构树表 -->
      <n-data-table
        :columns="columns"
        :data="treeData"
        :loading="loading"
        :row-key="(row: Organization) => row.id"
        default-expand-all
        flex-height
      />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal
      v-model:show="showModal"
      preset="card"
      :title="isEdit ? '编辑机构' : '新增机构'"
      :style="{ width: '480px' }"
    >
      <n-form ref="formRef" :model="form" :rules="formRules" label-placement="left" label-width="90" size="medium">
        <n-form-item label="机构名称" path="org_name">
          <n-input v-model:value="form.org_name" placeholder="如：客户服务部" />
        </n-form-item>
        <n-form-item label="机构编码" path="org_code">
          <n-input v-model:value="form.org_code" placeholder="如：CS（创建后不可修改）" :disabled="isEdit" />
        </n-form-item>
        <n-form-item label="上级机构">
          <n-select
            v-model:value="form.parent_id"
            :options="parentOptions"
            placeholder="无（顶级机构）"
            clearable
          />
        </n-form-item>
        <n-form-item label="负责人">
          <n-input v-model:value="form.leader_name" placeholder="负责人姓名（可选）" />
        </n-form-item>
        <n-form-item label="排序">
          <n-input-number v-model:value="form.sort" :min="0" style="width: 160px" />
        </n-form-item>
        <n-form-item label="状态">
          <n-switch :value="statusSwitch" @update:value="statusSwitch = $event" />
          <span style="margin-left: 8px; color: #6b7280">{{ statusSwitch ? '启用' : '禁用' }}</span>
        </n-form-item>
        <n-form-item label="备注">
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
import { ref, reactive, computed, h, onMounted } from 'vue'
import { NButton, NTag, type DataTableColumns } from 'naive-ui'
import { message, dialog } from '@/utils/naive'
import {
  getOrganizationAll,
  createOrganization,
  updateOrganization,
  deleteOrganization,
} from '@/api/agent'
import type { Organization } from '@/types/agent'

const loading = ref(false)
const orgList = ref<Organization[]>([])
const showModal = ref(false)
const saving = ref(false)
const isEdit = ref(false)
const editingId = ref<number>(0)
const statusSwitch = ref(true)

const form = reactive({
  org_name: '',
  org_code: '',
  parent_id: null as number | null,
  leader_name: '',
  sort: 0,
  remark: '',
})

const formRules = {
  org_name: { required: true, message: '请输入机构名称', trigger: 'blur' },
}

// 加载全部机构
async function loadData() {
  loading.value = true
  try {
    const list = await getOrganizationAll()
    orgList.value = list || []
  } finally {
    loading.value = false
  }
}

// 扁平列表转树
function buildTree(list: Organization[], parentId = 0): Organization[] {
  return list
    .filter((o) => (o.parent_id || 0) === parentId)
    .map((o) => {
      const children = buildTree(list, o.id)
      return children.length ? { ...o, children } : { ...o }
    })
}

// 查询表单：输入经"查询"按钮应用（appliedKeyword）
const queryForm = reactive({ keyword: '' })
const appliedKeyword = ref('')

// 按关键字过滤扁平列表：命中节点（名称/编码/负责人）连同其全部祖先链保留，树结构不被截断
const filteredOrgList = computed(() => {
  const kw = appliedKeyword.value.trim()
  if (!kw) return orgList.value
  const kept = new Set<number>()
  orgList.value.forEach((o) => {
    const hit =
      o.org_name.includes(kw) ||
      o.org_code.includes(kw) ||
      (o.leader_name || '').includes(kw)
    if (!hit) return
    let cur: Organization | undefined = o
    while (cur && !kept.has(cur.id)) {
      kept.add(cur.id)
      cur = orgList.value.find((x) => x.id === (cur!.parent_id || 0))
    }
  })
  return orgList.value.filter((o) => kept.has(o.id))
})

const treeData = computed(() => buildTree(filteredOrgList.value))

function handleSearch() {
  appliedKeyword.value = queryForm.keyword
}

function handleReset() {
  queryForm.keyword = ''
  appliedKeyword.value = ''
}

// 上级机构选项：编辑时排除自己及其子孙
const parentOptions = computed(() => {
  const disabledIds = new Set<number>()
  if (isEdit.value) {
    const collect = (id: number) => {
      disabledIds.add(id)
      orgList.value.filter((o) => o.parent_id === id).forEach((c) => collect(c.id))
    }
    collect(editingId.value)
  }
  return orgList.value
    .filter((o) => !disabledIds.has(o.id))
    .map((o) => ({ label: `${'　'.repeat(Math.max(0, o.level - 1))}${o.org_name}`, value: o.id }))
})

function resetForm() {
  form.org_name = ''
  form.org_code = ''
  form.parent_id = null
  form.leader_name = ''
  form.sort = 0
  form.remark = ''
  statusSwitch.value = true
}

// 新增：parent 为父节点（点行内"新增下级"）或 null（顶级）
function handleAdd(parent: Organization | null) {
  resetForm()
  isEdit.value = false
  editingId.value = 0
  if (parent) form.parent_id = parent.id
  showModal.value = true
}

function handleEdit(row: Organization) {
  resetForm()
  isEdit.value = true
  editingId.value = row.id
  form.org_name = row.org_name
  form.org_code = row.org_code
  form.parent_id = row.parent_id || null
  form.leader_name = row.leader_name
  form.sort = row.sort
  form.remark = row.remark
  statusSwitch.value = row.status === 1
  showModal.value = true
}

async function handleSave() {
  saving.value = true
  try {
    const payload = {
      org_name: form.org_name,
      org_code: form.org_code,
      parent_id: form.parent_id || 0,
      leader_name: form.leader_name,
      sort: form.sort,
      status: statusSwitch.value ? 1 : 0,
      remark: form.remark,
    }
    if (isEdit.value) {
      await updateOrganization({ ...payload, id: editingId.value })
      message.success('已更新')
    } else {
      await createOrganization(payload)
      message.success('已创建')
    }
    showModal.value = false
    await loadData()
  } catch (e: any) {
    if (e?.message) message.error(e.message)
  } finally {
    saving.value = false
  }
}

function handleDelete(row: Organization) {
  dialog.warning({
    title: '删除确认',
    content: `确认删除机构「${row.org_name}」？`,
    onPositiveClick: async () => {
      try {
        await deleteOrganization(row.id)
        message.success('已删除')
        loadData()
      } catch (e: any) {
        message.error(e?.message || '删除失败')
      }
    },
  })
}

const columns: DataTableColumns<Organization> = [
  {
    title: '机构名称',
    key: 'org_name',
    render: (row) => h('span', row.org_name),
  },
  { title: '编码', key: 'org_code', width: 120 },
  { title: '负责人', key: 'leader_name', width: 120, render: (row) => row.leader_name || '-' },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row) =>
      h(NTag, { size: 'small', type: row.status === 1 ? 'success' : 'default' }, {
        default: () => (row.status === 1 ? '启用' : '禁用'),
      }),
  },
  {
    title: '操作',
    key: 'actions',
    width: 200,
    render: (row) => [
      h(NButton, { size: 'small', quaternary: true, type: 'primary', style: 'margin-right:6px', onClick: () => handleAdd(row) }, { default: () => '新增下级' }),
      h(NButton, { size: 'small', quaternary: true, style: 'margin-right:6px', onClick: () => handleEdit(row) }, { default: () => '编辑' }),
      h(NButton, { size: 'small', quaternary: true, type: 'error', onClick: () => handleDelete(row) }, { default: () => '删除' }),
    ],
  },
]

onMounted(loadData)
</script>

<style scoped>
.org-page { display: flex; flex-direction: column; gap: 16px; height: 100%; }
.search-card :deep(.n-card-content) { padding-bottom: 8px; }

/* 数据卡填满搜索区之外的剩余高度，树表主体内部滚动 */
.table-card { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.table-card :deep(.n-card-content) { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.table-card :deep(.n-data-table) { flex: 1; min-height: 0; }
.table-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.table-title { font-size: 14px; font-weight: 600; }
</style>
