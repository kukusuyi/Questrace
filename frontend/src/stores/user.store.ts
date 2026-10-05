import { defineStore } from 'pinia'
import { ref } from 'vue'

import { getCurrentUser, updateEducationStage } from '@/api/user.api'
import type { UserMeResponse } from '@/types/user'
import { clearStoredAuthUser, setStoredAuthUser } from '@/utils/auth'

export const useUserStore = defineStore('user', () => {
  const profile = ref<UserMeResponse | null>(null)
  const loading = ref(false)

  async function fetchProfile() {
    loading.value = true
    try {
      profile.value = await getCurrentUser()
      setStoredAuthUser({
        user_id: profile.value.user_id,
        username: profile.value.username,
      })
    } finally {
      loading.value = false
    }
  }

  async function saveEducationStage(stage:string) { profile.value = await updateEducationStage(stage) }

  function clearProfile() {
    profile.value = null
    clearStoredAuthUser()
  }

  return {
    profile,
    loading,
    fetchProfile,
    saveEducationStage,
    clearProfile,
  }
})
