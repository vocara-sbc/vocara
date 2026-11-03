// 线路管理相关类型（字段与后端 tb_line 模型一致，snake_case）

export interface LineInfo {
  id: number
  tenant_id: number
  line_name: string
  line_code: string
  // 线路类型：1-SIP中继 2-PRI中继 3-模拟线
  line_type: number
  provider?: string
  gateway?: string
  port?: number
  username?: string
  caller_id?: string
  max_channels: number
  used_channels: number
  status: number
  // 注册状态：0-未注册 1-已注册
  register_status: number
  skill_ids?: string
  fs_instance_id?: number
  last_register_at?: string
  remark?: string
  created_at?: string
  updated_at?: string
}

// 线路分页查询参数
export interface LinePageQuery {
  page?: number
  page_size?: number
  keyword?: string
  line_type?: number
  provider?: string
  status?: number
  register_status?: number
}
