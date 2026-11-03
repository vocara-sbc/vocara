<template>
  <div class="system-config-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="配置键/配置值/描述"
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

    <n-card :bordered="false" class="config-card">
      <div class="table-header">
        <div class="table-title">参数配置项</div>
        <div class="table-actions">
          <n-button size="small" @click="loadData">
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
            新增配置项
          </n-button>
        </div>
      </div>

      <!-- 后端配置按 config_type 区分类型，页内按类型分区展示 -->
      <n-tabs v-model:value="activeType" type="line" animated>
        <n-tab-pane
          v-for="t in typeTabs"
          :key="t.value"
          :name="t.value"
          :tab="t.label"
        >
          <n-data-table
            :columns="columns"
            :data="filteredData"
            :loading="loading"
            :row-key="(row: SystemConfigItem) => row.id"
            flex-height
          />
        </n-tab-pane>
      </n-tabs>
    </n-card>

    <!-- 新增/编辑配置项弹窗 -->
    <n-modal v-model:show="showModal" preset="card" :title="isEdit ? '编辑配置项' : '新增配置项'" :style="{ width: '560px' }">
      <n-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-placement="left"
        label-width="90"
        size="medium"
      >
        <n-form-item label="配置键" path="config_key">
          <n-input v-model:value="form.config_key" placeholder="如：system.name（保存后不可修改）" :disabled="isEdit" />
        </n-form-item>
        <n-form-item label="配置值" path="config_value">
          <n-input v-model:value="form.config_value" type="textarea" :rows="3" placeholder="配置内容" />
        </n-form-item>
        <n-form-item label="配置类型" path="config_type">
          <n-select v-model:value="form.config_type" :options="configTypeOptions" style="width: 200px" />
        </n-form-item>
        <n-form-item label="配置描述" path="config_desc">
          <n-input v-model:value="form.config_desc" placeholder="可选，说明该配置的用途" />
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
  getConfigList,
  saveConfig,
  deleteConfig,
  type SystemConfigItem,
} from '@/api/system'

const loading = ref(false)
const saving = ref(false)

// 类型分区 tab：全部 + 后端支持的四种配置类型
const typeTabs = [
  { label: '全部', value: 'all' },
  { label: '字符串', value: 'string' },
  { label: '数值', value: 'int' },
  { label: '开关', value: 'bool' },
  { label: 'JSON', value: 'json' },
]
const activeType = ref('all')

const configTypeOptions = [
  { label: '字符串', value: 'string' },
  { label: '数值', value: 'int' },
  { label: '开关', value: 'bool' },
  { label: 'JSON', value: 'json' },
]

const typeTextMap: Record<string, string> = {
  string: '字符串',
  int: '数值',
  bool: '开关',
  json: 'JSON',
}

const tableData = ref<SystemConfigItem[]>([])

// 查询表单：输入经"查询"按钮应用（appliedKeyword）
const queryForm = reactive({ keyword: '' })
const appliedKeyword = ref('')

// 过滤：配置类型分区 + 关键字（匹配配置键/配置值/描述）叠加
const filteredData = computed(() => {
  const kw = appliedKeyword.value.trim()
  return tableData.value.filter((c) => {
    if (activeType.value !== 'all' && (c.config_type || 'string') !== activeType.value) {
      return false
    }
    if (
      kw &&
      !(c.config_key || '').includes(kw) &&
      !(c.config_value || '').includes(kw) &&
      !(c.config_desc || '').includes(kw)
    ) {
      return false
    }
    return true
  })
})

function handleSearch() {
  appliedKeyword.value = queryForm.keyword
}

function handleReset() {
  queryForm.keyword = ''
  appliedKeyword.value = ''
}

// ===== 表格列 =====
const columns = [
  { title: '配置键', key: 'config_key', width: 220 },
  {
    title: '配置值',
    key: 'config_value',
    ellipsis: { tooltip: true },
    render: (row: SystemConfigItem) => row.config_value || '-',
  },
  {
    title: '类型',
    key: 'config_type',
    width: 90,
    render: (row: SystemConfigItem) => typeTextMap[row.config_type] || row.config_type || '字符串',
  },
  {
    title: '描述',
    key: 'config_desc',
    width: 200,
    ellipsis: { tooltip: true },
    render: (row: SystemConfigItem) => row.config_desc || '-',
  },
  {
    title: '更新时间',
    key: 'updated_at',
    width: 170,
    render: (row: SystemConfigItem) =>
      row.updated_at ? row.updated_at.replace('T', ' ').slice(0, 19) : '-',
  },
  {
    title: '操作',
    key: 'actions',
    width: 130,
    fixed: 'right' as const,
    render: (row: SystemConfigItem) => {
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
const editingId = ref(0)
const formRef = ref()
const emptyForm = () => ({
  config_key: '',
  config_value: '',
  config_type: 'string',
  config_desc: '',
})
const form = reactive(emptyForm())

const formRules = {
  config_key: { required: true, message: '请输入配置键', trigger: 'blur' },
}

// ===== 数据加载 =====
async function loadData() {
  loading.value = true
  try {
    const res = await getConfigList()
    tableData.value = res || []
  } catch {
    // 错误提示由请求拦截器统一处理，列表保持空态
    tableData.value = []
  } finally {
    loading.value = false
  }
}

// ===== 新增/编辑/删除 =====
function handleAdd() {
  Object.assign(form, emptyForm())
  editingId.value = 0
  isEdit.value = false
  showModal.value = true
}

function handleEdit(row: SystemConfigItem) {
  Object.assign(form, emptyForm(), {
    config_key: row.config_key,
    config_value: row.config_value,
    config_type: row.config_type || 'string',
    config_desc: row.config_desc || '',
  })
  editingId.value = row.id
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
    // 保存接口按配置键写入新记录，编辑时先移除旧记录再写入
    if (isEdit.value && editingId.value > 0) {
      await deleteConfig(editingId.value)
    }
    await saveConfig({ ...form })
    message.success(isEdit.value ? '更新成功' : '创建成功')
    showModal.value = false
    loadData()
  } catch {
    // 错误提示由请求拦截器统一处理
  } finally {
    saving.value = false
  }
}

function handleDelete(row: SystemConfigItem) {
  dialog.warning({
    title: '删除确认',
    content: `确定要删除配置项「${row.config_key}」吗？删除后读取该配置的功能将使用默认值。`,
    positiveText: '确定删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      try {
        await deleteConfig(row.id)
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
.system-config-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  height: 100%;
}

.search-card :deep(.n-card-content) {
  padding-bottom: 8px;
}

/* 配置卡填满搜索区之外的剩余高度，类型页签内的表格主体内部滚动 */
.config-card {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.config-card :deep(.n-card-content) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.config-card :deep(.n-tabs) {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.config-card :deep(.n-tabs-pane-wrapper) {
  flex: 1;
  min-height: 0;
}

.config-card :deep(.n-tab-pane) {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.config-card :deep(.n-data-table) {
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
