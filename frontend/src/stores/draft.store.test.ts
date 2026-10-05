import { beforeEach, expect, it } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import { useDraftStore } from './draft.store'
import type { AnalyzeWrongQuestionResponse } from '@/types/ai'
beforeEach(()=>{sessionStorage.clear();setActivePinia(createPinia())})
it('restores original content when analysis is rejected, including after reload',async()=>{let store=useDraftStore();const original=store.ensureDraft();original.question_json.question_core='手动整理的内容';original.chapter='原章节';store.applyAnalysis({chapter:'AI章节',tags:original.tags,semantic_summary:'AI总结',mistake_summary:'AI错因'} as AnalyzeWrongQuestionResponse);await nextTick();expect(store.currentDraft?.chapter).toBe('AI章节');setActivePinia(createPinia());store=useDraftStore();store.discardAnalysis();expect(store.currentDraft?.chapter).toBe('原章节');expect(store.currentDraft?.question_json.question_core).toBe('手动整理的内容');store.resetDraft();expect(sessionStorage.getItem('questrace:draft:before-analysis')).toBeNull()})
it('does not discard a draft when choosing another entry point',()=>{const store=useDraftStore();store.ensureDraft().question_json.question_core='待整理';store.ensureDraft('upload');expect(store.currentDraft?.question_json.question_core).toBe('待整理')})
it('deleting a resumed draft remains deleted after reload', async()=>{
 let store=useDraftStore();store.ensureDraft().question_json.question_core='未完成';await nextTick();
 setActivePinia(createPinia());store=useDraftStore();expect(store.currentDraft).not.toBeNull();
 store.resetDraft();await nextTick();setActivePinia(createPinia());expect(useDraftStore().currentDraft).toBeNull();
})
it('does not default to math and restores classification when analysis is rejected',async()=>{
 const store=useDraftStore();const draft=store.ensureDraft();expect(draft.subject_id).toBe('');expect(draft.subject).toBe('待分类');
 draft.subject_id='highschool_physics';draft.subject='高中物理';draft.course_id='';draft.classification_status='confirmed';
 store.applyAnalysis({subject_id:'cs408',subject:'408',course_id:'operating_systems',chapter:'内存管理',classification_status:'confirmed',tags:draft.tags,semantic_summary:'分页',mistake_summary:''});
 await nextTick();expect(store.currentDraft?.course_id).toBe('operating_systems');
 setActivePinia(createPinia());const recovered=useDraftStore();expect(recovered.currentDraft?.subject_id).toBe('cs408');recovered.discardAnalysis();expect(recovered.currentDraft?.subject_id).toBe('highschool_physics');expect(recovered.currentDraft?.course_id).toBe('');
})
it('new upload discards previous classification and AI confirmation',()=>{const store=useDraftStore();const draft=store.ensureDraft();draft.subject_id='math_grad';draft.analysis_confirmed=true;store.setUploadedImage(1,'/image');expect(store.currentDraft?.subject_id).toBe('');expect(store.currentDraft?.analysis_confirmed).toBe(false)})
