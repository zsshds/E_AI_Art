import { post } from './client'

export interface LoginResponse {
  token: string
  user_id: string
  username: string
  role: string
}

export interface RegisterResponse {
  user_id: string
  username: string
  role: string
}

export function login(username: string, password: string): Promise<LoginResponse> {
  return post<LoginResponse>('/auth/login', { username, password })
}

export function signup(username: string, password: string): Promise<LoginResponse> {
  return post<LoginResponse>('/auth/signup', { username, password })
}

export function register(username: string, password: string, role: string): Promise<RegisterResponse> {
  return post<RegisterResponse>('/auth/register', { username, password, role })
}
