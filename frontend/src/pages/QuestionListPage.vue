<template>
    <div class="page-shell">
        <header class="page-header">
            <div>
                <h2 class="page-title">错题列表</h2>
                <p class="page-subtitle">
                    整理每一次思考，找到值得再练的一道题。
                </p>
            </div>
            <div class="header-actions"><RouterLink to="/reviews"><el-button>复习与组卷</el-button></RouterLink>
                <el-radio-group v-if="!isMobile"
                    :model-value="questionStore.preferredListView"
                    @update:model-value="questionStore.setPreferredListView"
                >
                    <el-radio-button label="card">卡片视图</el-radio-button>
                    <el-radio-button label="table">表格视图</el-radio-button>
                </el-radio-group>
            </div>
        </header>

        <div v-if="isMobile" class="mobile-search"><el-input v-model="filters.keyword" placeholder="搜索题目与标签" aria-label="搜索题目与标签" clearable @keyup.enter="applyFilters"/><el-button @click="applyFilters">搜索</el-button><el-button @click="filterOpen=true">筛选</el-button></div>
        <div class="filter-chips"><template v-for="(label,key) in filterLabels" :key="key"><el-tag v-if="filters[key]" closable @close="clearFilter(key)">{{label}}：{{filterDisplay(key,filters[key])}}</el-tag></template><el-tag v-if="activeTagHint" closable @close="clearTagFilters">{{activeTagHint}}</el-tag></div>
        <component :is="isMobile?ElDrawer:'section'" v-model="filterOpen" title="筛选错题" direction="btt" size="auto" class="paper-card filter-card">
            <el-form label-position="top">
                <div class="filter-grid">
                    <el-form-item v-if="!isMobile" label="关键词">
                        <el-input
                            v-model="filters.keyword"
                            placeholder="搜索题目主干、标签等关键信息"
                        />
                    </el-form-item>
                    <el-form-item label="标签"><TagFilter v-model="selectedTags" :subject-id="filters.subject_id" :course-id="filters.course_id"/></el-form-item>
