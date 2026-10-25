import { request } from '@/utils/request'

/** 授权绑定信息 */
export interface LicenseFingerprint {
  /** 绑定模式：host 主机指纹 / install 集群安装码 / none 不绑定 */
  mode?: string
  /** 绑定的指纹值集合 */
  values?: string[]
}

/** 授权容量配额，0 表示不限制 */
export interface LicenseLimits {
  /** 座席数 */
  agents?: number
  /** 并发通话数 */
  concurrent_calls?: number
  /** 线路数 */
  lines?: number
  /** 租户数 */
  tenants?: number
}

/** 授权信息 */
export interface LicenseInfo {
  /** 授权状态：valid/expiring/grace/expired/invalid */
  state: string
  /** 状态说明 */
  message: string
  /** 剩余天数；永久为 -1 */
  days_left: number
  /** 授权编号 */
  license_id?: string
  /** 客户名称 */
  customer?: string
  /** 产品版本 */
  edition?: string
  /** 授权类型 formal/trial */
  license_type?: string
  /** 签发时间 */
  issued_at?: string
  /** 生效时间 */
  not_before?: string
  /** 到期时间 */
  expire_at?: string
  /** 是否永久授权 */
  perpetual?: boolean
  /** 宽限天数 */
  grace_days?: number
  /** 功能模块开关 */
  modules?: Record<string, boolean>
  /** 容量配额 */
  limits?: LicenseLimits
  /** 授权绑定信息 */
  fingerprint?: LicenseFingerprint
  /** 安装标识（install 模式，客户凭此向厂商申请授权） */
  install_id?: string
  /** 本机机器码 */
  local_machine?: string
}

/** 查询当前授权信息 */
export function getLicenseInfo() {
  return request<LicenseInfo>({
    url: '/v1/admin/license/info',
    method: 'get',
  })
}

/** 导入 License 文本，校验通过后落盘并热加载 */
export function importLicense(content: string) {
  return request<LicenseInfo>({
    url: '/v1/admin/license/import',
    method: 'post',
    data: { content },
  })
}

/** 放弃内存状态，从磁盘重新加载 License */
export function reloadLicense() {
  return request<LicenseInfo>({
    url: '/v1/admin/license/reload',
    method: 'post',
  })
}
