<template>
    <div class="page-shell">
        <header class="page-header">
            <div>
                <h2 class="page-title">相似题检索</h2>
                <p class="page-subtitle">
                    支持语义向量和错因向量两种查询方式。
                </p>
            </div>
        </header>

        <section class="paper-card filter-card">
            <div class="filter-grid"><el-form-item v-if="detail?.subject_id==='cs408'" label="召回范围"><el-select v-model="recallScope"><el-option label="同一课程" value="course"/><el-option label="整个408" value="subject"/></el-select></el-form-item>
                <el-form-item label="向量类型">
                    <el-select v-model="vectorType">
                        <el-option label="语义相似" value="semantic" />
                        <el-option label="错因相似" value="mistake" />
                    </el-select>
                </el-form-item>
                <el-form-item label="召回数量">
                    <el-input-number v-model="limit" :min="1" :max="20" />
                </el-form-item>
                <el-form-item label="标签过滤">
                    <el-switch v-model="useTagFilter" />
                </el-form-item>
            </div>
            <div class="filter-actions">
                <el-button
                    type="primary"
                    :loading="loading"
                    @click="loadSimilar"
                    >重新检索</el-button
                >
            </div>
        </section>

        <section v-if="detail" class="paper-card current-card">
            <div class="meta-text">当前错题</div>
            <h3>{{ detail.question_core }}</h3>
        </section>

        <section class="similar-list">
            <SimilarQuestionCard
                v-for="item in list"
                :key="item.question_id"
                :item="item"
            />
            <el-empty
                v-if="!loading && !list.length"
                :description="message"
            />
        </section>
    </div>
</template>

<script setup lang="ts">
import { ElMessage } from "element-plus";
import { onMounted, ref } from "vue";
import { useRoute } from "vue-router";

import SimilarQuestionCard from "@/components/SimilarQuestionCard/index.vue";
import { findSimilarQuestions, getQuestionDetail } from "@/api/question.api";
import type { VectorType } from "@/types/common";
import type { QuestionDetail, SimilarQuestionItem } from "@/types/question";
import { getErrorMessage } from "@/utils/error";

const route = useRoute();
const detail = ref<QuestionDetail | null>(null);
const list = ref<SimilarQuestionItem[]>([]);
const vectorType = ref<VectorType>("semantic");
const recallScope=ref<"course"|"subject">("course");
const limit = ref(10);
const useTagFilter = ref(true);
const loading = ref(false);
const message=ref("暂未找到同学科相似题");

function getQuestionID() {
    return Number(route.params.id);
}

async function loadSimilar() {
    loading.value = true;
    try {
        const response = await findSimilarQuestions(getQuestionID(), {
                recall_scope:recallScope.value,
                vector_type: vectorType.value,
                limit: limit.value,
                use_tag_filter: useTagFilter.value,
            });
        list.value=response.list;message.value=response.status==='indexing'?'同学科索引正在生成或等待重试':'暂未找到同学科相似题';
    } catch (error) {
        list.value=[];message.value=getErrorMessage(error,"相似题加载失败");
    } finally {
        loading.value = false;
    }
}

onMounted(async () => {
    try {
        detail.value = await getQuestionDetail(getQuestionID());
    } catch {
        // 如果详情失败，检索页仍尝试继续显示检索结果。
    }

    await loadSimilar();
});
</script>

<style scoped>
.filter-card,
.current-card {
    padding: 20px;
}

.filter-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 240px));
    gap: 16px;
}

.filter-actions {
    margin-top: 8px;
}

.current-card h3 {
    margin: 10px 0 0;
    line-height: 1.7;
}

.similar-list {
    display: grid;
    gap: 14px;
}

@media (max-width: 900px) {
    .filter-grid {
        grid-template-columns: 1fr;
    }
}
</style>
