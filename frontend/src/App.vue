<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from './stores/auth'

const router = useRouter()
const auth = useAuthStore()

function handleLogout() {
  auth.logout()
  router.replace('/login')
}
</script>

<template>
  <div class="app-container">
    <header class="app-header" v-if="auth.isAuthenticated">
      <h1>E_AI_Art</h1>
      <nav>
        <router-link v-if="auth.isAdmin" to="/style-editor">风格配置</router-link>
        <router-link to="/generate">生图</router-link>
        <router-link to="/tasks">任务队列</router-link>
        <router-link v-if="auth.isAdmin" to="/projects">项目管理</router-link>
        <router-link v-if="auth.isAdmin" to="/users">用户管理</router-link>
        <router-link v-if="auth.isAdmin" to="/settings">系统设置</router-link>
      </nav>
      <div class="user-area">
        <span class="user-name">{{ auth.username }} ({{ auth.isAdmin ? '管理员' : '用户' }})</span>
        <button class="btn btn-logout" @click="handleLogout">退出</button>
      </div>
    </header>
    <main class="app-main">
      <router-view />
    </main>
  </div>
</template>

<style>
:root {
  --color-primary: #6366f1;
  --color-primary-hover: #4f46e5;
  --color-bg: #f8fafc;
  --color-surface: #ffffff;
  --color-border: #e2e8f0;
  --color-text: #1e293b;
  --color-text-muted: #64748b;
  --radius: 8px;
  --shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: var(--color-bg);
  color: var(--color-text);
}

.app-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  height: 56px;
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}

.app-header h1 {
  font-size: 18px;
  font-weight: 600;
}

.app-header nav {
  display: flex;
  gap: 16px;
}

.app-header nav a {
  color: var(--color-text-muted);
  text-decoration: none;
  font-size: 14px;
  padding: 6px 12px;
  border-radius: var(--radius);
}

.app-header nav a:hover,
.app-header nav a.router-link-exact-active {
  color: var(--color-primary);
  background: rgba(99, 102, 241, 0.08);
}

.user-area {
  display: flex;
  align-items: center;
  gap: 12px;
}

.user-name {
  font-size: 13px;
  color: var(--color-text-muted);
}

.btn-logout {
  padding: 4px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 13px;
  cursor: pointer;
  background: var(--color-surface);
  color: var(--color-text-muted);
}
.btn-logout:hover {
  color: #991b1b;
  border-color: #fca5a5;
}

.app-main {
  max-width: 1200px;
  margin: 0 auto;
  padding: 24px;
}
</style>
