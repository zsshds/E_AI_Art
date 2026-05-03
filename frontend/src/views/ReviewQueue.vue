<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { listTasks, type Task } from '../api/task'

const tasks = ref<Task[]>([])
const filter = ref<'all' | 'pending' | 'processing' | 'done' | 'failed'>('all')
const loading = ref(false)
const expandedId = ref<string | null>(null)
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

function toggleExpand(taskId: string) {
  expandedId.value = expandedId.value === taskId ? null : taskId
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
    a.download = `imagegen-${taskId.slice(-8)}.png`
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

const expandedTask = computed(() => {
  return tasks.value.find(t => t.id === expandedId.value) || null
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
      <h2>审核队列</h2>
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
        :class="{ expanded: expandedId === task.id }"
        @click="toggleExpand(task.id!)"
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

    <!-- Detail panel -->
    <div v-if="expandedTask" class="detail-panel">
      <div class="detail-header">
        <h3>任务详情 — {{ expandedTask.id?.slice(-8) }}</h3>
        <button class="btn-close" @click="expandedId = null">&times;</button>
      </div>

      <div class="detail-grid">
        <div class="detail-item">
          <label>任务ID</label>
          <code>{{ expandedTask.id }}</code>
        </div>
        <div class="detail-item">
          <label>创建者</label>
          <span>{{ expandedTask.created_by }}</span>
        </div>
        <div class="detail-item">
          <label>模型</label>
          <span>{{ expandedTask.model || '-' }}</span>
        </div>
        <div class="detail-item">
          <label>状态</label>
          <span class="status-badge" :class="expandedTask.status">{{ statusLabels[expandedTask.status] }}</span>
        </div>
        <div class="detail-item">
          <label>创建时间</label>
          <span>{{ new Date(expandedTask.created_at).toLocaleString() }}</span>
        </div>
        <div class="detail-item">
          <label>重试次数</label>
          <span>{{ expandedTask.retry_count }}</span>
        </div>
        <div class="detail-item full-width">
          <label>用户输入</label>
          <pre>{{ expandedTask.user_input }}</pre>
        </div>
        <div class="detail-item full-width">
          <label>最终 Prompt</label>
          <pre>{{ expandedTask.final_prompt }}</pre>
        </div>
      </div>

      <!-- Failed task: error message -->
      <div v-if="expandedTask.status === 'failed'" class="detail-error">
        <h4>失败原因</h4>
        <p>{{ expandedTask.error_message || '未知错误' }}</p>
      </div>

      <!-- Done task: image + download -->
      <div v-if="expandedTask.status === 'done' && expandedTask.result_image_url" class="detail-image">
        <h4>生成结果</h4>
        <img :src="expandedTask.result_image_url" alt="生成结果" />
        <button class="btn btn-primary" @click.stop="handleDownload(expandedTask.id!)" :disabled="downloading">
          {{ downloading ? '下载中...' : '下载图片' }}
        </button>
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
.table-row.expanded { background: rgba(99, 102, 241, 0.06); }

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

/* Detail panel */
.detail-panel {
  margin-top: 16px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 20px;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.detail-header h3 { font-size: 16px; }

.btn-close {
  border: none;
  background: none;
  font-size: 24px;
  cursor: pointer;
  color: var(--color-text-muted);
  line-height: 1;
}

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.detail-item {
  min-width: 0;
}
.detail-item.full-width { grid-column: 1 / -1; }
.detail-item label {
  display: block;
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 4px;
}
.detail-item code {
  font-size: 12px;
  word-break: break-all;
}
.detail-item pre {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 13px;
  line-height: 1.5;
  background: var(--color-bg);
  padding: 8px;
  border-radius: 4px;
  margin: 0;
}

.detail-error {
  margin-top: 16px;
  padding: 12px;
  background: #fee2e2;
  border-radius: var(--radius);
}
.detail-error h4 { font-size: 14px; color: #991b1b; margin-bottom: 4px; }
.detail-error p { font-size: 13px; color: #7f1d1d; margin: 0; white-space: pre-wrap; word-break: break-word; }

.detail-image {
  margin-top: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}
.detail-image h4 { font-size: 14px; }
.detail-image img {
  max-width: 100%;
  max-height: 600px;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
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
</style>
