<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { listStyleProfiles, type StyleProfile } from '../api/style'
import { createTask, getTask, subscribeTask, type Task } from '../api/task'
import { useAuthStore } from '../stores/auth'
import { useAvailableModels } from '../composables/useAvailableModels'
import PromptAssistant from '../components/PromptAssistant.vue'

const { availableModels, loading: modelsLoading } = useAvailableModels()

const sizeOptions = [
  { value: 'auto', label: 'auto (默认)' },
  { value: '1024x1024', label: '1024×1024 (1:1 正方形)' },
  { value: '1536x1024', label: '1536×1024 (3:2 横向)' },
  { value: '1024x1536', label: '1024×1536 (2:3 竖屏)' },
  { value: '2048x2048', label: '2048×2048 (1:1 2K)' },
  { value: '2048x1152', label: '2048×1152 (16:9 2K)' },
  { value: '3840x2160', label: '3840×2160 (16:9 4K)' },
  { value: '2160x3840', label: '2160×3840 (9:16 4K)' },
]

const qualityOptions = [
  { value: 'low', label: '低质量 (快速)' },
  { value: 'medium', label: '中等质量 (推荐)' },
  { value: 'high', label: '高质量 (较慢)' },
]

const profiles = ref<StyleProfile[]>([])
const selectedProfileId = ref('')
const selectedModel = ref('')
const selectedSize = ref('auto')
const selectedQuality = ref('medium')
const userInput = ref('')
const generating = ref(false)
const currentTask = ref<Task | null>(null)
const resultImage = ref('')
const error = ref('')
let unsubscribe: (() => void) | null = null

onMounted(async () => {
  const auth = useAuthStore()
  try {
    profiles.value = await listStyleProfiles()
    // Admin sees all profiles, user only sees locked ones
    if (!auth.isAdmin) {
      profiles.value = profiles.value.filter(p => p.is_locked)
    }
  } catch {
    // Handle silently
  }
})

onUnmounted(() => {
  unsubscribe?.()
})

const downloading = ref(false)

