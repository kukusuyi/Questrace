import type {
  DraftFlowMode,
  DraftStatus,
  MasteryStatus,
  OCRConfidence,
  PageResult,
  SimilarityType,
  SourceType,
  VectorType,
} from './common'

export interface QuestionJSON {
  question_core: string
  standard_solution: string
  wrong_solution: string
}

export interface TagGroups {
  knowledge_points: string[]
  problem_type: string[]
  method: string[]
  mistake_reason: string[]
}

export interface OCRContext {
  ocr_confidence: OCRConfidence
  uncertain_parts: string[]
}

export interface QuestionDraft {
 warnings?:string[]
 suggested_subject?:string
  subject_id?: string
  course_id?: string
  classification_status?: string
  analysis_stale?: boolean
  analysis_confirmed?: boolean
  revision?: number

  provider_name: string
  model_name: string
  source_type: SourceType
  source_image_id?: number
  source_image_url: string
  subject: string
  chapter: string
  chapter_locked: boolean
  question_json: QuestionJSON
  tags: TagGroups
  semantic_summary: string
  mistake_summary: string
  difficulty_level: number
  mastery_status: MasteryStatus
  ocr_context?: OCRContext
  flow_mode: DraftFlowMode
  status: DraftStatus
}

export interface CreateWrongQuestionPayload {
  subject_id?: string
  course_id?: string
  classification_status?: string
  analysis_stale?: boolean
  analysis_confirmed?: boolean
  revision?: number

  source_type: SourceType
  source_image_id?: number
  source_image_url: string
  subject: string
  chapter: string
  question_json: QuestionJSON
  tags: TagGroups
  semantic_summary: string
  mistake_summary: string
  difficulty_level: number
  mastery_status: MasteryStatus
}

export interface UpdateWrongQuestionPayload {
  subject_id?: string
  course_id?: string
  classification_status?: string
  analysis_stale?: boolean
  analysis_confirmed?: boolean
  revision?: number

  question_json: QuestionJSON
  subject: string
  chapter: string
  tags: TagGroups
  source_image_id?: number
  source_image_url: string
  semantic_summary: string
  mistake_summary: string
  difficulty_level: number
  mastery_status: MasteryStatus
}

export interface CreateWrongQuestionResponse {
  question_id: number
}

export interface UpdateWrongQuestionResponse {
  question_id: number
  updated: boolean
}

export interface DeleteWrongQuestionResponse {
  question_id: number
  deleted: boolean
}

export interface QuestionListItem {
  subject_id?: string
  course_id?: string
  classification_status?: string
  analysis_stale?: boolean
  analysis_confirmed?: boolean
  revision?: number

  question_id: number
  question_core: string
  source_image_id?: number
  source_image_url: string
  subject: string
  chapter: string
  tags: TagGroups
  difficulty_level: number
  mastery_status: MasteryStatus
  created_at: string
}

export interface QuestionDetail extends QuestionListItem {
  standard_solution: string
  wrong_solution: string
  semantic_summary: string
  mistake_summary: string
  source_type: SourceType
  updated_at: string
}

export interface SimilarQuestionRequest {
 recall_scope?: "course" | "subject"
  vector_type: VectorType
  limit: number
  use_tag_filter: boolean
}

export interface SimilarQuestionItem {
 subject?:string
 chapter?:string
  subject_id?: string
  course_id?: string
  classification_status?: string
  analysis_stale?: boolean
  analysis_confirmed?: boolean
  revision?: number

  question_id: number
  score: number
  similarity_type: SimilarityType
  question_core: string
  source_image_id?: number
  source_image_url: string
  matched_tags: string[]
  reason: string
  tags: TagGroups
}

export interface SimilarQuestionResponse {
 status?:string
  list: SimilarQuestionItem[]
}

export interface SimilarByJSONPayload extends SimilarQuestionRequest {
  chapter?: string
  subject_id?: string
  course_id?: string
  classification_status?: string
  analysis_stale?: boolean
  analysis_confirmed?: boolean
  revision?: number

  question_json: QuestionJSON
  tags: TagGroups
}

export interface ListQuestionFilter {
  subject_id?: string
  course_id?: string
  classification_status?: string
  analysis_stale?: boolean
  analysis_confirmed?: boolean
  revision?: number

  page?: number
  page_size?: number
  subject?: string
  chapter?: string
  keyword?: string
  tag_ids?: string
  mastery_status?: MasteryStatus | ''
  difficulty_level?: number
  source_type?: SourceType | ''
}

export type QuestionPageResult = PageResult<QuestionListItem>
