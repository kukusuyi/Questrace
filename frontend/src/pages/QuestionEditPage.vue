<template>
  <div class="page-shell" v-loading="loading">
    <header class="page-header">
      <div>
        <h2 class="page-title">编辑错题</h2>
        <p class="page-subtitle">
          完善题目、解法和标签。保存后会在后台更新相似题索引。
        </p>
      </div>
      <div class="edit-action-bar" v-if="draft">
        <el-button @click="goBack">返回详情</el-button>
        <el-button type="primary" :loading="saving" @click="saveChanges">保存修改</el-button>
      </div>
    </header>

    <QuestionForm
      v-if="draft"
      :model="draft"
      :show-analysis-fields="true"
      :tag-options="tagStore.groupedOptions"
    />

    <section v-if="draft" class="paper-card upload-card">
      <div class="upload-head">
        <h3>替换绑定图片</h3>
        <p class="page-subtitle">
          上传新图片替换原图，保存修改后生效。
        </p>
      </div>
      <UploadPanel
        :uploaded-image="{
          image_id: draft.source_image_id,
          image_url: draft.source_image_url,
        }"
        @success="handleImageUploaded"
      />
    </section>

    <el-empty v-else-if="!loading" description="未找到可编辑的错题数据" />
  </div>
</template>

<script setup lang="ts">
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, ref } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'

import QuestionForm from '@/components/QuestionForm/index.vue'
import UploadPanel from '@/components/UploadPanel/index.vue'
import { getQuestionDetail, updateQuestion } from '@/api/question.api'
import { useTagStore } from '@/stores/tag.store'
import type { UploadedImage } from '@/types/file'
import type { QuestionDraft, UpdateWrongQuestionPayload } from '@/types/question'
import { getErrorMessage } from '@/utils/error'

const route = useRoute()
const router = useRouter()
const draft = ref<QuestionDraft | null>(null)
const loading = ref(false)
const saving = ref(false)
let savedSnapshot = ""
onBeforeRouteLeave(async () => {
 if(!draft.value || JSON.stringify(draft.value)===savedSnapshot)return true
 try {await ElMessageBox.confirm("尚未保存修改，确认离开？", "放弃修改", {confirmButtonText:"放弃并离开",cancelButtonText:"继续编辑"});return true}catch{return false}
})
const tagStore = useTagStore()

function getQuestionID() {
  return Number(route.params.id)
}

function goBack() {
  router.push(`/questions/${getQuestionID()}`)
}

function handleImageUploaded(payload: UploadedImage) {
  if (!draft.value) {
    return
  }

  draft.value.source_image_id = payload.image_id
  draft.value.source_image_url = payload.image_url
  ElMessage.success('已替换当前绑定图片')
}

async function loadDetail() {
  loading.value = true
  try {
    const detail = await getQuestionDetail(getQuestionID())
    draft.value = {
      provider_name: '',
      model_name: '',
      source_type: detail.source_type,
      source_image_id: detail.source_image_id,
      source_image_url: detail.source_image_url,
      subject: detail.subject,
 subject_id:detail.subject_id,course_id:detail.course_id,classification_status:detail.classification_status,analysis_stale:detail.analysis_stale,revision:detail.revision,
      chapter: detail.chapter,
      chapter_locked: false,
      question_json: {
        question_core: detail.question_core,
        standard_solution: detail.standard_solution,
        wrong_solution: detail.wrong_solution,
      },
      tags: detail.tags,
      semantic_summary: detail.semantic_summary,
      mistake_summary: detail.mistake_summary,
      difficulty_level: detail.difficulty_level,
      mastery_status: detail.mastery_status,
      flow_mode: 'manual',
      status: 'saved',
    }
    savedSnapshot = JSON.stringify(draft.value)
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '错题编辑数据加载失败'))
  } finally {
    loading.value = false
  }
}

async function saveChanges() {
  if (!draft.value) {
    return
  }

  const payload: UpdateWrongQuestionPayload = {
    revision:draft.value.revision,
    question_json: draft.value.question_json,
    subject: draft.value.subject,
 subject_id:draft.value.subject_id,course_id:draft.value.course_id,classification_status:draft.value.classification_status,analysis_stale:draft.value.analysis_stale,analysis_confirmed:draft.value.analysis_confirmed,
    chapter: draft.value.chapter,
    tags: draft.value.tags,
    source_image_id: draft.value.source_image_id,
    source_image_url: draft.value.source_image_url,
    semantic_summary: draft.value.semantic_summary,
    mistake_summary: draft.value.mistake_summary,
    difficulty_level: draft.value.difficulty_level,
    mastery_status: draft.value.mastery_status,
  }

  saving.value = true
  try {
    await updateQuestion(getQuestionID(), payload)
    savedSnapshot = JSON.stringify(draft.value)
    ElMessage.success('错题更新成功')
    router.push(`/questions/${getQuestionID()}`)
  } catch (error) {
    ElMessage.error(getErrorMessage(error, '错题更新失败'))
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  try {
    await tagStore.fetchTags()
  } catch {
    // 标签候选加载失败不阻断编辑。
  }

  await loadDetail()
})
</script>

<style scoped>
.header-actions {
  display: flex;
  gap: 12px;
}

.upload-card {
  margin-top: 20px;
  padding: 20px;
}

.upload-head h3 {
  margin: 0;
}
</style>
