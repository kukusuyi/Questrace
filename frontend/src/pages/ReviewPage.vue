<template>
<div class="page-shell">
 <header class="page-header"><div><h2 class="page-title">复习</h2><p class="page-subtitle">回忆、核对，再把掌握变得扎实。</p></div><el-button v-if="session" @click="router.push('/reviews')">返回复习</el-button></header>
 <el-alert v-if="error" :title="error" type="error" :closable="false"/>
 <el-skeleton v-if="loading" :rows="4" animated/>
 <template v-else-if="session">
 <section class="paper-card review-panel"><p>已完成 {{done}} / {{session.items.length}} 题 · 进度自动保存</p><el-progress :percentage="Math.round(done/session.items.length*100)"/><el-button @click="print">导出本卷 PDF</el-button>
 <template v-if="current"><h3>第 {{session.items.indexOf(current)+1}} 题</h3><ClassificationBadge :item="current"/><LatexRenderer :content="current.question_core"/><img v-if="current.source_image_url" class="source-image" :src="current.source_image_url" alt="题目原图"/>
 <el-button v-if="!revealed" type="primary" @click="revealed=true">完成思考，查看答案</el-button>
 <template v-else><h3>答案与解析</h3><LatexRenderer :content="current.standard_solution||'暂无标准答案，请结合自己的解题过程自评。'"/><el-input v-model="note" type="textarea" placeholder="复习笔记（可选）" maxlength="10000" aria-label="复习笔记"/><div class="review-actions"><el-button v-for="(label,value) in results" :key="value" :disabled="busy" @click="submit(value)">{{label}}</el-button></div></template>
 </template><template v-else><el-result icon="success" title="本次练习已完成" sub-title="下次复习已根据自评结果安排。"/><div v-for="item in session.items" :key="item.question_id" class="session-row"><RouterLink v-if="!item.deleted" :to="`/questions/${item.question_id}`">题目 {{item.question_id}}</RouterLink><span>{{item.deleted?'题目已删除':results[item.result as keyof typeof results]}}</span></div></template>
 </section></template>
 <template v-else>
 <section class="paper-card review-panel"><h3>待复习 {{summary.due}} 题</h3><p class="meta-text">优先到期题目，再补充未掌握与学习中的题目。连续三次有效答对后标为已掌握。</p>
 <el-form label-position="top" class="review-filters"><SubjectPicker :model="form" filter @change="form.tag_ids=[]"/><el-form-item label="标签"><TagFilter v-model="form.tag_ids" :subject-id="form.subject_id" :course-id="form.course_id"/></el-form-item><el-form-item label="掌握状态"><el-select v-model="form.mastery_status" clearable placeholder="到期及待掌握"><el-option v-for="(label,value) in mastery" :key="value" :value="value" :label="label"/></el-select></el-form-item><el-form-item label="题数"><el-input-number v-model="form.count" :min="1" :max="100"/></el-form-item></el-form><el-button type="primary" :loading="busy" @click="create">自动组卷并开始</el-button></section>
 <section class="paper-card review-panel"><h3>我的练习</h3><el-empty v-if="!summary.sessions.length" description="开始第一份练习吧"/><div v-for="s in summary.sessions" :key="s.id" class="session-row"><span>{{new Date(s.created_at*1000).toLocaleString()}} · {{s.done}} / {{s.total}} 题</span><el-button @click="router.push({path:'/reviews',query:{session:s.id}})">{{s.done<s.total?'继续练习':'查看练习'}}</el-button></div></section>
 <section class="paper-card review-panel"><h3>最近复习记录</h3><el-empty v-if="!history.length" description="还没有复习记录"/><article v-for="(h,i) in history" :key="i" class="session-row"><RouterLink :to="`/questions/${h.question_id}`">题目 {{h.question_id}}</RouterLink><span>{{results[h.result as keyof typeof results]}} · {{mastery[h.mastery_status as keyof typeof mastery]}}<small v-if="h.note"> · {{h.note}}</small></span></article></section>
 </template>
</div>
</template>
<script setup lang="ts">
import ClassificationBadge from "@/components/ClassificationBadge/index.vue"
import SubjectPicker from "@/components/SubjectPicker/index.vue";
import {computed,reactive,ref,watch} from 'vue'
import {useRoute,useRouter,RouterLink} from 'vue-router'
import {ElMessage} from 'element-plus'
import LatexRenderer from '@/components/LatexRenderer/index.vue'
import TagFilter from '@/components/TagFilter/index.vue'
import {reviewSummary,reviewHistory,getReview,createReview,submitReview,type ReviewSession,type ReviewSummary,type ReviewHistory} from '@/api/review.api'
import {buildQuestionExportPrintURL} from '@/api/question.api'
import {getErrorMessage} from '@/utils/error'
const router=useRouter(),route=useRoute(),loading=ref(false),busy=ref(false),error=ref(''),session=ref<ReviewSession|null>(null),revealed=ref(false),note=ref('')
const summary=ref<ReviewSummary>({due:0,sessions:[]}),history=ref<ReviewHistory[]>([])
const form=reactive({subject:'',subject_id:'',course_id:'',chapter:'',tag_ids:[] as number[],mastery_status:'',count:10})
const mastery={unmastered:'未掌握',learning:'学习中',mastered:'已掌握'},results={forgot:'不会',partial:'模糊',correct:'会了'}
const current=computed(()=>session.value?.items.find(i=>!i.result&&!i.deleted)),done=computed(()=>session.value?.items.filter(i=>i.result||i.deleted).length||0)
let submissionID=''
watch(()=>current.value?.question_id,()=>{revealed.value=false;note.value='';submissionID=globalThis.crypto?.randomUUID?.()||`${Date.now()}-${Math.random().toString(36).slice(2)}`})
async function load(){loading.value=true;error.value='';try{session.value=null;if(route.query.session)session.value=await getReview(Number(route.query.session));else [summary.value,history.value]=await Promise.all([reviewSummary(),reviewHistory()])}catch(e){error.value=getErrorMessage(e,'复习加载失败')}finally{loading.value=false}}
watch(()=>route.query.session,load,{immediate:true})
async function create(){busy.value=true;try{const s=await createReview(form);if(s.items.length<form.count)ElMessage.info(`符合条件的题目共 ${s.items.length} 道，已全部加入`);await router.push({path:'/reviews',query:{session:s.id}})}catch(e){error.value=getErrorMessage(e,'组卷失败')}finally{busy.value=false}}
async function submit(result:string){if(!session.value||!current.value)return;busy.value=true;error.value='';try{const outcome=await submitReview(session.value.id,{question_id:current.value.question_id,submission_id:submissionID,result,note:note.value});current.value.result=result;ElMessage.success(`${mastery[outcome.mastery_status as keyof typeof mastery]} · 下次复习 ${new Date(outcome.due_at*1000).toLocaleDateString()}`)}catch(e){error.value=getErrorMessage(e,'提交失败，可重试或返回刷新')}finally{busy.value=false}}
function print(){if(session.value)window.open(buildQuestionExportPrintURL(session.value.items.filter(i=>!i.deleted).map(i=>i.question_id),'questions_only'),'_blank','noopener,noreferrer')}
</script>
<style scoped>
.review-panel{padding:24px}.review-panel>*+*{margin-top:16px}.review-filters{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:16px}.review-actions,.session-row{display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap}.session-row{padding:12px 0;border-bottom:1px solid var(--line)}.source-image{max-width:100%;max-height:500px}.review-actions .el-button{margin:0;min-height:44px}@media(max-width:600px){.review-filters{grid-template-columns:1fr}.review-panel{padding:16px}}
</style>
