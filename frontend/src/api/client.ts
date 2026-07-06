const BASE_URL = '/api/v1'

interface APIResponse<T = any> {
  code: number
  message: string
  data: T
}

function getToken(): string {
  return localStorage.getItem('token') || ''
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const extraHeaders = options?.headers ? Object.fromEntries(new Headers(options.headers).entries()) : {}
  const res = await fetch(`${BASE_URL}${url}`, {
    headers: {
      ...headers,
      ...extraHeaders,
    },
    ...options,
  })

  if (res.status === 401) {
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    localStorage.removeItem('role')
    window.location.href = '/login'
    throw new Error('登录已过期，请重新登录')
  }

  if (res.status === 413) {
    throw new Error('上传图片过大，请压缩图片或减少上传数量后重试')
  }

  const contentType = res.headers.get('content-type') || ''
  const bodyText = await res.text()
  if (!contentType.includes('application/json')) {
    throw new Error(res.ok ? '服务器返回了非 JSON 数据' : `请求失败（HTTP ${res.status}）`)
  }

  let json: APIResponse<T>
  try {
    json = JSON.parse(bodyText) as APIResponse<T>
  } catch {
    throw new Error(res.ok ? '服务器返回了无效的 JSON 数据' : `请求失败（HTTP ${res.status}）`)
  }

  if (!res.ok || json.code !== 0) {
    throw new Error(json.message || `请求失败（HTTP ${res.status}）`)
  }
  return json.data
}

export function get<T>(url: string): Promise<T> {
  return request<T>(url)
}

export function post<T>(url: string, body: any): Promise<T> {
  return request<T>(url, {
    method: 'POST',
    body: JSON.stringify(body),
  })
}

export function put<T>(url: string, body: any): Promise<T> {
  return request<T>(url, {
    method: 'PUT',
    body: JSON.stringify(body),
  })
}

export function del<T>(url: string): Promise<T> {
  return request<T>(url, { method: 'DELETE' })
}
