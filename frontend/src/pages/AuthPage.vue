<template><main class="auth-page"><section class="auth-intro"><h1>题迹 Questrace</h1><p>让每一道错题<br/>都有收获。</p><span class="meta-text">收集 · 整理 · 理解 · 再练习</span></section><section class="auth-card paper-card">
                <div class="auth-header">
                    <div>
                        <h2>
                            {{
                                activeTab === "login" ? "登录账号" : "注册账号"
                            }}
                        </h2>
                    </div>
                </div>

                <el-tabs v-model="activeTab" stretch>
                    <el-tab-pane label="登录" name="login">
                        <el-form label-position="top" @submit.prevent>
                            <el-form-item label="用户名">
                                <el-input
                                    v-model="loginForm.username" autocomplete="username"
                                    placeholder="请输入用户名"
                                />
                            </el-form-item>
                            <el-form-item label="密码">
                                <el-input
                                    v-model="loginForm.password" autocomplete="current-password"
                                    type="password"
                                    show-password
                                    placeholder="请输入密码"
                                    @keyup.enter="submitLogin"
                                />
                            </el-form-item>
                            <el-button
                                type="primary"
                                class="submit-btn"
                                :loading="submitting"
                                @click="submitLogin"
                            >
                                登录并进入系统
                            </el-button>
                        </el-form>
                    </el-tab-pane>

                    <el-tab-pane v-if="registrationEnabled" label="注册" name="register">
                        <el-form label-position="top" @submit.prevent>
                            <el-form-item label="用户名">
                                <el-input
                                    v-model="registerForm.username"
                                    placeholder="至少便于识别的用户名"
                                />
                            </el-form-item>
                            <el-form-item label="邮箱">
                                <el-input
                                    v-model="registerForm.email"
                                    placeholder="例如 name@example.com"
                                />
                            </el-form-item>
                            <el-form-item label="密码">
                                <el-input
                                    v-model="registerForm.password"
                                    type="password"
                                    show-password
                                    placeholder="至少 6 位"
                                    @keyup.enter="submitRegister"
                                />
                            </el-form-item>
                            <EducationStagePicker v-model="registerForm.education_stage"/>
                            <el-button
                                type="primary"
                                class="submit-btn"
                                :loading="submitting"
                                @click="submitRegister"
                            >
                                注册并直接登录
                            </el-button>
                        </el-form>
                    </el-tab-pane>
                </el-tabs>
            </section></main></template>
<script setup lang="ts">
import EducationStagePicker from "@/components/EducationStagePicker/index.vue";
import { ElMessage } from "element-plus";
import { computed, reactive, ref, onMounted } from "vue";
import { useRoute, useRouter } from "vue-router";

import { useAuthStore } from "@/stores/auth.store";
import { getErrorMessage } from "@/utils/error";

const route = useRoute();
const router = useRouter();
const authStore = useAuthStore();
const registrationEnabled=ref(false);
onMounted(async()=>{try{const r=await fetch("/api/v1/system/status");registrationEnabled.value=(await r.json()).data.registration_enabled}catch{}});
const activeTab = ref<"login" | "register">("login");
const submitting = computed(() => authStore.loading);

const loginForm = reactive({
    username: "",
    password: "",
});

const registerForm = reactive({
 education_stage: "",
    username: "",
    email: "",
    password: "",
});

function resolveNextPath() {
    const redirect = route.query.redirect;
    return typeof redirect === "string" && redirect.startsWith("/")
        ? redirect
        : "/dashboard";
}

async function submitLogin() {
    if (!loginForm.username.trim() || !loginForm.password) {
        ElMessage.warning("请输入用户名和密码");
        return;
    }

    try {
        await authStore.loginWithPassword(loginForm);
        ElMessage.success("登录成功");
        router.push(resolveNextPath());
    } catch (error) {
        ElMessage.error(getErrorMessage(error, "登录失败"));
    }
}

async function submitRegister() {
 if(!registerForm.education_stage){ElMessage.warning("请选择当前学习阶段：大学或高中");return;}
    if (
        !registerForm.username.trim() ||
        !registerForm.email.trim() ||
        !registerForm.password
    ) {
        ElMessage.warning("请完整填写用户名、邮箱和密码");
        return;
    }

    if (registerForm.password.length < 6) {
        ElMessage.warning("密码长度不能少于 6 位");
        return;
    }

    try {
        await authStore.registerAccount(registerForm);
        ElMessage.success("注册成功，已自动登录");
        router.push(resolveNextPath());
    } catch (error) {
        ElMessage.error(getErrorMessage(error, "注册失败"));
    }
}
</script>

<style scoped>
.auth-page{min-height:100dvh;display:grid;grid-template-columns:1fr 1fr;align-items:center;gap:64px;max-width:1050px;margin:auto;padding:48px}.auth-intro{padding:24px}.auth-intro h1{font-size:24px;margin-top:28px}.auth-intro p{font-size:38px;line-height:1.4;letter-spacing:-1px}.auth-card{padding:32px}.auth-header h2{font-size:24px;margin-top:0}.submit-btn{width:100%;margin-top:12px}@media(max-width:767px){.auth-page{display:block;padding:24px 16px}.auth-intro{padding:8px 8px 24px}.auth-intro p,.auth-intro .meta-text{display:none}.auth-intro h1{font-size:24px;margin:12px 0 0}.auth-card{padding:24px}}
</style>