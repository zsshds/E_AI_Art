<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { getSettings, updateSettings, fetchModels, type AppSettings, type ModelInfo } from '../api/setting'

const settings = ref<AppSettings>({})
const loading = ref(false)
const saving = ref(false)
const message = ref('')

// Model management state
const allModels = ref<ModelInfo[]>([])
const fetchingModels = ref(false)
const modelMsg = ref('')
const modelSearch = ref('')

// Fuzzy search: match each character/word in query against model id (case-insensitive)
const filteredModels = computed(() => {
  const q = modelSearch.value.trim().toLowerCase()
  if (!q) return allModels.value
  // Split query into individual characters for fuzzy matching
  const chars = q.split('')
  return allModels.value.filter(m => {
    const id = m.id.toLowerCase()
    return chars.every(c => id.includes(c))
  })
})

onMounted(async () => {
  loading.value = true
  try {
    settings.value = await getSettings()
  } catch {
    // ignore
  } finally {
    loading.value = false
  }
})

// Parse available_models from settings (JSON array string)
const selectedModels = computed({
  get: (): string[] => {
    if (!settings.value.available_models) return []
    try {
      return JSON.parse(settings.value.available_models) as string[]
    } catch {
      return []
    }
  },
  set: (val: string[]) => {
    settings.value.available_models = JSON.stringify(val)
  },
})

function isModelSelected(modelId: string): boolean {
  return selectedModels.value.includes(modelId)
}

function toggleModel(modelId: string) {
  const current = [...selectedModels.value]
  const idx = current.indexOf(modelId)
  if (idx >= 0) {
    current.splice(idx, 1)
  } else {
    current.push(modelId)
  }
  selectedModels.value = current
}

function selectAllModels() {
  const targets = modelSearch.value.trim() ? filteredModels.value : allModels.value
  const targetIds = new Set(targets.map(m => m.id))
  const current = selectedModels.value.filter(id => !targetIds.has(id))
  selectedModels.value = [...current, ...targets.map(m => m.id)]
}

function deselectAllModels() {
  const targets = modelSearch.value.trim() ? filteredModels.value : allModels.value
  const targetIds = new Set(targets.map(m => m.id))
  selectedModels.value = selectedModels.value.filter(id => !targetIds.has(id))
}

async function handleFetchModels() {
  fetchingModels.value = true
  modelMsg.value = ''
  try {
    const result = await fetchModels(settings.value.model_filter_type)
    allModels.value = result.data || []
    if (allModels.value.length === 0) {
      modelMsg.value = '未获取到模型列表'
    } else {
      modelMsg.value = `成功获取 ${allModels.value.length} 个模型`
    }
  } catch (e: any) {
    modelMsg.value = '获取模型列表失败: ' + (e.message || '未知错误')
  } finally {
    fetchingModels.value = false
  }
}

