import { get, post } from './client'

export interface Task {
  id?: string
  style_profile_id: string
  user_input: string
  model: string
  final_prompt: string
  status: 'pending' | 'processing' | 'done' | 'failed'
  result_image_url: string
  error_message: string
  retry_count: number
  created_by: string
  created_at: string
  updated_at: string
}

export function createTask(styleProfileId: string, userInput: string, model?: string): Promise<Task> {
  return post<Task>('/tasks', {
    style_profile_id: styleProfileId,
    user_input: userInput,
    model: model || '',
  })
}

export function getTask(id: string): Promise<Task> {
  return get<Task>(`/tasks/${id}`)
}

export function listTasks(createdBy?: string): Promise<Task[]> {
  const query = createdBy ? `?created_by=${encodeURIComponent(createdBy)}` : ''
  return get<Task[]>(`/tasks${query}`)
}

export function subscribeTask(taskId: string, onUpdate: (data: any) => void): () => void {
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const ws = new WebSocket(`${protocol}//${location.host}/ws/tasks/${taskId}`)

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      onUpdate(data)
    } catch {
      // ignore parse errors
    }
  }

  return () => ws.close()
}
