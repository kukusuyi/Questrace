<template><span class="classification"><span>{{classificationLabel(item)}}</span><el-tag v-if="item.classification_status==='legacy_pending'" type="warning" size="small">历史分类待复核</el-tag><el-tag v-else-if="item.classification_status!=='confirmed'" type="info" size="small">待分类</el-tag><el-tag v-if="item.analysis_stale" type="info" size="small">分析待更新</el-tag></span></template>
<script setup lang="ts">import {classificationLabel,type Classification} from '@/stores/subjects.store';defineProps<{item:Classification}>();
</script>
<style scoped>.classification{display:inline-flex;align-items:center;flex-wrap:wrap;gap:6px;overflow-wrap:anywhere}</style>