async function handleSave() {
  saving.value = true
  message.value = ''
  try {
    const result = await updateSettings(settings.value)
    settings.value = result
    message.value = '保存成功，配置已实时生效'
  } catch (e: any) {
    message.value = '保存失败: ' + (e.message || '未知错误')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="settings-page">
    <h2>系统设置</h2>
    <p class="subtitle">配置 AI 平台连接参数，保存后即时生效</p>

    <div v-if="loading" class="loading">加载中...</div>

    <div v-else class="settings-form">
      <div class="form-group">
        <label>API Base URL</label>
        <input v-model="settings.api_base_url" type="text" placeholder="https://api.openai.com" class="text-input" />
        <small>第三方 AI 平台的 API 端点地址</small>
      </div>

      <div class="form-group">
        <label>API Key</label>
        <input v-model="settings.api_key" type="password" placeholder="sk-..." class="text-input" />
        <small>第三方平台的 API 密钥</small>
      </div>

      <div class="form-group">
        <label>生图端点 Path</label>
        <input v-model="settings.api_generation_path" type="text" placeholder="/v1/images/generations/tasks" class="text-input" />
        <small>相对路径模式下，提交文生图任务的端点路径</small>
      </div>

      <div class="form-group">
        <label>生图端点完整 URL（优先）</label>
        <input v-model="settings.api_generation_url" type="text" placeholder="https://your-platform.example.com/v1/images/generations" class="text-input" />
        <small>如第三方平台接口不遵循统一 Base URL + Path，可直接填完整地址；填写后会优先生效</small>
      </div>

      <div class="form-group">
        <label>轮询端点 Path</label>
        <input v-model="settings.api_poll_path" type="text" placeholder="/v1/images/tasks/" class="text-input" />
        <small>相对路径模式下，查询任务结果的端点路径；末尾的 / 可不填</small>
      </div>

      <div class="form-group">
        <label>轮询端点完整 URL（优先）</label>
        <input v-model="settings.api_poll_url" type="text" placeholder="https://your-platform.example.com/v1/images/tasks" class="text-input" />
        <small>填写后会直接用该地址作为任务查询前缀，系统会自动拼接 task_id</small>
      </div>

      <div class="form-group">
        <label>Chat 端点 Path</label>
        <input v-model="settings.api_chat_path" type="text" placeholder="/v1/chat/completions" class="text-input" />
        <small>带参考图/对话式生成时使用的相对路径</small>
      </div>

      <div class="form-group">
        <label>Chat 端点完整 URL（优先）</label>
        <input v-model="settings.api_chat_url" type="text" placeholder="https://your-platform.example.com/v1/chat/completions" class="text-input" />
        <small>如图生图或带图提示走独立网关，可在此配置完整地址</small>
      </div>

      <div class="form-group">
        <label>Banana 模型完整 URL（优先）</label>
        <input v-model="settings.api_banana_generation_url" type="text" placeholder="https://your-platform.example.com/v1/images/generations" class="text-input" />
        <small>仅 Banana 类模型使用；留空则继续跟随生图端点/默认逻辑</small>
      </div>

      <!-- Model management section -->
      <div class="model-section">
        <div class="model-header">
          <h3>可用模型配置</h3>
          <button class="btn btn-secondary" @click="handleFetchModels" :disabled="fetchingModels">
            {{ fetchingModels ? '获取中...' : '获取模型列表' }}
          </button>
        </div>
        <small>配置模型获取地址与返回数据筛选方式，保存后点击"获取模型列表"生效</small>

        <div class="form-row">
          <div class="form-group form-group-inline">
            <label>模型获取地址</label>
            <input v-model="settings.model_fetch_url" type="text" placeholder="/v1/models" class="text-input" />
            <small>相对路径以 API Base URL 为基础，也可填写完整 URL</small>
          </div>
          <div class="form-group form-group-inline">
            <label>数据筛选方式</label>
            <select v-model="settings.model_filter_type" class="text-input">
              <option value="none">全部模型</option>
              <option value="image">仅生图模型</option>
            </select>
            <small>获取后自动按此规则筛选模型列表</small>
          </div>
        </div>

        <div v-if="modelMsg" class="model-message" :class="{ error: modelMsg.includes('失败') }">{{ modelMsg }}</div>

        <div v-if="allModels.length > 0" class="model-list">
          <div class="model-actions">
            <input
              v-model="modelSearch"
              type="text"
              class="model-search-input"
              placeholder="搜索模型..."
            />
            <button class="btn-text" @click="selectAllModels">全选</button>
            <button class="btn-text" @click="deselectAllModels">取消全选</button>
            <span class="model-count">已选 {{ selectedModels.length }} / {{ allModels.length }} 个模型</span>
          </div>
          <div v-if="filteredModels.length === 0" class="model-empty">
            无匹配模型
          </div>
          <label
            v-for="model in filteredModels"
            :key="model.id"
            class="model-item"
            :class="{ selected: isModelSelected(model.id) }"
          >
            <input
              type="checkbox"
              :checked="isModelSelected(model.id)"
              @change="toggleModel(model.id)"
            />
            <span class="model-id">{{ model.id }}</span>
            <span v-if="model.owned_by" class="model-owner">{{ model.owned_by }}</span>
          </label>
        </div>
        <div v-else-if="!fetchingModels && !modelMsg" class="model-empty">
          点击"获取模型列表"从 AI 平台同步可用模型
        </div>
      </div>

      <div v-if="message" class="message" :class="{ error: message.includes('失败') }">{{ message }}</div>

      <button class="btn btn-primary" @click="handleSave" :disabled="saving">
        {{ saving ? '保存中...' : '保存配置' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.settings-page {
  max-width: 600px;
  margin: 0 auto;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  padding: 24px;
}

.settings-page h2 {
  font-size: 20px;
  margin-bottom: 4px;
}

.subtitle {
  font-size: 14px;
  color: var(--color-text-muted);
  margin-bottom: 24px;
}

.form-group {
  margin-bottom: 20px;
}

.form-group label {
  display: block;
  font-size: 14px;
  font-weight: 500;
  margin-bottom: 6px;
}

.form-group small {
  display: block;
  font-size: 12px;
  color: var(--color-text-muted);
  margin-top: 4px;
}

.text-input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  background: var(--color-surface);
}

.message {
  margin-bottom: 16px;
  padding: 8px 12px;
  background: #d1fae5;
  color: #065f46;
  border-radius: var(--radius);
  font-size: 13px;
}
.message.error {
  background: #fee2e2;
  color: #991b1b;
}

/* Model management */
.model-section {
  margin-top: 28px;
  padding-top: 24px;
  border-top: 1px solid var(--color-border);
}

.model-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 4px;
}

.form-row {
  display: flex;
  gap: 16px;
  margin-bottom: 12px;
}

.form-group-inline {
  flex: 1;
  min-width: 0;
}

.model-header h3 {
  font-size: 16px;
  font-weight: 500;
  margin: 0;
}

.model-section > small {
  display: block;
  font-size: 12px;
  color: var(--color-text-muted);
  margin-bottom: 12px;
}

.model-message {
  padding: 6px 10px;
  background: #d1fae5;
  color: #065f46;
  border-radius: var(--radius);
  font-size: 13px;
  margin-bottom: 12px;
}
.model-message.error {
  background: #fee2e2;
  color: #991b1b;
}

.model-list {
  max-height: 320px;
  overflow-y: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  background: var(--color-surface);
}

.model-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-bg);
  position: sticky;
  top: 0;
}

