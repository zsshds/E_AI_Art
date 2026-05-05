import { ref } from 'vue'
import { getAvailableModels, type AvailableModel } from '../api/setting'

const models = ref<AvailableModel[]>([])
const loading = ref(false)
const loaded = ref(false)
const error = ref('')

export function useAvailableModels() {
  if (!loaded.value && !loading.value) {
    loading.value = true
    getAvailableModels()
      .then((data) => {
        models.value = data
        loaded.value = true
      })
      .catch((e) => {
        error.value = e.message || '获取模型列表失败'
      })
      .finally(() => {
        loading.value = false
      })
  }

  return { availableModels: models, loading, error }
}
