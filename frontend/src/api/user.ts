import { get, post, del } from './client'

export interface User {
  id: string
  username: string
  role: string
  created_at: string
}

export const listUsers = () => get<User[]>('/users')
export const createUser = (username: string, password: string, role: string) =>
  post<User>('/auth/register', { username, password, role })
export const deleteUser = (id: string) => del<null>(`/users/${id}`)
