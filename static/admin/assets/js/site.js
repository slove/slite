/**
 * 网站设置组件 (SiteComponent)
 * 处理网站的视觉识别系统配置，包括 LOGO、名称、标语及版权信息
 */
const SiteComponent = {
    props: ['site_config', 'is_saving', 'save_success'],
    data() {
        return {
            /* 内部状态：用于控制失败提示的显示 */
            save_error: false
        };
    },
    computed: {
        /*
         * 回退图标生成
         * 若用户未配置 LOGO 图片地址，则取站点名称的首字符作为默认展示图标
         */
        fallbackLogo() {
            if (!this.site_config.site_name) return 'S';
            return this.site_config.site_name.charAt(0).toUpperCase();
        }
    },
    methods: {
        /*
         * 触发保存动作
         * 重置本地错误状态，并向父组件发送 saveConfig 指令
         */
        handleSave() {
            this.save_error = false;
            this.$emit('action', 'saveConfig');
        }
    },
    template: `
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-10 animate-in fade-in slide-in-from-bottom-4 relative">
        
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

        <div class="space-y-8">
            <div class="grid grid-cols-1 gap-6">
                <div>
                    <label class="text-[10px] font-black uppercase text-custom-gray-400 block mb-2 tracking-[0.2em]">网站标识</label>
                    <input type="text" v-model="site_config.site_logo" placeholder="请输入图片链接 (留空则显示首字母图标)" 
                           class="input-custom w-full p-3 text-[11px] font-mono focus:outline-none focus:bg-gray-50 transition-all">
                </div>
                <div>
                    <label class="text-[10px] font-black uppercase text-custom-gray-400 block mb-2 tracking-[0.2em]">网站名称</label>
                    <input type="text" v-model="site_config.site_name" placeholder="SLITE MONITORING"
                           class="input-custom w-full p-3 font-black focus:outline-none focus:bg-gray-50 transition-all">
                </div>
                <div>
                    <label class="text-[10px] font-black uppercase text-custom-gray-400 block mb-2 tracking-[0.2em]">网站标语</label>
                    <input type="text" v-model="site_config.site_slogan" placeholder="你的每一次心跳，皆有回响。"
                           class="input-custom w-full p-3 text-xs font-bold focus:outline-none focus:bg-gray-50 transition-all">
                </div>
                <div>
                    <label class="text-[10px] font-black uppercase text-custom-gray-400 block mb-2 tracking-[0.2em]">页脚版权内容</label>
                    <textarea v-model="site_config.site_footer" placeholder="© 2026 SLITE. 所有权利保留。"
                              class="input-custom w-full p-3 h-32 resize-none text-[11px] font-mono focus:outline-none focus:bg-gray-50 transition-all"></textarea>
                </div>
            </div>
            
            <div class="pt-4">
                <button @click="handleSave" :disabled="is_saving"
                        class="btn-primary w-full py-4 text-[11px] font-black uppercase tracking-[0.3em] disabled:bg-gray-400 transition-all shadow-[6px_6px_0px_rgba(0,0,0,0.1)] active:shadow-none active:translate-x-[2px] active:translate-y-[2px]">
                    {{ is_saving ? '正在保存...' : '确认保存' }}
                </button>
            </div>
        </div>

        <div class="card-custom p-10 flex flex-col items-center justify-center bg-custom-white relative overflow-hidden">
            <div class="absolute top-0 left-0 bg-custom-black text-custom-white px-4 py-1 text-[9px] font-black uppercase tracking-widest">
                实时视觉预览
            </div>
            
            <div class="text-center space-y-6 z-10 w-full">
                <div class="mb-4 flex justify-center">
                    <img v-if="site_config.site_logo" :src="site_config.site_logo" class="h-12 grayscale object-contain">
                    <div v-else class="w-16 h-16 bg-custom-black text-custom-white flex items-center justify-center text-3xl font-black">
                        {{ fallbackLogo }}
                    </div>
                </div>

                <h2 class="text-1xl font-black uppercase tracking-tighter leading-none border-b-4 border-custom-black pb-2 break-all">
                    {{ site_config.site_name || '未命名站点' }}
                </h2>
                <p class="text-[12px] text-custom-gray-500 font-bold uppercase tracking-[0.4em]">
                    {{ site_config.site_slogan || '系统准备就绪' }}
                </p>
                <div class="mt-8 pt-8 border-t border-dashed border-gray-300 w-full text-[9px] text-custom-gray-400 font-mono break-all" v-html="site_config.site_footer || '页脚内容预览区'">
                </div>
            </div>
        </div>
    </div>`
};

window.SiteComponent = SiteComponent;