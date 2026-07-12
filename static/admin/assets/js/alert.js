/**
 * 告警通知管理组件
 * 核心功能：管理节点资源阈值、系统负载监控及 Telegram 推送通道
 */
const AlertComponent = {
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
                    当前状态: 
                    <span :class="config.engine.enabled ? 'text-custom-green-600' : 'text-custom-red-600'">
                        {{ config.engine.enabled ? '引擎已就绪' : '引擎已停用' }}
                    </span>
                </p>
            </div>
            <div class="flex items-center pb-1">
                <button @click="config.engine.enabled = !config.engine.enabled" 
                        :class="['text-[10px] border-custom-black px-8 py-2 font-black uppercase transition-all tracking-widest', 
                                config.engine.enabled ? 'bg-custom-black text-custom-white' : 'bg-custom-white text-custom-black']">
                    {{ config.engine.enabled ? '停用监控引擎' : '激活监控引擎' }}
                </button>
            </div>
        </header>

        <div class="card-custom bg-custom-white space-y-6">
            <div class="flex justify-between items-center border-b border-gray-100 pb-4">
                <h4 class="text-sm font-black uppercase flex items-center gap-3 text-custom-black">
                    <span class="bg-custom-black w-1.5 h-5"></span> 
                    受控节点配置
                </h4>
                <div class="flex gap-4">
                    <button @click="toggleAllNodes" class="text-[9px] font-black uppercase border-b border-custom-black hover:text-red-600 transition-colors">
                        {{ config.engine.targetNodes.length === availableNodes.length ? '取消全选' : '全选所有' }}
                    </button>
                </div>
            </div>

            <div class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
                <div v-for="node in availableNodes" :key="node.id"
                     @click="toggleNode(node.id)"
                     :class="['cursor-pointer border-custom p-3 transition-all flex flex-col gap-1',
                             config.engine.targetNodes.includes(node.id) ? 'border-custom-black bg-custom-black text-custom-white' : 'border-gray-200 bg-gray-50 text-custom-gray-400 hover:border-gray-400']">
                    <span class="text-[10px] font-black uppercase truncate">{{ node.name }}</span>
                    <span class="text-[8px] font-mono opacity-60">ID: {{ node.id }}</span>
                </div>
            </div>
            <p class="text-[9px] text-custom-gray-400 font-bold uppercase">
                注意: 仅选中的节点在达到阈值时会触发推送任务。当前已选中: {{ config.engine.targetNodes.length }} 个节点。
            </p>
        </div>
        
        <div class="card-custom bg-custom-white space-y-10">
            <div class="flex justify-between items-center border-b border-gray-100 pb-4">
                <h4 class="text-sm font-black uppercase flex items-center gap-3 text-custom-black">
                    <span class="bg-custom-red-600 w-1.5 h-5"></span> 
                    资源监控配置
                </h4>
            </div>
            
            <div class="space-y-8">
                <label class="text-[10px] font-black uppercase tracking-widest text-custom-gray-500 block">基础资源阈值</label>
                <div class="grid grid-cols-1 md:grid-cols-3 gap-12">
                    <div class="space-y-4">
                        <div class="flex justify-between items-center">
                            <span class="text-[10px] font-black uppercase text-custom-gray-400">CPU占用</span>
                            <span class="text-lg font-black text-custom-red-600">{{ config.metrics.cpu }}%</span>
                        </div>
                        <input type="range" v-model="config.metrics.cpu" min="1" max="100" class="w-full h-[2px] bg-gray-200 appearance-none accent-black cursor-pointer">
                    </div>

                    <div class="space-y-4">
                        <div class="flex justify-between items-center">
                            <span class="text-[10px] font-black uppercase text-custom-gray-400">内存占用</span>
                            <span class="text-lg font-black text-custom-red-600">{{ config.metrics.memory }}%</span>
                        </div>
                        <input type="range" v-model="config.metrics.memory" min="1" max="100" class="w-full h-[2px] bg-gray-200 appearance-none accent-black cursor-pointer">
                    </div>

                    <div class="space-y-4">
                        <div class="flex justify-between items-center">
                            <span class="text-[10px] font-black uppercase text-custom-gray-400">硬盘占用</span>
                            <span class="text-lg font-black text-custom-red-600">{{ config.metrics.disk }}%</span>
                        </div>
                        <input type="range" v-model="config.metrics.disk" min="1" max="100" class="w-full h-[2px] bg-gray-200 appearance-none accent-black cursor-pointer">
                    </div>
                </div>
            </div>

            <div class="space-y-8 pt-6 border-t border-gray-100">
                <label class="text-[10px] font-black uppercase tracking-widest text-custom-gray-500 block">系统核心指标</label>
                <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                    <div class="p-4 bg-gray-50 border border-gray-100">
                        <span class="text-[9px] font-black text-custom-gray-400 uppercase block mb-2">系统负载阈值</span>
                        <input type="number" step="0.1" v-model="config.metrics.load" 
                               class="input-custom w-full bg-transparent border-b border-gray-200 text-sm font-black text-custom-black outline-none focus:border-custom-black transition-colors"
                               placeholder="默认值: 4.0">
                        <p class="text-[8px] text-custom-gray-400 mt-1">正常范围: 小于CPU核心数</p>
                    </div>
                    
                    <div class="p-4 bg-gray-50 border border-gray-100">
                        <span class="text-[9px] font-black text-custom-gray-400 uppercase block mb-2">Inode使用率(%)</span>
                        <input type="number" v-model="config.metrics.inode" 
                               class="input-custom w-full bg-transparent border-b border-gray-200 text-sm font-black text-custom-black outline-none focus:border-custom-black transition-colors"
                               placeholder="默认值: 85">
                        <p class="text-[8px] text-custom-gray-400 mt-1">警戒线: 85%</p>
                    </div>
                </div>
            </div>

            <div class="space-y-8 pt-6 border-t border-gray-100">
                <label class="text-[10px] font-black uppercase tracking-widest text-custom-gray-500 block">网络与流量监控</label>
                <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
                    <div class="p-4 bg-gray-50 border border-gray-100">
                        <span class="text-[9px] font-black text-custom-gray-400 uppercase block mb-2">连接数阈值</span>
                        <input type="number" v-model="config.metrics.connections" 
                               class="input-custom w-full bg-transparent border-b border-gray-200 text-sm font-black text-custom-black outline-none focus:border-custom-black transition-colors"
                               placeholder="默认值: 5000">
                        <p class="text-[8px] text-custom-gray-400 mt-1">正常范围: 几十到几百</p>
                    </div>
                    
                    <div class="p-4 bg-gray-50 border border-gray-100">
                        <span class="text-[9px] font-black text-custom-gray-400 uppercase block mb-2">月度上传(GB)</span>
                        <input type="number" v-model="config.metrics.monthUp" 
                               class="input-custom w-full bg-transparent border-b border-gray-200 text-sm font-black text-custom-black outline-none focus:border-custom-black transition-colors"
                               placeholder="默认值: 1000">
                        <p class="text-[8px] text-custom-gray-400 mt-1">流量配额监控</p>
                    </div>
                    
                    <div class="p-4 bg-gray-50 border border-gray-100">
                        <span class="text-[9px] font-black text-custom-gray-400 uppercase block mb-2">月度下载(GB)</span>
                        <input type="number" v-model="config.metrics.monthDown" 
                               class="input-custom w-full bg-transparent border-b border-gray-200 text-sm font-black text-custom-black outline-none focus:border-custom-black transition-colors"
                               placeholder="默认值: 2000">
                        <p class="text-[8px] text-custom-gray-400 mt-1">流量配额监控</p>
                    </div>
                </div>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-1 gap-16 pt-8 border-t border-gray-100">
                <div class="space-y-6">
                    <label class="text-[10px] font-black uppercase tracking-widest text-custom-gray-500 block mb-4">应用服务监控</label>
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-4">
                        <div class="p-4 bg-gray-50 border border-gray-100">
                            <span class="text-[9px] font-black text-custom-gray-400 uppercase block mb-2">关键进程名称</span>
                            <input type="text" v-model="config.metrics.processName" 
                                   class="input-custom w-full bg-transparent border-b border-gray-200 text-sm font-black text-custom-black outline-none focus:border-custom-black transition-colors"
                                   placeholder="例如: nginx, mysqld">
                            <p class="text-[8px] text-custom-gray-400 mt-1">监控指定进程是否存在</p>
                        </div>
                        
                        <div class="p-4 bg-gray-50 border border-gray-100">
                            <span class="text-[9px] font-black text-custom-gray-400 uppercase block mb-2">端口服务检测</span>
                            <input type="text" v-model="config.metrics.portService" 
                                   class="input-custom w-full bg-transparent border-b border-gray-200 text-sm font-black text-custom-black outline-none focus:border-custom-black transition-colors"
                                   placeholder="例如: 80, 443, 3306">
                            <p class="text-[8px] text-custom-gray-400 mt-1">监控指定端口是否开放</p>
                        </div>
                    </div>
                </div>
            </div>

            <div class="space-y-6 pt-8 border-t border-gray-100">
                <label class="text-[10px] font-black uppercase tracking-widest text-custom-gray-500 block mb-4">状态实时监测</label>
                <div class="grid grid-cols-1 md:grid-cols-4 gap-x-8 gap-y-4">
                    <label class="flex justify-between items-center cursor-pointer group text-custom-black">
                        <span class="text-[10px] font-bold uppercase group-hover:text-red-600 transition-colors">节点通信断开预警</span>
                        <input type="checkbox" v-model="config.switches.nodeOffline" class="w-4 h-4 accent-black">
                    </label>
                    <label class="flex justify-between items-center cursor-pointer group text-custom-black">
                        <span class="text-[10px] font-bold uppercase group-hover:text-red-600 transition-colors">服务重启检测</span>
                        <input type="checkbox" v-model="config.switches.serviceRestart" class="w-4 h-4 accent-black">
                    </label>
                    <label class="flex justify-between items-center cursor-pointer group text-custom-black">
                        <span class="text-[10px] font-bold uppercase group-hover:text-red-600 transition-colors">SWAP使用率监控</span>
                        <input type="checkbox" v-model="config.switches.swap" class="w-4 h-4 accent-black">
                    </label>
                    <label class="flex justify-between items-center cursor-pointer group text-custom-black">
                        <span class="text-[10px] font-bold uppercase group-hover:text-red-600 transition-colors">时间偏移检测</span>
                        <input type="checkbox" v-model="config.switches.timeDiff" class="w-4 h-4 accent-black">
                    </label>
                </div>
            </div>
        </div>

        <div class="card-custom bg-custom-white space-y-6">
            <div class="flex justify-between items-center border-b border-gray-100 pb-4">
                <h4 class="text-sm font-black uppercase flex items-center gap-3 text-custom-black">
                    <span class="bg-gray-600 w-1.5 h-5"></span> 
                    推送策略配置
                </h4>
            </div>
            
            <div class="grid grid-cols-1 md:grid-cols-3 gap-8">
                <div class="space-y-3">
                    <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-widest block">重复推送间隔 (分钟)</label>
                    <div class="h-[48px] flex items-center">
                        <input type="number" v-model="config.engine.rateLimit" 
                               class="input-custom bg-transparent text-center font-black text-sm outline-none focus:text-red-600 transition-colors w-full h-full">
                    </div>
                </div>
                
                <div class="space-y-3 md:col-span-2">
                    <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-widest block">静默推送时间段</label>
                    <div class="grid grid-cols-2 gap-4">
                        <div class="flex items-center gap-3 bg-gray-50 p-3 border border-gray-100 h-[48px]">
                            <span class="text-[9px] font-black text-custom-gray-400 uppercase">开始时间</span>
                            <input type="time" v-model="config.engine.quietStart" class="bg-transparent font-black text-xs outline-none flex-1 h-full">
                        </div>
                        <div class="flex items-center gap-3 bg-gray-50 p-3 border border-gray-100 h-[48px]">
                            <span class="text-[9px] font-black text-custom-gray-400 uppercase">结束时间</span>
                            <input type="time" v-model="config.engine.quietEnd" class="bg-transparent font-black text-xs outline-none flex-1 h-full">
                        </div>
                    </div>
                    <p class="text-[8px] text-custom-gray-400 font-bold uppercase mt-1">
                        静默时段内仅发送紧急告警
                    </p>
                </div>
            </div>
        </div>

        <div class="card-custom bg-custom-white relative overflow-hidden group">
            <div class="absolute -right-4 -top-6 text-[100px] font-black text-gray-50 select-none group-hover:text-blue-50/50 transition-colors pointer-events-none">TG</div>
            <div class="relative z-10">
                <div class="flex justify-between items-center mb-10">
                    <h4 class="text-sm font-black uppercase flex items-center gap-3 text-custom-black">
                        <span class="bg-custom-blue-500 w-1.5 h-5"></span> 
                        TELEGRAM机器人推送配置
                    </h4>
                    <div class="flex items-center gap-2">
                        <span :class="['text-[9px] font-black px-2 py-0.5 uppercase tracking-tighter transition-all', 
                                       config.channels.telegram.enabled ? 'text-custom-blue-500' : 'text-custom-gray-300']">
                            {{ config.channels.telegram.enabled ? '已启用' : '已停用' }}
                        </span>
                        <label class="relative inline-flex items-center cursor-pointer">
                            <input type="checkbox" v-model="config.channels.telegram.enabled" class="sr-only peer">
                            <div class="w-9 h-5 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-blue-500"></div>
                        </label>
                    </div>
                </div>

                <div class="grid grid-cols-1 md:grid-cols-2 gap-8 items-end">
                    <div class="space-y-2">
                        <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-[0.2em] flex items-center gap-2">
                            机器人 接口令牌(API TOKEN) <span class="text-custom-red-600">*</span>
                        </label>
                        <input type="password" v-model="config.channels.telegram.botToken" 
                                class="input-custom w-full p-4 font-mono text-xs focus:bg-gray-50 outline-none transition-all h-[52px]"
                                placeholder="请输入 Telegram 机器人令牌...">
                    </div>
                    <div class="space-y-2">
                        <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-[0.2em]">对话标识符(CHAT ID)</label>
                        <div class="flex gap-3">
                            <input type="text" v-model="config.channels.telegram.chatId" 
                                    class="input-custom flex-1 p-4 font-mono text-xs focus:bg-gray-50 outline-none transition-all h-[52px]"
                                    placeholder="例如: -100123456789">
                            <button @click="test_warn_push" :disabled="!config.channels.telegram.botToken || !config.channels.telegram.chatId"
                                    class="btn-primary px-8 text-[10px] font-black uppercase hover:bg-white hover:text-black transition-all disabled:opacity-30 disabled:cursor-not-allowed h-[52px]">
                                发送测试
                            </button>
                        </div>
                    </div>
                </div>
                <div class="h-4 mt-2">
                    <p v-if="warnTestSuccess" class="text-[10px] font-black text-custom-green-600 uppercase animate-pulse">
                        成功：测试消息已成功推送至终端。
                    </p>
                </div>
            </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <button @click="save_warn_config" 
                    class="btn-primary py-4 text-[11px] font-black uppercase tracking-[0.2em] shadow-[4px_4px_0px_rgba(0,0,0,0.1)]">
                应用并持久化预警配置
            </button>
            <button @click="reset_warn_config" 
                    class="btn-secondary py-4 text-[11px] font-black uppercase tracking-[0.2em] shadow-[4px_4px_0px_rgba(0,0,0,0.05)]">
                恢复系统默认规则
            </button>
        </div>

        <div class="card-custom bg-custom-gray-50 p-6">
            <h5 class="text-[9px] font-black text-custom-gray-400 uppercase mb-4 tracking-widest flex items-center gap-2">
                <span class="w-1 h-1 bg-custom-gray-400 rounded-full"></span> 
                推送数据载荷预览
            </h5>
            <pre class="text-[10px] font-mono leading-relaxed text-custom-gray-600 overflow-x-auto">
