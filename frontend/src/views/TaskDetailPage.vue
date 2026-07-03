<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getTask, getTaskConversation, sendChatMessage, subscribeTask, type Task } from '../api/task'

const route = useRoute()
const router = useRouter()

const task = ref<Task | null>(null)
const conversation = ref<Task[]>([])
const currentMessage = ref('')
const sending = ref(false)
const loading = ref(true)
const error = ref('')
const downloading = ref(false)
const chatContainer = ref<HTMLElement | null>(null)
const activeChildIds = ref<Set<string>>(new Set())
const unsubscribeMap = ref<Map<string, () => void>>(new Map())

const statusLabels: Record<string, string> = {
  pending: '排队中',
  processing: '生成中',
  done: '已完成',
  failed: '失败',
}

const canChat = computed(() => {
  return task.value?.status === 'done' && activeChildIds.value.size === 0
})

const latestImage = computed(() => {
  for (let i = conversation.value.length - 1; i >= 0; i--) {
    if (conversation.value[i].status === 'done' && conversation.value[i].result_image_url) {
      return conversation.value[i].result_image_url
    }
  }
  return ''
})

const latestDoneTaskId = computed(() => {
  for (let i = conversation.value.length - 1; i >= 0; i--) {
    if (conversation.value[i].status === 'done') {
      return conversation.value[i].id
    }
  }
  return task.value?.id
})

onMounted(async () => {
  const id = route.params.id as string
  try {
    task.value = await getTask(id)
    await loadConversation(id)
  } catch (e: any) {
    error.value = e.message || '加载任务失败'
  } finally {
    loading.value = false
  }
})

onUnmounted(() => {
  unsubscribeMap.value.forEach(fn => fn())
})

async function loadConversation(id: string) {
  try {
    const chain = await getTaskConversation(id)
    conversation.value = chain

    // Subscribe to any active (pending/processing) tasks in the chain
    for (const t of chain) {
      if (t.status === 'pending' || t.status === 'processing') {
        watchTask(t.id!)
      }
    }
  } catch {
    // If conversation endpoint fails, just show the root task alone
    conversation.value = task.value ? [task.value] : []
  }
  await nextTick()
  scrollToBottom()
}

function watchTask(childId: string) {
  if (activeChildIds.value.has(childId)) return
  activeChildIds.value.add(childId)

  const unsub = subscribeTask(childId, (update) => {
    const idx = conversation.value.findIndex(t => t.id === childId)
    if (idx !== -1) {
      const existing = conversation.value[idx]
      if (update.status) existing.status = update.status
      if (update.result_image_url) existing.result_image_url = update.result_image_url
      if (update.error_message) existing.error_message = update.error_message
      if (update.progress !== undefined) existing.progress = update.progress

      if (update.status === 'done' || update.status === 'failed') {
        activeChildIds.value.delete(childId)
        unsubscribeMap.value.get(childId)?.()
        unsubscribeMap.value.delete(childId)
      }
    }
    scrollToBottom()
  })

  unsubscribeMap.value.set(childId, unsub)

  // Fallback polling
  pollTask(childId)
}

async function pollTask(childId: string, maxAttempts = 60) {
  for (let i = 0; i < maxAttempts; i++) {
    if (!activeChildIds.value.has(childId)) return
    await new Promise(resolve => setTimeout(resolve, 2000))
    try {
      const t = await getTask(childId)
      const idx = conversation.value.findIndex(c => c.id === childId)
      if (idx !== -1) {
        conversation.value[idx] = t
      }
      if (t.status === 'done' || t.status === 'failed') {
        activeChildIds.value.delete(childId)
        unsubscribeMap.value.get(childId)?.()
        unsubscribeMap.value.delete(childId)
        scrollToBottom()
        return
      }
    } catch {
      // continue polling
    }
  }
}

async function handleSend() {
  const msg = currentMessage.value.trim()
  if (!msg || sending.value || !latestDoneTaskId.value) return

  currentMessage.value = ''
  sending.value = true

  try {
    const child = await sendChatMessage(latestDoneTaskId.value, msg)
    conversation.value.push(child)
    await nextTick()
    scrollToBottom()
    watchTask(child.id!)
  } catch (e: any) {
    alert(e.message || '发送失败')
  } finally {
    sending.value = false
  }
}

