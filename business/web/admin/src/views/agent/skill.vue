<template>
  <div class="skill-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="技能组名称/编码"
            clearable
            style="width: 220px"
            @keyup.enter="handleSearch"
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
          <n-button type="primary" @click="handleSearch">查询</n-button>
          <n-button style="margin-left: 8px" @click="handleReset">重置</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 操作栏 + 表格 -->
    <n-card :bordered="false" class="table-card">
      <div class="table-header">
        <div class="table-title">技能组列表</div>
        <div class="table-actions">
          <n-button type="primary" size="small" @click="handleAdd">
            新增技能组
          </n-button>
        </div>
      </div>

      <n-data-table
        :columns="columns"
        :data="tableData"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: SkillGroup) => row.id"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="showModal" preset="card" :title="isEdit ? '编辑技能组' : '新增技能组'" :style="{ width: '480px' }">
      <n-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-placement="left"
        label-width="90"
        size="medium"
      >
        <n-form-item label="技能名称" path="skill_name">
          <n-input v-model:value="form.skill_name" placeholder="如：客服技能" />
        </n-form-item>
        <n-form-item label="技能编码" path="skill_code">
          <n-input v-model:value="form.skill_code" placeholder="如：S001（创建后不可修改）" :disabled="isEdit" />
        </n-form-item>
        <n-form-item label="技能类型" path="skill_type">
          <n-select v-model:value="form.skill_type" :options="skillTypeOptions" style="width: 200px" />
        </n-form-item>
        <n-form-item label="优先级" path="priority">
          <n-input-number v-model:value="form.priority" :min="0" style="width: 160px" />
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

    <!-- 座席配置抽屉 -->
    <n-drawer v-model:show="showAgentDrawer" :width="560">
      <n-drawer-content :title="`座席配置 - ${currentSkill?.skill_name || ''}`" closable>
        <div class="agent-add-bar">
          <n-select
            v-model:value="addAgentId"
            :options="addableAgentOptions"
            placeholder="选择要添加的座席"
            clearable
            filterable
            style="flex: 1"
          />
          <n-input-number v-model:value="addLevel" :min="1" :max="10" placeholder="等级" style="width: 100px" />
          <n-button type="primary" :disabled="!addAgentId" @click="handleAddRow">添加</n-button>
        </div>
        <n-data-table
          :columns="agentColumns"
          :data="agentRows"
          :loading="agentLoading"
          :row-key="(row: AgentConfigRow) => row.agent_id"
          size="small"
        />
        <div class="drawer-footer">
          <span class="drawer-tip">等级范围 1~10，保存后列表内座席的技能等级按所填覆盖</span>
          <n-button type="primary" :loading="agentSaving" @click="handleSaveAgents">保存配置</n-button>
        </div>
      </n-drawer-content>
    </n-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, h } from 'vue'
import { NButton, NTag, NInputNumber } from 'naive-ui'
import { message, dialog } from '@/utils/naive'
import {
  getSkillGroupList,
  createSkillGroup,
  updateSkillGroup,
  deleteSkillGroup,
  getSkillGroupAgents,
  setSkillGroupAgents,
  getAgentAll,
} from '@/api/agent'
import { SkillTypeOptions, type SkillGroup, type AgentInfo } from '@/types/agent'

// 抽屉内的座席配置行（agent_id → 技能等级）
interface AgentConfigRow {
  agent_id: number
  agent_number: string
  agent_name: string
  skill_level: number
}

const loading = ref(false)
const saving = ref(false)

// ===== 查询 =====
const queryForm = reactive({
  keyword: '',
  status: undefined as number | undefined,
})

const statusOptions = [
  { label: '启用', value: 1 },
  { label: '禁用', value: 0 },
]

const skillTypeOptions = SkillTypeOptions

const tableData = ref<SkillGroup[]>([])
const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
})

