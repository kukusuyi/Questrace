<template><div class="page-shell upload-page"><header class="page-header"><div><h2 class="page-title">从图片开始</h2><p class="page-subtitle">上传一道题，识别文字或直接手动整理。</p></div><el-button text @click="router.push('/questions/create')">改用手动录入</el-button></header>
 <el-alert v-if="errorMessage" :title="errorMessage" type="error" :closable="false"/>
 <div class="upload-layout"><UploadPanel :uploaded-image="draft?{image_id:draft.source_image_id,image_url:draft.source_image_url}:undefined" @success="handleUploadSuccess"/><section class="paper-card status-panel"><h3>{{draft?.source_image_id?'图片已就绪':'选择题目图片'}}</h3><p class="meta-text">支持相册中的图片，也可以使用手机相机拍摄。原图会随错题一起保留。</p><el-alert v-if="!ocrEnabled" title="图片识别尚未配置。你仍可保留原图并手动填写题目。" type="info" :closable="false"/><div class="status-actions"><el-button v-if="ocrEnabled" :disabled="!draft?.source_image_id" :loading="processing" @click="runOCR">{{hasOCRResult?'重新识别':'识别图片文字'}}</el-button><el-button type="primary" :disabled="!draft?.source_image_id||processing" @click="router.push('/questions/create')">{{hasOCRResult?'确认并整理内容':'手动整理这张图片'}}</el-button></div><p v-if="draft?.ocr_context" class="meta-text">识别可能存在误差，请在整理页检查题干和公式。</p></section></div>
 <section v-if="hasOCRResult" class="paper-card status-panel"><h3>识别预览</h3><LatexRenderer :content="draft?.question_json.question_core" /></section>
 </div></template>
<script setup lang="ts">
import LatexRenderer from "@/components/LatexRenderer/index.vue";
import { ElMessage } from "element-plus";
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";

import { analyzeWrongQuestion, recognizeWrongQuestion } from "@/api/ai.api";
import AIModelSelector from "@/components/AIModelSelector/index.vue";
import QuestionForm from "@/components/QuestionForm/index.vue";
import UploadPanel from "@/components/UploadPanel/index.vue";
import { useAIStore } from "@/stores/ai.store";
import { useDraftStore } from "@/stores/draft.store";
import { getErrorMessage } from "@/utils/error";

const aiStore = useAIStore();
const draftStore = useDraftStore();
const router = useRouter();
const processing = ref(false);
const ocrEnabled=ref(false);
const errorMessage=ref("");

const draft = computed(() => draftStore.currentDraft);
const activeStep = computed(() => {
    if (!draft.value?.source_image_id) {
        return 0;
    }

    if (draft.value.status === "image_uploaded") {
        return 1;
    }

    if (draft.value.status === "ocr_processing") {
        return 1;
    }

    if (draft.value.status === "ocr_reviewing") {
        return 2;
    }

    if (draft.value.status === "ai_processing") {
        return 3;
    }

    if (draft.value.status === "ai_reviewing") {
        return 4;
    }

    return 0;
});
const hasOCRResult = computed(() => Boolean(draft.value?.ocr_context));
const providerName = computed({
    get: () => draft.value?.provider_name || "",
    set: (value: string) => {
        draftStore.updateAIModelSelection(value, draft.value?.model_name || "");
    },
});
const modelName = computed({
    get: () => draft.value?.model_name || "",
    set: (value: string) => {
        draftStore.updateAIModelSelection(
            draft.value?.provider_name || "",
            value,
        );
    },
});

function resetUploadDraft() {
    draftStore.initializeDraft("upload");
}

function handleUploadSuccess(payload: { image_id: number; image_url: string }) {
    draftStore.setUploadedImage(payload.image_id, payload.image_url);
}

function handleChapterChange(value: string) {
    draftStore.updateChapterSelection(value, value.trim().length > 0);
}

async function runOCR() {
    if(processing.value)return;
    errorMessage.value="";
    const current = draft.value;
    if (!current?.source_image_id || !current.source_image_url) {
        ElMessage.warning("请先完成图片上传");
        return;
    }

    processing.value = true;
    current.status = "ocr_processing";

    try {
        const ocrResult = await recognizeWrongQuestion({
            image_id: current.source_image_id,
            image_url: current.source_image_url,
        });

        current.question_json = {
            question_core: ocrResult.question_core,
            standard_solution: ocrResult.standard_solution,
            wrong_solution: ocrResult.wrong_solution,
        };
        current.ocr_context = {
            ocr_confidence: ocrResult.ocr_confidence,
            uncertain_parts: ocrResult.uncertain_parts,
        };
        current.status = "ocr_reviewing";
        ElMessage.success("OCR 识别完成，请先确认识别结果");
    } catch (error) {
        current.status = "image_uploaded";
        errorMessage.value=getErrorMessage(error, "识别失败，请重试或手动整理图片。");
    } finally {
        processing.value = false;
    }
}

async function runAnalysis() {
    const current = draft.value;
    if (!current) {
        return;
    }

    if (!current.question_json.question_core.trim()) {
        ElMessage.warning("question_core 不能为空");
        return;
    }

    processing.value = true;
    current.status = "ai_processing";

    try {
        const analysis = await analyzeWrongQuestion({
            subject_id:current.subject_id,course_id:current.course_id,
            provider_name: current.provider_name,
            model_name: current.model_name,
            chapter: current.chapter_locked
                ? (current.chapter || undefined)
                : undefined,
            question_json: current.question_json,
            ocr_context: current.ocr_context,
        });

        draftStore.applyAnalysis(analysis);
        router.push("/questions/ai-review");
    } catch (error) {
        current.status = "ocr_reviewing";
        ElMessage.error(getErrorMessage(error, "AI 分析失败"));
    } finally {
        processing.value = false;
    }
}

onMounted(async () => {
    try { const r=await fetch("/api/v1/system/status");ocrEnabled.value=(await r.json()).data.ocr_enabled; }catch{}
    draftStore.ensureDraft("upload");

    try {
        await aiStore.fetchChapters();
    } catch {
        // 章节列表加载失败不阻断 OCR 确认页使用。
    }
});
</script>

<style scoped>.upload-page{max-width:1100px}.upload-layout{display:grid;grid-template-columns:minmax(0,1.3fr) minmax(0,1fr);gap:24px}.status-panel{padding:24px}.status-actions{display:flex;gap:12px;flex-wrap:wrap;margin-top:24px}.ocr-preview{white-space:pre-wrap;overflow-wrap:anywhere}</style>