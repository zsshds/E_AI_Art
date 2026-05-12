import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { StyleProfile } from '../api/style'

export const useStyleProfileStore = defineStore('styleProfile', () => {
  const profiles = ref<StyleProfile[]>([])
  const currentProfile = ref<StyleProfile | null>(null)
  const loading = ref(false)

  function setProfiles(list: StyleProfile[]) {
    profiles.value = list
  }

  function setCurrent(profile: StyleProfile | null) {
    currentProfile.value = profile
  }

  function buildPromptPreview(profile: StyleProfile): string {
    const parts: string[] = []
    const push = (s: string) => { if (s?.trim()) parts.push(s.trim()) }

    push(profile.art_style)
    push(profile.color_tone)
    parts.push(...(profile.extra_tokens || []))

    return parts.join(', ')
  }

  return {
    profiles,
    currentProfile,
    loading,
    setProfiles,
    setCurrent,
    buildPromptPreview,
  }
})