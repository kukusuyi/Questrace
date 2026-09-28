import type { OCRContext, QuestionJSON, TagGroups } from './question'

export interface AnalyzeWrongQuestionPayload {
 subject_id?:string
 course_id?:string
  provider_name?: string
  model_name?: string
  chapter?: string
  question_json: QuestionJSON
  ocr_context?: OCRContext
}

export interface AnalyzeWrongQuestionResponse {
 subject_id?:string
 subject?:string
 course_id?:string
 classification_status?:string
 suggested_subject?:string
 warnings?:string[]
  chapter: string
  tags: TagGroups
  semantic_summary: string
  mistake_summary: string
}

export interface AIProviderItem {
  provider_name: string
  provider_type: string
  configured_model: string
}

export interface AIProviderListResponse {
  list: AIProviderItem[]
}

export interface AIProviderModelItem {
  model_name: string
}

export interface AIProviderModelListResponse {
  provider_name: string
  list: AIProviderModelItem[]
}

export interface AIChapterListResponse {
  list: string[]
}
