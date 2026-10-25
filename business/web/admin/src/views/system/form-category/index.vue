<template>
  <div class="form-category-page">
    <!-- 搜索栏 -->
    <n-card :bordered="false" class="search-card">
      <n-form inline label-placement="left" label-width="70" size="medium">
        <n-form-item label="关键字">
          <n-input
            v-model:value="queryForm.keyword"
            placeholder="分类名称/编码"
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

    <n-card :bordered="false" class="table-card">
      <div class="table-header">
        <div class="table-title">表单分类</div>
        <n-button type="primary" size="small" @click="openEdit()">新增分类</n-button>
      </div>

      <n-data-table
        :columns="columns"
        :data="displayRows"
        :loading="loading"
        :row-key="(row: Row) => row.code"
        flex-height
      />
    </n-card>

    <!-- 新增/编辑弹窗 -->
    <n-modal
      v-model:show="editVisible"
      preset="card"
      :title="editing ? '编辑分类' : '新增分类'"
      style="width: 480px"
    >
      <n-form label-placement="left" label-width="72" size="small">
        <n-form-item label="分类编码">
          <n-input
            v-model:value="form.code"
            :disabled="!!editing"
            placeholder="英文标识，如 follow"
          />
        </n-form-item>
        <n-form-item label="分类名称">
          <n-input v-model:value="form.name" placeholder="显示名称" />
        </n-form-item>
        <n-form-item label="排序">
          <n-input-number v-model:value="form.sort" :min="1" style="width: 100%" />
        </n-form-item>
        <n-form-item label="启用">
          <n-switch
            v-model:value="form.enabled"
            :disabled="form.code === 'general'"
          />
        </n-form-item>
      </n-form>
      <template #footer>
        <n-button @click="editVisible = false">取消</n-button>
        <n-button type="primary" @click="confirmEdit">确定</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { h, reactive, ref, computed, onMounted } from 'vue'
import { NButton, NSwitch, NTag } from 'naive-ui'
import { message, dialog } from '@/utils/naive'
import {
  getFormCategories,
  saveFormCategories,
  type FormCategoryInfo,
} from '@/api/form'

type Row = FormCategoryInfo

const rows = ref<Row[]>([])
const loading = ref(false)
const editVisible = ref(false)
const editing = ref<Row | null>(null)
const form = reactive({ code: '', name: '', sort: 1, enabled: true })

// 查询表单：输入经"查询"按钮应用（appliedKeyword）
const queryForm = reactive({ keyword: '' })
const appliedKeyword = ref('')

// 展示行：仅按关键字过滤显示，全量保存仍基于 rows
const displayRows = computed(() => {
  const kw = appliedKeyword.value.trim()
  if (!kw) return rows.value
  return rows.value.filter(
    (r) => r.name.includes(kw) || r.code.includes(kw),
  )
})

function handleSearch() {
  appliedKeyword.value = queryForm.keyword
}

function handleReset() {
  queryForm.keyword = ''
  appliedKeyword.value = ''
}

// 加载分类列表
async function load() {
  loading.value = true
  try {
    const res: any = await getFormCategories()
    rows.value = res?.data || res || []
  } catch {
    // 拦截器已统一提示
  } finally {
    loading.value = false
  }
}

// 打开新增/编辑弹窗
function openEdit(row?: Row) {
  editing.value = row || null
  if (row) {
    Object.assign(form, { code: row.code, name: row.name, sort: row.sort, enabled: row.enabled })
  } else {
    Object.assign(form, { code: '', name: '', sort: rows.value.length + 1, enabled: true })
  }
  editVisible.value = true
}

// 提交编辑结果（全量保存）
async function persistNext(next: Row[]) {
  const res: any = await saveFormCategories(next)
  const code = res?.code ?? 0
  return code === 0 || code === 200
}

function confirmEdit() {
  if (!form.name.trim()) {
    message.warning('请填写分类名称')
    return
  }
  const next = [...rows.value]
  if (editing.value) {
    const i = next.findIndex((r) => r.code === editing.value!.code)
    next[i] = { ...next[i], name: form.name, sort: form.sort, enabled: form.enabled }
  } else {
    if (!/^[A-Za-z][A-Za-z0-9_]{1,31}$/.test(form.code)) {
      message.warning('编码需以字母开头、仅含字母数字下划线、2–32 位')
      return
    }
    if (next.some((r) => r.code === form.code)) {
      message.error('编码已存在')
      return
    }
    next.push({
      code: form.code, name: form.name, sort: form.sort,
      enabled: form.enabled, builtin: false,
    })
  }
  persistNext(next).then((ok) => {
    if (ok) {
      message.success('已保存')
      editVisible.value = false
      load()
    }
  })
}

// 表格内切换启用状态（general 不允许停用，后端亦有校验）
function toggleEnabled(row: Row, enabled: boolean) {
  const next = rows.value.map((r) => (r.code === row.code ? { ...r, enabled } : r))
  persistNext(next).then((ok) => {
    if (ok) {
      message.success(enabled ? '已启用' : '已停用')
      load()
    }
  })
}

// 删除分类（内置禁用；引用检查由后端执行）
function remove(row: Row) {
  dialog.warning({
    title: '删除确认',
    content: `确定删除分类「${row.name}」吗？该分类下有模板时将无法删除。`,
    onPositiveClick: () =>
      persistNext(rows.value.filter((r) => r.code !== row.code)).then((ok) => {
        if (ok) {
          message.success('已删除')
          load()
        }
      }),
  })
}

const columns = [
  { title: '分类名称', key: 'name' },
  {
    title: '编码',
    key: 'code',
    render: (row: Row) => h(NTag, { size: 'small', type: 'info' }, { default: () => row.code }),
  },
  { title: '排序', key: 'sort', width: 80 },
  {
    title: '状态',
    key: 'enabled',
    width: 100,
    render: (row: Row) =>
      h(NSwitch, {
        value: row.enabled,
        disabled: row.code === 'general',
        onUpdateValue: (v: boolean) => toggleEnabled(row, v),
      }),
  },
  {
    title: '内置',
    key: 'builtin',
    width: 80,
    render: (row: Row) => (row.builtin ? '是' : '-'),
  },
  {
    title: '操作',
    key: 'actions',
    width: 100,
    render: (row: Row) =>
      h(
        NButton,
        {
          size: 'small', quaternary: true, type: 'error',
          disabled: row.builtin, onClick: () => remove(row),
        },
        { default: () => '删除' },
      ),
  },
]

onMounted(load)
</script>

<style scoped>
.form-category-page { display: flex; flex-direction: column; gap: 16px; height: 100%; }
.search-card :deep(.n-card-content) { padding-bottom: 8px; }

/* 数据卡填满搜索区之外的剩余高度，表格主体内部滚动 */
.table-card { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.table-card :deep(.n-card-content) { flex: 1; min-height: 0; display: flex; flex-direction: column; }
.table-card :deep(.n-data-table) { flex: 1; min-height: 0; }
.table-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; }
.table-title { font-size: 14px; font-weight: 600; }
</style>
