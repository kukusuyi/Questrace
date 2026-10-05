<template>
  <div class="page-shell" v-loading="loading">
    <header class="page-header">
      <div>
        <h2 class="page-title">错题详情</h2>
        <p class="page-subtitle">
          详情页回顾解题过程，找到下一次做对的关键。
        </p>
      </div>
      <div v-if="detail" class="header-actions">
        <RouterLink :to="`/questions/${detail.question_id}/edit`">
          <el-button>编辑</el-button>
        </RouterLink>
        <RouterLink :to="`/questions/${detail.question_id}/similar`">
          <el-button type="primary" plain>查找相似题</el-button>
        </RouterLink>
        <el-dropdown><el-button text>更多</el-button><template #dropdown><el-dropdown-menu><el-dropdown-item @click="removeQuestion">删除错题</el-dropdown-item></el-dropdown-menu></template></el-dropdown>
      </div>
    </header>

    <el-empty v-if="!loading && !detail" description="未找到这道错题" />

    <template v-else-if="detail">
      <section class="paper-card section-card">
        <div class="section-title">题目主干</div>
        <LatexRenderer :content="detail.question_core" allow-source-toggle />
      </section>
      <section class="summary-grid">


        <ImagePreviewer v-if="detail.source_image_url" :src="detail.source_image_url" />
      </section>



      <section class="content-split">
        <div class="paper-card section-card">
          <div class="section-title">标准解法</div>
          <LatexRenderer :content="detail.standard_solution" allow-source-toggle />
        </div>

        <div class="paper-card section-card">
          <div class="section-title">错误解法 / 错误思路</div>
          <LatexRenderer :content="detail.wrong_solution" allow-source-toggle />
        </div>
      </section>

      <section class="content-split">
        <div class="paper-card section-card">
          <div class="section-title">语义摘要</div>
          <div class="summary-text">{{ detail.semantic_summary || '暂无语义摘要' }}</div>
        </div>

        <div class="paper-card section-card">
          <div class="section-title">错因摘要</div>
          <div class="summary-text">{{ detail.mistake_summary || '暂无错因摘要' }}</div>
        </div>
      </section>

      <section class="paper-card section-card">
        <div class="section-title">标签分组</div>
        <TagGroup :tags="detail.tags" @tag-click="handleTagClick" />
      </section>

        <details class="paper-card summary-card"><summary>题目信息</summary>
          <div class="meta-text">题目元信息</div>
          <el-descriptions :column="1" border>
            <el-descriptions-item label="ID">{{ detail.question_id }}</el-descriptions-item>
            <el-descriptions-item label="学科"><ClassificationBadge :item="detail"/></el-descriptions-item>
            <el-descriptions-item label="章节">{{ detail.chapter || '--' }}</el-descriptions-item>
            <el-descriptions-item label="来源">{{ detail.source_type }}</el-descriptions-item>
            <el-descriptions-item label="掌握状态">{{ formatMasteryStatus(detail.mastery_status) }}</el-descriptions-item>
            <el-descriptions-item label="难度">{{ detail.difficulty_level }}</el-descriptions-item>
            <el-descriptions-item label="创建时间">
              {{ formatDateTime(detail.created_at) }}
            </el-descriptions-item>
            <el-descriptions-item label="更新时间">
              {{ formatDateTime(detail.updated_at) }}
            </el-descriptions-item>
          </el-descriptions>
        </details>
      <section class="paper-card section-card">
        <div class="section-title-row">
          <div class="section-title">相似题预览</div>
          <RouterLink :to="`/questions/${detail.question_id}/similar`">
            <el-button text>查看完整列表</el-button>
          </RouterLink>
        </div>
        <div class="similar-list">
          <SimilarQuestionCard
            v-for="item in similarList"
            :key="item.question_id"
            :item="item"
          />
          <el-empty v-if="!similarList.length" :description="similarMessage" />
        </div>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import ClassificationBadge from "@/components/ClassificationBadge/index.vue";
