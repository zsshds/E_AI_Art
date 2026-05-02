<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getSettings, updateSettings, type AppSettings } from '../api/setting'

const settings = ref<AppSettings>({})
const loading = ref(false)
const saving = ref(false)
const message = ref('')

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
