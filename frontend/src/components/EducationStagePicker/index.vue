<template>
  <el-form-item label="当前学习阶段" required>
    <el-radio-group :model-value="modelValue" aria-label="当前学习阶段" @update:model-value="emit('update:modelValue', String($event))">
      <el-radio-button value="university">大学</el-radio-button>
      <el-radio-button value="highschool">高中</el-radio-button>
    </el-radio-group>
    <p class="stage-hint">按阶段推荐学科，任何专业科目都可以添加。之后可在账户中修改。</p>
  </el-form-item>
</template>
<script setup lang="ts">
defineProps<{modelValue:string}>()
const emit=defineEmits<{'update:modelValue':[string]}>()
</script>
<style scoped>.stage-hint{width:100%;margin:8px 0 0;color:var(--text-secondary);font-size:13px;line-height:1.6}</style>
