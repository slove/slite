/**
 * 节点管理组件 (NodesComponent)
 * 处理监控节点的增删改查、排序、部署脚本生成以及状态实时监控
 * 支持节点分组(Group)的创建、关联、重命名与维护
 */
const NodesComponent = {
    props: ['site_config', 'is_saving', 'save_success'],
    data() {
        return {
            node_list: [],
            group_list: [],
            show_add_modal: false,
            show_edit_modal: false,
            show_group_modal: false,
            server_origin: window.location.origin,
            new_node: { node_id: '', name: '', group_id: 1 },
            edit_form: { id: '', name: '', location: '', is_visible: true, group_id: 1 },
            new_group: { name: '' },
            editing_group_id: null,
            editing_group_name: '',
            refresh_timer: null,
            drag_index: null,
            save_error: false,
            copy_success: false,
            copy_error: false
        }
    },
    methods: {
        /**
         * 异步获取节点和分组数据
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
                    if (this.drag_index === null) {
                        this.node_list = data.nodes || [];
                        this.group_list = data.groups || [];
                    }
                }
            } catch (err) {
                console.error("获取数据失败:", err);
            }
        },

        /**
         * 根据分组ID获取分组名称
         * @param {number} groupId - 分组ID
         * @returns {string} 分组名称
         */
        get_group_name(groupId) {
            const group = this.group_list.find(g => g.id === groupId);
            return group ? group.name : '默认分组';
        },

        /**
         * 创建新分组
         */
        async create_group() {
            if (!this.new_group.name.trim()) return;
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/admin/groups', {
                    method: 'POST',
                    headers: { 
                        'Content-Type': 'application/json',
                        'X-Admin-Token': adminToken
                    },
                    body: JSON.stringify(this.new_group)
                });
                if (res.ok) {
                    this.new_group.name = '';
                    this.$emit('action', 'triggerSuccess');
                    await this.fetch_nodes();
                }
            } catch (err) {
                this.save_error = true;
            }
        },

        /**
         * 开始编辑分组名称
         * @param {Object} group - 分组对象
         */
        start_edit_group(group) {
            this.editing_group_id = group.id;
            this.editing_group_name = group.name;
        },

        /**
         * 保存分组名称修改
         * @param {number} groupId - 分组ID
         */
        async save_group_name(groupId) {
            if (!this.editing_group_name.trim()) return;
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch(`/api/admin/groups/${groupId}`, {
                    method: 'PUT',
                    headers: { 
                        'Content-Type': 'application/json',
                        'X-Admin-Token': adminToken
                    },
                    body: JSON.stringify({ name: this.editing_group_name })
                });
                if (res.ok) {
                    this.editing_group_id = null;
                    this.$emit('action', 'triggerSuccess');
                    await this.fetch_nodes();
                }
            } catch (err) {
                this.save_error = true;
            }
        },

        /**
         * 删除分组
         * @param {number} groupId - 分组ID
         */
        async delete_group(groupId) {
            if (groupId === 1) {
                alert("默认分组不可删除");
                return;
            }
            if (!confirm("确定删除该分组吗？属于该分组的节点将自动移至默认分组。")) return;
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch(`/api/admin/groups/${groupId}`, {
                    method: 'DELETE',
                    headers: { 'X-Admin-Token': adminToken }
                });
                if (res.ok) {
                    this.$emit('action', 'triggerSuccess');
                    await this.fetch_nodes();
                }
            } catch (err) {
                this.save_error = true;
            }
        },

        /**
         * 开始拖拽节点
         * @param {number} index - 节点索引
         */
        handle_drag_start(index) {
            this.drag_index = index;
        },

        /**
         * 处理拖拽经过事件
         * @param {number} index - 目标位置索引
         */
        handle_drag_over(index) {
            if (this.drag_index === null || this.drag_index === index) return;
            const list = [...this.node_list];
            const dragItem = list[this.drag_index];
            list.splice(this.drag_index, 1);
            list.splice(index, 0, dragItem);
            this.drag_index = index;
            this.node_list = list;
        },

        /**
         * 处理拖拽结束事件
         */
        async handle_drop_end() {
            this.drag_index = null;
            this.save_error = false;
            const order_ids = this.node_list.map(n => n.id);
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/admin/nodes/reorder', {
                    method: 'POST',
                    headers: { 
                        'Content-Type': 'application/json',
                        'X-Admin-Token': adminToken
                    },
                    body: JSON.stringify({ ids: order_ids })
                });
                if (res.ok) {
                    this.$emit('action', 'triggerSuccess');
                } else {
                    this.save_error = true;
                }
            } catch (err) {
                this.save_error = true;
                console.error("保存排序失败:", err);
            }
        },

        /**
         * 准备添加新节点，获取节点ID
         */
        async prepare_add_node() {
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/admin/nodes/next-id', {
                    headers: { 'X-Admin-Token': adminToken }
                });
                if (res.ok) {
                    const data = await res.json();
                    this.new_node.node_id = data.id; 
                    this.new_node.name = ''; 
                    this.new_node.group_id = this.group_list.length > 0 ? this.group_list[0].id : 1;
                    this.show_add_modal = true;
                    
                    const simulatedNode = {
                        id: data.id,
                        name: '等待部署...',
                        group_id: this.new_node.group_id,
                        online: false,
                        ip: '0.0.0.0',
                        location: '等待上报...',
                        is_visible: true
                    };
                    
                    this.node_list.unshift(simulatedNode);
                }
            } catch (err) {
                console.error("无法获取部署 ID");
            }
        },

        /**
         * 生成Linux部署命令
         * @param {boolean} isEdit - 是否为编辑模式
         * @returns {string} Linux部署命令
         */
        gen_linux_cmd(isEdit = false) {
            const id = isEdit ? this.edit_form.id : this.new_node.node_id;
            const name = isEdit ? (this.edit_form.name || 'Node_' + id) : (this.new_node.name || 'Node_' + id);
            return `curl -fsSL ${this.server_origin}/install.sh | bash -s -- --id ${id} --name '${name}' --key ${id} --url ${this.server_origin}`;
        },

        /**
         * 生成Windows部署命令
         * @param {boolean} isEdit - 是否为编辑模式
         * @returns {string} Windows部署命令
         */
        gen_win_cmd(isEdit = false) {
            const id = isEdit ? this.edit_form.id : this.new_node.node_id;
            const name = isEdit ? (this.edit_form.name || 'Node_' + id) : (this.new_node.name || 'Node_' + id);
            return `Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('${this.server_origin}/install.sh')) --id ${id} --name '${name}' --key ${id} --url ${this.server_origin}`;
        },

        /**
         * 复制文本到剪贴板
         * @param {string} text - 要复制的文本
         */
        copy_text(text) {
            this.copy_success = false;
            this.copy_error = false;
            
            if (navigator.clipboard && window.isSecureContext) {
                navigator.clipboard.writeText(text).then(() => {
                    this.copy_success = true;
                    setTimeout(() => {
                        this.copy_success = false;
                    }, 2000);
                }).catch(() => {
                    this.copy_error = true;
                    setTimeout(() => {
                        this.copy_error = false;
                    }, 2000);
                });
            } else {
                const input = document.createElement('textarea');
                input.value = text;
                document.body.appendChild(input);
                input.select();
                try {
                    document.execCommand('copy');
                    this.copy_success = true;
                    setTimeout(() => {
                        this.copy_success = false;
                    }, 2000);
                } catch (err) {
                    this.copy_error = true;
                    setTimeout(() => {
                        this.copy_error = false;
                    }, 2000);
                }
                document.body.removeChild(input);
            }
        },

        /**
         * 打开节点编辑对话框
         * @param {Object} node - 节点对象
         */
        open_edit(node) {
            this.edit_form = { 
                id: node.id, 
                name: node.name, 
                location: node.location || '',
                is_visible: node.hasOwnProperty('is_visible') ? node.is_visible : true,
                group_id: node.group_id || 1
            };
            this.show_edit_modal = true;
        },

        /**
         * 保存节点修改
         */
        async save_node() {
            this.save_error = false;
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/admin/nodes/update', {
                    method: 'POST',
                    headers: { 
                        'Content-Type': 'application/json',
                        'X-Admin-Token': adminToken
                    },
                    body: JSON.stringify(this.edit_form)
                });
                
                if (res.ok) {
                    this.show_edit_modal = false;
                    this.$emit('action', 'triggerSuccess');
                    await this.fetch_nodes();
                } else {
                    this.save_error = true;
                }
            } catch (err) {
                this.save_error = true;
            }
        },

        /**
         * 确认删除节点
         * @param {Object} node - 节点对象
         */
        async confirm_delete(node) {
            if (!confirm(`确定永久移除节点 [${node.name}] 吗？\n此操作不可撤销！`)) return;
            try {
                const adminToken = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch(`/api/admin/nodes/delete/${node.id}`, { 
                    method: 'DELETE',
                    headers: { 'X-Admin-Token': adminToken }
                });
                if (res.ok) {
                    this.$emit('action', 'triggerSuccess');
                    await this.fetch_nodes();
                }
            } catch (err) {
                this.save_error = true;
            }
        }
    },
    mounted() {
        this.fetch_nodes();
        this.refresh_timer = setInterval(this.fetch_nodes, 5000);
    },
    beforeUnmount() {
        if (this.refresh_timer) {
            clearInterval(this.refresh_timer);
        }
    },
    template: `
    <div class="animate-in fade-in slide-in-from-bottom-4 space-y-10 pb-20 relative">
        
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

        <transition enter-active-class="transition duration-300 ease-out"
                    enter-from-class="transform translate-y-[-10px] opacity-0"
                    enter-to-class="transform translate-y-0 opacity-100"
                    leave-active-class="transition duration-200 ease-in"
                    leave-from-class="opacity-100"
                    leave-to-class="opacity-0">
            
            <div v-if="copy_success" 
                 class="notification-success absolute top-[-100px] right-0 z-50 flex items-center gap-3">
                <span>脚本已复制到剪贴板</span>
            </div>
        </transition>

        <transition enter-active-class="transition duration-300 ease-out"
                    enter-from-class="transform translate-y-[-10px] opacity-0"
                    enter-to-class="transform translate-y-0 opacity-100"
                    leave-active-class="transition duration-200 ease-in"
                    leave-from-class="opacity-100"
                    leave-to-class="opacity-0">
            
            <div v-if="copy_error" 
                 class="notification-error absolute top-[-100px] right-0 z-50 flex items-center gap-3">
                <span>复制失败，请手动选择复制</span>
            </div>
        </transition>

        <header class="flex justify-between items-end mb-8 pb-4 border-b border-custom-black">
            <div>
                <p class="text-[10px] text-custom-gray-400 mt-2 uppercase tracking-widest">
                    节点总数: <span class="text-custom-black font-bold">{{ node_list.length }}</span> | 
                    分组数: <span class="text-custom-black font-bold">{{ group_list.length }}</span>
                </p>
            </div>
            <div class="flex gap-3 pb-1">
                <button @click="show_group_modal = true" class="btn-secondary text-[10px] px-4 py-2 font-bold tracking-widest uppercase">
                    分组管理
                </button>
                <button @click="prepare_add_node" class="btn-primary text-[10px] px-4 py-2 font-bold tracking-widest uppercase">
                    + 部署新节点
                </button>
            </div>
        </header>

        <div class="space-y-4" @dragover.prevent @drop="handle_drop_end">
            <div v-for="(node, index) in node_list" :key="node.id" 
                 draggable="true"
                 @dragstart="handle_drag_start(index)"
                 @dragover.prevent="handle_drag_over(index)"
                 class="card-custom transition-all flex flex-col md:flex-row justify-between items-center gap-6 cursor-move active:opacity-50 hover:shadow-md">
                
                <div class="flex items-center gap-6 flex-1 w-full">
                    <div :class="['input-custom w-10 h-10 flex items-center justify-center font-black text-[10px]', 
                                 node.online ? 'text-custom-green-600' : 'text-custom-red-600']">
                        {{ node.online ? '在线' : '离线' }}
                    </div>
                    
                    <div class="space-y-1">
                        <div class="flex items-center gap-2">
                            <span class="badge-gray">#{{ index + 1 }}</span>
                            <h3 class="text-base font-black tracking-wider text-custom-black">{{ node.name }}</h3>
                            <span class="badge-black uppercase text-[8px]">{{ get_group_name(node.group_id) }}</span>
                            <span v-if="!node.is_visible" class="badge-black">访客隐藏</span>
                        </div>
                        <div class="flex items-center gap-4 text-[9px] font-mono text-custom-gray-400 uppercase tracking-tighter">
                            <span>UID: {{ node.id }}</span>
                            <span class="border-l border-gray-200 pl-4" :class="{'text-custom-orange-500': !node.ip || node.ip === '0.0.0.0'}">
                                IP: {{ (node.ip && node.ip !== '0.0.0.0') ? node.ip : '等待上报...' }}
                            </span>
                        </div>
                    </div>
                </div>

                <div class="flex flex-col items-center md:items-start gap-1 w-32">
                    <span class="text-[9px] font-black text-custom-gray-300 tracking-[0.2em] uppercase">位置</span>
                    <span class="text-xs font-bold text-custom-black">{{ node.location || '自动上报' }}</span>
                </div>

                <div class="flex gap-3 w-full md:w-auto">
                    <button @click="open_edit(node)" 
                            class="btn-secondary flex-1 md:flex-none text-[9px] px-4 py-2 font-black uppercase">
                        配置
                    </button>
                    <button @click="confirm_delete(node)" 
                            class="btn-danger flex-1 md:flex-none text-[9px] px-4 py-2 font-black uppercase">
                        移除
                    </button>
                </div>
            </div>

            <div v-if="node_list.length === 0" class="card-custom border-dashed p-16 text-center">
                <p class="text-[10px] text-custom-gray-400 font-bold tracking-widest uppercase">当前无活跃节点，请点击右上角新增部署</p>
            </div>
        </div>

        <div v-if="show_group_modal" class="modal-overlay">
            <div class="modal-content max-w-lg">
                <div class="mb-10 flex justify-between items-start">
                    <div>
                        <h2 class="text-2xl font-bold tracking-[0.2em] mb-1 text-custom-black uppercase">分组管理</h2>
                        <div class="w-12 h-1 bg-custom-gray"></div>
                    </div>
                    <button @click="show_group_modal = false" class="text-xl font-black p-2 border-custom hover:bg-gray-50 transition-all text-custom-black">✕</button>
                </div>

                <div class="space-y-6">
                    <div class="flex gap-3">
                        <input v-model="new_group.name" placeholder="输入新分组名称..." 
                               class="input-custom flex-1 p-3 text-sm focus:outline-none font-bold bg-gray-50">
                        <button @click="create_group" class="btn-primary px-6 py-3 text-[10px] font-bold uppercase">新增分组</button>
                    </div>

                    <div class="border border-gray-100 rounded overflow-hidden">
                        <div v-for="group in group_list" :key="group.id" 
                             class="flex justify-between items-center p-4 bg-white border-b border-gray-50 last:border-0 hover:bg-gray-50 transition-colors">
                            <div class="flex-1 flex items-center gap-4">
                                <div v-if="editing_group_id === group.id" class="flex items-center gap-2 flex-1">
                                    <input v-model="editing_group_name" class="input-custom flex-1 p-1 text-sm font-bold bg-white focus:outline-none">
                                    <button @click="save_group_name(group.id)" class="text-[9px] font-black text-custom-green-600 uppercase">保存</button>
                                    <button @click="editing_group_id = null" class="text-[9px] font-black text-custom-gray-400 uppercase">取消</button>
                                </div>
                                <div v-else class="flex flex-col">
                                    <span class="text-sm font-bold text-custom-black">{{ group.name }}</span>
                                    <span class="text-[8px] text-custom-gray-300 font-mono uppercase">Group ID: {{ group.id }}</span>
                                </div>
                            </div>
                            <div class="flex gap-2 ml-4">
                                <button v-if="editing_group_id !== group.id" @click="start_edit_group(group)"
                                        class="text-[9px] font-black text-custom-black uppercase border border-gray-200 px-2 py-1 hover:bg-gray-100">
                                    编辑
                                </button>
                                <button v-if="group.id !== 1" @click="delete_group(group.id)" 
                                        class="text-[9px] font-black text-custom-red-600 uppercase border border-custom-red-200 px-2 py-1 hover:bg-custom-red-50">
                                    删除
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div v-if="show_add_modal" class="modal-overlay">
            <div class="modal-content">
                <div class="mb-10 flex justify-between items-start">
                    <div>
                        <h2 class="text-2xl font-bold tracking-[0.2em] mb-1 text-custom-black uppercase">部署引导</h2>
                        <div class="w-12 h-1 bg-custom-gray"></div>
                    </div>
                    <button @click="show_add_modal = false" class="text-xl font-black p-2 border-custom hover:bg-gray-50 transition-all text-custom-black">✕</button>
                </div>

                <div class="space-y-8">
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
                        <div class="space-y-3">
                            <label class="text-[11px] font-black text-custom-black uppercase">1. 节点名称</label>
                            <input v-model="new_node.name" placeholder="请输入节点显示名称..." 
                                   class="input-custom w-full p-3 text-sm focus:outline-none font-bold bg-gray-50">
                        </div>
                        <div class="space-y-3">
                            <label class="text-[11px] font-black text-custom-black uppercase">2. 选择分组</label>
                            <select v-model="new_node.group_id" class="input-custom w-full p-3 text-sm focus:outline-none font-bold bg-gray-50 cursor-pointer">
                                <option v-for="g in group_list" :value="g.id">{{ g.name }}</option>
                            </select>
                        </div>
                    </div>
                    
                    <p class="text-[9px] text-custom-gray-400 italic">自动分配 NodeID: <span class="text-custom-black font-bold">{{ new_node.node_id }}</span></p>

                    <div class="space-y-8">
                        <label class="text-[11px] font-black text-custom-black block uppercase">3. 复制并执行部署脚本</label>
                        
                        <div class="relative pt-2">
                            <div class="absolute -top-2 left-4 bg-custom-white border-custom px-2 py-0.5 z-10">
                                <span class="text-[9px] font-black tracking-tighter text-custom-black uppercase">Linux / MacOS (Bash)</span>
                            </div>
                            <div class="input-custom bg-[#1f2937] text-green-400 p-6 pt-8 font-mono text-[10px] break-all whitespace-pre-wrap leading-relaxed">
                                {{ gen_linux_cmd(false) }}
                            </div>
                            <button @click="copy_text(gen_linux_cmd(false))" 
                                    class="absolute bottom-3 right-3 text-[9px] font-black bg-custom-white border-custom px-3 py-1 hover:bg-gray-100 transition-all text-custom-black uppercase">
                                点击复制
                            </button>
                        </div>

                        <div class="relative pt-2">
                            <div class="absolute -top-2 left-4 bg-custom-white border-custom px-2 py-0.5 z-10">
                                <span class="text-[9px] font-black tracking-tighter text-custom-black uppercase">Windows (PowerShell)</span>
                            </div>
                            <div class="input-custom bg-[#1f2937] text-blue-300 p-6 pt-8 font-mono text-[10px] break-all whitespace-pre-wrap leading-relaxed">
                                {{ gen_win_cmd(false) }}
                            </div>
                            <button @click="copy_text(gen_win_cmd(false))" 
                                    class="absolute bottom-3 right-3 text-[9px] font-black bg-custom-white border-custom px-3 py-1 hover:bg-gray-100 transition-all text-custom-black uppercase">
                                点击复制
                            </button>
                        </div>
                    </div>

                    <button @click="show_add_modal = false" class="btn-secondary w-full py-4 text-[11px] font-black tracking-[0.2em] uppercase">
                        已完成部署
                    </button>
                </div>
            </div>
        </div>

        <div v-if="show_edit_modal" class="modal-overlay">
            <div class="modal-content max-w-md">
                <div class="mb-8 text-center">
                    <h2 class="text-xl font-bold tracking-widest uppercase text-custom-black">修改配置</h2>
                    <p class="text-[9px] text-custom-gray-400 mt-1 italic uppercase">NodeID: {{ edit_form.id }}</p>
                </div>
                
                <div class="space-y-6">
                    <div class="space-y-1">
                        <label class="text-[10px] font-black text-custom-gray-400 ml-1 uppercase">显示名称</label>
                        <input v-model="edit_form.name" class="input-custom w-full p-3 text-sm focus:outline-none font-bold text-center text-custom-black">
                    </div>

                    <div class="space-y-1">
                        <label class="text-[10px] font-black text-custom-gray-400 ml-1 uppercase">地理位置</label>
                        <input v-model="edit_form.location" class="input-custom w-full p-3 text-sm focus:outline-none font-bold text-center text-custom-black" placeholder="如: 中国上海">
                    </div>

                    <div class="space-y-1">
                        <label class="text-[10px] font-black text-custom-gray-400 ml-1 uppercase">所属分组</label>
                        <select v-model="edit_form.group_id" class="input-custom w-full p-3 text-sm focus:outline-none font-bold text-center text-custom-black bg-white cursor-pointer">
                            <option v-for="g in group_list" :value="g.id">{{ g.name }}</option>
                        </select>
                    </div>
                    
                    <div class="input-custom bg-gray-50 flex items-center justify-between p-3">
                        <label class="text-[10px] font-black text-custom-black uppercase cursor-pointer select-none" for="node_visible_toggle">对访客公开显示</label>
                        <input id="node_visible_toggle" type="checkbox" v-model="edit_form.is_visible" class="w-4 h-4 accent-black cursor-pointer">
                    </div>
                    
                    <div class="space-y-3 border-t border-gray-200 pt-6">
                        <label class="text-[10px] font-black text-custom-gray-400 ml-1 uppercase">部署脚本（重装用）</label>
                        
                        <div class="space-y-3">
                            <div class="relative">
                                <div class="absolute -top-1 left-2 bg-custom-white px-1">
                                    <span class="text-[8px] font-black text-custom-gray-500 tracking-tight uppercase">Linux / Mac</span>
                                </div>
                                <div class="input-custom bg-gray-50 p-3 font-mono text-[9px] break-all whitespace-pre-wrap leading-relaxed text-custom-black">
                                    {{ gen_linux_cmd(true) }}
                                </div>
                                <button @click="copy_text(gen_linux_cmd(true))" 
                                        class="absolute bottom-2 right-2 text-[8px] font-bold bg-custom-white border border-gray-300 px-2 py-0.5 hover:bg-gray-100 transition-all text-custom-black uppercase">
                                    复制
                                </button>
                            </div>
                            
                            <div class="relative">
                                <div class="absolute -top-1 left-2 bg-custom-white px-1">
                                    <span class="text-[8px] font-black text-custom-gray-500 tracking-tight uppercase">Windows</span>
                                </div>
                                <div class="input-custom bg-gray-50 p-3 font-mono text-[9px] break-all whitespace-pre-wrap leading-relaxed text-custom-black">
                                    {{ gen_win_cmd(true) }}
                                </div>
                                <button @click="copy_text(gen_win_cmd(true))" 
                                        class="absolute bottom-2 right-2 text-[8px] font-bold bg-custom-white border border-gray-300 px-2 py-0.5 hover:bg-gray-100 transition-all text-custom-black uppercase">
                                    复制
                                </button>
                            </div>
                        </div>
                    </div>
                    
                    <div class="flex gap-4 pt-4">
                        <button @click="save_node" class="btn-primary flex-1 py-3 text-[10px] font-bold uppercase">保存</button>
                        <button @click="show_edit_modal = false" class="btn-secondary flex-1 py-3 text-[10px] font-bold uppercase">取消</button>
                    </div>
                </div>
            </div>
        </div>
    </div>
    `
};

window.NodesComponent = NodesComponent;