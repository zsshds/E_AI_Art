<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { signup, register as apiRegister } from '../api/auth'

const router = useRouter()
const auth = useAuthStore()

const mode = ref<'login' | 'signup'>('login')
const username = ref('')
const password = ref('')
const error = ref('')
const success = ref('')
const loading = ref(false)

// Admin register form
const showAdminRegister = ref(false)
const regUsername = ref('')
const regPassword = ref('')
const regRole = ref('user')
const regError = ref('')
const regSuccess = ref('')

async function handleLogin() {
  if (!username.value || !password.value) return
  loading.value = true
  error.value = ''
  try {
    await auth.login(username.value, password.value)
    router.replace(auth.isAdmin ? '/style-editor' : '/generate')
  } catch (e: any) {
    error.value = e.message || '登录失败'
  } finally {
    loading.value = false
  }
}

async function handleSignup() {
  if (!username.value || !password.value) return
  if (username.value.length < 3) {
    error.value = '用户名至少3个字符'
    return
  }
  if (password.value.length < 6) {
    error.value = '密码至少6个字符'
    return
  }
  loading.value = true
  error.value = ''
  try {
    const data = await signup(username.value, password.value)
    auth.setAuth(data)
    router.replace('/generate')
  } catch (e: any) {
    error.value = e.message || '注册失败'
  } finally {
    loading.value = false
  }
}

function switchMode(m: 'login' | 'signup') {
  mode.value = m
  error.value = ''
  success.value = ''
}

async function handleAdminRegister() {
  if (!regUsername.value || !regPassword.value) return
  regError.value = ''
  regSuccess.value = ''
  try {
    await apiRegister(regUsername.value, regPassword.value, regRole.value)
    regSuccess.value = `用户 "${regUsername.value}" 创建成功`
    regUsername.value = ''
    regPassword.value = ''
  } catch (e: any) {
    regError.value = e.message || '注册失败'
  }
}
</script>

<template>
  <div class="login-container">
    <div class="login-card">
      <h2>AI 生图平台</h2>

      <!-- Tabs -->
      <div class="tabs">
        <button
          :class="['tab', { active: mode === 'login' }]"
          @click="switchMode('login')"
        >登录</button>
        <button
          :class="['tab', { active: mode === 'signup' }]"
          @click="switchMode('signup')"
        >注册</button>
      </div>

      <div class="form-group">
        <label>用户名</label>
        <input v-model="username" type="text" placeholder="请输入用户名" class="text-input"
          @keyup.enter="mode === 'login' ? handleLogin() : handleSignup()" :disabled="loading" />
      </div>

      <div class="form-group">
        <label>密码</label>
        <input v-model="password" type="password" placeholder="请输入密码" class="text-input"
          @keyup.enter="mode === 'login' ? handleLogin() : handleSignup()" :disabled="loading" />
      </div>

      <div v-if="error" class="error-msg">{{ error }}</div>
      <div v-if="success" class="success-msg">{{ success }}</div>

      <button
        v-if="mode === 'login'"
        class="btn btn-primary btn-large"
        @click="handleLogin"
        :disabled="loading || !username || !password"
      >
        {{ loading ? '登录中...' : '登录' }}
      </button>
      <button
        v-else
        class="btn btn-primary btn-large"
        @click="handleSignup"
        :disabled="loading || !username || !password"
      >
        {{ loading ? '注册中...' : '注册' }}
      </button>

      <!-- Admin: Register new users -->
      <div v-if="auth.isAdmin" class="admin-register-section">
        <hr />
        <button class="btn btn-text" @click="showAdminRegister = !showAdminRegister">
          {{ showAdminRegister ? '收起' : '+ 管理用户注册' }}
        </button>
        <div v-if="showAdminRegister" class="admin-register-form">
          <div class="form-group">
            <label>用户名</label>
            <input v-model="regUsername" type="text" placeholder="新用户名" class="text-input" />
          </div>
          <div class="form-group">
            <label>密码</label>
            <input v-model="regPassword" type="password" placeholder="新密码" class="text-input" />
          </div>
          <div class="form-group">
            <label>角色</label>
            <select v-model="regRole" class="select-input">
              <option value="user">普通用户</option>
              <option value="admin">管理员</option>
            </select>
          </div>
          <div v-if="regError" class="error-msg">{{ regError }}</div>
          <div v-if="regSuccess" class="success-msg">{{ regSuccess }}</div>
          <button class="btn btn-secondary" @click="handleAdminRegister" :disabled="!regUsername || !regPassword">
            创建用户
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 80vh;
}

.login-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 32px;
  width: 100%;
  max-width: 400px;
}

.login-card h2 {
  font-size: 24px;
  text-align: center;
  margin-bottom: 4px;
}

.tabs {
  display: flex;
  gap: 0;
  margin: 20px 0;
  border-radius: var(--radius);
  overflow: hidden;
  border: 1px solid var(--color-border);
}

.tab {
  flex: 1;
  padding: 8px;
  border: none;
  background: var(--color-bg);
  font-size: 14px;
  cursor: pointer;
  color: var(--color-text-muted);
}
.tab.active {
  background: var(--color-primary);
  color: white;
}

.form-group {
  margin-bottom: 16px;
}

.form-group label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 6px;
}

.text-input,
.select-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  background: var(--color-surface);
}

.btn-large {
  width: 100%;
  padding: 12px;
  font-size: 16px;
}

.error-msg {
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #fee2e2;
  color: #991b1b;
  border-radius: var(--radius);
  font-size: 13px;
}

.success-msg {
  margin-bottom: 12px;
  padding: 8px 12px;
  background: #d1fae5;
  color: #065f46;
  border-radius: var(--radius);
  font-size: 13px;
}

.admin-register-section {
  margin-top: 24px;
}

.admin-register-section hr {
  border: none;
  border-top: 1px solid var(--color-border);
  margin-bottom: 12px;
}

.admin-register-form {
  margin-top: 12px;
}

.btn {
  padding: 8px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  cursor: pointer;
  background: var(--color-surface);
}
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary { background: var(--color-primary); color: white; border-color: var(--color-primary); }
.btn-primary:hover:not(:disabled) { background: var(--color-primary-hover); }
.btn-secondary:hover:not(:disabled) { background: var(--color-bg); }
.btn-text {
  border: none;
  background: none;
  color: var(--color-primary);
  padding: 4px 0;
  font-size: 13px;
}
.btn-text:hover { text-decoration: underline; }
</style>
