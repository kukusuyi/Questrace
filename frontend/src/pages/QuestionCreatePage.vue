<template>
 <div class="page-shell editor-page">
  <header class="page-header"><div><h2 class="page-title">整理错题</h2><p class="page-subtitle">写下题目与思路，先保存，再慢慢完善。</p></div><div class="header-actions"><el-button text @click="router.push('/questions/upload')">改用图片录入</el-button><el-dropdown><el-button text>更多</el-button><template #dropdown><el-dropdown-menu><el-dropdown-item @click="advanced=!advanced">高级编辑</el-dropdown-item><el-dropdown-item @click="resetDraft">清空草稿</el-dropdown-item></el-dropdown-menu></template></el-dropdown></div></header>
  <el-alert v-if="errorMessage" :title="errorMessage" type="error" :closable="false" role="alert"/>
  <section v-if="advanced" class="paper-card mode-card"><el-radio-group v-model="editorMode"><el-radio-button label="form">表单编辑</el-radio-button><el-radio-button label="json">JSON 编辑</el-radio-button></el-radio-group></section>
  <QuestionForm v-if="draft && editorMode==='form'" :model="draft" :tag-options="tagStore.groupedOptions"/>
  <QuestionJsonEditor v-else-if="draft" v-model="draft.question_json" @validation-change="jsonValid=$event"/>
  <el-collapse class="paper-card upload-card"><el-collapse-item title="附上原图（可选）" name="image"><UploadPanel :uploaded-image="draft?{image_id:draft.source_image_id,image_url:draft.source_image_url}:undefined" @success="handleImageUploaded"/></el-collapse-item></el-collapse>
  <div class="edit-action-bar"><el-button :disabled="submitting" @click="showAI=true">AI 辅助分析</el-button><el-button type="primary" :loading="submitting" @click="saveDirectly">保存错题</el-button></div>
  <el-dialog append-to-body align-center v-model="showAI" title="AI 辅助分析" width="560px"><p class="meta-text">分析结果将先供你检查，不会直接保存到题库。</p><AIModelSelector v-if="draft" v-model:provider-name="providerName" v-model:model-name="modelName"/><template #footer><el-button @click="showAI=false">返回编辑</el-button><el-button type="primary" :loading="submitting" @click="analyzeDraft">生成分析建议</el-button></template></el-dialog>
 </div>
</template>
<script setup lang="ts">
import { ElMessage, ElMessageBox } from "element-plus";
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";

import { createQuestion } from "@/api/question.api";
import { analyzeWrongQuestion } from "@/api/ai.api";
import AIModelSelector from "@/components/AIModelSelector/index.vue";
import QuestionForm from "@/components/QuestionForm/index.vue";
import QuestionJsonEditor from "@/components/QuestionJsonEditor/index.vue";
import UploadPanel from "@/components/UploadPanel/index.vue";
import { useDraftStore } from "@/stores/draft.store";
import { useTagStore } from "@/stores/tag.store";
import { getErrorMessage } from "@/utils/error";

const router = useRouter();
const draftStore = useDraftStore();
const tagStore = useTagStore();
const editorMode = ref<"form" | "json">("form");
const jsonValid = ref(true);
const submitting = ref(false);

const draft = computed(() => draftStore.currentDraft);
const providerName = computed({
    get: () => draft.value?.provider_name || "",
    set: (value: string) => {
        draftStore.updateAIModelSelection(value, draft.value?.model_name || "");
    },
});
const showAI = ref(false);
const advanced = ref(false);
const errorMessage = ref("");
const modelName = computed({
    get: () => draft.value?.model_name || "",
    set: (value: string) => {
        draftStore.updateAIModelSelection(
            draft.value?.provider_name || "",
            value,
        );
    },
});

async function resetDraft() {
 try { await ElMessageBox.confirm('这会清空正在整理的内容。','清空草稿',{type:'warning'}); draftStore.initializeDraft("manual"); } catch {}
}

function handleImageUploaded(payload: { image_id: number; image_url: string }) {
    const current = draftStore.ensureDraft("manual");
    current.source_type = "image";
    current.source_image_id = payload.image_id;
    current.source_image_url = payload.image_url;
    ElMessage.success("图片已绑定到当前草稿");
}

async function saveDirectly() {
 if(submitting.value)return;
 const d=draft.value;if(!d || !d.question_json.question_core.trim() || !jsonValid.value){ElMessage.warning('请填写有效的题目主干');return}
 submitting.value=true
 try {const result=await createQuestion({source_type:d.source_type,source_image_id:d.source_image_id,source_image_url:d.source_image_url,subject:d.subject,subject_id:d.subject_id,course_id:d.course_id,analysis_stale:d.analysis_stale,analysis_confirmed:d.analysis_confirmed,chapter:d.chapter,question_json:d.question_json,tags:d.tags,semantic_summary:d.semantic_summary||d.question_json.question_core,mistake_summary:d.mistake_summary,difficulty_level:d.difficulty_level,mastery_status:d.mastery_status});draftStore.resetDraft();router.push(`/questions/${result.question_id}`)}catch(e){errorMessage.value=getErrorMessage(e,'保存失败')}finally{submitting.value=false}
}

async function analyzeDraft() {
    if(submitting.value)return;
    errorMessage.value="";
    const current = draft.value;
    if (!current) {
        return;
    }

    if (!jsonValid.value) {
        ElMessage.warning("请先修复 JSON 格式错误");
        return;
    }

    if (!current.question_json.question_core.trim()) {
        ElMessage.warning("请先填写题目内容");
        return;
    }

    if (!current.subject.trim()) {
        ElMessage.warning("请填写学科");
        return;
    }

    if (!current.provider_name.trim() || !current.model_name.trim()) {
        ElMessage.warning("请先选择模型厂商和模型名称");
        return;
    }

    current.status = "ai_processing";
    submitting.value = true;

    try {
        const result = await analyzeWrongQuestion({
            subject_id:current.subject_id,course_id:current.course_id,
            provider_name: current.provider_name,
            model_name: current.model_name,
            chapter:current.chapter_locked?current.chapter:undefined,
            question_json: current.question_json,
        });

        draftStore.applyAnalysis(result);
        router.push("/questions/ai-review");
    } catch (error) {
        errorMessage.value=getErrorMessage(error, "AI 分析失败，请重试；你的内容已保留。");
    } finally {
        submitting.value = false;
    }
}

onMounted(async () => {
    draftStore.ensureDraft("manual");

    try {
        await tagStore.fetchTags();
    } catch {
        // 标签选项加载失败不阻断录入。
    }
});
</script>

<style scoped>.editor-page{max-width:1100px}.mode-card,.upload-card{padding:16px 24px}.header-actions{display:flex;gap:8px;flex-wrap:wrap}</style>
