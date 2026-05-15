import { get, put } from './client'

export interface AppSettings {
  api_base_url?: string
  api_key?: string
  api_generation_path?: string
  api_poll_path?: string
  available_models?: string
  model_fetch_url?: string
  model_filter_type?: string
}

export function getSettings(): Promise<AppSettings> {
  return get<AppSettings>('/settings')
}

const updatableSettingKeys = [
  'api_base_url',
  'api_key',
  'api_generation_path',
  'api_poll_path',
  'available_models',
  'model_fetch_url',
  'model_filter_type',
] as const

function buildUpdatePayload(data: AppSettings): AppSettings {
  const payload: AppSettings = {}
  for (const key of updatableSettingKeys) {
    const value = data[key]
    if (value !== undefined) {
      payload[key] = value
    }
  }
  return payload
}

export function updateSettings(data: AppSettings): Promise<AppSettings> {
  return put<AppSettings>('/settings', buildUpdatePayload(data))
}

// --- Model list ---

export interface ModelInfo {
  id: string
  object: string
  created: number
  owned_by: string
}

export interface ListModelsResponse {
  object: string
  data: ModelInfo[]
}

export function fetchModels(filterType?: string): Promise<ListModelsResponse> {
  const query = filterType ? `?filter=${filterType}` : ''
  return get<ListModelsResponse>('/settings/models' + query)
}

// --- Available models (configured by admin) ---

export interface AvailableModel {
  id: string
  type: string  // "gpt" | "gemini" | "mj" (banana merged into gemini)
  label: string
}

export function getAvailableModels(): Promise<AvailableModel[]> {
  return get<AvailableModel[]>('/settings/models/available')
}