async function handleDownload(taskId?: string) {
  const id = taskId || latestDoneTaskId.value
  if (!id) return
  downloading.value = true
  try {
    const token = localStorage.getItem('token')
    const resp = await fetch(`/api/v1/tasks/${id}/download`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!resp.ok) throw new Error('下载失败')
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `E_AI_Art-${id.slice(-8)}.png`
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

function openImage(url: string) {
  const a = document.createElement('a')
  a.href = url
  a.target = '_blank'
  a.click()
}

function scrollToBottom() {
  nextTick(() => {
    if (chatContainer.value) {
      chatContainer.value.scrollTop = chatContainer.value.scrollHeight
    }
  })
}
</script>

<template>
  <div class="task-detail-page">
    <button class="btn-back" @click="router.push('/tasks')">&larr; 返回任务列表</button>

    <div v-if="loading" class="state-box">加载中...</div>
    <div v-else-if="error" class="state-box error">{{ error }}</div>
    <div v-else-if="!task" class="state-box">任务不存在</div>

    <div v-else class="page-layout">
      <!-- Left sidebar: task info -->
      <aside class="detail-sidebar">
        <h2>任务详情</h2>
        <div class="detail-grid">
          <div class="detail-item">
            <label>任务ID</label>
            <code>{{ task.id }}</code>
          </div>
          <div class="detail-item">
            <label>创建者</label>
            <span>{{ task.created_by }}</span>
          </div>
          <div class="detail-item">
            <label>模型</label>
            <span>{{ task.model || '-' }}</span>
          </div>
          <div class="detail-item">
            <label>尺寸</label>
            <span>{{ task.size || '-' }}</span>
          </div>
          <div class="detail-item">
            <label>状态</label>
            <span class="status-badge" :class="task.status">{{ statusLabels[task.status] }}</span>
          </div>
          <div class="detail-item">
            <label>创建时间</label>
            <span>{{ new Date(task.created_at).toLocaleString() }}</span>
          </div>
          <div class="detail-item full-width">
            <label>原始提示词</label>
            <pre>{{ task.user_input }}</pre>
          </div>
          <div v-if="task.final_prompt" class="detail-item full-width">
            <label>最终 Prompt</label>
            <pre>{{ task.final_prompt }}</pre>
          </div>
        </div>

        <div v-if="task.status === 'failed'" class="detail-error">
          <h4>失败原因</h4>
          <p>{{ task.error_message || '未知错误' }}</p>
        </div>

        <div v-if="latestImage" class="current-image">
          <h3>当前结果</h3>
          <img :src="latestImage" alt="当前结果" />
          <button class="btn btn-primary" @click="handleDownload()" :disabled="downloading">
            {{ downloading ? '下载中...' : '下载图片' }}
          </button>
        </div>
      </aside>

      <!-- Right main: chat thread -->
      <section class="chat-section">
        <h2>迭代编辑</h2>
        <div class="chat-messages" ref="chatContainer">
          <div v-if="task.status === 'pending'" class="chat-status">任务排队中...</div>
          <div v-else-if="task.status === 'processing'" class="chat-status">正在生成初始图片...</div>
          <div v-else-if="task.status === 'failed'" class="chat-status error">任务失败</div>

          <template v-else>
            <div v-if="conversation.length === 0" class="chat-status">
              暂无对话记录。在下方输入框描述你想如何修改当前图片。
            </div>

            <div v-for="turn in conversation" :key="turn.id" class="chat-turn">
              <div class="chat-bubble user">
                <div class="bubble-label">你</div>
                <div class="bubble-content">{{ turn.user_input }}</div>
              </div>
              <div class="chat-bubble ai">
                <div class="bubble-label">AI</div>
                <div v-if="turn.status === 'pending'" class="bubble-content status-pending">排队中...</div>
                <div v-else-if="turn.status === 'processing'" class="bubble-content status-processing">
                  <span class="spinner"></span> 生成中{{ turn.progress ? ` ${turn.progress}%` : '' }}...
                </div>
                <div v-else-if="turn.status === 'failed'" class="bubble-content status-failed">
                  失败: {{ turn.error_message || '未知错误' }}
                </div>
                <div v-else-if="turn.status === 'done' && turn.result_image_url" class="bubble-content">
                  <img :src="turn.result_image_url" alt="编辑结果" @click="openImage(turn.result_image_url)" />
                  <button class="btn btn-small" @click="handleDownload(turn.id!)">下载</button>
                </div>
                <div v-else class="bubble-content status-pending">等待结果...</div>
              </div>
            </div>
          </template>
        </div>

        <div v-if="canChat" class="chat-input-area">
          <textarea
            v-model="currentMessage"
            placeholder="描述你想如何修改当前图片，例如：把天空调暗、添加一只猫..."
            :disabled="sending"
            @keydown.enter.exact.prevent="handleSend"
            rows="3"
          />
          <button
            class="btn btn-primary"
            @click="handleSend"
            :disabled="sending || !currentMessage.trim()"
          >
            {{ sending ? '发送中...' : '发送' }}
          </button>
        </div>
        <div v-else-if="activeChildIds.size > 0" class="chat-input-hint">
          正在处理上一条编辑请求...
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.task-detail-page {
  max-width: 1200px;
  margin: 0 auto;
  padding: 20px;
}

.btn-back {
  border: none;
  background: none;
  color: var(--color-primary);
  cursor: pointer;
  font-size: 14px;
  padding: 0;
  margin-bottom: 16px;
}
.btn-back:hover { text-decoration: underline; }

.state-box {
  text-align: center;
  padding: 48px;
  color: var(--color-text-muted);
}
.state-box.error { color: #991b1b; }

.page-layout {
  display: grid;
  grid-template-columns: 340px 1fr;
  gap: 24px;
  align-items: start;
}

/* Sidebar */
.detail-sidebar {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 20px;
  position: sticky;
  top: 20px;
}
.detail-sidebar h2 { font-size: 18px; margin-bottom: 16px; }

.detail-grid {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.detail-item label {
  display: block;
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 2px;
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
  margin: 4px 0 0;
  max-height: 120px;
  overflow-y: auto;
}

.status-badge {
  padding: 2px 8px;
  border-radius: 10px;
  font-size: 12px;
}
.status-badge.pending { background: #fef3c7; color: #92400e; }
.status-badge.processing { background: #dbeafe; color: #1e40af; }
.status-badge.done { background: #d1fae5; color: #065f46; }
.status-badge.failed { background: #fee2e2; color: #991b1b; }

.detail-error {
  margin-top: 12px;
  padding: 12px;
  background: #fee2e2;
  border-radius: var(--radius);
}
.detail-error h4 { font-size: 14px; color: #991b1b; margin-bottom: 4px; }
.detail-error p { font-size: 13px; color: #7f1d1d; margin: 0; }

.current-image {
  margin-top: 16px;
}
.current-image h3 { font-size: 14px; margin-bottom: 8px; }
.current-image img {
  width: 100%;
  max-height: 300px;
  object-fit: contain;
  border-radius: var(--radius);
  border: 1px solid var(--color-border);
  background: var(--color-bg);
}

/* Chat section */
.chat-section {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  display: flex;
  flex-direction: column;
  min-height: calc(100vh - 120px);
}
.chat-section h2 {
  font-size: 18px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-border);
  margin: 0;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
  gap: 16px;
  max-height: calc(100vh - 320px);
}

.chat-status {
  text-align: center;
  padding: 32px;
  color: var(--color-text-muted);
  font-size: 14px;
}
.chat-status.error { color: #991b1b; }

.chat-turn {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.chat-bubble {
  max-width: 85%;
}
.chat-bubble.user {
  align-self: flex-end;
}
.chat-bubble.ai {
  align-self: flex-start;
}

.bubble-label {
  font-size: 11px;
  color: var(--color-text-muted);
  margin-bottom: 2px;
  padding: 0 4px;
}

.bubble-content {
  padding: 10px 14px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.5;
}
.chat-bubble.user .bubble-content {
  background: var(--color-primary);
  color: white;
  border-bottom-right-radius: 4px;
}
.chat-bubble.ai .bubble-content {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-bottom-left-radius: 4px;
}
.chat-bubble.ai .bubble-content img {
  max-width: 100%;
  max-height: 400px;
  display: block;
  border-radius: 6px;
  cursor: pointer;
}
.chat-bubble.ai .bubble-content.status-pending { color: var(--color-text-muted); }
.chat-bubble.ai .bubble-content.status-failed { background: #fee2e2; color: #991b1b; }

.status-processing {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--color-primary);
}
.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  display: inline-block;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Chat input */
.chat-input-area {
  padding: 12px 20px;
  border-top: 1px solid var(--color-border);
  display: flex;
  gap: 8px;
  align-items: flex-end;
}
.chat-input-area textarea {
  flex: 1;
  padding: 10px 14px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  resize: vertical;
  min-height: 44px;
  font-family: inherit;
  background: var(--color-bg);
  color: var(--color-text);
}
.chat-input-area textarea:focus {
  outline: none;
  border-color: var(--color-primary);
}
.chat-input-area textarea:disabled {
  opacity: 0.5;
}

.chat-input-hint {
  padding: 12px 20px;
  border-top: 1px solid var(--color-border);
  text-align: center;
  font-size: 13px;
  color: var(--color-text-muted);
}

/* Buttons */
.btn {
  padding: 8px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  cursor: pointer;
  background: var(--color-surface);
  white-space: nowrap;
}
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary {
  background: var(--color-primary);
  color: white;
  border-color: var(--color-primary);
}
.btn-primary:hover:not(:disabled) { background: var(--color-primary-hover); }
.btn-small {
  padding: 4px 10px;
  font-size: 12px;
  margin-top: 8px;
  display: inline-block;
}

@media (max-width: 768px) {
  .page-layout {
    grid-template-columns: 1fr;
  }
  .detail-sidebar {
    position: static;
  }
}
</style>
