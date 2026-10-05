<template>
 <div class="classification-fields">
  <el-form-item label="学科">
   <template v-if="proposal!==null && !filter && !model.subject_id">
    <el-input :model-value="proposal" :disabled="busy" maxlength="64" placeholder="确认或修改识别出的学科" @update:model-value="editProposal" @keyup.enter="confirmProposal"/>
    <div class="suggestion-actions"><span>AI 已填写建议学科，确认后加入学科列表。</span><el-button type="primary" :loading="busy" :disabled="!proposal.trim()" @click="confirmProposal">确认学科</el-button><el-button text :disabled="busy" @click="dismissProposal">选择已有学科</el-button></div>
   </template>
   <el-select v-else :model-value="model.subject_id||''" filterable clearable :allow-create="!filter" :loading="busy" :placeholder="filter?'全部学科':'自动识别，或输入任意学科名称'" @change="selectSubject">
   <el-option v-for="s in visibleSubjects" :key="s.id" :value="s.id" :label="s.name"/>
  </el-select></el-form-item>
  <el-form-item v-if="subject?.courses.length" label="课程"><el-select :model-value="model.course_id||''" clearable :placeholder="filter?'全部课程':'自动识别课程'" @change="selectCourse"><el-option v-for="c in subject.courses" :key="c.id" :value="c.id" :label="c.name"/></el-select></el-form-item>
  <el-form-item v-if="model.subject_id" label="章节">
   <el-select v-if="subject?.specialized" :model-value="model.chapter||''" clearable filterable :disabled="!!subject.courses.length&&!model.course_id" :placeholder="filter?'全部章节':'自动识别章节'" @change="selectChapter"><el-option v-for="c in chapters" :key="c" :value="c" :label="c"/></el-select>
   <el-input v-else :model-value="model.chapter||''" clearable placeholder="可选：章节或主题" @update:model-value="selectChapter"/>
  </el-form-item>
 </div>
 <div v-if="!filter" class="subject-help"><span>{{store.educationStage==='highschool'?'高中阶段 · 也可添加语文、英语等其他科目':'大学阶段 · 支持材料、机械、医学等任意专业科目'}}</span><el-button text :loading="busy" @click="addSubject">添加学科</el-button></div>
 <p v-if="error" role="alert">{{error}} <el-button text @click="load">重试</el-button></p>
</template>
<script setup lang="ts">
import {ElMessageBox} from 'element-plus'
import {computed,onMounted,ref,watch} from 'vue'
import {useSubjectsStore,type Classification} from '@/stores/subjects.store'
import {getErrorMessage} from '@/utils/error'
const props=defineProps<{model:Classification;filter?:boolean}>()
const emit=defineEmits<{change:[]}>()
const store=useSubjectsStore(),error=ref(''),busy=ref(false)
const proposal=ref<string|null>(null)
watch(()=>[props.model.suggested_subject,props.model.subject_id],()=>{if(props.model.subject_id){proposal.value=null}else if(props.model.suggested_subject){proposal.value=props.model.suggested_subject}}, {immediate:true})
function editProposal(value:string){proposal.value=value;props.model.suggested_subject=value}
function dismissProposal(){proposal.value=null;props.model.suggested_subject=''}
async function confirmProposal(){if(busy.value||!proposal.value?.trim())return;await selectSubject(proposal.value.trim())}
const subject=computed(()=>store.items.find(s=>s.id===props.model.subject_id))
const visibleSubjects=computed(()=>store.items.filter(s=>!s.education_stage||s.education_stage===store.educationStage||s.id===props.model.subject_id))
async function addSubject(){try{const {value}=await ElMessageBox.prompt('填写学科或专业课程名称，例如材料力学、机械原理。','添加学科',{inputPlaceholder:'学科名称',inputValidator:(v:string)=>!!v.trim()&&v.trim().length<=64||'请输入1至64个字符',confirmButtonText:'添加',cancelButtonText:'取消'});await selectSubject(value.trim())}catch{/* 用户取消 */}}
const chapters=computed(()=>subject.value?.courses.find(c=>c.id===props.model.course_id)?.chapters||subject.value?.chapters||[])
async function load(){try{await store.fetch();error.value=''}catch(e){error.value=getErrorMessage(e,'学科目录加载失败')}}
async function selectSubject(id:string){
 if(busy.value)return
 busy.value=true;error.value=''
 try{let s=store.items.find(x=>x.id===id);if(id&&!s&&!props.filter)s=await store.create(id)
 const m=props.model;m.subject_id=s?.id||'';if(!props.filter)m.subject=s?.name||'待分类';m.course_id='';m.chapter='';changed();if(s){m.suggested_subject='';proposal.value=null}
 }catch(e){error.value=getErrorMessage(e,'创建学科失败')}finally{busy.value=false}
}
function invalidateAnalysis(){props.model.analysis_stale=!!(props.model.analysis_stale||props.model.analysis_confirmed||props.model.revision);props.model.analysis_confirmed=false}
function changed(){if(!props.filter){invalidateAnalysis();props.model.classification_status=props.model.subject_id?'confirmed':'pending';props.model.analysis_confirmed=false;props.model.chapter_locked=false}emit('change')}
function selectCourse(id:string){props.model.course_id=id||'';props.model.chapter='';changed()}
function selectChapter(chapter:string){props.model.chapter=chapter||'';if(!props.filter){props.model.chapter_locked=!!chapter;invalidateAnalysis()}emit('change')}
onMounted(load)
</script>
<style scoped>.classification-fields{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:16px;width:100%}.classification-fields :deep(.el-select){width:100%}.suggestion-actions{display:flex;align-items:center;gap:8px;flex-wrap:wrap;margin-top:8px;font-size:13px;color:var(--text-secondary)}.subject-help{display:flex;align-items:center;justify-content:space-between;gap:12px;flex-wrap:wrap;font-size:13px;color:var(--text-secondary);margin-bottom:16px}p{color:var(--text-secondary)}</style>
