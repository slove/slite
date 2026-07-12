/**
 * 日志记录管理组件 (LogsComponent)
 */
const LogsComponent = {
    props: ['site_config'],
    template: `
    <div class="animate-in fade-in slide-in-from-bottom-4 space-y-6 pb-20 font-sans text-slate-900 relative">
        
        <transition enter-active-class="transition duration-300 ease-out"
                    enter-from-class="transform translate-y-[-10px] opacity-0"
                    enter-to-class="transform translate-y-0 opacity-100"
                    leave-active-class="transition duration-200 ease-in"
                    leave-from-class="opacity-100"
                    leave-to-class="opacity-0">
            
            <div v-if="save_success || save_error" 
                 :class="[save_success ? 'notification-success' : 'notification-error']"
                 class="absolute top-[-100px] right-0 z-50 flex items-center gap-3">
                <span>{{ save_success ? '操作成功' : '操作失败' }}</span>
            </div>
        </transition>

        <section class="card-custom-no-padding overflow-hidden">
            <div class="p-8 md:p-10 flex flex-col md:flex-row justify-between items-start md:items-center gap-6">
                <div>
                    <div class="flex items-center gap-3 mb-4">
                        <div class="w-1.5 h-6 bg-custom-black"></div>
                        <h2 class="text-sm font-black uppercase tracking-widest text-custom-black">系统运行日志 / SYSTEM_LOGS</h2>
                    </div>
                    <h1 class="text-4xl font-black italic tracking-tighter text-custom-black uppercase">
                        实时遥测数据流
                    </h1>
                    <p class="text-[11px] text-custom-gray-400 mt-2 font-bold uppercase tracking-[0.2em]">
                        当前监控节点: {{ site_config.site_name || 'PRIMARY_NODE' }} // 自动同步已开启
                    </p>
                </div>

                <div class="flex gap-3">
                    <button @click="fetchLogs" 
                            class="btn-primary px-8 py-3 text-[11px] font-black uppercase tracking-widest shadow-[4px_4px_0px_rgba(0,0,0,0.2)]">
                        强制刷新
                    </button>
                    <button @click="clearLogs" 
                            class="btn-danger px-8 py-3 text-[11px] font-black uppercase tracking-widest shadow-[4px_4px_0px_rgba(220,38,38,0.1)]">
                        擦除历史记录
                    </button>
                </div>
            </div>
            <div class="h-1.5 bg-custom-black w-full"></div>
        </section>

        <section class="card-custom p-6">
            <div class="flex flex-col md:flex-row gap-8">
                <div class="flex-grow">
                    <div class="flex items-center gap-2 mb-3">
                        <div class="w-2 h-2 bg-custom-black"></div>
                        <label class="text-[10px] font-black uppercase tracking-widest text-custom-gray-400">关键字检索</label>
                    </div>
                    <input type="text" v-model="filter.query" 
                           class="input-custom w-full border-b border-[#666666] py-2 font-mono text-xs outline-none focus:border-black transition-colors"
                           placeholder="输入指令、IP或事件描述...">
                </div>
                <div class="shrink-0">
                    <div class="flex items-center gap-2 mb-3">
                        <div class="w-2 h-2 bg-custom-black"></div>
                        <label class="text-[10px] font-black uppercase tracking-widest text-custom-gray-400">事件分类过滤</label>
                    </div>
                    <div class="input-custom flex p-1 bg-gray-50">
                        <button v-for="t in categories" :key="t.id"
                                @click="filter.type = t.id"
                                :class="['px-4 py-1.5 text-[10px] font-black transition-all', 
                                         filter.type === t.id ? 'bg-custom-black text-custom-white' : 'text-custom-gray-400 hover:text-black']">
                            {{ t.name }}
                        </button>
                    </div>
                </div>
            </div>
        </section>

        <section class="card-custom-no-padding overflow-hidden">
            <table class="w-full text-left border-collapse">
                <thead>
                    <tr class="bg-custom-black text-custom-white">
                        <th class="py-4 px-6 text-[10px] font-black uppercase tracking-[0.2em] w-48">时间戳 (TIMESTAMP)</th>
                        <th class="py-4 px-6 text-[10px] font-black uppercase tracking-[0.2em] w-32 text-center">级别</th>
                        <th class="py-4 px-6 text-[10px] font-black uppercase tracking-[0.2em]">日志载荷与事件描述</th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-gray-100 font-mono">
                    <tr v-for="log in filteredLogs" :key="log.id" class="group hover:bg-gray-50 transition-colors">
                        <td class="py-4 px-6 text-[11px] text-custom-gray-400 italic">{{ formatTime(log.timestamp) }}</td>
                        <td class="py-4 px-6 text-center">
                            <span :class="['text-[9px] font-black px-2 py-0.5 border-custom uppercase inline-block', getLevelStyle(log.level)]">
                                {{ translateLevel(log.level) }}
                            </span>
                        </td>
                        <td class="py-4 px-6 text-[12px] font-medium text-black dark:text-white">
                            {{ log.message }}
                        </td>
                    </tr>
                </tbody>
            </table>

            <div v-if="filteredLogs.length === 0" class="py-24 text-center">
                <div class="inline-block w-8 h-8 border-2 border-custom-black border-t-transparent animate-spin mb-4"></div>
                <p class="text-[10px] font-black text-custom-gray-300 uppercase tracking-[0.4em]">等待遥测流接入 (IDLE)...</p>
            </div>
        </section>

        <footer class="pt-4 flex justify-between items-center text-[10px] font-black text-custom-gray-400 uppercase tracking-[0.2em]">
            <div class="flex gap-8">
                <div class="flex items-center gap-2">
                    <span class="text-custom-gray-300">已处理事件:</span>
                    <span class="text-custom-black italic underline">{{ filteredLogs.length }}</span>
                </div>
                <div class="flex items-center gap-2">
                    <span class="text-custom-gray-300">最后同步:</span>
                    <span class="text-custom-black italic underline">{{ syncTime }}</span>
                </div>
            </div>
            <div class="hidden md:block">
                TELEMETRY_SUBSYSTEM_V2
            </div>
        </footer>
    </div>
    `,
    data() {
        return {
            timer: null,
            syncTime: '',
            filter: { query: '', type: 'all' },
            categories: [
                { id: 'all', name: '全部' },
                { id: 'system', name: '系统' },
                { id: 'network', name: '网络' },
                { id: 'security', name: '安全' },
                { id: 'warn', name: '警报' }
            ],
            logs: [],
            save_success: false,
            save_error: false
        }
    },
    computed: {
        /**
         * 过滤后的日志列表
         */
        filteredLogs() {
            return this.logs
                .filter(log => {
                    const matchType = this.filter.type === 'all' || log.type === this.filter.type;
                    const matchQuery = !this.filter.query || 
                                       log.message.toLowerCase().includes(this.filter.query.toLowerCase());
                    return matchType && matchQuery;
                })
                .sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
        }
    },
    mounted() {
        this.fetchLogs();
        this.timer = setInterval(this.fetchLogs, 10000);
        this.updateSyncTime();
    },
    beforeUnmount() {
        if (this.timer) clearInterval(this.timer);
    },
    methods: {
        /**
         * 显示通知提示
         */
        showNotification(type) {
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
         * 更新同步时间显示
         */
        updateSyncTime() {
            this.syncTime = new Date().toLocaleTimeString('zh-CN', { hour12: false });
        },

        /**
         * 获取日志列表
         */
        async fetchLogs() {
            try {
                const res = await fetch('/api/admin/logs');
                if (res.ok) {
                    const data = await res.json();
                    this.logs = data || [];
                    this.updateSyncTime();
                }
            } catch (e) { 
                console.error("LOG_SYNC_ERROR"); 
            }
        },

        /**
         * 格式化时间显示
         * @param {string} ts - 时间戳字符串
         * @returns {string} 格式化后的时间字符串
         */
        formatTime(ts) {
            const d = new Date(ts);
            return `${d.getFullYear()}-${(d.getMonth()+1).toString().padStart(2,'0')}-${d.getDate().toString().padStart(2,'0')} ${d.getHours().toString().padStart(2,'0')}:${d.getMinutes().toString().padStart(2,'0')}:${d.getSeconds().toString().padStart(2,'0')}`;
        },

        /**
         * 翻译日志级别为中文
         * @param {string} level - 日志级别
         * @returns {string} 中文翻译
         */
        translateLevel(level) {
            const map = { 'error': '错误', 'warn': '警报', 'info': '常规', 'debug': '调试' };
            return map[level.toLowerCase()] || level;
        },

        /**
         * 获取日志级别的样式类
         * @param {string} level - 日志级别
         * @returns {string} 样式类名
         */
        getLevelStyle(level) {
            switch(level.toLowerCase()) {
                case 'error': return 'badge-red';
                case 'warn': return 'badge-orange';
                case 'info': return 'badge-black';
                default: return 'border-gray-200 text-custom-gray-400 bg-white';
            }
        },

        /**
         * 清除所有日志记录
         */
        async clearLogs() {
            if (confirm('确认操作：执行完整内存抹除程序？此操作不可撤销。')) {
                try {
                    const res = await fetch('/api/admin/logs', { method: 'DELETE' });
                    if (res.ok) {
                        this.logs = [];
                        this.updateSyncTime();
                        this.showNotification('success');
                    }
                } catch (e) { 
                    this.showNotification('error');
                    alert("指令执行失败"); 
                }
            }
        }
    }
};

window.LogsComponent = LogsComponent;