// ===== 表格列 =====
const columns = [
  { title: '技能编码', key: 'skill_code', width: 100 },
  { title: '技能名称', key: 'skill_name', width: 150 },
  {
    title: '技能类型',
    key: 'skill_type',
    width: 100,
    render: (row: SkillGroup) =>
      skillTypeOptions.find((t) => t.value === row.skill_type)?.label || '-',
  },
  { title: '优先级', key: 'priority', width: 80 },
  { title: '座席数', key: 'agent_count', width: 80 },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row: SkillGroup) =>
      h(
        NTag,
        { size: 'small', type: row.status === 1 ? 'success' : 'default' },
        () => (row.status === 1 ? '启用' : '禁用')
      ),
  },
  {
    title: '备注',
    key: 'remark',
    ellipsis: { tooltip: true },
    render: (row: SkillGroup) => row.remark || '-',
  },
  {
    title: '创建时间',
    key: 'created_at',
    width: 160,
    render: (row: SkillGroup) => formatTime(row.created_at),
  },
  {
    title: '操作',
    key: 'actions',
    width: 200,
    fixed: 'right' as const,
    render: (row: SkillGroup) =>
      h('div', { style: { display: 'flex', gap: '8px' } }, [
        h(NButton, { size: 'small', onClick: () => handleEdit(row) }, () => '编辑'),
        h(NButton, { size: 'small', onClick: () => openAgentDrawer(row) }, () => '座席配置'),
        h(
          NButton,
          { size: 'small', type: 'error', quaternary: true, onClick: () => handleDelete(row) },
          () => '删除'
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
  skill_name: '',
  skill_code: '',
  skill_type: 1,
  priority: 0,
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
  skill_name: { required: true, message: '请输入技能名称', trigger: 'blur' },
  skill_code: { required: true, message: '请输入技能编码', trigger: 'blur' },
}

// ===== 座席配置 =====
const showAgentDrawer = ref(false)
const currentSkill = ref<SkillGroup | null>(null)
const agentRows = ref<AgentConfigRow[]>([])
const agentLoading = ref(false)
const agentSaving = ref(false)
const allAgents = ref<AgentInfo[]>([])
const addAgentId = ref<number | undefined>(undefined)
const addLevel = ref<number>(5)

// 可添加的座席：排除已在配置列表中的座席
const addableAgentOptions = computed(() => {
  const rowIds = new Set(agentRows.value.map((r) => r.agent_id))
  return allAgents.value
    .filter((a) => !rowIds.has(a.id))
    .map((a) => ({ label: `${a.agent_name}（${a.agent_number}）`, value: a.id }))
})

const agentColumns = [
  { title: '工号', key: 'agent_number', width: 100 },
  { title: '姓名', key: 'agent_name', width: 110 },
  {
    title: '技能等级',
    key: 'skill_level',
    width: 140,
    render: (row: AgentConfigRow) =>
      h(NInputNumber, {
        size: 'small',
        min: 1,
        max: 10,
        value: row.skill_level,
        onUpdateValue: (v: number | null) => {
          row.skill_level = v ?? 5
        },
      }),
  },
]

// ===== 数据加载 =====
async function loadData() {
  loading.value = true
  try {
    const res = await getSkillGroupList({
      page: pagination.page,
      page_size: pagination.pageSize,
      keyword: queryForm.keyword || undefined,
      status: queryForm.status,
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

function handleEdit(row: SkillGroup) {
  Object.assign(form, emptyForm(), {
    id: row.id,
    skill_name: row.skill_name,
    skill_code: row.skill_code,
    skill_type: row.skill_type || 1,
    priority: row.priority,
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
      await updateSkillGroup({
        id: form.id,
        skill_name: form.skill_name,
        skill_type: form.skill_type,
        priority: form.priority,
        status: form.status,
        remark: form.remark,
      })
      message.success('更新成功')
    } else {
      await createSkillGroup({
        skill_name: form.skill_name,
        skill_code: form.skill_code,
        skill_type: form.skill_type,
        priority: form.priority,
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

function handleDelete(row: SkillGroup) {
  dialog.warning({
    title: '删除确认',
    content: `确定删除技能组「${row.skill_name}」吗？组内座席的技能配置将一并清除。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteSkillGroup(row.id)
        message.success('删除成功')
        loadData()
      } catch (e) {
        // 错误提示由拦截器统一处理
      }
    },
  })
}

// ===== 座席配置 =====
async function openAgentDrawer(row: SkillGroup) {
  currentSkill.value = row
  showAgentDrawer.value = true
  addAgentId.value = undefined
  addLevel.value = 5
  agentLoading.value = true
  try {
    const [agents, configured] = await Promise.all([getAgentAll(), getSkillGroupAgents(row.id)])
    allAgents.value = agents || []
    agentRows.value = (configured || []).map((c) => ({
      agent_id: c.agent_id,
      agent_number: c.agent_number,
      agent_name: c.agent_name,
      skill_level: c.skill_level || 5,
    }))
  } catch (e) {
    agentRows.value = []
  } finally {
    agentLoading.value = false
  }
}

function handleAddRow() {
  if (!addAgentId.value) return
  const agent = allAgents.value.find((a) => a.id === addAgentId.value)
  if (!agent) return
  agentRows.value.push({
    agent_id: agent.id,
    agent_number: agent.agent_number,
    agent_name: agent.agent_name,
    skill_level: addLevel.value || 5,
  })
  addAgentId.value = undefined
  addLevel.value = 5
}

// 保存座席配置
// 后端接口为增量覆盖语义：按提交的 agent_levels 更新各座席等级，不提供移除单个座席的入口，
// 因此抽屉仅支持添加座席与调整等级
async function handleSaveAgents() {
  if (!currentSkill.value) return
  agentSaving.value = true
  try {
    const levels: Record<number, number> = {}
    agentRows.value.forEach((r) => {
      levels[r.agent_id] = r.skill_level || 5
    })
    await setSkillGroupAgents(currentSkill.value.id, levels)
    message.success('保存成功')
    showAgentDrawer.value = false
    loadData()
  } catch (e) {
    // 错误提示由拦截器统一处理
  } finally {
    agentSaving.value = false
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped lang="css">
.skill-page {
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

.agent-add-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
}

.drawer-footer {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 12px;
}

.drawer-tip {
  font-size: 12px;
  color: #9ca3af;
}
</style>
