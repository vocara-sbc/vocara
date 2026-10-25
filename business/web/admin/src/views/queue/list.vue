<template>
  <div class="queue-list-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="队列名称/编码"
            clearable
            style="width: 220px"
          />
        </n-form-item>
        <n-form-item label="类型">
          <n-select
            v-model:value="queryForm.queue_type"
            :options="queueTypeOptions"
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
        <div class="table-title">队列配置列表</div>
        <div class="table-actions">
          <n-button type="primary" size="small" @click="handleAdd">
            <template #icon>
              <n-icon size="14">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                  <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
                </svg>
              </n-icon>
            </template>
            新增队列
          </n-button>
        </div>
      </div>

      <n-data-table
        :columns="columns"
        :data="tableData"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: QueueInfo) => row.id"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="showModal" preset="card" :title="isEdit ? '编辑队列' : '新增队列'" :style="{ width: '640px' }">
      <n-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-placement="left"
        label-width="120"
        size="medium"
      >
        <n-form-item label="队列名称" path="queue_name">
          <n-input v-model:value="form.queue_name" placeholder="如：客服队列" />
        </n-form-item>
        <n-form-item label="队列编码" path="queue_code">
          <n-input v-model:value="form.queue_code" placeholder="如：QUEUE-CS（创建后不可修改）" :disabled="isEdit" />
        </n-form-item>
        <n-form-item label="队列类型" path="queue_type">
          <n-radio-group v-model:value="form.queue_type">
            <n-radio :value="1">呼入队列</n-radio>
            <n-radio :value="2">呼出队列</n-radio>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="分配策略" path="strategy">
          <n-select v-model:value="form.strategy" :options="strategyOptions" style="width: 240px" />
        </n-form-item>
        <n-form-item label="最大等待时长" path="max_wait_time">
          <n-input-number v-model:value="form.max_wait_time" :min="0" style="width: 180px" />
          <span style="margin-left: 8px; color: #6b7280">秒，超时按溢出策略处理</span>
        </n-form-item>
        <n-form-item label="最大等待人数" path="max_wait_num">
          <n-input-number v-model:value="form.max_wait_num" :min="0" style="width: 180px" />
          <span style="margin-left: 8px; color: #6b7280">0 表示不限制</span>
        </n-form-item>
        <n-form-item label="溢出处理" path="overflow_type">
          <n-select v-model:value="form.overflow_type" :options="overflowTypeOptions" style="width: 240px" />
        </n-form-item>
        <n-form-item v-if="form.overflow_type !== 0" label="溢出目标" path="overflow_value">
          <n-input v-model:value="form.overflow_value" placeholder="转接目标（队列/IVR/座席标识）" />
        </n-form-item>
        <n-form-item label="无座席处理" path="no_agent_type">
          <n-select v-model:value="form.no_agent_type" :options="noAgentTypeOptions" style="width: 240px" />
        </n-form-item>
        <n-form-item label="会后处理时长" path="wrap_up_time">
          <n-input-number v-model:value="form.wrap_up_time" :min="0" style="width: 180px" />
          <span style="margin-left: 8px; color: #6b7280">秒</span>
        </n-form-item>
        <n-form-item label="关联技能组" path="skillIds">
          <n-select
            v-model:value="form.skillIds"
            :options="skillGroupOptions"
            placeholder="可选，不选则不限技能组"
            multiple
            clearable
            filterable
          />
        </n-form-item>
        <n-form-item label="等待音文件" path="sound_file">
          <n-input v-model:value="form.sound_file" placeholder="可选，如 wait.wav" />
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
import { useRouter } from 'vue-router'
import { message, dialog } from '@/utils/naive'
import {
  getQueueList,
  createQueue,
  updateQueue,
  deleteQueue,
} from '@/api/queue'
import type { QueueInfo } from '@/types/queue'
import { getSkillGroupAll } from '@/api/agent'

const router = useRouter()

const loading = ref(false)
const saving = ref(false)

// ===== 查询 =====
const queryForm = reactive({
  keyword: '',
  queue_type: undefined as number | undefined,
  status: undefined as number | undefined,
})

const queueTypeOptions = [
  { label: '呼入队列', value: 1 },
  { label: '呼出队列', value: 2 },
]

const statusOptions = [
  { label: '启用', value: 1 },
  { label: '停用', value: 0 },
]

const strategyOptions = [
  { label: '轮询分配', value: 'round_robin' },
  { label: '最长空闲', value: 'longest_idle' },
  { label: '技能等级', value: 'skill_level' },
]

const overflowTypeOptions = [
  { label: '挂断', value: 0 },
  { label: '转队列', value: 1 },
  { label: '转 IVR', value: 2 },
  { label: '转座席', value: 3 },
]

const noAgentTypeOptions = [
  { label: '继续等待', value: 0 },
  { label: '溢出处理', value: 1 },
  { label: '挂断', value: 2 },
]

const strategyTextMap: Record<string, string> = {
  round_robin: '轮询分配',
  longest_idle: '最长空闲',
  skill_level: '技能等级',
}

