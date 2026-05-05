import { get, post, put, del } from './client'

export interface Project {
  id?: string
  name: string
  created_by: string
  members: string[]
  created_at?: string
  updated_at?: string
}

export function listProjects(): Promise<Project[]> {
  return get<Project[]>('/projects')
}

export function createProject(name: string, members: string[]): Promise<Project> {
  return post<Project>('/projects', { name, members })
}

export function updateProject(id: string, data: { name?: string; members?: string[] }): Promise<Project> {
  return put<Project>(`/projects/${id}`, data)
}

export function deleteProject(id: string): Promise<void> {
  return del<void>(`/projects/${id}`)
}