.model-search-input {
  flex: 1;
  padding: 4px 8px;
  border: 1px solid var(--color-border);
  border-radius: 4px;
  font-size: 13px;
  background: var(--color-surface);
  outline: none;
  min-width: 0;
}
.model-search-input:focus {
  border-color: var(--color-primary);
}

.btn-text {
  font-size: 13px;
  color: var(--color-primary);
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
}
.btn-text:hover {
  text-decoration: underline;
}

.model-count {
  font-size: 12px;
  color: var(--color-text-muted);
  margin-left: auto;
}

.model-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  border-bottom: 1px solid var(--color-border);
  font-size: 14px;
}
.model-item:last-child {
  border-bottom: none;
}
.model-item:hover {
  background: var(--color-bg);
}
.model-item.selected {
  background: #ecfdf5;
}

.model-id {
  flex: 1;
  font-family: monospace;
  font-size: 13px;
}

.model-owner {
  font-size: 12px;
  color: var(--color-text-muted);
}

.model-empty {
  padding: 24px;
  text-align: center;
  font-size: 13px;
  color: var(--color-text-muted);
}

.btn {
  padding: 8px 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
  font-size: 14px;
  cursor: pointer;
}
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-primary { background: var(--color-primary); color: white; border-color: var(--color-primary); }
.btn-primary:hover:not(:disabled) { background: var(--color-primary-hover); }
</style>
