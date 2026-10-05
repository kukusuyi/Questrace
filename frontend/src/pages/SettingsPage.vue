<template>
<div class="page-shell settings-page">
 <header class="page-header"><div><h2 class="page-title">设置与账户</h2><p class="page-subtitle">让题迹适合你的学习习惯。</p></div></header>
 <el-alert v-if="error" :title="error" type="error" :closable="false" role="alert"/>
 <el-tabs v-model="active" class="settings-tabs">
  <el-tab-pane label="个人信息" name="account"><section class="paper-card panel"><h3>个人信息</h3><template v-if="profileStore.profile"><p>{{profileStore.profile.username}} · {{profileStore.profile.email}}</p><el-form label-position="top"><EducationStagePicker v-model="educationStage"/></el-form><p class="meta-text">当前阶段：{{profileStore.profile.education_stage==='highschool'?'高中':'大学'}}。更改阶段不会修改已有题目的分类。</p><el-button type="primary" :loading="stageBusy" :disabled="educationStage===profileStore.profile.education_stage" @click="saveStage">保存学习阶段</el-button></template><el-button v-else @click="loadProfile">加载个人信息</el-button></section></el-tab-pane>
  <el-tab-pane label="外观" name="appearance"><AppearancePanel/></el-tab-pane>
  <el-tab-pane label="电脑连接" name="connection"><section class="paper-card panel"><h3>连接这台电脑</h3><p class="meta-text">手机与电脑连接同一网络，优先使用固定域名地址，换网后可自动解析。</p><p v-if="status.discovery?.state === 'unavailable'" role="status">自动发现暂不可用，请使用下方 IP 地址连接，并检查局域网与防火墙权限。</p><div v-for="url in status.urls" :key="url" class="address"><code>{{url}}</code><el-button @click="copy(url)">复制地址</el-button></div><el-alert title="127.0.0.1 仅供本机访问。连接失败时，请确认电脑正在运行并允许防火墙的局域网连接。" type="info" :closable="false"/></section></el-tab-pane>
  <el-tab-pane v-if="settings" label="模型服务" name="models"><section class="paper-card panel"><h3>模型服务</h3><div class="settings-actions"><a href="https://account.aliyun.com/register/qr_register.htm" target="_blank" rel="noopener noreferrer">注册阿里云账号 ↗</a><a href="https://bailian.console.aliyun.com/" target="_blank" rel="noopener noreferrer">开通 Qwen / 获取 API Key ↗</a><a href="https://help.aliyun.com/zh/model-studio/first-api-call-to-qwen" target="_blank" rel="noopener noreferrer">配置指南 ↗</a></div><p class="student-link"><a href="https://www.aliyun.com/activity/ecs/campus-deal" target="_blank" rel="noopener noreferrer">高校学生可免费领取阿里云 300 元优惠券 ↗</a></p><p class="meta-text">按需配置。没有模型也能录题、上传图片和复习。</p><el-form label-position="top">
   <el-collapse><el-collapse-item title="OCR · 图片识别" name="ocr"><el-form-item label="API Key"><SecretField v-model="settings.ocr.api_key"/></el-form-item><ModelConfigField kind="ocr" v-model="settings.ocr.model" :config="settings.ocr"/></el-collapse-item>
   <el-collapse-item title="AI · 分析模型" name="ai"><div v-for="(m,i) in settings.models" :key="i" class="model"><el-form-item label="服务名称"><el-input v-model="m.name"/></el-form-item><el-form-item label="接口地址"><el-input v-model="m.base_url" placeholder="https://example.com/v1"/></el-form-item><el-form-item label="API Key"><SecretField v-model="m.api_key"/></el-form-item><ModelConfigField kind="analysis" v-model="m.model" :config="m" :saved-name="savedNames.get(m)"/><el-button type="danger" text @click="settings.models.splice(i,1)">移除模型</el-button></div><el-button @click="settings.models.push({name:'qwen'+(settings.models.length?'-'+(settings.models.length+1):''),provider_type:'qwen',base_url:'https://dashscope.aliyuncs.com/compatible-mode/v1',model:'qwen3.8-flash',api_key:''})">添加分析模型</el-button></el-collapse-item>
   <el-collapse-item title="Embedding · 相似题" name="embedding"><el-form-item label="接口地址"><el-input v-model="settings.embedding.base_url"/></el-form-item><el-form-item label="模型名称"><el-input v-model="settings.embedding.model"/></el-form-item><el-form-item label="API Key"><SecretField v-model="settings.embedding.api_key"/></el-form-item><el-alert title="更换模型或地址会重新生成索引，可能产生服务商调用费用。" type="info" :closable="false"/></el-collapse-item></el-collapse>
   <div class="settings-actions"><el-button :loading="busy" @click="test">测试连接</el-button><el-button type="primary" :loading="busy" @click="save">保存设置</el-button></div><p v-for="(value,key) in results" :key="key" role="status">{{key}}：{{value}}</p>
  </el-form></section></el-tab-pane>
  <el-tab-pane label="索引任务" name="jobs"><section class="paper-card panel"><h3>相似题索引</h3><p>{{status.embedding_enabled?'后台生成索引，不影响保存与查看错题。':'尚未配置模型，手动录题和图片查看仍可使用。'}}</p><div class="job-stats"><div><strong>{{jobs.pending||0}}</strong><span>等待处理</span></div><div><strong>{{jobs.done||0}}</strong><span>已经完成</span></div><div><strong>{{jobs.failed||0}}</strong><span>待重试</span></div></div><div class="settings-actions"><el-button :loading="busy" :disabled="!jobs.failed" @click="retry">重试失败任务</el-button><el-button @click="load">刷新</el-button></div></section></el-tab-pane>
  <el-tab-pane v-if="settings" label="用户管理" name="users"><section class="paper-card panel"><h3>账户与注册</h3><el-switch v-model="settings.registration_enabled" active-text="允许公开注册"/><el-button class="registration-save" :loading="busy" @click="save">保存</el-button><div class="user-list"><article v-for="u in users" :key="u.id"><strong>{{u.username}}</strong><span>{{u.email}}</span><el-tag>{{u.role==='admin'?'管理员':'成员'}}</el-tag></article></div><el-collapse><el-collapse-item title="创建新账户"><el-form label-position="top" @submit.prevent="createUser"><el-form-item label="用户名"><el-input v-model="user.username"/></el-form-item><el-form-item label="邮箱"><el-input v-model="user.email" type="email"/></el-form-item><el-form-item label="密码"><el-input v-model="user.password" type="password" autocomplete="new-password"/></el-form-item><EducationStagePicker v-model="user.education_stage"/><el-button native-type="submit" type="primary" :loading="busy">创建账户</el-button></el-form></el-collapse-item></el-collapse></section></el-tab-pane>
  <el-tab-pane label="关于与账户" name="about"><section class="paper-card panel"><h3>题迹 Questrace</h3><p>让每一道错题都有收获</p><el-switch :model-value="autoUpdate" active-text="自动检查更新" @update:model-value="setAutoUpdate(Boolean($event))"/><el-button :loading="updateBusy" @click="checkUpdate(true)">检查更新</el-button><p role="status">{{updateMessage}}</p><p class="meta-text">题目和图片保存在当前电脑。保持电脑在线，即可从同一局域网的手机访问。</p><template v-if="settings"><el-form-item label="APK 下载地址"><el-input v-model="settings.download_url" placeholder="可选"/></el-form-item><el-button :loading="busy" @click="save">保存下载地址</el-button></template><el-divider/><el-button @click="logout">退出当前账户</el-button></section></el-tab-pane>
 </el-tabs>
