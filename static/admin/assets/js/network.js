/**
 * 网络探测管理组件
 * 管理页面：任务列表显示、拖拽排序、增删改查功能
 * 与后端handlers.go和routes.go配合，确保排序一致性
 */
const NetworkComponent = {
    props: ['site_config'],
    data() {
        return {
            tasks: [],
            available_nodes: [],
            show_modal: false,
            is_edit: false,
            drag_index: null,
            is_saving: false,
            save_success: false,
            save_error: false,
            form: {
                id: null,
                name: '',
                type: 'icmp',
                target: '',
                port: 80,
                node_ids: [],
                is_active: true
            }
        }
    },
    methods: {
        /**
         * 触发通知提示
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
         * 刷新任务列表
         * 从routes.go的GET /api/admin/tasks接口获取数据
         * 数据已按照数据库中的sort_index排序
         */
        async fetch_tasks() {
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/admin/tasks', {
                    headers: {
                        'X-Admin-Token': adminToken
                    }
                });
                if (res.ok) {
                    const data = await res.json();
                    if (this.drag_index === null) {
                        /**
                         * 后端已经按照sort_index排序，前端直接使用
                         * routes.go: DB.Order("sort_index ASC, created_at DESC")
                         */
                        this.tasks = data;
                    }
                } else if (res.status === 401) {
                    window.location.href = '/admin/login';
                }
                
                const node_res = await fetch('/api/stats');
                if (node_res.ok) this.available_nodes = await node_res.json();
            } catch (err) {
                console.error("网络请求错误:", err);
            }
        },

        /**
         * 拖拽事件：开始
         */
        handle_drag_start(index) {
            this.drag_index = index;
        },

        /**
         * 拖拽事件：悬停处理
         * 实时更新本地数组的视觉排序
         */
        handle_drag_over(index) {
            if (this.drag_index === null || this.drag_index === index) return;
            const list = [...this.tasks];
            const dragItem = list[this.drag_index];
            list.splice(this.drag_index, 1);
            list.splice(index, 0, dragItem);
            this.drag_index = index;
            this.tasks = list;
        },

        /**
         * 拖拽事件：结束
         * 将排序后的ID序列同步至服务器进行持久化
         * 调用routes.go的POST /api/admin/tasks/reorder接口
         */
        async handle_drop_end() {
            this.drag_index = null;
            const order_ids = this.tasks.map(t => t.id);
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/admin/tasks/reorder', {
                    method: 'POST',
                    headers: { 
                        'Content-Type': 'application/json', 
                        'X-Admin-Token': adminToken 
                    },
                    body: JSON.stringify({ ids: order_ids })
                });
                if (res.ok) {
                    this.show_notification('success');
                    /**
                     * 排序更新后，重新获取排序后的任务列表
                     * 确保前端显示与数据库一致
                     */
                    await this.fetch_tasks();
                } else {
                    this.show_notification('error');
                    /**
                     * 排序失败，重新获取原始排序的任务列表
                     */
                    await this.fetch_tasks();
                }
            } catch (err) {
                this.show_notification('error');
                console.error("排序持久化失败:", err);
                await this.fetch_tasks();
            }
        },

        /**
         * 准备新建任务
         */
        prepare_add_task() {
            this.is_edit = false;
            this.reset_form();
            this.show_modal = true;
        },

        /**
         * 打开编辑弹窗
         * 深拷贝任务数据至表单
         */
        open_edit(task) {
            this.is_edit = true;
            this.form = JSON.parse(JSON.stringify(task));
            if (!this.form.node_ids) this.form.node_ids = [];
            this.show_modal = true;
        },

        /**
         * 提交任务数据
         * 处理创建 (POST) 或更新 (PUT) 逻辑
         * 对应routes.go的POST /api/admin/tasks和PUT /api/admin/tasks/:id接口
         */
        async submit_task() {
            if (!this.form.name || !this.form.target) {
                alert("请输入完整信息");
                return;
            }
            
            this.is_saving = true;
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const url = this.is_edit ? `/api/admin/tasks/${this.form.id}` : '/api/admin/tasks';
                const method = this.is_edit ? 'PUT' : 'POST';

                const res = await fetch(url, {
                    method: method,
                    headers: { 
                        'Content-Type': 'application/json', 
                        'X-Admin-Token': adminToken 
                    },
                    body: JSON.stringify(this.form)
                });

                if (res.ok) {
                    this.show_notification('success');
                    this.close_modal();
                    await this.fetch_tasks();
                } else {
                    this.show_notification('error');
                    const err = await res.json();
                    alert("提交失败: " + (err.message || "未知错误"));
                }
            } catch (err) {
                this.show_notification('error');
                alert("请求服务器失败");
            } finally {
                this.is_saving = false;
            }
        },

        /**
         * 切换任务的活跃状态
         * 调用routes.go的PUT /api/admin/tasks/:id接口
         */
        async toggle_task(task) {
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const new_state = !task.is_active;
                const res = await fetch(`/api/admin/tasks/${task.id}`, {
                    method: 'PUT',
                    headers: { 
                        'Content-Type': 'application/json', 
                        'X-Admin-Token': adminToken 
                    },
                    body: JSON.stringify({ ...task, is_active: new_state })
                });
                if (res.ok) {
                    task.is_active = new_state;
                    this.show_notification('success');
                } else {
                    this.show_notification('error');
                }
            } catch (err) {
                this.show_notification('error');
                alert("更新失败");
            }
        },

        /**
         * 移除任务
         * 调用routes.go的DELETE /api/admin/tasks/:id接口
         */
        async confirm_delete(task) {
            if (!confirm(`确定移除任务 [${task.name}] 吗？`)) return;
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch(`/api/admin/tasks/${task.id}`, {
                    method: 'DELETE',
                    headers: { 'X-Admin-Token': adminToken }
                });
                if (res.ok) {
                    this.show_notification('success');
                    await this.fetch_tasks();
                } else {
                    this.show_notification('error');
                }
            } catch (err) {
                this.show_notification('error');
                alert("移除失败");
            }
        },

        /**
         * 辅助方法：获取节点展示名称
         */
        get_node_name(node_id) {
            const n = this.available_nodes.find(node => node.id === node_id);
            return n ? n.name : `节点(${node_id})`;
        },

        /**
         * 重置表单内容
         */
        reset_form() {
            this.form = {
                id: null,
                name: '',
                type: 'icmp',
                target: '',
                port: 80,
                node_ids: [],
                is_active: true
            };
        },

        /**
         * 关闭模态框
         */
        close_modal() {
            this.show_modal = false;
            this.reset_form();
        }
    },
    mounted() {
        this.fetch_tasks();
    },
    template: `
    <div class="animate-in fade-in slide-in-from-bottom-4 space-y-8 sm:space-y-10 pb-20 relative">
        
        <transition enter-active-class="transition duration-300 ease-out"
                    enter-from-class="transform translate-y-[-10px] opacity-0"
                    enter-to-class="transform translate-y-0 opacity-100"
                    leave-active-class="transition duration-200 ease-in"
                    leave-from-class="opacity-100"
                    leave-to-class="opacity-0">
            
            <div v-if="save_success || save_error" 
                 :class="[save_success ? 'notification-success' : 'notification-error']"
                 class="absolute top-[-60px] sm:top-[-100px] right-0 z-50 flex items-center gap-3">
                <span>{{ save_success ? '操作成功' : '操作失败' }}</span>
            </div>
        </transition>

        <header class="flex flex-col sm:flex-row sm:justify-between sm:items-end gap-4 mb-8 pb-4 border-b border-custom-black">
            <div>
                <p class="text-[10px] text-custom-gray-400 mt-2">
                    当前配置探测任务数: <span class="text-custom-black font-bold">{{ tasks.length }}</span>
                </p>
            </div>
            <div class="flex gap-3 flex-wrap sm:pb-1">
                <button @click="fetch_tasks" class="btn-secondary text-[10px] px-4 py-2 font-bold tracking-widest uppercase whitespace-nowrap">
                    刷新列表
                </button>
                <button @click="prepare_add_task" class="btn-primary text-[10px] px-4 py-2 font-bold tracking-widest uppercase whitespace-nowrap">
                    + 创建新任务
                </button>
            </div>
        </header>

        <div class="grid grid-cols-1 gap-4" @dragover.prevent @drop="handle_drop_end">
            <div v-for="(task, index) in tasks" :key="task.id" 
                 draggable="true"
                 @dragstart="handle_drag_start(index)"
                 @dragover.prevent="handle_drag_over(index)"
                 class="card-custom w-full p-5 bg-custom-white flex flex-col md:flex-row justify-between items-start md:items-center gap-6 relative overflow-hidden isolate cursor-move active:opacity-50 transition-all hover:shadow-md">
                
                <div class="absolute -right-2 -bottom-4 text-[40px] font-black text-gray-50 select-none uppercase italic text-custom-gray">{{ task.type }}</div>
                
                <div class="flex items-center gap-6 flex-1 w-full relative z-10">
                    <div :class="['input-custom w-12 h-12 flex flex-col items-center justify-center font-black leading-none shrink-0', 
                                 task.is_active ? 'bg-custom-green-50 text-custom-green-600' : 'bg-gray-100 text-custom-gray-400']">
                        <span class="text-[8px] uppercase tracking-tighter">{{ task.is_active ? '运行中' : '已静音' }}</span>
                    </div>
                    
                    <div class="space-y-1 min-w-0">
                        <div class="flex items-center gap-2 flex-wrap">
                            <span class="badge-black">{{ task.type }}</span>
                            <h3 class="text-base font-black tracking-wider text-custom-black truncate">{{ task.name }}</h3>
                        </div>
                        <div class="flex flex-col gap-1">
                            <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-[9px] font-mono text-custom-gray-400 uppercase tracking-tighter">
                                <span>唯一标识: {{ task.id }}</span>
                                <span class="sm:border-l border-gray-200 sm:pl-4">目标: {{ task.target }}{{ task.port ? ':' + task.port : '' }}</span>
                            </div>
                            <div class="flex flex-wrap gap-1 mt-1 items-center">
                                <span class="text-[8px] text-custom-gray-400 uppercase font-bold">执行范围:</span>
                                <span v-if="!task.node_ids || task.node_ids.length === 0" class="badge-orange">全部节点</span>
                                <template v-else>
                                    <span v-for="nid in task.node_ids" :key="nid" class="badge-blue">
                                        {{ get_node_name(nid) }}
                                    </span>
                                </template>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="flex gap-3 flex-wrap w-full md:w-auto relative z-10">
                    <button @click="open_edit(task)" 
                            class="btn-secondary text-[9px] px-4 py-2 font-black uppercase">
                        配置
                    </button>
                    <button @click="toggle_task(task)" 
                            :class="['btn-secondary text-[9px] px-4 py-2 font-black uppercase', 
                                    task.is_active ? 'text-custom-orange-500 hover' : 'text-custom-green-600 hover']">
                        {{ task.is_active ? '停用' : '启用' }}
                    </button>
                    <button @click="confirm_delete(task)" 
                            class="btn-danger text-[9px] px-4 py-2 font-black uppercase">
                        移除
                    </button>
                </div>
            </div>

            <div v-if="tasks.length === 0" class="card-custom border-dashed p-16  text-center">
                <p class="text-[10px] text-custom-gray-400 font-bold tracking-widest uppercase">暂无探测任务，点击右上角创建</p>
            </div>
        </div>

        <!-- 弹窗通过 teleport 直接挂载到 body 下，彻底脱离侧边栏/内容区容器的层级与 overflow 影响，
             确保无论祖先元素有何种 CSS（overflow、position 等），弹窗永远盖在最上层，不会被内容遮挡或裹挟。 -->
        <teleport to="body">
            <div v-if="show_modal" class="modal-overlay">
                <div class="modal-content max-w-xl">
                    <div class="mb-8 text-center">
                        <h2 class="text-xl font-bold tracking-[0.2em] uppercase text-custom-black">{{ is_edit ? '修改探测任务' : '新建探测任务' }}</h2>
                        <div class="w-12 h-1 bg-custom-gray mx-auto mt-2"></div>
                    </div>

                    <div class="space-y-6">
                        <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
                            <div class="space-y-1">
                                <label class="text-[10px] font-black text-custom-gray-400 uppercase ml-1">任务名称</label>
                                <input v-model="form.name" placeholder="如: 核心API监测" 
                                       class="input-custom w-full p-3 text-sm focus:outline-none font-bold bg-gray-50">
                            </div>
                            <div class="space-y-1">
                                <label class="text-[10px] font-black text-custom-gray-400 uppercase ml-1">探测类型</label>
                                <select v-model="form.type" 
                                        class="input-custom w-full p-3 text-sm focus:outline-none font-bold bg-gray-50 appearance-none cursor-pointer">
                                    <option value="icmp">ICMP (Ping)</option>
                                    <option value="tcp">TCP (端口检查)</option>
                                    <option value="http">HTTP (状态检查)</option>
                                </select>
                            </div>
                        </div>

                        <div class="space-y-1">
                            <label class="text-[10px] font-black text-custom-gray-400 uppercase ml-1">目标地址 (Target)</label>
                            <input v-model="form.target" placeholder="8.8.8.8 或 https://api.site.com" 
                                   class="input-custom w-full p-3 text-sm focus:outline-none font-bold bg-gray-50">
                        </div>

                        <div v-if="form.type === 'tcp'" class="space-y-1">
                            <label class="text-[10px] font-black text-custom-gray-400 uppercase ml-1">端口 (Port)</label>
                            <input v-model.number="form.port" type="number" placeholder="80" 
                                   class="input-custom w-full p-3 text-sm focus:outline-none font-bold bg-gray-50">
                        </div>

                        <div class="space-y-3">
                            <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-1">
                                <label class="text-[10px] font-black text-custom-gray-400 uppercase ml-1">指派执行节点 (可选多项)</label>
                                <span class="text-[9px] text-custom-gray-400 italic">不勾选则默认分发至所有节点</span>
                            </div>
                            <div class="input-custom grid grid-cols-1 sm:grid-cols-2 gap-2 p-4 bg-gray-50 max-h-48 overflow-y-auto">
                                <label v-for="node in available_nodes" :key="node.id" 
                                       class="flex items-center gap-3 p-2 border-transparent hover:border-gray-300 transition-all cursor-pointer">
                                    <input type="checkbox" :value="node.id" v-model="form.node_ids" class="w-4 h-4 accent-black cursor-pointer">
                                    <div class="flex flex-col min-w-0">
                                        <span class="text-[11px] font-bold leading-none text-custom-black truncate">{{ node.name }}</span>
                                        <span class="text-[8px] text-custom-gray-400 uppercase mt-1">{{ node.location || '未知位置' }}</span>
                                    </div>
                                </label>
                                <div v-if="available_nodes.length === 0" class="col-span-full py-4 text-center text-[10px] text-custom-gray-400 font-bold uppercase tracking-widest">
                                    暂无在线节点
                                </div>
                            </div>
                        </div>

                        <div class="flex flex-col sm:flex-row gap-3 sm:gap-4 pt-4">
                            <button @click="submit_task" :disabled="is_saving"
                                    class="btn-primary flex-1 text-[11px] font-black uppercase tracking-widest disabled:bg-gray-400">
                                {{ is_saving ? '处理中...' : (is_edit ? '确认修改' : '立即创建') }}
                            </button>
                            <button @click="close_modal" 
                                    class="btn-secondary flex-1 text-[11px] font-black uppercase tracking-widest">
                                取消
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </teleport>
    </div>
    `
};

window.NetworkComponent = NetworkComponent;