<SubjectPicker :model="filters" filter @change="classificationChanged"/>
<el-form-item label="分类状态"><el-select v-model="filters.classification_status" placeholder="全部分类状态" clearable><el-option label="历史分类待复核" value="legacy_pending"/><el-option label="待分类" value="pending"/><el-option label="已分类" value="confirmed"/></el-select></el-form-item>
                    <el-form-item label="掌握状态">
                        <el-select v-model="filters.mastery_status" placeholder="全部掌握状态" clearable>
                            <el-option label="未掌握" value="unmastered" />
                            <el-option label="学习中" value="learning" />
                            <el-option label="已掌握" value="mastered" />
                        </el-select>
                    </el-form-item>
                    <el-form-item label="来源类型">
                        <el-select v-model="filters.source_type" placeholder="全部来源" clearable>
                            <el-option label="手动录入" value="manual" />
                            <el-option label="图片识别" value="image" />
                            <el-option label="导入" value="import" />
                        </el-select>
                    </el-form-item>
                </div>

                <div class="filter-actions">
                    <span v-if="activeTagHint" class="meta-text">
                        当前通过标签筛选：{{ activeTagHint }}
                    </span>
                    <div class="grow"></div>
                    <el-button @click="resetFilters">重置</el-button>
                    <el-button type="primary" @click="applyFilters"
                        >查询</el-button
                    >
                </div>
            </el-form>
        </component>

        <div :class="{'question-workspace':wide&&previewID}"><section class="paper-card list-card">
            <div class="list-toolbar">
                <div class="toolbar-left">
                    <div class="meta-text">共 {{ total }} 条错题</div>
                    <div v-if="selectedQuestionIds.length" class="selection-summary">
                        <span class="meta-text">已选 {{ selectedQuestionIds.length }} 题</span>
                        <el-button @click="batchOpen=true">复核／修改分类</el-button><el-button text @click="clearSelection">清空已选</el-button>
                    </div>
                </div>
                <div class="toolbar-actions">
                    <el-button type="primary" plain :disabled="!selectedQuestionIds.length" @click="exportSelectedQuestions('questions_only')">导出 PDF · 每页两题</el-button>

                </div>
            </div>

            <div v-if="loading" class="loading-block">
                <el-skeleton :rows="6" animated />
            </div>

            <template v-else>
                <div
                    v-if="isMobile || questionStore.preferredListView === 'card'"
                    class="card-grid"
                >
                    <QuestionCard selectable
                        v-for="item in list"
                        :key="item.question_id"
                        :item="item" :preview="wide" @open="openQuestion(item.question_id)"
                        :selected="isQuestionSelected(item.question_id)"
                        @toggle-select="toggleSelection(item.question_id)"
                    />
                </div>

                <el-table v-else :data="list" class="question-table">
                    <el-table-column label="选择" width="78">
                        <template #default="{ row }">
                            <el-checkbox
                                :model-value="isQuestionSelected(row.question_id)"
                                @change="toggleSelection(row.question_id)"
                            />
                        </template>
                    </el-table-column>
                    <el-table-column prop="question_id" label="ID" width="88" />
                    <el-table-column label="题目">
                        <template #default="{ row }">
                            {{ truncateText(row.question_core, 72) }}
                        </template>
                    </el-table-column>
                    <el-table-column label="分类" min-width="200"><template #default="{row}"><ClassificationBadge :item="row"/></template></el-table-column>
                    <el-table-column label="掌握状态" width="120">
                        <template #default="{ row }">
                            {{ formatMasteryStatus(row.mastery_status) }}
                        </template>
                    </el-table-column>
                    <el-table-column
                        prop="created_at"
                        label="创建时间"
                        width="180"
                    >
                        <template #default="{ row }">
                            {{ formatDateTime(row.created_at) }}
                        </template>
                    </el-table-column>
                    <el-table-column label="操作" width="180">
                        <template #default="{ row }">
                            <RouterLink :to="`/questions/${row.question_id}`">
                                <el-button text>详情</el-button>
                            </RouterLink>
                            <RouterLink
                                :to="`/questions/${row.question_id}/edit`"
                            >
                                <el-button text>编辑</el-button>
                            </RouterLink>
                        </template>
                    </el-table-column>
                </el-table>

                <el-empty
                    v-if="!list.length"
                    description="暂无符合条件的错题"
                />

                <div class="pagination-row">
                    <el-pagination
                        background
                        layout="prev, pager, next"
                        :total="total"
                        :page-size="filters.page_size"
                        :current-page="filters.page"
                        @current-change="handlePageChange"
                    />
                </div>
            </template>
        </section><aside v-if="wide&&previewID" class="detail-pane"><el-button text @click="router.replace({query:{...route.query,preview:undefined}})">关闭预览</el-button><QuestionDetailPage :question-id="previewID"/></aside></div>
 <el-dialog v-model="batchOpen" title="批量复核分类" width="min(600px, 95vw)"><p>将为当前选中的 {{selectedQuestionIds.length}} 道题确认以下分类。原标签和摘要保留供复核。</p><el-form label-position="top"><SubjectPicker :model="batchTarget"/></el-form><template #footer><el-button @click="batchOpen=false">取消</el-button><el-button type="primary" :disabled="!batchTarget.subject_id" :loading="batchBusy" @click="applyClassification">确认修改 {{selectedQuestionIds.length}} 道题</el-button></template></el-dialog>
    </div>
</template>

<script setup lang="ts">
import SubjectPicker from '@/components/SubjectPicker/index.vue'
import ClassificationBadge from '@/components/ClassificationBadge/index.vue'
import {httpPost} from '@/api/http'
import {type Classification,subjectNames,courseNames,useSubjectsStore} from '@/stores/subjects.store'

