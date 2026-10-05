<template>
  <article class="similar-card paper-panel">
    <ClassificationBadge :item="item"/>
    <div class="similar-head">
      <div>
        <div class="meta-text">相似类型 · {{ item.similarity_type }}</div>
        <h3>{{ truncateText(item.question_core, 120) }}</h3>
      </div>
      <div class="score-block">
        <span class="meta-text">相似度</span>
        <strong>{{ item.score.toFixed(2) }}</strong>
      </div>
    </div>

    <TagGroup :tags="item.tags" @tag-click="handleTagClick" />

    <div class="reason-block">
      <div class="meta-text">命中标签</div>
      <div>{{ item.matched_tags.length ? item.matched_tags.join(' / ') : '暂无命中标签说明' }}</div>
      <div class="meta-text reason">{{ item.reason || '后端暂未返回相似原因摘要。' }}</div>
    </div>

    <div class="actions">
      <RouterLink :to="`/questions/${item.question_id}`">
        <el-button type="primary" plain>查看详情</el-button>
      </RouterLink>
    </div>
  </article>
</template>

<script setup lang="ts">
import ClassificationBadge from "@/components/ClassificationBadge/index.vue";
import { useRouter } from 'vue-router'

import TagGroup from '@/components/TagGroup/index.vue'
import type { SimilarQuestionItem } from '@/types/question'
import { truncateText } from '@/utils/format'

const props = defineProps<{
  item: SimilarQuestionItem
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
.similar-card {
  display: grid;
  gap: 16px;
  padding: 18px;
}

.similar-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
}

.similar-head h3 {
  margin: 8px 0 0;
  font-size: 18px;
}

.score-block {
  display: grid;
  justify-items: end;
  min-width: 88px;
}

.score-block strong {
  font-size: 28px;
  color: var(--primary);
}

.reason-block {
  display: grid;
  gap: 8px;
}

.reason {
  line-height: 1.6;
}

.actions {
  display: flex;
  justify-content: flex-end;
}
</style>
