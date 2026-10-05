import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

import type { DraftFlowMode, DraftStatus, MasteryStatus, SourceType } from '@/types/common'
import type { AnalyzeWrongQuestionResponse } from '@/types/ai'
import type { OCRContext, QuestionDraft, QuestionJSON, TagGroups } from '@/types/question'

const STORAGE_KEY = 'questrace:draft'

function createEmptyTags(): TagGroups {
  return {
    knowledge_points: [],
    problem_type: [],
    method: [],
    mistake_reason: [],
  }
}

function createEmptyQuestionJSON(): QuestionJSON {
  return {
    question_core: '',
    standard_solution: '',
    wrong_solution: '',
  }
}

export function createEmptyDraft(flowMode: DraftFlowMode = 'manual'): QuestionDraft {
  const sourceType: SourceType = flowMode === 'upload' ? 'image' : 'manual'
  const masteryStatus: MasteryStatus = 'unmastered'
  const status: DraftStatus = 'draft'

  return {
    provider_name: '',
    model_name: '',
    source_type: sourceType,
    source_image_url: '',
    subject: '待分类',
    subject_id:'',course_id:'',classification_status:'pending',analysis_stale:false,
    chapter: '',
    chapter_locked: false,
    question_json: createEmptyQuestionJSON(),
    tags: createEmptyTags(),
    semantic_summary: '',
    mistake_summary: '',
    difficulty_level: 3,
    mastery_status: masteryStatus,
    flow_mode: flowMode,
    status,
  }
}

export const useDraftStore = defineStore('draft', () => {
  const currentDraft = ref<QuestionDraft | null>(null)

  function loadDraft() {
    const raw = window.sessionStorage.getItem(STORAGE_KEY)
    if (!raw) {
      return
    }

    try {
      const parsed = JSON.parse(raw) as Partial<QuestionDraft>
      const base = createEmptyDraft(parsed.flow_mode || 'manual')

      currentDraft.value = {
        ...base,
        ...parsed,
        ...(parsed.subject_id===undefined?{subject_id:'',classification_status:'pending',analysis_stale:true,analysis_confirmed:false}:{}),
        question_json: {
          ...base.question_json,
          ...parsed.question_json,
        },
        tags: {
          ...base.tags,
          ...parsed.tags,
        },
      }
    } catch {
      window.sessionStorage.removeItem(STORAGE_KEY)
    }
  }

  function ensureDraft(flowMode: DraftFlowMode = 'manual') {
    if (!currentDraft.value) {
      currentDraft.value = createEmptyDraft(flowMode)
    }

    return currentDraft.value
  }

  function initializeDraft(flowMode: DraftFlowMode) {
    window.sessionStorage.removeItem(STORAGE_KEY+':before-analysis')
    currentDraft.value = createEmptyDraft(flowMode)
  }

  function updateQuestionJSON(questionJSON: QuestionJSON) {
    ensureDraft().question_json = questionJSON
  }

  function updateOCRContext(ocrContext?: OCRContext) {
    ensureDraft().ocr_context = ocrContext
  }

  function updateAIModelSelection(providerName: string, modelName: string) {
    const draft = ensureDraft()
    draft.provider_name = providerName
    draft.model_name = modelName
  }

  function setUploadedImage(imageID: number, imageURL: string) {
    const draft = ensureDraft('upload')
    draft.source_type = 'image'
    draft.source_image_id = imageID
    draft.source_image_url = imageURL
    draft.question_json = createEmptyQuestionJSON()
    draft.ocr_context = undefined
    draft.subject_id='';draft.subject='待分类';draft.course_id='';draft.classification_status='pending';draft.analysis_stale=false;draft.analysis_confirmed=false;draft.warnings=[];draft.suggested_subject='';
    draft.chapter = ''
    draft.chapter_locked = false
    draft.tags = createEmptyTags()
    draft.semantic_summary = ''
    draft.mistake_summary = ''
    draft.status = 'image_uploaded'
  }

  function discardAnalysis() {
    const raw=window.sessionStorage.getItem(STORAGE_KEY+':before-analysis')
    if(raw){try{currentDraft.value={...JSON.parse(raw),status:'draft'}}catch{};window.sessionStorage.removeItem(STORAGE_KEY+':before-analysis')}
  }
  function applyAnalysis(result: AnalyzeWrongQuestionResponse) {
    const draft = ensureDraft()
    if(!window.sessionStorage.getItem(STORAGE_KEY+':before-analysis'))window.sessionStorage.setItem(STORAGE_KEY+':before-analysis',JSON.stringify(draft))
    currentDraft.value = {
      ...draft,
      subject_id:result.subject_id||'',subject:result.subject||'待分类',course_id:result.course_id||'',classification_status:result.classification_status||'pending',analysis_stale:false,analysis_confirmed:true,warnings:result.warnings,suggested_subject:result.suggested_subject,
      chapter: result.chapter,
      chapter_locked: draft.chapter_locked,
      tags: {
        ...result.tags,
      },
      semantic_summary: result.semantic_summary,
      mistake_summary: result.mistake_summary,
      status: 'ai_reviewing',
    }
  }

  function resetDraft() {
    currentDraft.value = null
    window.sessionStorage.removeItem(STORAGE_KEY+':before-analysis')
    window.sessionStorage.removeItem(STORAGE_KEY)
  }

  function updateChapterSelection(chapter: string, locked: boolean) {
    const draft = ensureDraft()
    currentDraft.value = {
      ...draft,
      chapter,
      chapter_locked: locked,
    }
  }

  watch(
    currentDraft,
    (value) => {
      if (!value) {
        window.sessionStorage.removeItem(STORAGE_KEY)
        return
      }

      window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(value))
    },
    { deep: true },
  )

  loadDraft()

  return {
    currentDraft,
    initializeDraft,
    ensureDraft,
    updateQuestionJSON,
    updateOCRContext,
    updateAIModelSelection,
    setUploadedImage,
    applyAnalysis,
    discardAnalysis,
    updateChapterSelection,
    resetDraft,
  }
})