[监控告警] {{ site_config.site_name || '生产环境边缘节点' }}
---------------------------
节点状态: {{ config.engine.enabled ? '监控中' : '已停用' }}
生效节点: {{ config.engine.targetNodes.length }} 台主机
告警间隔: {{ config.engine.rateLimit }} 分钟
静默时段: {{ config.engine.quietStart }} - {{ config.engine.quietEnd }}

关键指标:
- CPU使用率: {{ config.metrics.cpu }}%
- 内存使用率: {{ config.metrics.memory }}%
- 磁盘使用率: {{ config.metrics.disk }}%
- 系统负载: {{ config.metrics.load }}
- Inode使用率: {{ config.metrics.inode }}%
- 连接数阈值: {{ config.metrics.connections }}
- 月度上传配额: {{ config.metrics.monthUp }}GB
- 月度下载配额: {{ config.metrics.monthDown }}GB
- 监控进程: {{ config.metrics.processName }}
- 监控端口: {{ config.metrics.portService }}

监控状态:
- 节点离线检测: {{ config.switches.nodeOffline ? '开启' : '关闭' }}
- 服务重启检测: {{ config.switches.serviceRestart ? '开启' : '关闭' }}
- SWAP使用率监控: {{ config.switches.swap ? '开启' : '关闭' }}
- 时间偏移检测: {{ config.switches.timeDiff ? '开启' : '关闭' }}