const props=defineProps<{questionId?:number}>();
import { ElMessage, ElMessageBox } from 'element-plus'
import { ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'

import ImagePreviewer from '@/components/ImagePreviewer/index.vue'
import LatexRenderer from '@/components/LatexRenderer/index.vue'
import SimilarQuestionCard from '@/components/SimilarQuestionCard/index.vue'
import TagGroup from '@/components/TagGroup/index.vue'
import { httpGet } from '@/api/http'
import { deleteQuestion, findSimilarQuestions, getQuestionDetail } from '@/api/question.api'
import type { QuestionDetail, SimilarQuestionItem } from '@/types/question'
import { getErrorMessage } from '@/utils/error'
import { formatDateTime, formatMasteryStatus } from '@/utils/format'

const route = useRoute()
const router = useRouter()
const loading = ref(false)
const detail = ref<QuestionDetail | null>(null)
const similarMessage = ref('暂未找到相似题')
const similarList = ref<SimilarQuestionItem[]>([])

function getQuestionID() {
  return props.questionId||Number(route.params.id)
}

function handleTagClick(payload: { type: string; name: string }) {
  router.push({
    path: '/questions',
    query: {
      subject_id:detail.value?.subject_id||undefined,course_id:detail.value?.course_id||undefined,
      tagName: payload.name,
      tagType: payload.type,
    },
  })
}

async function loadDetail() {
  detail.value = null
  similarList.value = []
  loading.value = true
  try {
    detail.value = await getQuestionDetail(getQuestionID())
    if(detail.value.classification_status!=='confirmed'){similarMessage.value='请先编辑并确认题目学科，再查找相似题';return}
    const status=await httpGet<{embedding_enabled:boolean}>('/api/v1/system/status')
    if(!status.embedding_enabled){similarMessage.value='相似题功能尚未配置，请管理员在设置中添加 Embedding 模型';return}
    similarMessage.value='暂未找到同学科相似题'
    const similarResponse = await findSimilarQuestions(getQuestionID(), {
      vector_type: 'semantic',
      limit: 3,
      use_tag_filter: false,
    })
    similarList.value = similarResponse.list
    if(similarResponse.status==='indexing')similarMessage.value='同学科题目的索引正在生成或等待重试，请稍后查看'
  } catch (error) {
    if(detail.value)similarMessage.value=getErrorMessage(error,'相似题加载失败');else ElMessage.error(getErrorMessage(error, '错题详情加载失败'))
  } finally {
    loading.value = false
  }
}

async function removeQuestion() {
  const current = detail.value
  if (!current) {
    return
  }

  try {
    await ElMessageBox.confirm(
      '删除后该错题将无法在列表中查看，关联向量也会被删除。是否继续？',
      '确认删除',
      { type: 'warning' },
    )

    await deleteQuestion(current.question_id)
    ElMessage.success('错题已删除')
    router.push('/questions')
  } catch (error) {
    if (error instanceof Error && error.message !== 'cancel') {
      ElMessage.error(getErrorMessage(error, '删除失败'))
    }
  }
}

watch(
  () => props.questionId||route.params.id,
  () => {
    void loadDetail()
  },
  { immediate: true },
)
</script>

<style scoped>
.header-actions {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}

.summary-grid,
.content-split {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 20px;
}

.summary-card,
.section-card {
  padding: 20px;
}

.section-title,
.section-title-row {
  margin-bottom: 14px;
  font-size: 20px;
  font-weight: 700;
}

.section-title-row {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: center;
}

.summary-text {
  min-height: 120px;
  line-height: 1.8;
  white-space: pre-wrap;
  color: var(--text-main);
}

.similar-list {
  display: grid;
  gap: 14px;
}

@media (max-width: 1080px) {
  .summary-grid,
  .content-split {
    grid-template-columns: 1fr;
  }
}
</style>
