/**
 * login.js - 监控页面登录验证逻辑
 * 核心功能：处理访问密钥提交、Token 持久化存储及页面跳转
 */

// Vue 应用
const { createApp, ref, onMounted } = Vue;

createApp({
    setup() {
        const loginKey = ref('');
        const isSubmitting = ref(false);
        const errorMessage = ref('');

        /**
         * 处理登录逻辑
         */
        const handleLogin = async () => {
            if (!loginKey.value || isSubmitting.value) return;

            // 重置状态
            isSubmitting.value = true;
            errorMessage.value = '';

            try {
                // 向后端发起验证请求
                const response = await fetch('/api/auth/view', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({
                        key: loginKey.value
                    })
                });

                if (response.ok) {
                    const data = await response.json();
                    
                    // 保存Token
                    localStorage.setItem('slite_view_token', data.token);
                    
                    // 跳转到主页
                    setTimeout(() => {
                        window.location.href = '/';
                    }, 150);
                } else {
                    const errorData = await response.json().catch(() => ({}));
                    errorMessage.value = errorData.message === "INVALID ACCESS KEY" 
                        ? "认证失败，请检查访问密钥是否正确" 
                        : (errorData.message || "登录失败，请稍后再试");
                    loginKey.value = '';
                }
            } catch (err) {
                errorMessage.value = '无法连接到认证服务器，请稍后再试';
            } finally {
                isSubmitting.value = false;
            }
        };

        /**
         * 监听键盘事件，支持回车键提交
         */
        const handleKeyPress = (event) => {
            if (event.key === 'Enter' && !isSubmitting.value && loginKey.value) {
                handleLogin();
            }
        };

        onMounted(() => {
            window.addEventListener('keypress', handleKeyPress);
            
            return () => {
                window.removeEventListener('keypress', handleKeyPress);
            };
        });

        onMounted(() => {
            const existingToken = localStorage.getItem('slite_view_token');
            
            // 检查URL中的错误参数
            const urlParams = new URLSearchParams(window.location.search);
            const error = urlParams.get('error');
            if (error === 'session_expired') {
                errorMessage.value = '会话已过期，请重新登录';
                localStorage.removeItem('slite_view_token');
            } else if (error === 'unauthorized') {
                errorMessage.value = '未授权访问，请重新验证';
                localStorage.removeItem('slite_view_token');
            } else if (error === 'auth_required') {
                errorMessage.value = '需要身份验证才能访问';
            }
        });

        return {
            loginKey,
            isSubmitting,
            errorMessage,
            handleLogin,
            handleKeyPress
        };
    },
    
    mounted() {
        const input = this.$el.querySelector('input[type="password"]');
        if (input) {
            input.focus();
        }
    }
}).mount('#login-app');