import TagFilter from '@/components/TagFilter/index.vue';
import {useViewport} from "@/composables/useViewport";
import QuestionDetailPage from "@/pages/QuestionDetailPage.vue";
import { ElMessage, ElDrawer } from "element-plus";
import { computed, onMounted, reactive, ref, watch } from "vue";
import { RouterLink, useRoute, useRouter } from "vue-router";

import QuestionCard from "@/components/QuestionCard/index.vue";
import {
    buildQuestionExportPrintURL,
    type QuestionExportMode,
    listQuestions,
} from "@/api/question.api";
import { listTags } from "@/api/tag.api";
import { useQuestionStore } from "@/stores/question.store";
import type { MasteryStatus, SourceType } from "@/types/common";
import type { ListQuestionFilter, QuestionListItem } from "@/types/question";
import { getErrorMessage } from "@/utils/error";
import {
    formatDateTime,
    formatMasteryStatus,
    truncateText,
} from "@/utils/format";

const route = useRoute();
const router = useRouter();
const questionStore = useQuestionStore();
const subjectsStore=useSubjectsStore();
const loading = ref(false);
const isMobile=useViewport(), wide=useViewport('(min-width: 1280px)'),filterOpen=ref(false);
const previewID=computed(()=>Number(route.query.preview)||undefined);
const filterLabels={subject_id:'学科',course_id:'课程',chapter:'章节',classification_status:'分类状态',mastery_status:'掌握状态',source_type:'来源'} as const;
function openQuestion(id:number){if(wide.value)router.push({path:'/questions',query:{...route.query,preview:String(id)}});else router.push(`/questions/${id}`)}
function clearTagFilters(){selectedTags.value=[];filters.tag_ids='';activeTagHint.value='';applyFilters()}
function clearFilter(key:keyof typeof filterLabels){filters[key]='';if(key==='subject_id'){filters.course_id='';filters.chapter='';selectedTags.value=[]}if(key==='course_id'){filters.chapter='';selectedTags.value=[]}applyFilters()}
function applyFilters(){clearSelection();filterOpen.value=false;router.push({path:'/questions',query:{keyword:filters.keyword||undefined,subject_id:filters.subject_id||undefined,course_id:filters.course_id||undefined,classification_status:filters.classification_status||undefined,chapter:filters.chapter||undefined,mastery_status:filters.mastery_status||undefined,source_type:filters.source_type||undefined,tag_ids:selectedTags.value.join(',')||undefined,page:'1'}})}

const selectedTags=ref<number[]>([]);
const total = ref(0);
const list = ref<QuestionListItem[]>([]);
const activeTagHint = ref("");
const selectedQuestionIds = ref<number[]>([]);
const selectedRevisions=ref<Record<number,number>>({});
const allowedMasteryStatus: MasteryStatus[] = ["unmastered", "learning", "mastered"];
const allowedSourceType: SourceType[] = ["manual", "image", "import"];

const filters = reactive<
    Required<Pick<ListQuestionFilter, "page" | "page_size">> & {
        keyword: string;
        subject: string;
        subject_id:string;course_id:string;classification_status:string;
        chapter: string;
        mastery_status: MasteryStatus | "";
        source_type: SourceType | "";
        tag_ids: string;
    }
>({
    page: 1,
    page_size: 10,
    keyword: "",
    subject: "",subject_id:"",course_id:"",classification_status:"",
    chapter: "",
    mastery_status: "",
    source_type: "",
    tag_ids: "",
});

