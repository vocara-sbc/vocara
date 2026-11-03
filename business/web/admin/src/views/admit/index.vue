<template>
  <div class="admit-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form :model="queryForm" inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="配置名称/编码/接入号"
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
          <n-button type="primary" @click="handleSearch">查询</n-button>
          <n-button style="margin-left: 8px" @click="handleReset">重置</n-button>
        </n-form-item>
      </n-form>
    </n-card>

    <!-- 操作栏 + 表格 -->
    <n-card :bordered="false" class="table-card">
      <div class="table-header">
        <div class="table-title">呼入接待（DID 路由）</div>
        <div class="table-actions">
          <n-button type="primary" size="small" @click="handleAdd">
            新增呼入接待
          </n-button>
        </div>
      </div>

      <n-data-table
        :columns="columns"
        :data="tableData"
        :pagination="pagination"
        :loading="loading"
        :row-key="(row: InboundRouteInfo) => row.id"
        flex-height
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
      />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal v-model:show="showModal" preset="card" :title="isEdit ? '编辑呼入接待' : '新增呼入接待'" :style="{ width: '640px' }">
      <n-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-placement="left"
        label-width="110"
        size="medium"
      >
        <n-form-item label="配置名称" path="route_name">
          <n-input v-model:value="form.route_name" placeholder="如：客服热线 6000" />
        </n-form-item>
        <n-form-item label="配置编码" path="route_code">
          <n-input v-model:value="form.route_code" placeholder="如：ADMIT-6000（创建后不可修改）" :disabled="isEdit" />
        </n-form-item>
        <n-form-item label="呼入接入号" path="did">
          <n-input v-model:value="form.did" placeholder="客户拨打的号码（4~12 位数字，如 6000 或运营商 DID）" />
        </n-form-item>
        <n-form-item label="目标队列" path="queue_id">
          <n-select
            v-model:value="form.queue_id"
            :options="queueOptions"
            placeholder="选择呼入进入的队列"
            clearable
            filterable
          />
        </n-form-item>
        <n-form-item label="目标技能组" path="skill_group_id">
          <n-select
            v-model:value="form.skill_group_id"
            :options="skillGroupOptions"
            placeholder="可选，不选则不限技能组"
            clearable
            filterable
          />
        </n-form-item>
        <n-form-item label="状态" path="status">
          <n-switch v-model:value="statusSwitch" />
          <span style="margin-left: 8px; color: #6b7280">{{ statusSwitch ? '启用' : '停用' }}</span>
        </n-form-item>
        <n-form-item label="排序" path="sort">
          <n-input-number v-model:value="form.sort" :min="0" style="width: 160px" />
        </n-form-item>
        <n-form-item label="备注" path="remark">
          <n-input v-model:value="form.remark" type="textarea" :rows="2" placeholder="可选" />
        </n-form-item>

        <n-collapse>
          <n-collapse-item title="高级设置（欢迎音 / 工作时间 / 非工作时间处理）" name="advanced">
            <n-form-item label="欢迎音文件" path="welcome_sound">
              <n-input v-model:value="form.welcome_sound" placeholder="可选，如 welcome.wav" />
            </n-form-item>
            <n-form-item label="等待音文件" path="wait_sound">
              <n-input v-model:value="form.wait_sound" placeholder="可选，如 wait.wav" />
            </n-form-item>
            <n-form-item label="工作时间" path="work_time_type">
              <n-radio-group v-model:value="form.work_time_type">
                <n-radio :value="0">全天候</n-radio>
                <n-radio :value="1">自定义</n-radio>
              </n-radio-group>
            </n-form-item>
            <n-form-item v-if="form.work_time_type === 1" label="时间配置" path="work_time_config">
              <n-input v-model:value="form.work_time_config" type="textarea" :rows="2" placeholder='JSON，如 {"mon":"09:00-18:00"}' />
            </n-form-item>
            <n-form-item label="非工作处理" path="off_work_type">
              <n-select
                v-model:value="form.off_work_type"
                :options="offWorkOptions"
                style="width: 220px"
              />
            </n-form-item>
            <n-form-item label="处理值" path="off_work_value">
              <n-input v-model:value="form.off_work_value" placeholder="转接目标（座席/队列/留言信箱）" />
            </n-form-item>
            <n-form-item label="提示音" path="off_work_sound">
              <n-input v-model:value="form.off_work_sound" placeholder="可选" />
            </n-form-item>
          </n-collapse-item>
        </n-collapse>
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
import { NButton, NTag, NSwitch } from 'naive-ui'
import { message, dialog } from '@/utils/naive'
import {
  getInboundRoutePage,
  createInboundRoute,
  updateInboundRoute,
  deleteInboundRoute,
  type InboundRouteInfo,
} from '@/api/inboundRoute'
import { getQueueAll } from '@/api/queue'
import { getSkillGroupAll } from '@/api/agent'

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

const tableData = ref<InboundRouteInfo[]>([])
const pagination = reactive({
  page: 1,
  pageSize: 10,
  itemCount: 0,
  showSizePicker: true,
  pageSizes: [10, 20, 50],
})

// ===== 下拉选项 =====
const queueOptions = ref<Array<{ label: string; value: number }>>([])
const skillGroupOptions = ref<Array<{ label: string; value: number }>>([])

