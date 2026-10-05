import type {Classification} from '@/stores/subjects.store'
import {httpGet,httpPost} from './http'
export type ReviewItem=Classification & {question_id:number;question_core:string;standard_solution:string;source_image_url:string;result:string;deleted:boolean}
export type ReviewSession={id:number;requested_count:number;created_at:number;items:ReviewItem[]}
export type ReviewSummary={due:number;sessions:{id:number;created_at:number;total:number;done:number}[]}
export type ReviewHistory={question_id:number;result:string;mastery_status:string;reviewed_at:string;note:string;due_at:number}
export const reviewSummary=()=>httpGet<ReviewSummary>('/api/v1/reviews/summary')
export const reviewHistory=()=>httpGet<ReviewHistory[]>('/api/v1/reviews/history')
export const getReview=(id:number)=>httpGet<ReviewSession>(`/api/v1/reviews/sessions/${id}`)
export const createReview=(input:{subject:string;subject_id?:string;course_id?:string;chapter?:string;tag_ids:number[];mastery_status:string;count:number})=>httpPost<ReviewSession>('/api/v1/reviews/sessions',input)
export const submitReview=(id:number,input:{question_id:number;submission_id:string;result:string;note:string})=>httpPost<{mastery_status:string;due_at:number}>(`/api/v1/reviews/sessions/${id}/results`,input)
