// 真实挂载冒烟：happy-dom 加载构建产物，挂载渲染器与设计器，
// 按 DOM 产物判断 naive 组件是否真正解析渲染、栅格容器是否工作。
// 运行：npm run test:e2e（需先 npm run build）
import { Window } from 'happy-dom'
import { readFileSync } from 'fs'

const code = readFileSync('../admin/public/form/vocara-form.js', 'utf8')
const window = new Window({ url: 'https://localhost/' })
// 在 happy-dom 的全局作用域执行 IIFE，内部 document 等均为 happy-dom 实现
window.eval(code)

const tick = (ms = 100) => new Promise((r) => window.setTimeout(r, ms))

let pass = 0
const fail = []
function assert(cond, msg) {
  if (cond) pass++
  else fail.push(msg)
}

// 基础字段节点（测试数据）
function fieldNode(partial) {
  return {
    id: `f_${Math.random().toString(36).slice(2)}`,
    type: 'input', name: 'x', label: 'X', span: 24,
    props: {}, rules: {}, defaultValue: '', showInTable: true,
    tableWidth: 0, searchable: false, sort: 1,
    ...partial,
  }
}

// ===== 渲染器：一个 input + 一个 select =====
const schema = JSON.stringify({
  version: 1,
  formConfig: { labelWidth: 90, labelPlacement: 'left', size: 'medium' },
  fields: [
    fieldNode({ name: 'customer_name', label: '客户姓名', span: 12, props: { placeholder: '请输入', maxlength: 50 } }),
    fieldNode({ id: 'f_sel', type: 'select', name: 'tag', label: '标签', span: 12,
      props: { options: [{ label: 'A', value: 'a' }], multiple: false, placeholder: '请选择' }, defaultValue: null }),
  ],
})
const rEl = window.document.createElement('div')
window.document.body.appendChild(rEl)
window.VocaraForm.mountRenderer(rEl, { apiBase: '', getToken: () => '', schema })
await tick(200)

assert(rEl.querySelectorAll('input').length >= 1, '渲染器应渲染 NInput 的 <input>')
assert(!!rEl.querySelector('.n-input'), '应有 naive NInput 容器 .n-input')
assert(!!rEl.querySelector('.n-base-selection, .n-select'), '应有 naive NSelect 结构')

// ===== 渲染器：含栅格 schema 输出嵌套网格 =====
const gschema = JSON.stringify({
  version: 1, formConfig: {},
  fields: [{
    id: 'g1', type: 'grid',
    columns: [
      { span: 12, fields: [fieldNode({ name: 'a', label: 'A' })] },
      { span: 12, fields: [] },
    ],
  }],
})
const grEl = window.document.createElement('div')
window.document.body.appendChild(grEl)
window.VocaraForm.mountRenderer(grEl, { apiBase: '', getToken: () => '', schema: gschema })
await tick(200)
assert(grEl.querySelectorAll('.n-grid').length >= 1, '含栅格 schema 应渲染嵌套网格')
assert(!!grEl.querySelector('.n-grid input'), '栅格列内字段应可填')

// ===== 渲染器：日期字段点击弹出选择面板（初始值必须为 null，空串会导致不弹） =====
const dschema = JSON.stringify({
  version: 1, formConfig: {},
  fields: [{
    id: 'dd1', type: 'date', name: 'bday', label: '生日', span: 12,
    props: { format: 'yyyy-MM-dd' }, rules: {}, defaultValue: null,
    showInTable: true, tableWidth: 0, searchable: false, sort: 1,
  }],
})
const ddEl = window.document.createElement('div')
window.document.body.appendChild(ddEl)
window.VocaraForm.mountRenderer(ddEl, { apiBase: '', getToken: () => '', schema: dschema })
await tick(200)
const panelsBefore = window.document.body.querySelectorAll('.n-date-panel').length
;['mousedown', 'mouseup', 'click'].forEach((type) =>
  ddEl.querySelector('.n-input').dispatchEvent(new window.MouseEvent(type, { bubbles: true })),
)
await tick(400)
assert(window.document.body.querySelectorAll('.n-date-panel').length > panelsBefore, '点击日期框应弹出选择面板')

// ===== 设计器：空模板 =====
const dEl = window.document.createElement('div')
window.document.body.appendChild(dEl)
window.VocaraForm.mountDesigner(dEl, { apiBase: '', getToken: () => '' })
await tick(200)

const widgetItems = dEl.querySelectorAll('.widget-item')
assert(widgetItems.length >= 12, `设计器左侧应列出含栅格的全部字段类型，实际 ${widgetItems.length}`)
assert(!!dEl.querySelector('.designer-toolbar'), '设计器工具栏应渲染')

// 按标题找组件项
const widgetByTitle = (title) =>
  [...dEl.querySelectorAll('.widget-item')].find((w) => (w.textContent || '').includes(title))

// 双击「单行文本」加入画布
widgetByTitle('单行文本').dispatchEvent(new window.MouseEvent('dblclick', { bubbles: true }))
await tick(150)
const fieldNodes = dEl.querySelectorAll('.field-node')
assert(fieldNodes.length === 1, `双击后画布应有 1 个字段卡，实际 ${fieldNodes.length}`)
assert(!!dEl.querySelector('.field-node input'), '字段卡应渲染真实 input 控件')

// 点击字段 → 属性面板
fieldNodes[0].dispatchEvent(new window.MouseEvent('click', { bubbles: true }))
await tick(150)
const dText = dEl.textContent || ''
assert(dText.includes('字段属性'), '选中字段后属性面板应显示「字段属性」')
assert(dText.includes('字段标题'), '属性面板应渲染属性表单项')

// 点预览 → modal（teleport 到 body）
const previewBtn = [...dEl.querySelectorAll('.n-button')].find((b) => (b.textContent || '').includes('预览'))
assert(!!previewBtn, '应有预览按钮')
previewBtn.dispatchEvent(new window.MouseEvent('click', { bubbles: true }))
await tick(400)
assert((window.document.body.textContent || '').includes('表单预览'), '点预览应弹出「表单预览」弹窗')
// 关闭弹窗，避免影响后续
window.document.body.dispatchEvent(new window.MouseEvent('keydown', { key: 'Escape' }))
await tick(100)

// 双击「栅格」加入画布
widgetByTitle('栅格').dispatchEvent(new window.MouseEvent('dblclick', { bubbles: true }))
await tick(200)
assert(dEl.querySelectorAll('.grid-container').length === 1, '栅格容器应渲染')
assert(dEl.querySelectorAll('.grid-col').length === 2, '栅格默认应有两列')

// 加列 → 3 列
dEl.querySelector('.add-col').dispatchEvent(new window.MouseEvent('click', { bubbles: true }))
await tick(150)
assert(dEl.querySelectorAll('.grid-col').length === 3, '加列后应有三列')

console.log(`通过 ${pass} 项`)
if (fail.length) {
  console.log('失败项:')
  fail.forEach((f) => console.log('  ✗', f))
  process.exit(1)
}
console.log('全部 E2E 断言通过：naive 组件真实渲染、栅格容器工作正常')