const overflowTextMap: Record<number, string> = {
  0: '挂断',
  1: '转队列',
  2: '转 IVR',
  3: '转座席',
}

// ===== 技能组下拉 =====
const skillGroupOptions = ref<Array<{ label: string; value: number }>>([])

const tableData = ref<QueueInfo[]>([])
const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50, 100],
})

// ===== 表格列 =====
const columns = [
  { title: '队列编码', key: 'queue_code', width: 120 },
  { title: '队列名称', key: 'queue_name', width: 150 },
  {
    title: '类型',
    key: 'queue_type',
    width: 100,
    render: (row: QueueInfo) => (row.queue_type === 2 ? '呼出队列' : '呼入队列'),
  },
  {
    title: '分配策略',
    key: 'strategy',
    width: 110,
    render: (row: QueueInfo) => strategyTextMap[row.strategy] || row.strategy || '-',
  },
  {
    title: '最大等待人数',
    key: 'max_wait_num',
    width: 110,
    render: (row: QueueInfo) => (row.max_wait_num > 0 ? row.max_wait_num : '不限'),
  },
  {
    title: '最大等待时长',
    key: 'max_wait_time',
    width: 110,
    render: (row: QueueInfo) => `${row.max_wait_time} 秒`,
  },
  {
    title: '溢出处理',
    key: 'overflow_type',
    width: 100,
    render: (row: QueueInfo) => overflowTextMap[row.overflow_type] ?? '-',
  },
  { title: '当前等待', key: 'wait_num', width: 90 },
  {
    title: '就绪/忙碌',
    key: 'agent_ready_num',
    width: 100,
    render: (row: QueueInfo) => `${row.agent_ready_num} / ${row.agent_busy_num}`,
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row: QueueInfo) => {
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
    width: 200,
    fixed: 'right' as const,
    render: (row: QueueInfo) => {
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
            style: { color: '#2080f0', cursor: 'pointer', fontSize: '13px' },
            onClick: () => handleMonitor(row),
          },
          '实时监控'
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
  queue_name: '',
  queue_code: '',
  queue_type: 1,
  strategy: 'round_robin',
  max_wait_time: 60,
  max_wait_num: 0,
  overflow_type: 0,
  overflow_value: '',
  no_agent_type: 0,
  wrap_up_time: 0,
  skillIds: [] as number[],
  sound_file: '',
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
  queue_name: { required: true, message: '请输入队列名称', trigger: 'blur' },
  queue_code: { required: true, message: '请输入队列编码', trigger: 'blur' },
  strategy: { required: true, message: '请选择分配策略', trigger: 'change' },
}

// skill_ids 后端存储为 JSON 字符串（如 "[1,2]"），表单中用数组承载
function parseSkillIds(raw?: string): number[] {
  if (!raw) return []
  try {
    const arr = JSON.parse(raw)
    return Array.isArray(arr) ? arr.map(Number).filter((n) => !Number.isNaN(n)) : []
  } catch {
    return []
  }
}

// ===== 数据加载 =====
async function loadSkillGroups() {
  try {
    const list: any[] = await getSkillGroupAll()
    skillGroupOptions.value = (list || []).map((g: any) => ({
      label: g.skill_name || g.name || `#${g.id}`,
      value: g.id,
    }))
  } catch {
    // 选项加载失败不阻塞表格
  }
}

async function loadData() {
  loading.value = true
  try {
    const res = await getQueueList({
      page: pagination.page,
      page_size: pagination.pageSize,
      keyword: queryForm.keyword || undefined,
      queue_type: queryForm.queue_type,
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
  queryForm.queue_type = undefined
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

function handleEdit(row: QueueInfo) {
  Object.assign(form, emptyForm(), {
    ...row,
    skillIds: parseSkillIds(row.skill_ids),
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
    queue_name: form.queue_name,
    queue_code: form.queue_code,
    queue_type: form.queue_type,
    strategy: form.strategy,
    max_wait_time: form.max_wait_time,
    max_wait_num: form.max_wait_num,
    overflow_type: form.overflow_type,
    overflow_value: form.overflow_value,
    no_agent_type: form.no_agent_type,
    wrap_up_time: form.wrap_up_time,
    skill_ids: JSON.stringify(form.skillIds),
    sound_file: form.sound_file,
    status: form.status,
    remark: form.remark,
  }
  try {
    if (isEdit.value) {
      await updateQueue(payload)
      message.success('更新成功')
    } else {
      await createQueue(payload)
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

function handleMonitor(row: QueueInfo) {
  router.push('/queue/monitor')
}

function handleDelete(row: QueueInfo) {
  dialog.warning({
    title: '删除确认',
    content: `确定要删除队列「${row.queue_name}」吗？删除后进入该队列的呼叫将无法排队。`,
    positiveText: '确定删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteQueue(row.id)
        message.success('删除成功')
        loadData()
      } catch {
        // 错误提示由请求拦截器统一处理
      }
    },
  })
}

onMounted(() => {
  loadSkillGroups()
  loadData()
})
</script>

<style scoped lang="css">
.queue-list-page {
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
