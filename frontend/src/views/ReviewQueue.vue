<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { listTasks, type Task } from '../api/task'

const tasks = ref<Task[]>([])
const filter = ref<'all' | 'pending' | 'processing' | 'done' | 'failed'>('all')
const loading = ref(false)

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
        <span class="col-input">用户输入</span>
        <span class="col-prompt">最终Prompt</span>
        <span class="col-status">状态</span>
        <span class="col-time">创建时间</span>
        <span class="col-result">结果</span>
      </div>
      <div v-for="task in filteredTasks" :key="task.id" class="table-row">
        <span class="col-id" :title="task.id">{{ task.id?.slice(-8) }}</span>
        <span class="col-input">{{ task.user_input }}</span>
        <span class="col-prompt" :title="task.final_prompt">{{ task.final_prompt?.slice(0, 60) }}{{ task.final_prompt?.length > 60 ? '...' : '' }}</span>
        <span class="col-status">
          <span class="status-badge" :class="task.status">{{ statusLabels[task.status] }}</span>
        </span>
        <span class="col-time">{{ new Date(task.created_at).toLocaleString() }}</span>
        <span class="col-result">
          <img v-if="task.result_image_url" :src="task.result_image_url" class="thumb" />
          <span v-else-if="task.status === 'failed'" class="error-text" :title="task.error_message">失败</span>
          <span v-else>-</span>
        </span>
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
  grid-template-columns: 80px 1fr 1.5fr 80px 140px 80px;
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
}
.table-row:last-child { border-bottom: none; }
.table-row:hover { background: rgba(99, 102, 241, 0.02); }

.col-id { font-family: monospace; }
.col-input, .col-prompt {
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

.thumb { width: 40px; height: 40px; object-fit: cover; border-radius: 4px; }
.error-text { color: #991b1b; font-size: 12px; }

.btn {
  padding: 8px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  cursor: pointer;
  background: var(--color-surface);
}
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
