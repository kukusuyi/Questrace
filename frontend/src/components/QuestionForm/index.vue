<template>
    <div class="question-form-grid" :class="{ stacked }">
        <section class="paper-card form-card">
            <div class="section-head">
                <h3>题目基础信息</h3>
                <p>先写下题目，解法与分类可以稍后完善。</p>
            </div>

            <el-form label-position="top">
                <el-form-item label="题目主干">
                    <el-input
                        v-model="model.question_json.question_core"
                        type="textarea"
                        :autosize="{ minRows: 4, maxRows: 10 }"
                        placeholder="支持普通文本与 LaTeX 混排"
                    />
                </el-form-item>

                <el-form-item label="标准题解">
                    <el-input
                        v-model="model.question_json.standard_solution"
                        type="textarea"
                        :autosize="{ minRows: 4, maxRows: 10 }"
                        placeholder="可以为空"
                    />
                </el-form-item>

                <SolutionOCR v-model="model.question_json.standard_solution" />

                <el-form-item label="错误解法 / 错误思路">
                    <el-input
                        v-model="model.question_json.wrong_solution"
                        type="textarea"
                        :autosize="{ minRows: 4, maxRows: 10 }"
                        placeholder="可以为空，但建议保留学生原始错误过程"
                    />
                </el-form-item>

                <details class="secondary-fields" open><summary>分类与掌握状态</summary>
                <SubjectPicker :model="model" @change="emit('chapter-change', model.chapter)" />
                <p v-if="model.warnings?.length" role="alert">{{ model.warnings.join('；') }}</p>
                <el-checkbox v-if="showAnalysisFields && model.analysis_stale" v-model="model.analysis_confirmed">我已复核现有标签和摘要，适用于当前学科</el-checkbox>

                <div class="form-row">
                    <el-form-item label="来源类型" class="grow">
                        <el-select v-model="model.source_type" :disabled="lockSourceType">
                            <el-option label="手动录入" value="manual" />
                            <el-option label="图片识别" value="image" />
                            <el-option label="导入" value="import" />
                        </el-select>
                    </el-form-item>
                    <el-form-item label="掌握状态" class="grow">
                        <el-select v-model="model.mastery_status">
                            <el-option label="未掌握" value="unmastered" />
                            <el-option label="学习中" value="learning" />
                            <el-option label="已掌握" value="mastered" />
                        </el-select>
                    </el-form-item>
                </div>

                </details>
                <template v-if="showAnalysisFields">
                    <div class="section-divider"></div>

                    <div class="section-head compact">
                        <h3>标签与总结</h3>
                        <p>
                            标签与摘要都允许用户确认、修改，再保存到正式错题。
                        </p>
                    </div>

                    <el-form-item label="题目摘要">
                        <el-input
                            v-model="model.semantic_summary"
                            type="textarea"
                            :autosize="{ minRows: 3, maxRows: 8 }"
                        />
                    </el-form-item>

                    <el-form-item label="错因总结">
                        <el-input
                            v-model="model.mistake_summary"
                            type="textarea"
                            :autosize="{ minRows: 3, maxRows: 8 }"
                        />
                    </el-form-item>

                    <div class="form-row tags-row">
                        <el-form-item label="知识点标签" class="grow">
                            <el-select
                                v-model="model.tags.knowledge_points"
                                multiple
                                filterable
                                allow-create
                                default-first-option
                            >
                                <el-option
                                    v-for="item in scopedTagOptions.knowledge_points ||
                                    []"
                                    :key="item"
                                    :label="item"
                                    :value="item"
                                />
                            </el-select>
                        </el-form-item>

                        <el-form-item label="题型标签" class="grow">
                            <el-select
                                v-model="model.tags.problem_type"
                                multiple
                                filterable
                                allow-create
                                default-first-option
                            >
                                <el-option
                                    v-for="item in scopedTagOptions.problem_type ||
                                    []"
                                    :key="item"
                                    :label="item"
                                    :value="item"
                                />
                            </el-select>
                        </el-form-item>
                    </div>

                    <div class="form-row tags-row">
                        <el-form-item label="解法标签" class="grow">
                            <el-select
                                v-model="model.tags.method"
                                multiple
                                filterable
                                allow-create
                                default-first-option
                            >
                                <el-option
                                    v-for="item in scopedTagOptions.method || []"
                                    :key="item"
                                    :label="item"
                                    :value="item"
                                />
                            </el-select>
                        </el-form-item>

                        <el-form-item label="错因标签" class="grow">
                            <el-select
                                v-model="model.tags.mistake_reason"
                                multiple
                                filterable
                                allow-create
                                default-first-option
                            >
                                <el-option
                                    v-for="item in scopedTagOptions.mistake_reason ||
                                    []"
                                    :key="item"
                                    :label="item"
                                    :value="item"
                                />
                            </el-select>
                        </el-form-item>
                    </div>
                </template>
            </el-form>
        </section>

        <details class="preview-column"><summary>展开内容预览</summary>
            <div class="preview-stack">
                <div class="paper-card preview-card">
                    <div class="preview-head">题目预览</div>
                    <LatexRenderer
                        :content="model.question_json.question_core"
                        allow-source-toggle
                    />
                </div>

                <div class="paper-card preview-card">
                    <div class="preview-head">标准题解预览</div>
                    <LatexRenderer
                        :content="model.question_json.standard_solution"
                        allow-source-toggle
                    />
                </div>

                <div class="paper-card preview-card">
                    <div class="preview-head">错误解法预览</div>
                    <LatexRenderer
                        :content="model.question_json.wrong_solution"
                        allow-source-toggle
                    />
                </div>

                <div v-if="model.source_image_url" class="paper-card preview-card">
                    <div class="preview-head">原始图片预览</div>
                    <ImagePreviewer :src="model.source_image_url" />
                </div>
            </div>
        </details>
    </div>
