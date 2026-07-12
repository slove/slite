/**
 * 安全设置管理组件
 */
const SecurityComponent = {
    props: ['site_config'],
    template: `
    <div class="animate-in fade-in slide-in-from-bottom-4 space-y-8 pb-20 relative">
        
        <transition enter-active-class="transition duration-300 ease-out"
                    enter-from-class="transform translate-y-[-10px] opacity-0"
                    enter-to-class="transform translate-y-0 opacity-100"
                    leave-active-class="transition duration-200 ease-in"
                    leave-from-class="opacity-100"
                    leave-to-class="opacity-0">
            
            <div v-if="save_success || save_error" 
                 :class="[save_success ? 'notification-success' : 'notification-error']"
                 class="absolute top-[-100px] right-0 z-50 flex items-center gap-3">
                <span>{{ save_success ? '保存成功' : '保存失败' }}</span>
            </div>
        </transition>

        <header class="flex justify-between items-end mb-8 pb-4 border-b border-custom-black">
            <div>
                <p class="text-[10px] text-custom-gray-400 mt-2 font-bold uppercase tracking-widest">
                    安全状态: 
                    <span :class="isUnlocked ? 'text-custom-green-600' : 'text-custom-red-600'">
                        {{ isUnlocked ? '● 敏感操作已解锁' : '○ 高级设置已锁定' }}
                    </span>
                </p>
            </div>
            <div class="flex items-center pb-1">
                <button v-if="isUnlocked" @click="lockSettings" 
                        class="btn-primary text-[10px] px-8 py-2 font-black uppercase tracking-widest">
                    放弃并锁定
                </button>
            </div>
        </header>

        <div class="card-custom bg-custom-white space-y-6"
             :class="{'opacity-50 grayscale transition-all': isUnlocked}">
            <div class="flex justify-between items-center border-b border-gray-100 pb-4">
                <h4 class="text-sm font-black uppercase flex items-center gap-3 text-custom-black">
                    <span class="bg-custom-black w-1.5 h-5"></span> 
                    身份凭据校验
                </h4>
            </div>
            <div class="space-y-4">
                <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-[0.2em] block">
                    当前管理员密钥
                </label>
                <div class="flex gap-3">
                    <input type="password" v-model="verifyKey" :disabled="isUnlocked"
                           @keyup.enter="handleVerify"
                           class="input-custom flex-1 p-4 font-mono text-xs focus:bg-gray-50 outline-none transition-all h-[52px]"
                           placeholder="输入当前管理员密钥以获取修改权限...">
                    <button @click="handleVerify" :disabled="isUnlocked || !verifyKey"
                             class="btn-primary px-8 text-[10px] font-black uppercase hover:bg-white hover:text-black transition-all disabled:opacity-30 h-[52px]">
                        验证权限
                    </button>
                </div>
            </div>
        </div>

        <div class="card-custom bg-custom-white space-y-10 transition-all duration-500"
             :class="isUnlocked ? 'opacity-100' : 'opacity-20 blur-sm pointer-events-none select-none'">
            
            <div class="flex justify-between items-center border-b border-gray-100 pb-4">
                <h4 class="text-sm font-black uppercase flex items-center gap-3 text-custom-black">
                    <span class="bg-custom-red-600 w-1.5 h-5"></span> 
                    管理密钥重置
                </h4>
            </div>

            <div class="w-full space-y-8">
                <div class="space-y-8">
                    <div class="space-y-3">
                        <label class="text-[10px] font-black uppercase tracking-widest text-custom-gray-500 block">新管理密钥</label>
                        <div class="flex gap-2 w-full">
                            <input type="text" v-model="form.newAdminKey" 
                                   class="input-custom flex-1 border-b border-custom-black p-3 font-mono text-xs outline-none focus:bg-gray-50"
                                   placeholder="输入新的管理访问密钥">
                            <button @click="generateKey" class="btn-secondary text-[9px] font-black px-6 whitespace-nowrap">
                                随机生成
                            </button>
                        </div>
                    </div>
                </div>

                <div class="bg-custom-red-50 p-4 border-l-4 border-custom-red-600 w-full">
                    <p class="text-[10px] text-custom-red-700 font-bold leading-relaxed uppercase tracking-tight">
                        关键警告：修改管理员密钥会导致当前所有在线管理员的会话立即失效。
                        保存成功后，系统将强制跳转至登录界面，请务必牢记新密钥。
                    </p>
                </div>
            </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <button @click="save_security_config" :disabled="!isUnlocked || !form.newAdminKey"
                    class="btn-primary py-4 text-[11px] font-black uppercase tracking-[0.2em] shadow-[4px_4px_0px_rgba(0,0,0,0.1)] disabled:opacity-20">
                更新管理凭据并重新登录
            </button>
            <button @click="lockSettings" 
                    class="btn-secondary py-4 text-[11px] font-black uppercase tracking-[0.2em] shadow-[4px_4px_0px_rgba(0,0,0,0.05)]">
                取消操作
            </button>
        </div>
    </div>
    `,
    data() {
        return {
            isUnlocked: false,
            save_success: false,
            save_error: false,
            verifyKey: '',
            form: {
                newAdminKey: ''
            },
            config: {
                admin_auth_key: ''
            }
        }
    },
    methods: {
        /**
         * 触发通知提示
         */
        show_notification(type) {
            if (type === 'success') {
                this.save_success = true;
                this.save_error = false;
                setTimeout(() => { this.save_success = false; }, 3000);
            } else {
                this.save_error = true;
                this.save_success = false;
                setTimeout(() => { this.save_error = false; }, 3000);
            }
        },

        /**
         * 验证当前管理密钥以解锁高级功能
         */
        async handleVerify() {
            if (!this.verifyKey) return;
            try {
                const res = await fetch('/api/auth/admin', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ key: this.verifyKey })
                });

                if (res.ok) {
                    this.isUnlocked = true;
                    this.verifyKey = '';
                    this.show_notification('success');
                    this.fetch_security_config();
                } else {
                    this.show_notification('error');
                    alert("身份验证失败：输入的当前管理密钥不正确");
                }
            } catch (err) {
                this.show_notification('error');
                alert("安全网关通讯故障，请检查后端服务状态");
            }
        },

        /**
         * 重置安全视图状态并锁定组件
         */
        lockSettings() {
            this.isUnlocked = false;
            this.form.newAdminKey = '';
            this.verifyKey = '';
            this.show_notification('success');
        },

        /**
         * 随机生成安全强度较高的密钥字符串
         */
        generateKey() {
            const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
            let result = '';
            for (let i = 0; i < 16; i++) {
                result += chars.charAt(Math.floor(Math.random() * chars.length));
            }
            this.form.newAdminKey = result;
        },

        /**
         * 从服务端获取当前安全配置载荷
         */
        async fetch_security_config() {
            try {
                const res = await fetch('/api/admin/config');
                if (res.ok) {
                    const data = await res.json();
                    this.config = data;
                }
            } catch (err) {
                console.error("安全配置拉取异常:", err);
            }
        },

        /**
         * 执行核心凭据重置逻辑并清理本地会话
         */
        async save_security_config() {
            if (!this.form.newAdminKey || this.form.newAdminKey.length < 6) {
                alert("安全建议：新密钥长度不得小于 6 位");
                return;
            }

            try {
                const payload = {
                    ...this.config,
                    admin_auth_key: this.form.newAdminKey
                };

                const res = await fetch('/api/admin/config', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(payload)
                });

                if (res.ok) {
                    this.show_notification('success');
                    setTimeout(() => {
                        localStorage.removeItem('slite_admin_token');
                        window.location.href = '/admin/login';
                    }, 1500);
                } else {
                    this.show_notification('error');
                    alert("保存失败：权限校验或后端逻辑异常");
                }
            } catch (err) {
                this.show_notification('error');
                alert("持久化失败：网络通讯异常");
            }
        }
    }
};

window.SecurityComponent = SecurityComponent;