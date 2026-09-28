<template>
  <article class="question-card paper-card">
    <div class="card-top">
      <div>
        <div class="card-meta">
          <ClassificationBadge :item="item"/>
          <span class="difficulty">· 难度 {{ item.difficulty_level }}</span>
        </div>
        <div class="card-title"><LatexRenderer :content="item.question_core"/></div>
      </div>
      <div class="card-side">
        <el-checkbox v-if="selectable"
          :model-value="selected"
          class="select-box" :aria-label="`选择题目 ${item.question_id}`"
          @change="emit('toggle-select')"
        />
        <div class="card-badges">
        <span class="tag-pill badge">{{ formatMasteryStatus(item.mastery_status) }}</span>
        <span v-if="item.source_image_url" class="tag-pill badge">有原图</span>
        </div>
      </div>
    </div>

    <TagGroup :tags="item.tags" @tag-click="handleTagClick" />

    <div class="card-footer">
      <span class="meta-text">创建于 {{ formatDateTime(item.created_at) }}</span>
      <div class="card-actions">
        <el-button v-if="preview" text @click="emit('open')">详情</el-button><RouterLink v-else :to="`/questions/${item.question_id}`">
          <el-button text>详情</el-button>
        </RouterLink>
        <RouterLink :to="`/questions/${item.question_id}/edit`">
          <el-button text>编辑</el-button>
        </RouterLink>
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import ClassificationBadge from "@/components/ClassificationBadge/index.vue";
import { useRouter } from 'vue-router'

import LatexRenderer from '@/components/LatexRenderer/index.vue'
import TagGroup from '@/components/TagGroup/index.vue'
import type { QuestionListItem } from '@/types/question'
import { formatDateTime, formatMasteryStatus } from '@/utils/format'

const props = defineProps<{
  item: QuestionListItem
  selectable?: boolean
  selected?: boolean
  preview?:boolean
}>()

const emit = defineEmits<{
  (event: 'toggle-select'): void
  (event: 'open'): void
}>()

const router = useRouter()

function handleTagClick(payload: { type: string; name: string }) {
  router.push({
    path: '/questions',
    query: {
      subject_id: props.item.subject_id || undefined,
      course_id: props.item.course_id || undefined,
      tagType: payload.type,
      tagName: payload.name,
    },
  })
}
</script>

<style scoped>
.question-card {
  display: grid;
  gap: 16px;
  padding: 20px;
}

.card-top {
  display: flex;
  justify-content: space-between;
  gap: 16px;
}

.card-meta {
  color: var(--text-secondary);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.card-title {
  margin: 10px 0 0;
  font-size: 18px;
  line-height: 1.5;
}

.card-side,
.card-badges {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.card-side {
  align-items: flex-end;
  flex-direction: column;
}

.select-box {
  margin-right: -6px;
}

.badge {
  background: var(--line);
  border-color: var(--line);
}

.card-footer {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
}

.card-actions {
  display: flex;
  gap: 6px;
}
.card-top>div:first-child{min-width:0;flex:1}.card-title :deep(.latex-renderer){border:0;background:transparent}.card-title :deep(.renderer-preview){padding:0;min-height:0;max-height:144px;overflow:auto;font-weight:600}.card-title :deep(.latex-content){line-height:1.6}
.difficulty { white-space: nowrap; }
@media (max-width: 640px) {
  .card-top { flex-wrap: wrap; }
  .card-top > div:first-child { flex-basis: 100%; }
  .card-side { width: 100%; flex-direction: row; justify-content: space-between; align-items: center; }
}
</style>
