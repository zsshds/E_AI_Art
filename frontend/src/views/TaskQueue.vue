<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { listTasks, type Task } from '../api/task'

const router = useRouter()
const tasks = ref<Task[]>([])
const filter = ref<'all' | 'pending' | 'processing' | 'done' | 'failed'>('all')
const loading = ref(false)
const downloading = ref(false)

onMounted(async () => {
  await refresh()
})

async function refresh() {
  loading.value = true
  try {
    tasks.value = await listTasks()
  } finally {
    loading.value = false
  }
}

function goToTask(taskId: string) {
  router.push({ name: 'TaskDetail', params: { id: taskId } })
}

async function handleDownload(taskId: string) {
  if (!taskId) return
  downloading.value = true
  try {
    const token = localStorage.getItem('token')
    const resp = await fetch(`/api/v1/tasks/${taskId}/download`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!resp.ok) throw new Error('下载失败')
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `E_AI_Art-${taskId.slice(-8)}.png`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (e: any) {
    alert(e.message || '下载失败')
  } finally {
    downloading.value = false
  }
}

const filteredTasks = computed(() => {
  if (filter.value === 'all') return tasks.value
  return tasks.value.filter(t => t.status === filter.value)
})

const statusLabels: Record<string, string> = {
  pending: '排队中',
  processing: '生成中',
  done: '已完成',
  failed: '失败',
}
</script>

<template>
  <div class="review-queue">
    <div class="queue-header">
      <h2>任务队列</h2>
      <button class="btn btn-secondary" @click="refresh" :disabled="loading">
        {{ loading ? '刷新中...' : '刷新' }}
      </button>
    </div>

    <div class="filter-bar">
      <button
        v-for="(label, status) in { all: '全部', ...statusLabels }"
        :key="status"
        class="chip"
        :class="{ active: filter === status }"
        @click="filter = status as any"
      >{{ label }}</button>
    </div>

    <div v-if="filteredTasks.length === 0" class="empty-state">
      暂无任务记录
    </div>

    <div v-else class="task-table">
      <div class="table-header">
        <span class="col-id">任务ID</span>
        <span class="col-user">创建者</span>
        <span class="col-input">用户输入</span>
        <span class="col-status">状态</span>
        <span class="col-time">创建时间</span>
      </div>
      <div
        v-for="task in filteredTasks"
        :key="task.id"
        class="table-row"
        @click="goToTask(task.id!)"
      >
        <span class="col-id" :title="task.id">{{ task.id?.slice(-8) }}</span>
        <span class="col-user">{{ task.created_by }}</span>
        <span class="col-input">{{ task.user_input }}</span>
        <span class="col-status">
          <span class="status-badge" :class="task.status">{{ statusLabels[task.status] }}</span>
        </span>
        <span class="col-time">{{ new Date(task.created_at).toLocaleString() }}</span>
      </div>
    </div>

  </div>
</template>

<style scoped>
.review-queue h2 {
  font-size: 20px;
}

.queue-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.filter-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}

.chip {
  padding: 6px 16px;
  border: 1px solid var(--color-border);
  border-radius: 16px;
  background: var(--color-surface);
  font-size: 13px;
  cursor: pointer;
}
.chip.active {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.08);
  color: var(--color-primary);
}

.empty-state {
  text-align: center;
  padding: 48px;
  color: var(--color-text-muted);
}

.task-table {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  overflow-x: auto;
}

.table-header, .table-row {
  display: grid;
  grid-template-columns: 80px 80px 1fr 80px 160px;
  gap: 12px;
  align-items: center;
  padding: 10px 16px;
  font-size: 13px;
}

.table-header {
  background: var(--color-bg);
  font-weight: 600;
  border-bottom: 1px solid var(--color-border);
}

.table-row {
  border-bottom: 1px solid var(--color-border);
  cursor: pointer;
  transition: background 0.1s;
}
.table-row:last-child { border-bottom: none; }
.table-row:hover { background: rgba(99, 102, 241, 0.04); }
.col-id { font-family: monospace; }
.col-input {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.status-badge {
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
  white-space: nowrap;
}
.status-badge.pending { background: #fef3c7; color: #92400e; }
.status-badge.processing { background: #dbeafe; color: #1e40af; }
.status-badge.done { background: #d1fae5; color: #065f46; }
.status-badge.failed { background: #fee2e2; color: #991b1b; }

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
</style>
