const BASE_URL = '/api/v1'

interface APIResponse<T = any> {
  code: number
  message: string
  data: T
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${url}`, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  const json: APIResponse<T> = await res.json()
  if (json.code !== 0) {
    throw new Error(json.message)
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
