import {useUserStore} from './user.store'
import {useSubjectsStore} from './subjects.store'
import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { login, register } from '@/api/auth.api'
import type { AuthResponse, LoginPayload, RegisterPayload } from '@/types/auth'
import {
  clearAuthState,
  getAuthToken,
  getStoredAuthUser,
  setAuthToken,
  setStoredAuthUser,
} from '@/utils/auth'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(getAuthToken())
  const authUser = ref(getStoredAuthUser())
  const loading = ref(false)

  const isAuthenticated = computed(() => Boolean(token.value))

  function applyAuth(result: AuthResponse) {
    useUserStore().clearProfile();useSubjectsStore().reset()
    token.value = result.token
    authUser.value = {
      user_id: result.user_id,
      username: result.username,
    }
    setAuthToken(result.token)
    setStoredAuthUser(authUser.value)
  }

  async function loginWithPassword(payload: LoginPayload) {
    loading.value = true
    try {
      const result = await login(payload)
      applyAuth(result)
      return result
    } finally {
      loading.value = false
    }
  }

  async function registerAccount(payload: RegisterPayload) {
    loading.value = true
    try {
      const result = await register(payload)
      applyAuth(result)
      return result
    } finally {
      loading.value = false
    }
  }

  function logout() {
    useUserStore().clearProfile();useSubjectsStore().reset()
    token.value = ''
    authUser.value = null
    clearAuthState()
  }

  return {
    token,
    authUser,
    loading,
    isAuthenticated,
    applyAuth,
    loginWithPassword,
    registerAccount,
    logout,
  }
})
