<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { listTasks, type Task } from '../api/task'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()
const tasks = ref<Task[]>([])
const filter = ref<'all' | 'pending' | 'processing' | 'done' | 'failed'>('all')
const creatorFilter = ref('all')
const page = ref(1)
const total = ref(0)
const totalPages = ref(0)
const jumpPage = ref('1')
const pageSize = 20
const loading = ref(false)
const downloading = ref(false)

onMounted(async () => {
  await refresh()
})

watch([filter, creatorFilter], async () => {
  page.value = 1
  jumpPage.value = '1'
  await refresh(1)
})

async function refresh(targetPage = page.value) {
  loading.value = true
  try {
    const result = await listTasks({
      createdBy: creatorFilter.value === 'all' ? undefined : creatorFilter.value,
      status: filter.value,
      page: targetPage,
      pageSize,
    })
    tasks.value = result.items
    total.value = result.total
    page.value = result.page
    totalPages.value = result.total_pages
    jumpPage.value = String(result.page)
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

const creatorOptions = computed(() => {
  const creators = [...new Set(tasks.value.map(task => task.created_by).filter(Boolean))]
  creators.sort((a, b) => {
    if (a === auth.username) return -1
    if (b === auth.username) return 1
    return a.localeCompare(b, 'zh-CN')
  })
  return creators
})

const visiblePageNumbers = computed(() => {
  if (totalPages.value <= 0) return []
  const start = Math.max(1, page.value - 2)
  const end = Math.min(totalPages.value, page.value + 2)
  const pages: number[] = []
  for (let i = start; i <= end; i += 1) {
    pages.push(i)
  }
  return pages
})

const statusLabels: Record<string, string> = {
  pending: '排队中',
  processing: '生成中',
  done: '已完成',
  failed: '失败',
}

async function goToPrevPage() {
  if (page.value <= 1 || loading.value) return
  await refresh(page.value - 1)
}

async function goToNextPage() {
  if (page.value >= totalPages.value || loading.value) return
  await refresh(page.value + 1)
}

async function goToPage(targetPage: number) {
  if (loading.value || targetPage === page.value || targetPage < 1 || targetPage > totalPages.value) return
  await refresh(targetPage)
}

async function handleJumpPage() {
  const targetPage = Number.parseInt(jumpPage.value, 10)
  if (!Number.isFinite(targetPage) || targetPage < 1 || targetPage > totalPages.value) {
    alert(`请输入 1 到 ${totalPages.value || 1} 之间的页码`)
    jumpPage.value = String(page.value)
    return
  }
  await goToPage(targetPage)
}
</script>

<template>
  <div class="review-queue">
    <div class="queue-header">
      <h2>任务队列</h2>
      <button class="btn btn-secondary" @click="refresh()" :disabled="loading">
        {{ loading ? '刷新中...' : '刷新' }}
      </button>
    </div>

    <div class="filter-panel">
      <div class="filter-group">
        <span class="filter-label">状态</span>
        <div class="filter-bar">
          <button
            v-for="(label, status) in { all: '全部', ...statusLabels }"
            :key="status"
            class="chip"
            :class="{ active: filter === status }"
            @click="filter = status as any"
          >{{ label }}</button>
        </div>
      </div>

      <div class="filter-group">
        <span class="filter-label">创建者</span>
        <div class="filter-bar">
          <button
            class="chip"
            :class="{ active: creatorFilter === 'all' }"
            @click="creatorFilter = 'all'"
          >全部</button>
          <button
            v-for="creator in creatorOptions"
            :key="creator"
            class="chip"
            :class="{ active: creatorFilter === creator }"
            @click="creatorFilter = creator"
          >{{ creator === auth.username ? `${creator}（我）` : creator }}</button>
        </div>
      </div>
    </div>

    <div class="pagination-summary">
      <span>共 {{ total }} 条</span>
      <span>每页 {{ pageSize }} 条</span>
      <span v-if="totalPages > 0">第 {{ page }} / {{ totalPages }} 页</span>
    </div>

    <div v-if="tasks.length === 0" class="empty-state">
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
        v-for="task in tasks"
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

    <div v-if="totalPages > 0" class="pagination-bar">
      <div class="page-actions">
        <button class="btn btn-secondary" @click="goToPrevPage" :disabled="loading || page <= 1">上一页</button>
        <button
          v-for="pageNo in visiblePageNumbers"
          :key="pageNo"
          class="chip"
          :class="{ active: page === pageNo }"
          @click="goToPage(pageNo)"
        >{{ pageNo }}</button>
        <button class="btn btn-secondary" @click="goToNextPage" :disabled="loading || page >= totalPages">下一页</button>
      </div>

      <div class="jump-box">
        <span>跳到</span>
        <input
          v-model="jumpPage"
          type="number"
          min="1"
          :max="Math.max(totalPages, 1)"
          class="page-input"
          @keyup.enter="handleJumpPage"
        />
        <span>页</span>
        <button class="btn btn-secondary" @click="handleJumpPage" :disabled="loading">确定</button>
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

.filter-panel {
  display: grid;
  gap: 12px;
  margin-bottom: 16px;
}

.filter-group {
  display: grid;
  gap: 8px;
}

.filter-label {
  font-size: 13px;
  color: var(--color-text-muted);
}

.filter-bar {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.pagination-summary {
  display: flex;
  gap: 16px;
  margin-bottom: 12px;
  font-size: 13px;
  color: var(--color-text-muted);
  flex-wrap: wrap;
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

.pagination-bar {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  align-items: center;
  justify-content: space-between;
  margin-top: 16px;
}

.page-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.jump-box {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
}

.page-input {
  width: 72px;
  padding: 8px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
  font-size: 13px;
}
</style>
