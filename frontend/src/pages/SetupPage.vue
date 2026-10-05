<template>
  <main class="page-shell" style="max-width:560px;margin:48px auto;padding:0 16px">
    <section class="paper-card" style="padding:28px">
      <h1>欢迎使用题迹 2.0</h1><p>数据保存在这台电脑上。创建管理员账户后，就可以连接手机使用。</p>
      <el-alert v-if="error" :title="error" type="error" :closable="false" role="alert" />
      <el-form label-position="top" @submit.prevent="submit">
        <el-form-item label="初始化凭据"><el-input v-model="form.token" type="password" show-password placeholder="桌面程序自动填写；Linux 请查看启动输出" /></el-form-item>
        <el-form-item label="管理员用户名"><el-input v-model="form.username" autocomplete="username" /></el-form-item>
        <el-form-item label="邮箱"><el-input v-model="form.email" type="email" /></el-form-item>
        <el-form-item label="密码（至少 8 位）"><el-input v-model="form.password" type="password" show-password autocomplete="new-password" /></el-form-item>
        <EducationStagePicker v-model="form.education_stage"/>
        <el-button native-type="submit" type="primary" :loading="busy">创建管理员</el-button>
      </el-form>
    </section>
  </main>
</template>
<script setup lang="ts">
import EducationStagePicker from '@/components/EducationStagePicker/index.vue'
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { httpPost } from '@/api/http'
const router=useRouter()
const form=reactive({token:new URLSearchParams(location.hash.slice(1)).get('token')||'',username:'',email:'',password:'',education_stage:''})
if(location.hash) history.replaceState(null,'',location.pathname)
const busy=ref(false),error=ref('')
async function submit(){error.value='';if(!form.education_stage){error.value='请选择当前学习阶段：大学或高中';return}if(!form.token||!form.username||!form.email||form.password.length<8){error.value='请填写所有字段，密码至少 8 位';return}busy.value=true;try{await httpPost('/api/v1/system/setup',form);await router.replace('/auth')}catch(e){error.value=(e as Error).message}finally{busy.value=false}}
</script>
