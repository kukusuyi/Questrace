import { defineStore } from 'pinia'
import { ref } from 'vue'
import { httpGet, httpPost } from '@/api/http'
export interface SubjectCourse { id:string; name:string; chapters:string[] }
export interface Subject { education_stage?:string; id:string; name:string; specialized:boolean; courses:SubjectCourse[]; chapters:string[] }
export interface Classification { suggested_subject?:string; subject_id?:string; subject?:string; course_id?:string; chapter?:string; classification_status?:string; analysis_stale?:boolean; analysis_confirmed?:boolean; chapter_locked?:boolean; revision?:number }
export const useSubjectsStore=defineStore('subjects',()=>{
 const items=ref<Subject[]>([]),educationStage=ref('university')
 let generation=0
 async function fetch(){const current=++generation;const data=await httpGet<{list:Subject[];education_stage?:string}>('/api/v1/subjects');if(current!==generation)return;items.value=data.list;educationStage.value=data.education_stage||'university'}
 async function create(name:string){const item=await httpPost<Subject>('/api/v1/subjects',{name});if(!items.value.some(x=>x.id===item.id))items.value.push(item);return item}
 function reset(){generation++;items.value=[];educationStage.value='university'}
 return {items,educationStage,fetch,create,reset}
})
export const courseNames:Record<string,string>={data_structures:'数据结构',computer_organization:'计算机组成原理',operating_systems:'操作系统',computer_networks:'计算机网络'}
export const subjectNames:Record<string,string>={math_grad:'考研数学',cs408:'408',highschool_math:'高中数学',highschool_geography:'高中地理',highschool_biology:'高中生物',highschool_physics:'高中物理',highschool_chemistry:'高中化学'}
export function classificationLabel(x:Classification){return [subjectNames[x.subject_id||'']||x.subject||'待分类',courseNames[x.course_id||''],x.chapter].filter(Boolean).join(' · ')}
