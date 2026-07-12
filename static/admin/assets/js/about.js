/**
 * 关于系统组件 (AboutComponent)
 */
const AboutComponent = {
    props: ['site_config'],
    template: `
    <div class="animate-in fade-in slide-in-from-bottom-4 space-y-6 pb-20 font-sans text-slate-900">
        
        <section class="card-custom-no-padding overflow-hidden">
            <div class="p-8 md:p-12">
                <div class="flex items-center gap-3 mb-8">
                    <div class="w-1.5 h-6 bg-custom-black"></div>
                    <h2 class="text-sm font-black uppercase tracking-widest text-custom-black">系统设计哲学 / PHILOSOPHY</h2>
                </div>
                
                <div class="max-w-4xl">
                    <h1 class="text-4xl md:text-5xl font-black italic tracking-tighter mb-8 leading-tight text-custom-black">
                        {{ site_config.site_name || 'SLITE MONITOR' }}<br>
                        <span class="text-custom-gray-300 not-italic text-3xl md:text-4xl">消灭冗余，直击本质。</span>
                    </h1>
                    
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-8 text-[13px] leading-7 text-custom-gray-600 font-medium">
                        <p>
                            本监控系统诞生于对"过度设计"的反思。在运维领域，开发者往往深陷于沉重的仪表盘和复杂的配置陷阱。我们追求的是<b class="text-custom-black font-black underline underline-offset-4 decoration-1">瞬时感知</b>。
                        </p>
                        <p>
                            通过 <span class="text-custom-black font-bold">Go 语言</span> 强大的并发处理能力与 <span class="text-custom-black font-bold">WebSocket</span> 的实时特性，系统实现了节点状态的毫秒级分发。配合 <span class="text-custom-black font-bold">SQLite</span> 嵌入式数据库，实现了真正的零依赖、开箱即用。
                        </p>
                    </div>
                </div>
            </div>
            <div class="h-2 bg-custom-black w-full"></div>
        </section>

        <section class="card-custom p-8 md:p-12">
            <div class="flex items-center gap-3 mb-10">
                <div class="w-1.5 h-6 bg-custom-black"></div>
                <h2 class="text-sm font-black uppercase tracking-widest text-custom-black">核心功能特性 / FEATURES</h2>
            </div>

            <div class="grid grid-cols-1 md:grid-cols-2 gap-x-12 gap-y-6">
                <div v-for="feat in features" :key="feat.title" class="flex items-start gap-4 group">
                    <div class="mt-1.5">
                        <div class="w-2 h-2 bg-custom-black group-hover:scale-125 transition-transform"></div>
                    </div>
                    <div>
                        <h3 class="text-[13px] font-black text-custom-black uppercase mb-1">{{ feat.title }}</h3>
                        <p class="text-[11px] text-custom-gray-400 font-medium leading-relaxed">{{ feat.desc }}</p>
                    </div>
                </div>
            </div>
        </section>

        <section class="grid grid-cols-1 md:grid-cols-4 card-custom-no-padding text-custom-white">
            <div class="p-8 border-b md:border-b-0 md:border-r border-custom-gray-800 flex flex-col justify-between">
                <div>
                    <h2 class="text-[10px] font-black uppercase tracking-[0.2em] text-custom-gray-500 mb-2">架构基石</h2>
                    <p class="text-xs font-bold text-custom-gray-300">TRUSTED STACK</p>
                </div>
                <div class="mt-8">
                    <span class="text-[9px] font-mono text-custom-gray-600 block italic">VERSION CONTROL</span>
                    <span class="text-xl font-black tracking-tighter text-custom-yellow-300">{{ site_config.version || '2.1.0' }}</span>
                </div>
            </div>
            
            <div class="md:col-span-3 p-8">
                <div class="grid grid-cols-2 md:grid-cols-3 gap-x-8 gap-y-6">
                    <div v-for="tech in techStack" :key="tech.name" class="border-b border-custom-gray-800 pb-2 group">
                        <span class="text-[9px] font-black text-custom-gray-500 uppercase block mb-1 group-hover:text-custom-yellow-300 transition-colors">{{ tech.name }}</span>
                        <span class="text-xs font-mono font-bold tracking-tight text-custom-black">{{ tech.version }}</span>
                    </div>
                </div>
            </div>
        </section>

        <footer class="pt-4 flex justify-between items-center text-[10px] font-black text-custom-gray-400 uppercase tracking-[0.2em]">
            <div class="flex items-center gap-4">
                <span class="text-custom-black">SLITE PROJECT</span>
                <span class="text-custom-gray-200">/</span>
                <span>© 2026 ALL RIGHTS RESERVED.</span>
            </div>
            <div class="flex gap-6">
                <a href="#" class="hover:text-custom-black transition-colors underline underline-offset-4">文档说明</a>
                <a href="#" class="hover:text-custom-black transition-colors underline underline-offset-4">源代码</a>
            </div>
        </footer>
    </div>
    `,
    data() {
        return {
            features: [
                { title: '极轻量化架构', desc: '后端体积约 15MB，支持在 128MB 嵌入式设备运行。' },
                { title: '即时数据同步', desc: 'WebSocket 驱动，节点异动秒级推送到监控终端。' },
                { title: '核心审计安全', desc: '内置完整审计系统，记录所有配置修改与登录。' },
                { title: '数据长效持久', desc: '自动统计在线率及流量分布，生成 Uptime 历史记录。' },
                { title: '单机快捷部署', desc: '无依赖二进制文件，配合 SQLite 零成本起步。' },
                { title: '工业级 UI 界面', desc: '粗野主义线条设计，高对比度，专注于信息传达。' }
            ],
            techStack: [
                { name: '核心引擎', version: 'Golang 1.2x' },
                { name: '接口协议', version: 'Gin Framework' },
                { name: '持久化层', version: 'SQLite 3' },
                { name: '通讯协议', version: 'Gorilla WS' },
                { name: '前端框架', version: 'Vue 3.x' },
                { name: '样式引擎', version: 'Tailwind CSS' }
            ]
        }
    }
};

window.AboutComponent = AboutComponent;