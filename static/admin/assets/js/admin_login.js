/**
 * admin_login.js - 管理员登录页面逻辑
 * 功能：处理管理员登录验证、Token存储和页面跳转
 */

const { createApp, ref, onMounted } = Vue;

createApp({
    setup() {
        const loginKey = ref('');
        const isSubmitting = ref(false);
        const errorMessage = ref('');

        /*
         * 处理管理员登录
         * 验证管理员密钥并获取访问令牌
         */
        const handleLogin = async () => {
            if (!loginKey.value) {
                errorMessage.value = "请输入访问密钥";
                return;
            }

            isSubmitting.value = true;
            errorMessage.value = "";

            try {
                const res = await fetch('/api/auth/admin', {
                    method: 'POST',
                    headers: { 
                        'Content-Type': 'application/json',
                        'Accept': 'application/json' 
                    },
                    body: JSON.stringify({ key: loginKey.value })
                });

                if (res.ok) {
                    const data = await res.json();
                    
                    if (data.token) {
                        localStorage.setItem('slite_admin_token', data.token);
                        
                        /*
                         * 主题配置现在由theme-switcher.js统一管理
                         * 不再在此处处理主题设置
                         */
                        
                        setTimeout(() => {
                            window.location.replace('/admin');
                        }, 100);
                        
                    } else {
                        errorMessage.value = "服务器未返回有效的身份凭证";
                    }
                } else {
                    const errData = await res.json().catch(() => ({}));
                    errorMessage.value = errData.message === "INVALID ADMIN KEY" 
                        ? "密钥错误，请重新输入" 
                        : (errData.message || "登录失败，请重试");
                    
                    loginKey.value = "";
                }
            } catch (err) {
                errorMessage.value = "无法连接到系统，请检查网络或服务器状态";
            } finally {
                isSubmitting.value = false;
            }
        };

        /*
         * 监听键盘事件
         * 支持回车键提交表单
         */
        const handleKeyPress = (event) => {
            if (event.key === 'Enter' && !isSubmitting.value) {
                handleLogin();
            }
        };

        /*
         * 组件挂载生命周期
         * 设置输入框焦点并检查URL参数
         */
        onMounted(() => {
            const input = document.querySelector('input[type="password"]');
            if (input) {
                input.focus();
            }
            
            const savedToken = localStorage.getItem('slite_admin_token');
            if (savedToken) {
            }
            
            const urlParams = new URLSearchParams(window.location.search);
            const error = urlParams.get('error');
            if (error === 'session_expired') {
                errorMessage.value = '会话已过期，请重新登录';
                localStorage.removeItem('slite_admin_token');
            } else if (error === 'unauthorized') {
                errorMessage.value = '未授权访问，请重新验证';
                localStorage.removeItem('slite_admin_token');
            }
        });

        return {
            loginKey,
            isSubmitting,
            errorMessage,
            handleLogin,
            handleKeyPress
        };
    }
}).mount('#app');