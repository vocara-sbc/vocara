// schema rules → naive-ui FormItemRule：全部声明式翻译，不执行任何字符串脚本
import type { FormItemRule } from 'naive-ui'
import type { FormField } from './types'

const CHANGE_TRIGGER_TYPES = ['checkbox', 'select', 'radio', 'date', 'rate', 'switch']

export function buildNaiveRules(field: FormField): FormItemRule[] {
  const rules: FormItemRule[] = []
  const r = field.rules || {}
  const label = field.label || field.name
  const trigger = CHANGE_TRIGGER_TYPES.includes(field.type) ? ['change'] : ['blur', 'change']

  if (r.required) {
    rules.push({ required: true, message: `${label}不能为空`, trigger: trigger as any })
  }
  if (r.maxLength && r.maxLength > 0 && (field.type === 'input' || field.type === 'textarea')) {
    rules.push({
      validator: (_rule, value) => !value || String(value).length <= (r.maxLength || 0),
      message: `${label}长度不能超过${r.maxLength}字`,
      trigger: ['blur', 'change'],
    })
  }
  if (r.pattern && (field.type === 'input' || field.type === 'textarea')) {
    let regex: RegExp | null = null
    try {
      regex = new RegExp(r.pattern)
    } catch {
      regex = null // 非法正则视为不校验（设计器保存时后端已拦）
    }
    rules.push({
      validator: (_rule, value) => !value || !regex || regex.test(String(value)),
      message: r.patternHint || `${label}格式不正确`,
      trigger: ['blur', 'change'],
    })
  }
  if (field.type === 'number') {
    const min = field.props?.min
    const max = field.props?.max
    if (min !== undefined || max !== undefined) {
      rules.push({
        validator: (_rule, value) => {
          if (value === null || value === undefined || value === '') return true
          if (min !== undefined && value < min) return false
          if (max !== undefined && value > max) return false
          return true
        },
        message:
          min !== undefined && max !== undefined
            ? `${label}须在${min}~${max}之间`
            : min !== undefined
              ? `${label}不能小于${min}`
              : `${label}不能大于${max}`,
        trigger: ['blur', 'change'],
      })
    }
  }
  return rules
}
