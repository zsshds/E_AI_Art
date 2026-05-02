import { get, put } from './client'

export interface AppSettings {
  api_base_url?: string
  api_key?: string
}

export function getSettings(): Promise<AppSettings> {
  return get<AppSettings>('/settings')
}

export function updateSettings(data: AppSettings): Promise<AppSettings> {
  return put<AppSettings>('/settings', data)
}
