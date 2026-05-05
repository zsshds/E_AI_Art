import { get, post, put } from './client'

export interface StyleProfile {
  id?: string
  name: string
  created_by: string
  version: number
  is_locked: boolean
  model: string
  art_style: string
  color_tone: string
  extra_tokens: string[]
  project_id: string
  reference_image_url: string
  locked_prompt_prefix: string
  created_at?: string
  updated_at?: string
}

export function listStyleProfiles(createdBy?: string): Promise<StyleProfile[]> {
  const query = createdBy ? `?created_by=${encodeURIComponent(createdBy)}` : ''
  return get<StyleProfile[]>(`/style-profiles${query}`)
}

export function getStyleProfile(id: string): Promise<StyleProfile> {
  return get<StyleProfile>(`/style-profiles/${id}`)
}

export function createStyleProfile(data: Partial<StyleProfile>): Promise<StyleProfile> {
  return post<StyleProfile>('/style-profiles', data)
}

export function updateStyleProfile(id: string, data: Partial<StyleProfile>): Promise<StyleProfile> {
  return put<StyleProfile>(`/style-profiles/${id}`, data)
}

export function lockStyleProfile(id: string): Promise<{ locked_prompt_prefix: string }> {
  return put<{ locked_prompt_prefix: string }>(`/style-profiles/${id}/lock`, {})
}

export function previewStyleProfile(id: string, userInput: string): Promise<any> {
  return post<any>(`/style-profiles/${id}/preview`, { user_input: userInput })
}