触发时间: {{ new Date().toLocaleString() }}
建议操作: 请登录服务器查看详细状态
---------------------------
            </pre>
        </div>
    </div>
    `,
    data() {
        return {
            availableNodes: [],
            save_success: false,
            save_error: false,
            config: {
                engine: {
                    enabled: true,
                    rateLimit: 10,
                    quietStart: '01:00',
                    quietEnd: '06:00',
                    targetNodes: []
                },
                metrics: {
                    cpu: 90,
                    memory: 85,
                    disk: 90,
                    load: 4.0,
                    inode: 85,
                    connections: 5000,
                    monthUp: 1000,
                    monthDown: 2000,
                    processName: '',
                    portService: ''
                },
                switches: {
                    nodeOffline: true,
                    serviceRestart: true,
                    swap: true,
                    timeDiff: true
                },
                channels: {
                    telegram: {
                        enabled: false,
                        botToken: '',
                        chatId: ''
                    }
                }
            },
            warnTestSuccess: false
        }
    },
    methods: {
        /**
         * 显示通知提示
         * 根据传入类型显示成功或失败的状态条
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
         * 获取节点列表
         */
        async fetch_nodes() {
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/stats', {
                    headers: {
                        'X-Admin-Token': adminToken
                    }
                });
                if (res.ok) {
                    const data = await res.json();
                    this.availableNodes = data.nodes || [];
                }
            } catch (err) {
                console.error("加载节点列表失败:", err);
            }
        },

        /**
         * 切换节点的选中状态
         * @param {number} nodeId - 节点ID
         */
        toggleNode(nodeId) {
            const index = this.config.engine.targetNodes.indexOf(nodeId);
            if (index > -1) {
                this.config.engine.targetNodes.splice(index, 1);
            } else {
                this.config.engine.targetNodes.push(nodeId);
            }
        },

        /**
         * 切换所有节点的选中状态
         */
        toggleAllNodes() {
            if (this.config.engine.targetNodes.length === this.availableNodes.length) {
                this.config.engine.targetNodes = [];
            } else {
                this.config.engine.targetNodes = this.availableNodes.map(n => n.id);
            }
        },

        /**
         * 获取告警配置
         */
        async fetch_warn_config() {
            try {
                const res = await fetch('/api/admin/alert/config');
                if (res.ok) {
                    const data = await res.json();
                    if (!data.engine.targetNodes) data.engine.targetNodes = [];
                    
                    this.config = {
                        engine: {
                            enabled: data.engine.enabled !== undefined ? data.engine.enabled : true,
                            rateLimit: data.engine.rateLimit || 10,
                            quietStart: data.engine.quietStart || '01:00',
                            quietEnd: data.engine.quietEnd || '06:00',
                            targetNodes: data.engine.targetNodes || []
                        },
                        metrics: {
                            cpu: data.metrics?.cpu || 90,
                            memory: data.metrics?.memory || 85,
                            disk: data.metrics?.disk || 90,
                            load: data.metrics?.load || 4.0,
                            inode: data.metrics?.inode || 85,
                            connections: data.metrics?.connections || 5000,
                            monthUp: data.metrics?.monthUp || 1000,
                            monthDown: data.metrics?.monthDown || 2000,
                            processName: data.metrics?.processName || '',
                            portService: data.metrics?.portService || ''
                        },
                        switches: {
                            nodeOffline: data.switches?.nodeOffline !== undefined ? data.switches.nodeOffline : true,
                            serviceRestart: data.switches?.serviceRestart !== undefined ? data.switches.serviceRestart : true,
                            swap: data.switches?.swap !== undefined ? data.switches.swap : true,
                            timeDiff: data.switches?.timeDiff !== undefined ? data.switches.timeDiff : true
                        },
                        channels: {
                            telegram: {
                                enabled: data.channels?.telegram?.enabled || false,
                                botToken: data.channels?.telegram?.botToken || '',
                                chatId: data.channels?.telegram?.chatId || ''
                            }
                        }
                    };
                }
            } catch (err) {
                console.error("同步配置失败:", err);
            }
        },

        /**
         * 准备保存数据
         */
        prepare_payload() {
            return {
                id: Number(this.config.id) || 1,
                engine: {
                    enabled: Boolean(this.config.engine.enabled),
                    rateLimit: parseInt(this.config.engine.rateLimit) || 0,
                    quietStart: String(this.config.engine.quietStart || "00:00"),
                    quietEnd: String(this.config.engine.quietEnd || "00:00"),
                    targetNodes: Array.isArray(this.config.engine.targetNodes) ? this.config.engine.targetNodes : []
                },
                metrics: {
                    cpu: parseFloat(this.config.metrics.cpu) || 0,
                    memory: parseFloat(this.config.metrics.memory) || 0,
                    disk: parseFloat(this.config.metrics.disk) || 0,
                    load: parseFloat(this.config.metrics.load) || 0,
                    inode: parseFloat(this.config.metrics.inode) || 0,
                    connections: parseInt(this.config.metrics.connections) || 0,
                    monthUp: parseInt(this.config.metrics.monthUp) || 0,
                    monthDown: parseInt(this.config.metrics.monthDown) || 0,
                    processName: String(this.config.metrics.processName || ""),
                    portService: String(this.config.metrics.portService || "")
                },
                switches: {
                    nodeOffline: Boolean(this.config.switches.nodeOffline),
                    serviceRestart: Boolean(this.config.switches.serviceRestart),
                    swap: Boolean(this.config.switches.swap),
                    timeDiff: Boolean(this.config.switches.timeDiff)
                },
                channels: {
                    telegram: {
                        enabled: Boolean(this.config.channels.telegram.enabled),
                        botToken: String(this.config.channels.telegram.botToken || ""),
                        chatId: String(this.config.channels.telegram.chatId || "")
                    }
                }
            };
        },

        /**
         * 保存告警配置
         */
        async save_warn_config() {
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const payload = this.prepare_payload();
                
                const res = await fetch('/api/admin/alert/config', {
                    method: 'POST',
                    headers: { 
                        'Content-Type': 'application/json', 
                        'X-Admin-Token': adminToken 
                    },
                    body: JSON.stringify(payload)
                });
                
                if (res.ok) {
                    this.show_notification('success');
                } else {
                    this.show_notification('error');
                    const errorData = await res.json();
                    alert("保存失败: " + (errorData.message || "后端解析失败，请检查参数格式"));
                }
            } catch (err) {
                this.show_notification('error');
                alert("保存配置时发生网络故障");
            }
        },

        /**
         * 重置告警配置
         */
        reset_warn_config() {
            if (!confirm("确定要恢复所有参数至系统预设值吗？")) return;
            this.config.metrics = {
                cpu: 90,
                memory: 85,
                disk: 90,
                load: 4.0,
                inode: 85,
                connections: 5000,
                monthUp: 1000,
                monthDown: 2000,
                processName: '',
                portService: ''
            };
            this.config.switches = {
                nodeOffline: true,
                serviceRestart: true,
                swap: true,
                timeDiff: true
            };
            this.config.engine.rateLimit = 10;
            this.config.engine.quietStart = '01:00';
            this.config.engine.quietEnd = '06:00';
            this.config.engine.targetNodes = this.availableNodes.map(n => n.id);
            this.show_notification('success');
        },

        /**
         * 测试推送功能
         */
        async test_warn_push() {
            this.warnTestSuccess = false;
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const payload = this.prepare_payload();

                const res = await fetch('/api/admin/alert/test', {
                    method: 'POST',
                    headers: { 
                        'Content-Type': 'application/json', 
                        'X-Admin-Token': adminToken 
                    },
                    body: JSON.stringify(payload)
                });
                
                if (res.ok) {
                    this.warnTestSuccess = true;
                    this.show_notification('success');
                    setTimeout(() => { this.warnTestSuccess = false }, 5000);
                } else {
                    this.show_notification('error');
                    const data = await res.json();
                    alert("通道验证失败: " + (data.message || "请核对 Token"));
                }
            } catch (err) {
                this.show_notification('error');
                alert("无法连接推送网关");
            }
        }
    },
    mounted() {
        this.fetch_nodes();
        this.fetch_warn_config();
    }
};

window.AlertComponent = AlertComponent;