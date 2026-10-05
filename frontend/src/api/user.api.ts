import { httpGet, httpPut } from './http'

import type { UserMeResponse } from '@/types/user'

export function getCurrentUser() {
  return httpGet<UserMeResponse>('/api/v1/users/me')
}

export function updateEducationStage(education_stage:string) { return httpPut<UserMeResponse>('/api/v1/users/me',{education_stage}) }
