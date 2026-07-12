/**
 * 基本设置组件 (BasicComponent)
 * 处理系统核心访问权限、界面版块可见性、监控频率及主题设置
 */
const BasicComponent = {
    name: 'BasicComponent',
    props: ['basic_config', 'is_saving', 'save_success'],
    data() {
        return {
            save_error: false,
            defaultConfig: {
                view_auth_enabled: false,
                view_auth_key: "123456",
                admin_auth_key: "admin888",
                show_docker: true,
                show_net_delay: true,
                show_heatmap: true,
                node_refresh_interval: 5,
                task_show_count: 8,
                max_log_count: 100,
                admin_theme: "default",
                monitor_theme: "default",
                theme_mode: "fixed"
            },
            is_resetting: false,
            reset_success: false
        };
    },
    watch: {
        save_success(newVal) {
            if (newVal) {
                this.save_error = false;
            }
        }
    },
    computed: {
        adminThemeOptions() {
            return [
                { value: 'default', label: '默认主题', color: 'bg-gray-100' },
                { value: 'dark', label: '深色主题', color: 'bg-gray-900' }
            ];
        },
        monitorThemeOptions() {
            return [
                { value: 'default', label: '默认主题', color: 'bg-gray-100' },
                { value: 'dark', label: '深色主题', color: 'bg-gray-900' }
            ];
        },
        themeModeOptions() {
            return [
                { value: 'fixed', label: '固定主题' },
                { value: 'system', label: '跟随系统' }
            ];
        }
    },
    methods: {
        handleSave() {
            this.save_error = false;
            
            if (this.basic_config.view_auth_enabled && !this.basic_config.view_auth_key) {
                alert("开启验证后，验证密钥不能为空");
                return;
            }

            const configToSave = {
                view_auth_enabled: this.basic_config.view_auth_enabled || false,
                show_docker: this.basic_config.show_docker !== undefined ? this.basic_config.show_docker : true,
                show_net_delay: this.basic_config.show_net_delay !== undefined ? this.basic_config.show_net_delay : true,
                show_heatmap: this.basic_config.show_heatmap !== undefined ? this.basic_config.show_heatmap : true,
                node_refresh_interval: this.basic_config.node_refresh_interval || 5,
                task_show_count: this.basic_config.task_show_count || 8,
                max_log_count: this.basic_config.max_log_count || 100,
                admin_theme: this.basic_config.admin_theme || 'default',
                monitor_theme: this.basic_config.monitor_theme || 'default',
                theme_mode: this.basic_config.theme_mode || 'fixed'
            };

            if (this.basic_config.view_auth_enabled && this.basic_config.view_auth_key) {
                configToSave.view_auth_key = this.basic_config.view_auth_key;
            }

            if (this.basic_config.admin_auth_key) {
                configToSave.admin_auth_key = this.basic_config.admin_auth_key;
            }

            this.$emit('save-basic', configToSave);
            
            if (this.basic_config.theme_mode === 'system') {
                const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
                const themeName = prefersDark ? 'dark' : 'default';
                document.documentElement.setAttribute('data-theme', themeName);
                const themeLink = document.getElementById('theme-style');
                if (themeLink) {
                    themeLink.href = `/static/admin/assets/css/${themeName}.css`;
                }
            } else {
                const themeName = this.basic_config.admin_theme;
                document.documentElement.setAttribute('data-theme', themeName);
                const themeLink = document.getElementById('theme-style');
                if (themeLink) {
                    const themeFile = themeName === 'dark' ? 'dark' : 'default';
                    themeLink.href = `/static/admin/assets/css/${themeFile}.css`;
                }
            }
            
            document.cookie = `slite_admin_theme=${this.basic_config.admin_theme};path=/;max-age=604800`;
            document.cookie = `slite_theme_mode=${this.basic_config.theme_mode};path=/;max-age=604800`;
        },
        handleResetToDefault() {
            if (confirm("确定要恢复默认设置吗？当前未保存的修改将丢失。")) {
                this.is_resetting = true;
                
                const keys = Object.keys(this.defaultConfig);
                keys.forEach((field, index) => {
                    setTimeout(() => {
                        this.basic_config[field] = this.defaultConfig[field];
                        
                        if (index === keys.length - 1) {
                            setTimeout(() => {
                                this.is_resetting = false;
                                this.reset_success = true;
                                setTimeout(() => { this.reset_success = false; }, 3000);
                            }, 300);
                        }
                    }, index * 50);
                });
            }
        },
        getThemeLabel(themeValue, options) {
            const option = options.find(opt => opt.value === themeValue);
            return option ? option.label : '未知主题';
        },
        getThemeModeLabel(modeValue) {
            const modes = {
                'fixed': '固定主题',
                'system': '跟随系统'
            };
            return modes[modeValue] || '未知模式';
        }
    },
    template: `
    <div class="animate-in fade-in slide-in-from-bottom-4 space-y-8 sm:space-y-12 relative">
        
        <transition enter-active-class="transition duration-300 ease-out"
                    enter-from-class="transform translate-y-[-10px] opacity-0"
                    enter-to-class="transform translate-y-0 opacity-100"
                    leave-active-class="transition duration-200 ease-in"
                    leave-from-class="opacity-100"
                    leave-to-class="opacity-0">
            
            <div v-if="save_success || save_error" 
                 :class="[save_success ? 'notification-success' : 'notification-error']"
                 class="absolute top-[-60px] sm:top-[-100px] right-0 z-50 flex items-center gap-3">
                <span>{{ save_success ? '保存成功' : '保存失败' }}</span>
            </div>
        </transition>

        <transition enter-active-class="transition duration-300 ease-out"
                    enter-from-class="transform translate-y-[-10px] opacity-0"
                    enter-to-class="transform translate-y-0 opacity-100"
                    leave-active-class="transition duration-200 ease-in"
                    leave-from-class="opacity-100"
                    leave-to-class="opacity-0">
            
            <div v-if="reset_success" 
                 class="notification-success absolute top-[-60px] sm:top-[-100px] right-0 z-50 flex items-center gap-3">
                <span>已恢复默认设置</span>
            </div>
        </transition>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-8 sm:gap-16 items-stretch">
            
            <div class="flex flex-col h-full space-y-8 sm:space-y-10">
                <div class="space-y-6">
                    <h3 class="flex items-center gap-3 text-[12px] font-black uppercase tracking-[0.2em]">
                        <span class="w-2 h-6 bg-custom-red-600"></span>
                        首页访问权限
                    </h3>
                    <div class="card-custom bg-custom-gray-50 space-y-6 flex-grow">
                        <div class="flex items-center justify-between cursor-pointer" @click="basic_config.view_auth_enabled = !basic_config.view_auth_enabled">
                            <span class="text-[11px] font-black uppercase tracking-wider">强制密钥访问验证</span>
                            <div class="toggle-switch">
                                <input type="checkbox" v-model="basic_config.view_auth_enabled" class="sr-only" @click.stop>
                                <div class="toggle-slider" :class="{'active': basic_config.view_auth_enabled}"></div>
                            </div>
                        </div>
                        
                        <div v-if="basic_config.view_auth_enabled" class="space-y-2 animate-in slide-in-from-top-2">
                            <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-widest">验证密钥 (AUTH_KEY)</label>
                            <input type="text" v-model="basic_config.view_auth_key" placeholder="请输入访问密钥"
                                   class="input-custom w-full p-3 text-xs font-mono font-bold focus:bg-white outline-none">
                        </div>
                        <p class="text-[10px] text-custom-gray-400 font-bold italic leading-relaxed">
                            * 开启后，访客需输入正确密钥才能查看监控数据。
                        </p>
                    </div>
                </div>

                <div class="space-y-6">
                    <h3 class="flex items-center gap-3 text-[12px] font-black uppercase tracking-[0.2em]">
                        <span class="w-2 h-6 bg-custom-black"></span>
                        界面版块显示控制
                    </h3>
                    <div class="space-y-3">
                        <div v-for="(label, key) in { 'show_docker': '容器运行状态', 'show_net_delay': '网络链路检测', 'show_heatmap': '服务可用性检测' }" 
                               :key="key"
                               class="input-custom flex items-center justify-between p-4 cursor-pointer hover:bg-gray-50 transition-all group"
                               @click="basic_config[key] = !basic_config[key]">
                            <span class="text-[11px] font-black uppercase tracking-wider group-hover:translate-x-1 transition-transform">
                                 {{ label }}
                            </span>
                            <div class="toggle-switch">
                                <input type="checkbox" v-model="basic_config[key]" class="sr-only" @click.stop>
                                <div class="toggle-slider" :class="{'active': basic_config[key]}"></div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div class="flex flex-col h-full">
                <div class="space-y-6 flex-grow flex flex-col">
                    <h3 class="flex items-center gap-3 text-[12px] font-black uppercase tracking-[0.2em]">
                        <span class="w-2 h-6 bg-custom-black"></span>
                        监控性能与频率校准
                    </h3>
                    
                    <div class="card-custom p-5 sm:p-8 flex-grow space-y-6 sm:space-y-8 flex flex-col justify-between">
                        <div class="space-y-6 sm:space-y-8">
                            <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 sm:gap-6">
                                <div class="space-y-2">
                                    <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-widest">数据刷新频率 (秒)</label>
                                    <input type="number" v-model.number="basic_config.node_refresh_interval" 
                                           class="input-custom input-custom-black w-full p-3 text-sm font-mono font-bold focus:outline-none">
                                </div>
                                <div class="space-y-2">
                                    <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-widest">单节点展示任务数</label>
                                    <input type="number" v-model.number="basic_config.task_show_count" 
                                           class="input-custom input-custom-black w-full p-3 text-sm font-mono font-bold focus:outline-none">
                                </div>
                            </div>
                            
                            <div class="space-y-2">
                                <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-widest">历史日志保留行数 (最大值)</label>
                                <input type="number" v-model.number="basic_config.max_log_count" 
                                       class="input-custom input-custom-black w-full p-3 text-sm font-mono font-bold focus:outline-none">
                            </div>
                        </div>

                        <div class="card-custom bg-custom-yellow-50 flex gap-4 mt-8">
                            <div class="text-[10px] font-bold text-custom-black leading-relaxed uppercase">
                                严重警告：刷新频率设置过低会显著增加服务器资源占用。建议保持在 3s - 5s 之间。
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="space-y-6">
            <h3 class="flex items-center gap-3 text-[12px] font-black uppercase tracking-[0.2em]">
                <span class="w-2 h-6 bg-custom-black"></span>
                个性化视觉主题
            </h3>
            
            <div class="card-custom p-5 sm:p-8 space-y-6 sm:space-y-8">
                <div class="grid grid-cols-1 md:grid-cols-2 gap-6 sm:gap-8">
                    <div class="space-y-2">
                        <label class="text-[10px] font-black uppercase text-custom-gray-400 block mb-1 tracking-[0.1em]">后台管理主题</label>
                        <select v-model="basic_config.admin_theme" 
                                class="input-custom w-full p-3 text-[11px] font-mono focus:outline-none focus:bg-gray-50 transition-all">
                            <option v-for="option in adminThemeOptions" :value="option.value" :key="option.value">
                                {{ option.label }}
                            </option>
                        </select>
                        <p class="text-[9px] text-custom-gray-400 font-bold mt-1">
                            当前: {{ getThemeLabel(basic_config.admin_theme, adminThemeOptions) }}
                        </p>
                    </div>
                    
                    <div class="space-y-2">
                        <label class="text-[10px] font-black uppercase text-custom-gray-400 block mb-1 tracking-[0.1em]">监控页面主题</label>
                        <select v-model="basic_config.monitor_theme" 
                                class="input-custom w-full p-3 text-[11px] font-mono focus:outline-none focus:bg-gray-50 transition-all">
                            <option v-for="option in monitorThemeOptions" :value="option.value" :key="option.value">
                                {{ option.label }}
                            </option>
                        </select>
                        <p class="text-[9px] text-custom-gray-400 font-bold mt-1">
                            当前: {{ getThemeLabel(basic_config.monitor_theme, monitorThemeOptions) }}
                        </p>
                    </div>
                </div>
                
                <div class="space-y-2">
                    <label class="text-[10px] font-black uppercase text-custom-gray-400 block mb-1 tracking-[0.1em]">色彩渲染模式</label>
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
                        <label v-for="option in themeModeOptions" 
                               :key="option.value"
                               @click="basic_config.theme_mode = option.value"
                               :class="[
                                   'input-custom flex items-center justify-between p-4 cursor-pointer transition-all text-[11px] font-mono',
                                   basic_config.theme_mode === option.value ? 'bg-custom-blue-50 border-custom-blue-300' : 'hover:bg-gray-50'
                               ]">
                            <span class="font-bold uppercase tracking-wider">{{ option.label }}</span>
                            <div class="w-4 h-4 rounded-full border-2 flex items-center justify-center border-custom-black">
                                <div v-if="basic_config.theme_mode === option.value" class="w-2 h-2 rounded-full bg-custom-black"></div>
                            </div>
                        </label>
                    </div>
                    <p class="text-[9px] text-custom-gray-400 font-bold mt-1">
                        当前: {{ getThemeModeLabel(basic_config.theme_mode) }}
                    </p>
                </div>
            </div>
        </div>

        <!-- 操作按钮：小屏下纵向堆叠、占满宽度；桌面端保持原来的右对齐横排 -->
        <div class="flex flex-col sm:flex-row gap-3 sm:gap-4 justify-end items-stretch sm:items-center pt-6 sm:pt-8">
            <button @click="handleResetToDefault" 
                    :disabled="is_resetting || is_saving"
                    class="btn-secondary w-full sm:w-auto sm:min-w-[200px] py-3.5 sm:py-4 px-6 sm:px-8 text-[10px] sm:text-[11px] font-black uppercase tracking-[0.15em] sm:tracking-[0.2em] disabled:opacity-50 transition-all flex items-center justify-center gap-3 whitespace-nowrap">
                {{ is_resetting ? '正在重置...' : '恢复默认' }}
            </button>
            
            <button @click="handleSave" :disabled="is_saving" 
                    class="btn-primary w-full sm:w-auto sm:min-w-[200px] py-3.5 sm:py-4 px-6 sm:px-8 text-[10px] sm:text-[11px] font-black uppercase tracking-[0.15em] sm:tracking-[0.2em] disabled:bg-gray-400 transition-all flex items-center justify-center gap-3 whitespace-nowrap">
                <span v-if="is_saving" class="animate-spin w-4 h-4 border-2 border-white/30 border-t-white rounded-full"></span>
                {{ is_saving ? '正在保存数据...' : '确认并保存更改' }}
            </button>
        </div>

        <transition enter-active-class="transition duration-300" leave-active-class="transition duration-200"
                    enter-from-class="opacity-0" leave-to-class="opacity-0">
            <div v-if="is_resetting" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-[1000] p-4">
                <div class="bg-white p-8 sm:p-10 shadow-2xl text-center space-y-4">
                    <div class="animate-spin w-10 h-10 border-4 border-black border-t-transparent rounded-full mx-auto"></div>
                    <p class="text-[11px] font-black uppercase tracking-widest">System Resetting...</p>
                </div>
            </div>
        </transition>
    </div>`
};

window.BasicComponent = BasicComponent;