</template>

<script setup lang="ts">
import {ref,watch} from 'vue'
import {listTags} from '@/api/tag.api'
import SubjectPicker from '@/components/SubjectPicker/index.vue'
import SolutionOCR from '@/components/SolutionOCR/index.vue'
import ImagePreviewer from "@/components/ImagePreviewer/index.vue";
import LatexRenderer from "@/components/LatexRenderer/index.vue";
import type { OptionItem } from "@/types/common";
import type { QuestionDraft } from "@/types/question";

const emit = defineEmits<{
    "chapter-change": [value: string];
}>();

const props=withDefaults(
    defineProps<{
        model: QuestionDraft;
        showAnalysisFields?: boolean;
        stacked?: boolean;
        tagOptions?: Partial<Record<keyof QuestionDraft["tags"], string[]>>;
        lockSourceType?: boolean;
        chapterOptions?: OptionItem[];
        chapterPlaceholder?: string;
    }>(),
    {
        showAnalysisFields: false,
        tagOptions: () => ({}),
        lockSourceType: false,
        chapterOptions: () => [],
        chapterPlaceholder: "例如：函数极限与连续",
    },
);
const scopedTagOptions=ref<Partial<Record<keyof QuestionDraft['tags'],string[]>>>({})
let tagRequest=0
watch(()=>[props.model.subject_id,props.model.course_id],async()=>{
 const request=++tagRequest;scopedTagOptions.value={};if(!props.model.subject_id)return
 try{const r=await listTags({subject_id:props.model.subject_id,course_id:props.model.course_id});if(request!==tagRequest)return
 const map:Record<string,keyof QuestionDraft['tags']>={knowledge_point:'knowledge_points',problem_type:'problem_type',method:'method',mistake_reason:'mistake_reason'}
 for(const tag of r.list){const key=map[tag.tag_type];if(key)(scopedTagOptions.value[key]??=[]).push(tag.tag_name)}
 }catch{/* 手动标签仍可编辑。 */}
},{immediate:true})
</script>

<style scoped>
.question-form-grid {
    display: grid;
    grid-template-columns: minmax(0, 1.2fr) minmax(0, 0.8fr);
    gap: 20px;
}

.question-form-grid.stacked { grid-template-columns:minmax(0,1fr); }
.question-form-grid > * { min-width:0; }
.form-card,
.preview-card {
    padding: 20px;
}

.section-head h3,
.preview-head {
    margin: 0;
    font-size: 20px;
}

.section-head p {
    margin: 8px 0 0;
    color: var(--text-secondary);
}

.section-head.compact {
    margin-bottom: 14px;
}

.form-row {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
}

.grow {
    width: 100%;
}

.section-divider {
    margin: 8px 0 18px;
    border-top: 1px dashed var(--line-strong);
}

.preview-column,
.preview-stack {
    display: grid;
    gap: 16px;
    align-content: start;
}

.preview-head {
    margin-bottom: 12px;
}

@media (max-width: 1080px) {
    .question-form-grid {
        grid-template-columns: 1fr;
    }
}

@media (max-width: 720px) {
    .form-row {
        grid-template-columns: 1fr;
    }
}

.preview-column>summary{cursor:pointer;min-height:44px;padding:12px 0;font-weight:600}
.secondary-fields summary{cursor:pointer;min-height:44px;padding:12px 0;font-weight:600}.secondary-fields[open] summary{margin-bottom:16px}
</style>
