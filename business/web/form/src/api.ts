// SDK 内部 HTTP 封装：不依赖宿主 axios，apiBase 与 token 由挂载参数注入
import type { FormRecord, FormTemplate, PageData, RecordCondition } from './schema/types'

export interface FormApiOptions {
  apiBase: string
  getToken: () => string
}

interface BizResponse<T = any> {
  code: number
  message: string
  data: T
}

export function createFormApi(opts: FormApiOptions) {
  async function call<T = any>(path: string, init?: RequestInit): Promise<T> {
    const resp = await fetch(opts.apiBase + path, {
      ...init,
      headers: {
        'Content-Type': 'application/json;charset=utf-8',
        Authorization: `Bearer ${opts.getToken()}`,
        ...(init?.headers || {}),
      },
    })
    const res: BizResponse<T> = await resp.json()
    if (res.code !== 0 && res.code !== 200) {
      throw new Error(res.message || '请求失败')
    }
    return res.data
  }

  function qs(params: Record<string, any>): string {
    const search = new URLSearchParams()
    Object.entries(params).forEach(([k, v]) => {
      if (v !== undefined && v !== null && v !== '') search.set(k, String(v))
    })
    const s = search.toString()
    return s ? `?${s}` : ''
  }

  return {
    templatePage: (params: Record<string, any>) =>
      call<PageData<FormTemplate>>(`/templates/page${qs(params)}`),
    templateList: (category?: string) =>
      call<FormTemplate[]>(`/templates/list${qs({ category })}`),
    templateGet: (id: number) => call<FormTemplate>(`/templates/${id}`),
    templateSave: (data: Record<string, any>) =>
      call<FormTemplate>(`/templates/save`, { method: 'POST', body: JSON.stringify(data) }),
    templateUpdate: (data: Record<string, any>) =>
      call(`/templates/update`, { method: 'POST', body: JSON.stringify(data) }),
    templateDelete: (id: number) => call(`/templates/${id}`, { method: 'DELETE' }),

    categoryList: () =>
      call<{ code: string; name: string; enabled: boolean }[]>(`/categories`),
    // 按模板标识取当前版本（含 schema）
    templateSchemaByKey: (key: string) =>
      call<{ version: number; schema: string }>(`/templates/key/${encodeURIComponent(key)}`),
    // 按版本 ID 取指定版本（含 schema），用于编辑老记录
    templateVersionGet: (id: number) =>
      call<{ version: number; schema: string }>(`/templates/version/${id}`),

    recordPage: (templateKey: string, conditions: RecordCondition[], page: number, pageSize: number) =>
      call<PageData<FormRecord>>(
        `/records/page${qs({ template_key: templateKey, conditions: conditions.length ? JSON.stringify(conditions) : '', page, page_size: pageSize })}`,
      ),
    recordGet: (id: number) => call<FormRecord>(`/records/${id}`),
    recordSave: (data: { template_key: string; record_key?: string; data: Record<string, any> }) =>
      call<FormRecord>(`/records/save`, { method: 'POST', body: JSON.stringify(data) }),
    recordUpdate: (data: { id: number; record_key?: string; data: Record<string, any> }) =>
      call(`/records/update`, { method: 'POST', body: JSON.stringify(data) }),
    recordDelete: (id: number) => call(`/records/${id}`, { method: 'DELETE' }),

    // 导出返回原始 Response（blob 由调用方处理）
    recordExport: async (templateKey: string, conditions: RecordCondition[]) => {
      const resp = await fetch(
        `${opts.apiBase}/records/export${qs({ template_key: templateKey, conditions: conditions.length ? JSON.stringify(conditions) : '' })}`,
        { headers: { Authorization: `Bearer ${opts.getToken()}` } },
      )
      if (!resp.ok) throw new Error(`导出失败 (${resp.status})`)
      return resp
    },
  }
}

export type FormApi = ReturnType<typeof createFormApi>