const batchOpen=ref(false),batchBusy=ref(false),batchTarget=reactive<Classification>({subject_id:'',course_id:'',chapter:''})
function classificationChanged(){filters.subject='';selectedTags.value=[];filters.tag_ids='';activeTagHint.value='';filters.page=1;clearSelection();if(previewID.value)router.replace({query:{...route.query,preview:undefined}})}
function filterDisplay(key:string,value:string){if(key==='subject_id')return subjectsStore.items.find(s=>s.id===value)?.name||subjectNames[value]||value;if(key==='course_id')return courseNames[value]||value;if(key==='classification_status')return ({legacy_pending:'历史分类待复核',pending:'待分类',confirmed:'已分类'} as Record<string,string>)[value]||value;return value}
async function applyClassification(){batchBusy.value=true;try{await httpPost('/api/v1/wrong-questions/classification',{subject_id:batchTarget.subject_id,course_id:batchTarget.course_id,chapter:batchTarget.chapter,items:selectedQuestionIds.value.map(id=>({question_id:id,revision:selectedRevisions.value[id]}))});batchOpen.value=false;clearSelection();await loadQuestions();ElMessage.success('分类已更新，相似题索引将在后台生成')}catch(e){ElMessage.error(getErrorMessage(e,'分类更新失败'))}finally{batchBusy.value=false}}

const routeTagName = computed(() => {
    const value = route.query.tagName;
    return typeof value === "string" ? value : "";
});

const routeTagType = computed(() => {
    const value = route.query.tagType;
    return typeof value === "string" ? value : "";
});

function normalizeMasteryStatus(value: unknown): MasteryStatus | "" {
    return typeof value === "string" && allowedMasteryStatus.includes(value as MasteryStatus)
        ? (value as MasteryStatus)
        : "";
}

function normalizeSourceType(value: unknown): SourceType | "" {
    return typeof value === "string" && allowedSourceType.includes(value as SourceType)
        ? (value as SourceType)
        : "";
}

function syncFiltersFromRoute() {
 filters.subject_id=String(route.query.subject_id||'');filters.course_id=String(route.query.course_id||'');filters.classification_status=String(route.query.classification_status||'');
 selectedTags.value=String(route.query.tag_ids||'').split(',').map(Number).filter(n=>n>0);
    filters.page=Math.max(1,Number(route.query.page)||1);
    filters.keyword =
        typeof route.query.keyword === "string" ? route.query.keyword : "";
    filters.subject =
        typeof route.query.subject === "string" ? route.query.subject : "";
    filters.chapter =
        typeof route.query.chapter === "string" ? route.query.chapter : "";
    filters.mastery_status = normalizeMasteryStatus(route.query.mastery_status);
    filters.source_type = normalizeSourceType(route.query.source_type);
}

async function syncTagFilterFromRoute() {
 if(route.query.tag_ids){filters.tag_ids=String(route.query.tag_ids);activeTagHint.value=`已选 ${selectedTags.value.length} 个标签`;return;}
    if (!routeTagName.value) {
        filters.tag_ids = "";
        activeTagHint.value = "";
        return;
    }

    const tagTypeMap: Record<string, string> = {
        knowledge_points: "knowledge_point",
        knowledge_point: "knowledge_point",
        problem_type: "problem_type",
        method: "method",
        mistake_reason: "mistake_reason",
    };

    const response = await listTags({
        keyword: routeTagName.value,
        tag_type: tagTypeMap[routeTagType.value] || "",
    });

    const matched = response.list.find(
        (item) => item.tag_name === routeTagName.value,
    );
    filters.tag_ids = matched ? String(matched.tag_id) : "-1";
    selectedTags.value=matched?[matched.tag_id]:[];
    activeTagHint.value = matched
        ? routeTagName.value
        : "标签已不存在";
}

let loadGeneration=0;
async function loadQuestions() {
 const generation=++loadGeneration;
    loading.value = true;

    try {
        await syncTagFilterFromRoute();
        const response = await listQuestions({
            ...filters,
            mastery_status: filters.mastery_status || undefined,
            source_type: filters.source_type || undefined,
            tag_ids: filters.tag_ids || undefined,
            chapter: filters.chapter || undefined,
        });

        if(generation!==loadGeneration)return;
        list.value = response.list;
        total.value = response.total;
        questionStore.setRecentQuestions(response.list.slice(0, 4));
    } catch (error) {
        ElMessage.error(getErrorMessage(error, "错题列表加载失败"));
    } finally {
        if(generation===loadGeneration)loading.value = false;
    }
}

