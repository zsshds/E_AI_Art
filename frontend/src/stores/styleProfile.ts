import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { StyleProfile } from '../api/style'

export const useStyleProfileStore = defineStore('styleProfile', () => {
  const profiles = ref<StyleProfile[]>([])
  const currentProfile = ref<StyleProfile | null>(null)
  const loading = ref(false)

  const lockedProfiles = computed(() => profiles.value.filter(p => p.is_locked))

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
    push(profile.lighting)
    push(profile.quality_tags)
    push(profile.composition)
    parts.push(...(profile.extra_tokens || []))

    return parts.join(', ')
  }

  return {
    profiles,
    currentProfile,
    loading,
    lockedProfiles,
    setProfiles,
    setCurrent,
    buildPromptPreview,
  }
})