// 队列/技能组 id → 名称映射（表格展示用）
const queueNameMap = computed(() => {
  const m = new Map<number, string>()
  queueOptions.value.forEach((o) => m.set(o.value, o.label))
  return m
})
const skillGroupNameMap = computed(() => {
  const m = new Map<number, string>()
  skillGroupOptions.value.forEach((o) => m.set(o.value, o.label))
  return m
})

const offWorkOptions = [
  { label: '挂断', value: 0 },
  { label: '留言', value: 1 },
  { label: '转座席', value: 2 },
  { label: '转队列', value: 3 },
]

// ===== 表格列 =====
const columns = [
  { title: '配置名称', key: 'route_name', width: 160 },
  { title: '配置编码', key: 'route_code', width: 130 },
  {
    title: '呼入接入号',
    key: 'did',
    width: 120,
    render: (row: InboundRouteInfo) =>
      row.did ? h(NTag, { size: 'small', type: 'success' }, () => row.did) : '-',
  },
  {
    title: '目标队列',
    key: 'queue_id',
    width: 130,
    render: (row: InboundRouteInfo) =>
      row.queue_name || queueNameMap.value.get(row.queue_id || 0) || (row.queue_id ? `#${row.queue_id}` : '-'),
  },
  {
    title: '目标技能组',
    key: 'skill_group_id',
    width: 130,
    render: (row: InboundRouteInfo) =>
      row.skill_group_id
        ? skillGroupNameMap.value.get(row.skill_group_id) || `#${row.skill_group_id}`
        : '不限',
  },
  { title: '排序', key: 'sort', width: 70 },
  {
    title: '状态',
    key: 'status',
    width: 90,
    render: (row: InboundRouteInfo) =>
      h(
        NTag,
        { size: 'small', type: row.status === 1 ? 'success' : 'default' },
        () => (row.status === 1 ? '启用' : '停用')
      ),
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    render: (row: InboundRouteInfo) => [
      h(
        NButton,
        { size: 'small', style: 'margin-right: 8px', onClick: () => handleEdit(row) },
        () => '编辑'
      ),
      h(
        NButton,
        { size: 'small', type: 'error', quaternary: true, onClick: () => handleDelete(row) },
        () => '删除'
      ),
    ],
  },
]

// ===== 表单 =====
const showModal = ref(false)
const isEdit = ref(false)
const formRef = ref()
const emptyForm = () => ({
  id: 0,
  route_name: '',
  route_code: '',
  did: '',
  queue_id: undefined as number | undefined,
  skill_group_id: undefined as number | undefined,
  status: 1,
  sort: 0,
  remark: '',
  welcome_sound: '',
  wait_sound: '',
  work_time_type: 0,
  work_time_config: '',
  holiday_ids: '',
  off_work_type: 0,
  off_work_value: '',
  off_work_sound: '',
})
const form = reactive(emptyForm())

const statusSwitch = computed({
  get: () => form.status === 1,
  set: (v: boolean) => {
    form.status = v ? 1 : 0
  },
})

const formRules = {
  route_name: { required: true, message: '请输入配置名称', trigger: 'blur' },
  route_code: { required: true, message: '请输入配置编码', trigger: 'blur' },
  did: {
    required: true,
    pattern: /^\d{4,12}$/,
    message: '呼入接入号须为 4~12 位纯数字',
    trigger: 'blur',
  },
  queue_id: {
    required: true,
    type: 'number',
    message: '请选择目标队列',
    trigger: 'change',
  },
}

// ===== 数据加载 =====
async function loadOptions() {
  try {
    const [queues, skillGroups]: any[] = await Promise.all([getQueueAll(), getSkillGroupAll()])
    queueOptions.value = (queues || []).map((q: any) => ({
      label: q.queue_name || q.name || `#${q.id}`,
      value: q.id,
    }))
    skillGroupOptions.value = (skillGroups || []).map((g: any) => ({
      label: g.skill_name || g.name || `#${g.id}`,
      value: g.id,
    }))
  } catch (e) {
    // 选项加载失败不阻塞表格
  }
}

async function loadData() {
  loading.value = true
  try {
    const res: any = await getInboundRoutePage({
      page: pagination.page,
      page_size: pagination.pageSize,
      keyword: queryForm.keyword || undefined,
      status: queryForm.status,
    })
    tableData.value = res?.list || res?.data?.list || []
    pagination.itemCount = res?.total ?? res?.data?.total ?? 0
  } catch (e: any) {
    message.error(e?.message || '加载失败')
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

function handleEdit(row: InboundRouteInfo) {
  Object.assign(form, emptyForm(), {
    ...row,
    queue_id: row.queue_id || undefined,
    skill_group_id: row.skill_group_id || undefined,
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
      await updateInboundRoute({ ...form })
      message.success('更新成功')
    } else {
      await createInboundRoute({ ...form })
      message.success('创建成功')
    }
    showModal.value = false
    loadData()
  } catch (e: any) {
    message.error(e?.message || '保存失败')
  } finally {
    saving.value = false
  }
}

function handleDelete(row: InboundRouteInfo) {
  dialog.warning({
    title: '删除确认',
    content: `确定删除呼入接待「${row.route_name}」（接入号 ${row.did || '-'}）吗？删除后该号码的呼入将无法路由。`,
    positiveText: '删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteInboundRoute(row.id)
        message.success('删除成功')
        loadData()
      } catch (e: any) {
        message.error(e?.message || '删除失败')
      }
    },
  })
}

onMounted(() => {
  loadOptions()
  loadData()
})
</script>

<style scoped>
.admit-page {
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
  font-weight: 600;
  font-size: 14px;
}
</style>
