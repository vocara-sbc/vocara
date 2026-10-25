import type { FormField, FormNode } from './types'
import { isGridNode } from './types'

// walkFields 深度遍历：字段节点执行 fn，栅格节点逐列递归
export function walkFields(nodes: FormNode[], fn: (f: FormField) => void): void {
  nodes.forEach((node) => {
    if (isGridNode(node)) {
      node.columns.forEach((col) => walkFields(col.fields, fn))
    } else {
      fn(node)
    }
  })
}

// collectFields 拍平收集全部字段节点（与布局无关）
export function collectFields(nodes: FormNode[]): FormField[] {
  const out: FormField[] = []
  walkFields(nodes, (f) => out.push(f))
  return out
}