</div>
</template>
<script setup lang="ts">
import EducationStagePicker from '@/components/EducationStagePicker/index.vue'
import {useUserStore} from '@/stores/user.store'
import {useSubjectsStore} from '@/stores/subjects.store'
import {autoUpdate,setAutoUpdate,checkUpdate,updateBusy,updateMessage} from '@/update/updates'
import { onMounted, reactive, ref } from 'vue'
import ModelConfigField from '@/components/ModelConfigField/index.vue'
import {useAIStore} from '@/stores/ai.store'
import AppearancePanel from '@/components/AppearancePanel/index.vue'
import SecretField from '@/components/SecretField/index.vue'
import {useAuthStore} from '@/stores/auth.store'
import {useRouter} from 'vue-router'
const profileStore=useUserStore(),educationStage=ref(''),stageBusy=ref(false)
async function loadProfile(){try{await profileStore.fetchProfile();educationStage.value=profileStore.profile?.education_stage||'university'}catch(e){error.value=getErrorMessage(e)}}
async function saveStage(){stageBusy.value=true;try{await profileStore.saveEducationStage(educationStage.value);await useSubjectsStore().fetch();ElMessage.success('学习阶段已保存')}catch(e){error.value=getErrorMessage(e)}finally{stageBusy.value=false}}
const active=ref('account'), auth=useAuthStore(), router=useRouter()
function logout(){auth.logout();router.replace('/auth')}
import { ElMessage } from 'element-plus'
import { getErrorMessage } from '@/utils/error'
import { httpGet,httpPost,httpPut } from '@/api/http'
type Model={name:string;provider_type:string;base_url:string;model:string;api_key:string}
type Settings={registration_enabled:boolean;ocr:{name:string;model:string;api_key:string};models:Model[];embedding:Omit<Model,'name'>;download_url:string}
const status=ref<{urls:string[];embedding_enabled:boolean;discovery?:{state:string}}>({urls:[],embedding_enabled:false}),settings=ref<Settings|null>(null),jobs=ref<Record<string,number>>({}),users=ref<any[]>([]),results=ref<Record<string,string>>({}),error=ref(''),busy=ref(false)
const savedNames=new WeakMap<Model,string>()
const user=reactive({username:'',email:'',password:'',education_stage:''})
async function load(){try{status.value=await httpGet('/api/v1/system/status');jobs.value=await httpGet('/api/v1/vector-jobs');try{settings.value=await httpGet('/api/v1/admin/settings');settings.value?.models.forEach(m=>savedNames.set(m,m.name));users.value=await httpGet('/api/v1/admin/users')}catch{settings.value=null}}catch(e){error.value=getErrorMessage(e)}}
async function act(fn:()=>Promise<void>){busy.value=true;error.value='';try{await fn()}catch(e){error.value=getErrorMessage(e)}finally{busy.value=false}}
async function save(){await act(async()=>{await httpPut('/api/v1/admin/settings',{...settings.value,models:settings.value?.models.map(m=>({...m,saved_name:savedNames.get(m)}))});await useAIStore().fetchProviders(true);ElMessage.success('设置已保存');await load()})}
async function test(){await act(async()=>{const raw=await httpPost<Record<string,string>>('/api/v1/admin/settings/test',settings.value);results.value=Object.fromEntries(Object.entries(raw).map(([key,value])=>[key,/^(ok|success|成功|连接成功|服务连接成功)/i.test(value)?'连接成功':getErrorMessage(new Error(value),'连接测试失败，请检查服务配置。')]))})}
async function retry(){await act(async()=>{await httpPost('/api/v1/vector-jobs',{});await load()})}
async function createUser(){await act(async()=>{await httpPost('/api/v1/admin/users',user);user.password='';ElMessage.success('用户已创建');await load()})}
async function copy(text:string){try{await navigator.clipboard.writeText(text);ElMessage.success('地址已复制')}catch{ElMessage.info('请选中地址手动复制')}}
onMounted(()=>{load();loadProfile()})
</script>
<style scoped>
.student-link{font-size:14px;margin:12px 0;line-height:1.6}.settings-page{max-width:1000px}.panel{padding:24px}.address{display:flex;gap:12px;align-items:center;flex-wrap:wrap;padding:16px 0}.model{padding:16px;border:1px solid var(--line);border-radius:12px;margin-bottom:16px}.settings-actions{display:flex;gap:12px;flex-wrap:wrap;margin-top:24px}.job-stats{display:grid;grid-template-columns:repeat(3,1fr);gap:16px;margin:28px 0}.job-stats div{display:flex;flex-direction:column;gap:8px}.job-stats strong{font-size:28px}.job-stats span{color:var(--text-secondary);font-size:13px}.user-list{display:grid;margin:24px 0}.user-list article{display:flex;align-items:center;flex-wrap:wrap;gap:12px;padding:16px 0;border-bottom:1px solid var(--line)}.user-list article span{color:var(--text-secondary);flex:1;overflow-wrap:anywhere}.registration-save{margin-left:16px}:deep(.el-tabs__item){height:48px}
</style>
