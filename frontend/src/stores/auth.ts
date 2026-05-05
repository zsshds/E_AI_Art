import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as apiLogin, type LoginResponse } from '../api/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const username = ref(localStorage.getItem('username') || '')
  const role = ref(localStorage.getItem('role') || '')

  const isAuthenticated = computed(() => !!token.value)
  const isAdmin = computed(() => role.value === 'admin')

  function setAuth(data: LoginResponse) {
    token.value = data.token
    username.value = data.username
    role.value = data.role
    localStorage.setItem('token', data.token)
    localStorage.setItem('username', data.username)
    localStorage.setItem('role', data.role)
  }

  async function login(usernameInput: string, password: string) {
    const data = await apiLogin(usernameInput, password)
    setAuth(data)
  }

  function logout() {
    token.value = ''
    username.value = ''
    role.value = ''
    localStorage.removeItem('token')
    localStorage.removeItem('username')
    localStorage.removeItem('role')
  }

  return { token, username, role, isAuthenticated, isAdmin, setAuth, login, logout }
})