function isQuestionSelected(questionID: number) {
    return selectedQuestionIds.value.includes(questionID);
}

function toggleSelection(questionID: number) {
    if (isQuestionSelected(questionID)) {
        selectedQuestionIds.value = selectedQuestionIds.value.filter((item) => item !== questionID);
        return;
    }

    selectedQuestionIds.value = [...selectedQuestionIds.value, questionID];
    selectedRevisions.value[questionID]=list.value.find(x=>x.question_id===questionID)?.revision||0;
}

function clearSelection() {
    selectedQuestionIds.value = [];selectedRevisions.value={};
}

function exportSelectedQuestions(exportMode: QuestionExportMode) {
    if (!selectedQuestionIds.value.length) {
        ElMessage.warning("请先选择要导出的错题");
        return;
    }

    const exportURL = buildQuestionExportPrintURL(selectedQuestionIds.value, exportMode);
    window.open(exportURL, "_blank", "noopener,noreferrer");
}

async function resetFilters() {
    filters.page = 1;
    filters.keyword = "";
    filters.subject = "";filters.subject_id="";filters.course_id="";filters.classification_status="";clearSelection();
    filters.chapter = "";
    filters.mastery_status = "";
    filters.source_type = "";
    filters.tag_ids = "";
    selectedTags.value=[];
    activeTagHint.value = "";

    if (Object.keys(route.query).length) {
        await router.push({ path: "/questions", query: {} });
        return;
    }

    loadQuestions();
}

function handlePageChange(page: number) {
 clearSelection();
    filters.page = page;
    router.push({query:{...route.query,page:String(page)}});
}

onMounted(() => {
    syncFiltersFromRoute();
    loadQuestions();
});

watch(
    () => {const {preview,...query}=route.query;return JSON.stringify(query)},
    () => {
        clearSelection();
        syncFiltersFromRoute();
        loadQuestions();
    },
);
</script>

<style scoped>
.mobile-search{display:flex;gap:8px}.filter-chips{display:flex;gap:8px;flex-wrap:wrap}.filter-chips:empty{display:none}.question-workspace{display:grid;grid-template-columns:minmax(300px,.9fr) minmax(0,1.3fr);gap:24px;align-items:start}.question-workspace .card-grid{grid-template-columns:1fr}.detail-pane{min-width:0;padding:8px}.question-workspace .detail-pane :deep(.summary-grid),.question-workspace .detail-pane :deep(.content-split){grid-template-columns:1fr}
.header-actions {
    display: flex;
    gap: 12px;
}

.filter-card,
.list-card {
    padding: 20px;
}

.filter-grid :deep(.classification-fields){grid-column:1/-1}
.filter-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 16px;
}

.filter-actions,
.list-toolbar,
.pagination-row {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
}

.filter-actions {
    margin-top: 8px;
}

.grow {
    flex: 1;
}

.list-toolbar {
    justify-content: space-between;
    margin-bottom: 16px;
}

.toolbar-left,
.toolbar-actions,
.selection-summary {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-wrap: wrap;
}

.card-grid {
    display: grid;
    gap: 16px;
}

.pagination-row {
    justify-content: flex-end;
    margin-top: 18px;
}

@media (max-width: 1100px) {
    .filter-grid :deep(.classification-fields){grid-column:1/-1}
.filter-grid {
        grid-template-columns: repeat(2, minmax(0, 1fr));
    }
}

@media (max-width: 720px) {
    .filter-grid :deep(.classification-fields){grid-column:1/-1}
.filter-grid {
        grid-template-columns: 1fr;
    }
}
</style>
