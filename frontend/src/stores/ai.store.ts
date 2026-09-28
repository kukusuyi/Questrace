import { defineStore } from 'pinia'
import { computed, ref } from 'vue'

import { listAIChapters, listAIModelProviders, listAIProviderModels } from '@/api/ai.api'
import type { OptionItem } from '@/types/common'
import type { AIProviderItem, AIProviderModelItem } from '@/types/ai'

export const useAIStore = defineStore('ai', () => {
  const providers = ref<AIProviderItem[]>([])
  const chapters = ref<string[]>([])
  const providerModels = ref<Record<string, AIProviderModelItem[]>>({})
  const loadingProviders = ref(false)
  const loadingChapters = ref(false)
  const loadingModels = ref<Record<string, boolean>>({})

  const providerOptions = computed(() =>
    providers.value.map((item) => ({
      label: item.configured_model
        ? `${item.provider_name} · ${item.configured_model}`
        : item.provider_name,
      value: item.provider_name,
    })),
  )
  const chapterOptions = computed<OptionItem[]>(() =>
    chapters.value.map((item) => ({
      label: item,
      value: item,
    })),
  )
  const chapterOptionsWithAuto = computed<OptionItem[]>(() => [
    { label: '自动判断章节', value: '' },
    ...chapterOptions.value,
  ])

  function getModels(providerName: string) {
    return providerModels.value[providerName] || []
  }

  async function fetchProviders(force = false) {
    if (providers.value.length && !force) {
      return providers.value
    }

    loadingProviders.value = true
    try {
      const response = await listAIModelProviders()
      providers.value = response.list
      return response.list
    } finally {
      loadingProviders.value = false
    }
  }

  const chaptersByScope=ref<Record<string,string[]>>({})
  let chapterRequest=0
  async function fetchChapters(force=false,subjectId='',courseId='') {
    const token=++chapterRequest,key=`${subjectId}:${courseId}`
    chapters.value=[]
    if(!subjectId)return []
    if(!force && chaptersByScope.value[key]){chapters.value=chaptersByScope.value[key];return chapters.value}
    loadingChapters.value=true
    try{const response=await listAIChapters(subjectId,courseId);chaptersByScope.value[key]=response.list;if(token===chapterRequest)chapters.value=response.list;return response.list}
    finally{if(token===chapterRequest)loadingChapters.value=false}
  }

  async function fetchModels(providerName: string, force = false) {
    if (!providerName.trim()) {
      return []
    }

    if (providerModels.value[providerName] && !force) {
      return providerModels.value[providerName]
    }

    loadingModels.value = {
      ...loadingModels.value,
      [providerName]: true,
    }

    try {
      const response = await listAIProviderModels(providerName)
      providerModels.value = {
        ...providerModels.value,
        [providerName]: response.list,
      }
      return response.list
    } finally {
      loadingModels.value = {
        ...loadingModels.value,
        [providerName]: false,
      }
    }
  }

  function isLoadingModels(providerName: string) {
    return !!loadingModels.value[providerName]
  }

  return {
    providers,
    chapters,
    providerOptions,
    chapterOptions,
    chapterOptionsWithAuto,
    loadingProviders,
    loadingChapters,
    fetchProviders,
    fetchChapters,
    fetchModels,
    getModels,
    isLoadingModels,
  }
})
