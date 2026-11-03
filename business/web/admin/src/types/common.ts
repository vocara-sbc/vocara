// 通用 API 响应类型
export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}

// 分页数据
export interface PageData<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
  pages: number
}

// 分页查询参数
export interface PageQuery {
  page?: number
  pageSize?: number
  keyword?: string
  [key: string]: any
}

// 通用选项
export interface OptionItem {
  label: string
  value: string | number
  disabled?: boolean
}
