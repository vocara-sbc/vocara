/**
 * 文件下载工具：后端导出接口返回 blob（request 拦截器对 responseType=blob 直接返回完整响应），
 * 这里统一把响应体落盘，并从 Content-Disposition 提取文件名
 */

interface BlobResponse {
  data: Blob
  headers?: Record<string, string>
}

/** 从 Content-Disposition 头解析文件名，兼容 filename*=UTF-8'' 与 filename= 两种写法 */
function parseFileName(headers?: Record<string, string>): string {
  const disposition = headers?.['content-disposition'] || headers?.['Content-Disposition'] || ''
  const utf8Match = disposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8Match) {
    return decodeURIComponent(utf8Match[1].trim().replace(/"/g, ''))
  }
  const plainMatch = disposition.match(/filename=([^;]+)/i)
  if (plainMatch) {
    return plainMatch[1].trim().replace(/"/g, '')
  }
  return ''
}

/** 触发浏览器下载；fallbackName 在响应头未携带文件名时使用 */
export function downloadBlob(resp: BlobResponse, fallbackName: string) {
  const fileName = parseFileName(resp.headers) || fallbackName
  const url = URL.createObjectURL(resp.data)
  const a = document.createElement('a')
  a.href = url
  a.download = fileName
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}
