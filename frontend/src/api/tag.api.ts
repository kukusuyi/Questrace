import { httpDelete, httpGet, httpPost } from './http'

import type { CreateTagPayload, TagItem, TagListResponse } from '@/types/tag'

export function listTags(params?: { tag_type?: string; keyword?: string; subject_id?:string; course_id?:string }) {
  return httpGet<TagListResponse>('/api/v1/tags', params)
}

export function createTag(payload: CreateTagPayload) {
  return httpPost<TagItem>('/api/v1/tags', payload)
}

export function deleteTag(tagID: number) {
  return httpDelete<{ tag_id: number; deleted: boolean }>(`/api/v1/tags/${tagID}`)
}