async function handleDownload() {
  if (!currentTask.value?.id) return
  downloading.value = true
  try {
    const token = localStorage.getItem('token')
    const resp = await fetch(`/api/v1/tasks/${currentTask.value.id}/download`, {
      headers: { Authorization: `Bearer ${token}` },
    })
    if (!resp.ok) throw new Error('下载失败')
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `E_AI_Art-${currentTask.value.id.slice(-8)}.png`
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

function handleInsertToken(token: string) {
  const current = userInput.value.trim()
  userInput.value = current ? `${current}, ${token}` : token
}

async function handleGenerate() {
  if (!selectedProfileId.value || !userInput.value.trim()) return

  generating.value = true
  error.value = ''
  resultImage.value = ''
  currentTask.value = null

  try {
    const task = await createTask(selectedProfileId.value, userInput.value.trim(), selectedModel.value || undefined, selectedSize.value, selectedQuality.value)
    currentTask.value = task

    // Subscribe to WebSocket updates
    unsubscribe = subscribeTask(task.id!, (update) => {
      if (update.status === 'done') {
        resultImage.value = update.result_image_url
        generating.value = false
      } else if (update.status === 'failed') {
        error.value = update.error_message || '生图失败'
        generating.value = false
      }
      if (currentTask.value) {
        currentTask.value.status = update.status
      }
    })

    // Fallback polling
    pollTask(task.id!)
  } catch (e: any) {
    error.value = e.message
    generating.value = false
  }
}

async function pollTask(taskId: string) {
  const maxPolls = 60
  for (let i = 0; i < maxPolls; i++) {
    await new Promise(r => setTimeout(r, 2000))
    if (!generating.value) return
    try {
      const t = await getTask(taskId)
      currentTask.value = t
      if (t.status === 'done') {
        resultImage.value = t.result_image_url
        generating.value = false
        return
      }
      if (t.status === 'failed') {
        error.value = t.error_message || '生图失败'
        generating.value = false
        return
      }
    } catch {
      // Retry next poll
    }
  }
  generating.value = false
  error.value = '生图超时，请稍后重试'
}
</script>

<template>
  <div class="generate-page">
    <div class="generate-form">
      <h2>E_AI_Art 生图</h2>
      <p class="subtitle">选择锁定风格，输入自然语言描述即可生图</p>

      <div class="form-group">
        <label>风格选择</label>
        <select v-model="selectedProfileId" class="select-input" :disabled="generating"
          @change="(e) => {
            const p = profiles.find(p => p.id === (e.target as HTMLSelectElement).value)
            if (p && p.model) selectedModel = p.model
          }">
          <option value="">-- 选择风格 --</option>
          <option v-for="p in profiles" :key="p.id" :value="p.id">{{ p.name }} ({{ p.model }})</option>
        </select>
      </div>

      <div class="form-group">
        <label>AI 模型</label>
        <select v-model="selectedModel" class="select-input" :disabled="generating || modelsLoading">
          <option value="">
            {{ modelsLoading ? '加载中...' : availableModels.length === 0 ? '无可用模型，请联系管理员配置' : '-- 选择模型 --' }}
          </option>
          <option v-for="m in availableModels" :key="m.id" :value="m.id">{{ m.label }}</option>
        </select>
      </div>

      <div class="form-group">
        <label>分辨率</label>
        <select v-model="selectedSize" class="select-input" :disabled="generating">
          <option v-for="s in sizeOptions" :key="s.value" :value="s.value">{{ s.label }}</option>
        </select>
      </div>

      <div class="form-group">
        <label>生图质量</label>
        <select v-model="selectedQuality" class="select-input" :disabled="generating">
          <option v-for="q in qualityOptions" :key="q.value" :value="q.value">{{ q.label }}</option>
        </select>
      </div>

      <div class="form-group">
        <label>画面描述</label>
        <textarea
          v-model="userInput"
          placeholder="用自然语言描述你想生成的画面..."
          rows="4"
          class="text-input"
          :disabled="generating"
        ></textarea>
        <PromptAssistant @insert="handleInsertToken" />
      </div>

      <button
        class="btn btn-primary btn-large"
        @click="handleGenerate"
        :disabled="generating || !selectedProfileId || !userInput.trim()"
      >
        {{ generating ? '生成中...' : '开始生成' }}
      </button>

      <div v-if="currentTask" class="task-status">
        <span class="status-badge" :class="currentTask.status">
          {{ { pending: '排队中', processing: '生成中', done: '已完成', failed: '失败' }[currentTask.status] }}
        </span>
      </div>

      <div v-if="error" class="error-msg">{{ error }}</div>
    </div>

    <div class="generate-result">
      <div v-if="resultImage" class="result-image">
        <img :src="resultImage" alt="生成结果" />
        <div class="result-actions">
          <button class="btn btn-primary" @click="handleDownload" :disabled="downloading">
            {{ downloading ? '下载中...' : '下载图片' }}
          </button>
        </div>
      </div>
      <div v-else class="result-placeholder">
        <p>生成的图片将显示在这里</p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.generate-page {
  display: grid;
  grid-template-columns: 400px 1fr;
  gap: 24px;
  align-items: start;
}

.generate-form {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 24px;
}

.generate-form h2 {
  font-size: 20px;
  margin-bottom: 4px;
}

.subtitle {
  font-size: 14px;
  color: var(--color-text-muted);
  margin-bottom: 20px;
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

.select-input, .text-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  background: var(--color-surface);
}

textarea.text-input {
  resize: vertical;
  font-family: inherit;
}

.btn-large {
  width: 100%;
  padding: 12px;
  font-size: 16px;
}

.task-status {
  margin-top: 12px;
}

.status-badge {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 13px;
}
.status-badge.pending { background: #fef3c7; color: #92400e; }
.status-badge.processing { background: #dbeafe; color: #1e40af; }
.status-badge.done { background: #d1fae5; color: #065f46; }
.status-badge.failed { background: #fee2e2; color: #991b1b; }

.error-msg {
  margin-top: 12px;
  padding: 8px 12px;
  background: #fee2e2;
  color: #991b1b;
  border-radius: var(--radius);
  font-size: 13px;
}

.generate-result {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  min-height: 500px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.result-placeholder {
  color: var(--color-text-muted);
  font-size: 14px;
}

.result-image {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px;
}
.result-image img {
  max-width: 100%;
  border-radius: var(--radius);
  margin-bottom: 12px;
}
.result-actions {
  display: flex;
  gap: 8px;
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
</style>
