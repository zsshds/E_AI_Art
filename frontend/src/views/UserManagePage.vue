<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listUsers, createUser, deleteUser, type User } from '../api/user'

const users = ref<User[]>([])
const loading = ref(false)
const message = ref('')
const isError = ref(false)

const form = ref({ username: '', password: '', role: 'user' })
const submitting = ref(false)

async function load() {
  loading.value = true
  try {
    users.value = await listUsers()
  } catch (e: any) {
    showMsg(e.message, true)
  } finally {
    loading.value = false
  }
}

onMounted(load)

function showMsg(msg: string, error = false) {
  message.value = msg
  isError.value = error
  setTimeout(() => (message.value = ''), 3000)
}

async function handleCreate() {
  if (!form.value.username || !form.value.password) return
  submitting.value = true
  try {
    await createUser(form.value.username, form.value.password, form.value.role)
    form.value = { username: '', password: '', role: 'user' }
    showMsg('用户创建成功')
    await load()
  } catch (e: any) {
    showMsg(e.message, true)
  } finally {
    submitting.value = false
  }
}

async function handleDelete(user: User) {
  if (!confirm(`确认删除用户 "${user.username}"？`)) return
  try {
    await deleteUser(user.id)
    showMsg('用户已删除')
    await load()
  } catch (e: any) {
    showMsg(e.message, true)
  }
}
</script>

<template>
  <div class="user-manage">
    <h2>用户管理</h2>
    <p class="subtitle">创建账号并管理用户权限</p>

    <div class="card">
      <h3>创建用户</h3>
      <div class="form-row">
        <div class="form-group">
          <label>用户名</label>
          <input v-model="form.username" type="text" class="text-input" placeholder="至少 3 个字符" />
        </div>
        <div class="form-group">
          <label>密码</label>
          <input v-model="form.password" type="password" class="text-input" placeholder="至少 6 个字符" />
        </div>
        <div class="form-group">
          <label>角色</label>
          <select v-model="form.role" class="text-input">
            <option value="user">普通用户</option>
            <option value="admin">管理员</option>
          </select>
        </div>
      </div>
      <button class="btn btn-primary" @click="handleCreate" :disabled="submitting">
        {{ submitting ? '创建中...' : '创建用户' }}
      </button>
    </div>

    <div v-if="message" class="message" :class="{ error: isError }">{{ message }}</div>

    <div class="card">
      <h3>用户列表</h3>
      <div v-if="loading" class="empty">加载中...</div>
      <table v-else-if="users.length" class="user-table">
        <thead>
          <tr>
            <th>用户名</th>
            <th>角色</th>
            <th>创建时间</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="u in users" :key="u.id">
            <td>{{ u.username }}</td>
            <td><span class="role-badge" :class="u.role">{{ u.role === 'admin' ? '管理员' : '用户' }}</span></td>
            <td class="muted">{{ new Date(u.created_at).toLocaleDateString('zh-CN') }}</td>
            <td><button class="btn-del" @click="handleDelete(u)">删除</button></td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty">暂无用户</div>
    </div>
  </div>
</template>

<style scoped>
.user-manage { max-width: 800px; margin: 0 auto; }
.user-manage h2 { font-size: 20px; margin-bottom: 4px; }
.subtitle { font-size: 14px; color: var(--color-text-muted); margin-bottom: 24px; }

.card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 20px;
  margin-bottom: 16px;
}
.card h3 { font-size: 15px; font-weight: 500; margin-bottom: 16px; }

.form-row { display: flex; gap: 12px; margin-bottom: 16px; flex-wrap: wrap; }
.form-group { flex: 1; min-width: 140px; }
.form-group label { display: block; font-size: 13px; font-weight: 500; margin-bottom: 6px; }
.text-input { width: 100%; padding: 8px 10px; border: 1px solid var(--color-border); border-radius: var(--radius); font-size: 14px; background: var(--color-surface); }

.message { padding: 8px 12px; border-radius: var(--radius); font-size: 13px; margin-bottom: 16px; background: #d1fae5; color: #065f46; }
.message.error { background: #fee2e2; color: #991b1b; }

.user-table { width: 100%; border-collapse: collapse; font-size: 14px; }
.user-table th { text-align: left; padding: 8px 12px; font-size: 12px; color: var(--color-text-muted); border-bottom: 1px solid var(--color-border); }
.user-table td { padding: 10px 12px; border-bottom: 1px solid var(--color-border); }
.user-table tr:last-child td { border-bottom: none; }

.role-badge { padding: 2px 8px; border-radius: 12px; font-size: 12px; }
.role-badge.admin { background: #ede9fe; color: #5b21b6; }
.role-badge.user { background: #f1f5f9; color: var(--color-text-muted); }

.muted { color: var(--color-text-muted); font-size: 13px; }

.btn-del { background: none; border: 1px solid #fca5a5; color: #991b1b; border-radius: var(--radius); padding: 4px 10px; font-size: 13px; cursor: pointer; }
.btn-del:hover { background: #fee2e2; }

.empty { padding: 24px; text-align: center; font-size: 13px; color: var(--color-text-muted); }

.btn { padding: 8px 16px; border: 1px solid var(--color-border); border-radius: var(--radius); font-size: 14px; cursor: pointer; }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary { background: var(--color-primary); color: white; border-color: var(--color-primary); }
.btn-primary:hover:not(:disabled) { background: var(--color-primary-hover); }
</style>
