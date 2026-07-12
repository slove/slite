const BackupManagerComponent = {
    props: ['site_config', 'basic_config', 'backup_config', 'handle_backup_action'],
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
                <h2 class="text-2xl md:text-3xl font-black italic tracking-tighter text-custom-black">
                </h2>
                <p class="text-[10px] text-custom-gray-400 mt-2 font-bold uppercase tracking-widest">
                    当前状态: 
                    <span :class="config.auto_backup_enabled ? 'text-custom-green-600' : 'text-custom-red-600'">
                        {{ config.auto_backup_enabled ? '自动备份已启用' : '自动备份已停用' }}
                    </span>
                </p>
            </div>
            <div class="flex items-center pb-1">
                <button @click="toggleBackupEngine" 
                        :class="['text-[10px] border-custom-black px-8 py-2 font-black uppercase transition-all tracking-widest', 
                                config.auto_backup_enabled ? 'bg-custom-black text-custom-white' : 'bg-custom-white text-custom-black']">
                    {{ config.auto_backup_enabled ? '停用自动备份' : '启用自动备份' }}
                </button>
            </div>
        </header>

        <div class="card-custom bg-custom-white space-y-6">
            <div class="flex justify-between items-center border-b border-gray-100 pb-4">
                <h4 class="text-sm font-black uppercase flex items-center gap-3 text-custom-black">
                    <span class="bg-custom-black w-1.5 h-5"></span> 
                    备份操作
                </h4>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
                <div class="p-6 border border-gray-200 hover:border-custom-black transition-colors group">
                    <div class="flex items-center gap-3 mb-4">
                        <div class="w-1.5 h-5 bg-custom-yellow-300"></div>
                        <h3 class="text-sm font-black text-custom-black uppercase">即时快照</h3>
                    </div>
                    <p class="text-[11px] text-custom-gray-400 font-medium mb-6 leading-relaxed">
                        创建当前数据库的完整加密快照，适用于重要操作前的数据保全。
                    </p>
                    <div class="flex justify-between items-center">
                        <span class="text-[9px] font-mono text-custom-gray-500">SQLite 在线备份</span>
                        <button @click="createBackup" 
                                :disabled="backupInProgress"
                                class="px-4 py-2 text-xs font-black bg-custom-black text-custom-white hover:bg-custom-gray-800 disabled:opacity-50 disabled:cursor-not-allowed transition-colors">
                            {{ backupInProgress ? '备份中...' : '立即备份' }}
                        </button>
                    </div>
                </div>

                <div class="p-6 border border-gray-200 hover:border-custom-black transition-colors group">
                    <div class="flex items-center gap-3 mb-4">
                        <div class="w-1.5 h-5 bg-custom-yellow-300"></div>
                        <h3 class="text-sm font-black text-custom-black uppercase">自动备份</h3>
                    </div>
                    <p class="text-[11px] text-custom-gray-400 font-medium mb-6 leading-relaxed">
                        配置自动化备份策略，支持增量备份与全量备份组合。
                    </p>
                    <div class="space-y-3">
                        <div class="flex items-center gap-3">
                            <select v-model="config.backup_strategy" 
                                    class="text-xs bg-gray-50 border border-gray-200 text-custom-black px-3 py-1.5 flex-1 focus:border-custom-black focus:outline-none appearance-none text-center">
                                <option value="daily_full">每日全量备份</option>
                                <option value="weekly_full_inc">每周全量+每日增量备份</option>
                                <option value="monthly_archive">月度归档备份</option>
                            </select>
                        </div>
                    </div>
                </div>

                <div class="p-6 border border-gray-200 hover:border-custom-black transition-colors group">
                    <div class="flex items-center gap-3 mb-4">
                        <div class="w-1.5 h-5 bg-custom-yellow-300"></div>
                        <h3 class="text-sm font-black text-custom-black uppercase">数据导出</h3>
                    </div>
                    <p class="text-[11px] text-custom-gray-400 font-medium mb-6 leading-relaxed">
                        导出为可移植格式，支持 SQL 转储、CSV 数据集及 JSON 结构。
                    </p>
                    <div class="flex flex-col gap-2">
                        <button @click="exportData('sql')" 
                                class="w-full px-3 py-2 text-xs font-black border border-custom-black text-custom-black hover:bg-custom-black hover:text-custom-white transition-colors">
                            SQL 转储文件 (.sql)
                        </button>
                        <div class="flex gap-2">
                            <button @click="exportData('csv')" 
                                    class="flex-1 px-2 py-1.5 text-[10px] font-black border border-gray-200 text-custom-gray-600 hover:bg-gray-200 transition-colors">
                                CSV 数据集
                            </button>
                            <button @click="exportData('json')" 
                                    class="flex-1 px-2 py-1.5 text-[10px] font-black border border-gray-200 text-custom-gray-600 hover:bg-gray-200 transition-colors">
                                JSON 结构
                            </button>
                        </div>
                    </div>
                </div>
            </div>

            <div class="border-t border-gray-100 pt-8">
                <div class="flex items-center gap-3 mb-4">
                    <div class="w-1.5 h-5 bg-custom-black"></div>
                    <h3 class="text-xs font-black uppercase text-custom-black">备份系统状态</h3>
                </div>
                <div class="grid grid-cols-1 md:grid-cols-4 gap-4 text-[10px]">
                    <div class="p-4 bg-gray-50 border border-gray-100">
                        <div class="flex justify-between items-center mb-2">
                            <span class="font-bold text-custom-black">数据库状态</span>
                            <span class="font-mono" :class="dbStatus.online ? 'text-custom-green-500' : 'text-custom-red-500'">
                                {{ dbStatus.online ? '在线' : '离线' }}
                            </span>
                        </div>
                        <div class="text-custom-gray-500">{{ dbStatus.size || '--' }} / {{ dbStatus.rows || '--' }} 行</div>
                    </div>
                    <div class="p-4 bg-gray-50 border border-gray-100">
                        <div class="flex justify-between items-center mb-2">
                            <span class="font-bold text-custom-black">最后备份</span>
                            <span class="font-mono text-custom-gray-500">{{ backupStats.lastBackupTime || '无记录' }}</span>
                        </div>
                        <div class="h-1 bg-gray-200 overflow-hidden">
                            <div class="h-full bg-custom-yellow-300" :style="{width: backupStats.health + '%'}"></div>
                        </div>
                    </div>
                    <div class="p-4 bg-gray-50 border border-gray-100">
                        <div class="flex justify-between items-center mb-2">
                            <span class="font-bold text-custom-black">备份文件</span>
                            <span class="font-mono text-custom-black font-bold">{{ backupStats.totalCount || 0 }}</span>
                        </div>
                        <div class="text-custom-gray-500">{{ backupStats.totalSize || '0 Bytes' }}</div>
                    </div>
                    <div class="p-4 bg-gray-50 border border-gray-100">
                        <div class="flex justify-between items-center mb-2">
                            <span class="font-bold text-custom-black">加密状态</span>
                            <span class="font-mono text-custom-green-500 font-bold">AES-256-GCM</span>
                        </div>
                        <div class="text-custom-gray-500">完整性校验启用</div>
                    </div>
                </div>
            </div>
        </div>

        <div class="card-custom bg-custom-white">
            <div class="flex justify-between items-center border-b border-gray-100 pb-4 mb-6">
                <h4 class="text-sm font-black uppercase flex items-center gap-3 text-custom-black">
                    <span class="bg-custom-red-600 w-1.5 h-5"></span> 
                    备份档案管理
                </h4>
                <div class="flex items-center gap-4">
                    <div class="text-[10px] font-black text-custom-gray-400 uppercase">
                        共 {{ backups.length }} 个备份文件
                    </div>
                    <button @click="fetchBackupFiles" 
                            class="text-[10px] font-black px-3 py-1 border border-gray-200 text-custom-gray-600 hover:bg-gray-200 transition-colors">
                        刷新列表
                    </button>
                </div>
            </div>

            <div class="overflow-x-auto">
                <table class="w-full text-[11px]">
                    <thead>
                        <tr class="border-b border-gray-200">
                            <th class="text-left pb-3 font-black text-custom-black uppercase">文件名</th>
                            <th class="text-left pb-3 font-black text-custom-black uppercase">创建时间</th>
                            <th class="text-left pb-3 font-black text-custom-black uppercase">大小</th>
                            <th class="text-left pb-3 font-black text-custom-black uppercase">类型</th>
                            <th class="text-left pb-3 font-black text-custom-black uppercase">完整性</th>
                            <th class="text-left pb-3 font-black text-custom-black uppercase">操作</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr v-for="backup in backups" :key="backup.id" 
                            class="border-b border-gray-100 hover:bg-gray-50 transition-colors">
                            <td class="py-4">
                                <div class="flex items-center gap-2">
                                    <div :class="{
                                        'w-1.5 h-1.5 rounded-none': true,
                                        'bg-custom-yellow-300': backup.type === 'full',
                                        'bg-custom-blue-400': backup.type === 'incremental'
                                    }"></div>
                                    <div>
                                        <div class="font-mono font-bold text-custom-black">{{ backup.filename }}</div>
                                        <div v-if="backup.checksum" class="text-[9px] text-custom-gray-500 font-medium">{{ backup.checksum.substring(0, 16) }}...</div>
                                    </div>
                                </div>
                            </td>
                            <td class="py-4 font-medium text-custom-gray-600">{{ formatDateTime(backup.created_at) }}</td>
                            <td class="py-4 font-mono text-custom-gray-600">{{ formatFileSize(backup.size) }}</td>
                            <td class="py-4">
                                <span :class="{
                                    'px-2 py-1 text-[9px] font-black rounded-none': true,
                                    'bg-custom-yellow-300 text-custom-black': backup.type === 'full',
                                    'bg-custom-blue-400 text-custom-black': backup.type === 'incremental'
                                }">
                                    {{ backup.type === 'full' ? '完整备份' : '增量备份' }}
                                </span>
                            </td>
                            <td class="py-4">
                                <span :class="{
                                    'px-2 py-1 text-[9px] font-black rounded-none': true,
                                    'bg-green-100 text-green-700': backup.verified,
                                    'bg-red-100 text-red-700': !backup.verified
                                }">
                                    {{ backup.verified ? '已验证' : '未验证' }}
                                </span>
                            </td>
                            <td class="py-4">
                                <div class="flex gap-2">
                                    <button @click="verifyBackup(backup.id)" 
                                            class="px-2 py-1 text-[9px] font-black border border-green-500 text-green-600 hover:bg-green-500 hover:text-white transition-colors">
                                        验证
                                    </button>
                                    <button @click="selectForRestore(backup.id)" 
                                            class="px-2 py-1 text-[9px] font-black border border-custom-black text-custom-black hover:bg-custom-black hover:text-white transition-colors">
                                        恢复
                                    </button>
                                    <button @click="downloadBackup(backup.id)" 
                                            class="px-2 py-1 text-[9px] font-black border border-gray-400 text-gray-600 hover:bg-gray-400 hover:text-black transition-colors">
                                        下载
                                    </button>
                                    <button @click="deleteBackup(backup.id)" 
                                            class="px-2 py-1 text-[9px] font-black border border-red-300 text-red-500 hover:bg-red-300 hover:text-white transition-colors">
                                        删除
                                    </button>
                                </div>
                            </td>
                        </tr>
                        <tr v-if="backups.length === 0">
                            <td colspan="6" class="py-12 text-center">
                                <div class="flex flex-col items-center gap-3">
                                    <div class="w-12 h-12 bg-gray-200 rounded-none flex items-center justify-center">
                                        <div class="w-6 h-6 bg-gray-300 rounded-none"></div>
                                    </div>
                                    <div class="text-custom-gray-400 font-medium">暂无备份文件</div>
                                    <button @click="createBackup" 
                                            class="text-xs font-black px-4 py-2 bg-custom-black text-white hover:bg-gray-800 transition-colors">
                                        创建第一个备份
                                    </button>
                                </div>
                            </td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>

        <div class="card-custom bg-custom-white space-y-6">
            <div class="flex justify-between items-center border-b border-gray-100 pb-4">
                <h4 class="text-sm font-black uppercase flex items-center gap-3 text-custom-black">
                    <span class="bg-gray-600 w-1.5 h-5"></span> 
                    备份策略配置
                </h4>
            </div>
            
            <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
                <div class="space-y-6">
                    <div class="space-y-3">
                        <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-widest block">备份保留天数</label>
                        <div class="h-[56px] flex items-center">
                            <input type="number" v-model="config.backup_retention_days" 
                                   class="input-custom bg-transparent text-center font-black text-sm outline-none focus:text-red-600 transition-colors w-full h-full border-b border-gray-200">
                        </div>
                    </div>
                    
                    <div class="space-y-3">
                        <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-widest block">最大备份数量</label>
                        <div class="h-[56px] flex items-center">
                            <input type="number" v-model="config.backup_max_count" 
                                   class="input-custom bg-transparent text-center font-black text-sm outline-none focus:text-red-600 transition-colors w-full h-full border-b border-gray-200">
                        </div>
                    </div>
                </div>
                
                <div class="space-y-6">
                    <div class="space-y-3">
                        <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-widest block">备份执行时间</label>
                        <div class="h-[56px] flex items-center">
                            <input type="time" v-model="config.backup_schedule_time" 
                                   class="input-custom bg-transparent text-center font-black text-sm outline-none focus:text-red-600 transition-colors w-full h-full border-b border-gray-200">
                        </div>
                    </div>
                    
                    <div class="space-y-3">
                        <label class="text-[10px] font-black text-custom-gray-400 uppercase tracking-widest block">加密算法</label>
                        <div class="h-[56px] flex items-center">
                            <select v-model="config.backup_encryption_algorithm" 
                                    class="input-custom bg-transparent font-black text-sm outline-none w-full h-full border-b border-gray-200 appearance-none pr-8">
                                <option value="AES-256-GCM">AES-256-GCM 加密</option>
                                <option value="AES-256-CBC">AES-256-CBC 加密</option>
                                <option value="ChaCha20">ChaCha20 加密</option>
                            </select>
                        </div>
                    </div>
                </div>
            </div>

            <div class="space-y-6 pt-6 border-t border-gray-100">
                <label class="text-[10px] font-black uppercase tracking-widest text-custom-gray-500 block mb-4">功能开关</label>
                <div class="grid grid-cols-1 md:grid-cols-4 gap-x-8 gap-y-4">
                    <label class="flex justify-between items-center cursor-pointer group text-custom-black">
                        <span class="text-[10px] font-bold uppercase group-hover:text-red-600 transition-colors">自动备份</span>
                        <input type="checkbox" v-model="config.auto_backup_enabled" class="w-4 h-4 accent-black">
                    </label>
                    <label class="flex justify-between items-center cursor-pointer group text-custom-black">
                        <span class="text-[10px] font-bold uppercase group-hover:text-red-600 transition-colors">加密存储</span>
                        <input type="checkbox" v-model="config.backup_encryption_enabled" class="w-4 h-4 accent-black">
                    </label>
                    <label class="flex justify-between items-center cursor-pointer group text-custom-black">
                        <span class="text-[10px] font-bold uppercase group-hover:text-red-600 transition-colors">完整性校验</span>
                        <input type="checkbox" v-model="config.backup_verify_integrity" class="w-4 h-4 accent-black">
                    </label>
                    <label class="flex justify-between items-center cursor-pointer group text-custom-black">
                        <span class="text-[10px] font-bold uppercase group-hover:text-red-600 transition-colors">压缩存储</span>
                        <input type="checkbox" v-model="config.backup_compression_enabled" class="w-4 h-4 accent-black">
                    </label>
                </div>
            </div>
        </div>

        <div class="card-custom bg-custom-white relative overflow-hidden group">
            <div class="absolute -right-4 -top-6 text-[100px] font-black text-gray-50 select-none group-hover:text-red-50/50 transition-colors pointer-events-none">R</div>
            <div class="relative z-10">
                <div class="flex justify-between items-center mb-10">
                    <h4 class="text-sm font-black uppercase flex items-center gap-3 text-custom-black">
                        <span class="bg-custom-red-600 w-1.5 h-5"></span> 
                        数据恢复操作
                    </h4>
                </div>

                <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
                    <div>
                        <h3 class="text-lg font-black italic mb-4 text-custom-red-600">重要：恢复操作须知</h3>
                        <p class="text-[11px] text-custom-gray-400 font-medium leading-relaxed mb-6">
                            数据恢复操作将替换当前数据库的所有内容。这是一个不可逆的操作，请务必：
                        </p>
                        <ul class="space-y-3 text-[11px] text-custom-gray-400 font-medium mb-6">
                            <li class="flex items-start gap-2">
                                <div class="w-1.5 h-1.5 bg-custom-red-600 mt-1.5"></div>
                                <span>在业务低峰期或维护窗口执行</span>
                            </li>
                            <li class="flex items-start gap-2">
                                <div class="w-1.5 h-1.5 bg-custom-red-600 mt-1.5"></div>
                                <span>恢复前创建当前数据库的快照</span>
                            </li>
                            <li class="flex items-start gap-2">
                                <div class="w-1.5 h-1.5 bg-custom-red-600 mt-1.5"></div>
                                <span>验证备份文件的完整性和加密状态</span>
                            </li>
                        </ul>
                        <div class="bg-red-50 p-4 border border-red-100">
                            <div class="flex items-center gap-2 mb-2">
                                <div class="w-1.5 h-1.5 bg-custom-red-600"></div>
                                <span class="text-xs font-black text-red-600">注意：</span>
                            </div>
                            <p class="text-[10px] text-red-400">
                                恢复操作期间系统将进入维护模式，所有写操作将被暂停，预计耗时 1-3 分钟。
                            </p>
                        </div>
                    </div>

                    <div class="bg-gray-50 p-6 border border-gray-200">
                        <div class="mb-6">
                            <label class="block text-xs font-black mb-3 text-custom-gray-600">选择恢复源</label>
                            <div class="relative">
                                <select v-model="selectedRestoreBackup" 
                                        class="w-full bg-white text-custom-black text-sm px-4 py-3 border border-gray-300 focus:border-custom-red-600 focus:outline-none appearance-none">
                                    <option value="">请选择备份文件</option>
                                    <option v-for="backup in backups" :key="backup.id" :value="backup.id">
                                        {{ backup.filename }} ({{ formatFileSize(backup.size) }})
                                    </option>
                                </select>
                                <div class="absolute right-3 top-1/2 transform -translate-y-1/2 pointer-events-none">
                                    <div class="w-2 h-2 border-r border-b border-gray-500 transform rotate-45"></div>
                                </div>
                            </div>
                            <div v-if="selectedRestoreBackup" class="mt-3 p-3 bg-white border border-gray-200 rounded-none">
                                <div class="text-[10px] text-custom-gray-400">
                                    选中备份：{{ selectedBackupInfo }}
                                </div>
                            </div>
                        </div>

                        <div class="space-y-4 mb-6">
                            <label class="flex items-start gap-3 text-xs font-black text-custom-gray-600">
                                <input type="checkbox" v-model="confirmRestore" 
                                       class="mt-0.5 rounded-none border-gray-300 text-custom-red-600 focus:ring-custom-red-600">
                                <span>我已理解恢复操作的风险和后果</span>
                            </label>
                            <label class="flex items-start gap-3 text-xs font-black text-custom-gray-600">
                                <input type="checkbox" v-model="createPreRestoreBackup" 
                                       class="mt-0.5 rounded-none border-gray-300 text-custom-red-600 focus:ring-custom-red-600">
                                <span>恢复前自动创建当前数据库快照</span>
                            </label>
                            <label class="flex items-start gap-3 text-xs font-black text-custom-gray-600">
                                <input type="checkbox" v-model="verifyBeforeRestore" 
                                       class="mt-0.5 rounded-none border-gray-300 text-custom-red-600 focus:ring-custom-red-600">
                                <span>恢复前验证备份文件完整性</span>
                            </label>
                        </div>

                        <div class="space-y-3">
                            <button @click="executeRestore" 
                                    :disabled="!canExecuteRestore"
                                    :class="{
                                        'w-full py-3 text-sm font-black rounded-none transition-all duration-300': true,
                                        'bg-custom-red-600 text-white hover:bg-red-700 hover:shadow-lg': canExecuteRestore,
                                        'bg-gray-300 text-gray-400 cursor-not-allowed': !canExecuteRestore
                                    }">
                                <span v-if="!restoreInProgress">执行数据恢复</span>
                                <div v-else class="flex items-center justify-center gap-2">
                                    <div class="w-4 h-4 border-2 border-white border-t-transparent rounded-none animate-spin"></div>
                                    <span>恢复中，请勿关闭页面...</span>
                                </div>
                            </button>
                            <div v-if="restoreInProgress" class="text-center">
                                <div class="text-[10px] text-gray-500">进度: {{ restoreProgress }}%</div>
                                <div class="w-full h-1 bg-gray-200 overflow-hidden mt-1">
                                    <div class="h-full bg-custom-red-600 transition-all duration-300" 
                                         :style="{width: restoreProgress + '%'}"></div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
            <button @click="saveBackupConfig" 
                    class="btn-primary py-4 text-[11px] font-black uppercase tracking-[0.2em] shadow-[4px_4px_0px_rgba(0,0,0,0.1)]">
                保存备份配置
            </button>
            <button @click="resetBackupConfig" 
                    class="btn-secondary py-4 text-[11px] font-black uppercase tracking-[0.2em] shadow-[4px_4px_0px_rgba(0,0,0,0.05)]">
                恢复默认配置
            </button>
        </div>
    </div>
    `,
    data() {
        return {
            backupInProgress: false,
            restoreInProgress: false,
            restoreProgress: 0,
            selectedRestoreBackup: '',
            confirmRestore: false,
            createPreRestoreBackup: true,
            verifyBeforeRestore: true,
            save_success: false,
            save_error: false,
            backups: [],
            backupStats: {
                totalCount: 0,
                totalSize: '0 Bytes',
                lastBackupTime: null,
                health: 0
            },
            dbStatus: {
                online: true,
                size: '--',
                rows: '--'
            },
            config: {}
        };
    },
    watch: {
        backup_config: {
            immediate: true,
            handler(newVal) {
                if (newVal) {
                    this.config = { ...newVal };
                }
            }
        }
    },
    computed: {
        /**
         * 获取选中的备份文件信息
         */
        selectedBackupInfo() {
            if (!this.selectedRestoreBackup) return '';
            const backup = this.backups.find(b => b.id == this.selectedRestoreBackup);
            if (!backup) return '';
            return `${backup.filename} (${this.formatDateTime(backup.created_at)}, ${this.formatFileSize(backup.size)})`;
        },
        /**
         * 判断是否可以执行恢复操作
         */
        canExecuteRestore() {
            return this.selectedRestoreBackup && 
                   this.confirmRestore && 
                   !this.restoreInProgress;
        }
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
         * 获取备份配置
         */
        async fetchBackupConfig() {
            try {
                const token = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/admin/backup/config', {
                    headers: {
                        'X-Admin-Token': token
                    }
                });
                if (res.ok) {
                    const data = await res.json();
                    this.config = data;
                }
            } catch (err) {
                console.error("加载备份配置失败:", err);
            }
        },

        /**
         * 获取备份文件列表
         */
        async fetchBackupFiles() {
            try {
                const token = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/admin/backup/files', {
                    headers: {
                        'X-Admin-Token': token
                    }
                });
                if (res.ok) {
                    const data = await res.json();
                    this.backups = data.backups || [];
                    this.backupStats = data.stats || {};
                }
            } catch (err) {
                console.error("加载备份文件失败:", err);
            }
        },

        /**
         * 获取数据库状态
         */
        async fetchDatabaseStatus() {
            try {
                const token = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/admin/backup/status', {
                    headers: {
                        'X-Admin-Token': token
                    }
                });
                if (res.ok) {
                    const data = await res.json();
                    this.dbStatus = data;
                }
            } catch (err) {
                console.error("加载数据库状态失败:", err);
            }
        },

        /**
         * 切换备份引擎状态
         */
        toggleBackupEngine() {
            this.config.auto_backup_enabled = !this.config.auto_backup_enabled;
        },

        /**
         * 创建备份
         */
        async createBackup() {
            this.backupInProgress = true;
            try {
                const token = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/admin/backup/create', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-Admin-Token': token
                    }
                });
                if (res.ok) {
                    this.showNotification('success');
                    await this.fetchBackupFiles();
                } else {
                    this.showNotification('error');
                }
            } catch (err) {
                console.error("创建备份失败:", err);
                this.showNotification('error');
            } finally {
                this.backupInProgress = false;
            }
        },

        /**
         * 验证备份文件
         * @param {string} backupId - 备份文件ID
         */
        async verifyBackup(backupId) {
            try {
                const token = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch(`/api/admin/backup/verify/${backupId}`, {
                    method: 'POST',
                    headers: {
                        'X-Admin-Token': token
                    }
                });
                if (res.ok) {
                    const data = await res.json();
                    const index = this.backups.findIndex(b => b.id === backupId);
                    if (index > -1) {
                        this.backups[index].verified = data.verified;
                    }
                    this.showNotification('success');
                } else {
                    this.showNotification('error');
                }
            } catch (err) {
                console.error("验证备份失败:", err);
                this.showNotification('error');
            }
        },

        /**
         * 选择恢复备份
         * @param {string} backupId - 备份文件ID
         */
        selectForRestore(backupId) {
            this.selectedRestoreBackup = backupId;
        },

        /**
         * 下载备份文件
         * @param {string} backupId - 备份文件ID
         */
        async downloadBackup(backupId) {
            const backup = this.backups.find(b => b.id === backupId);
            if (backup) {
                const token = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const url = `/api/admin/backup/download/${backupId}?token=${encodeURIComponent(token)}`;
                window.open(url, '_blank');
            }
        },

        /**
         * 删除备份文件
         * @param {string} backupId - 备份文件ID
         */
        async deleteBackup(backupId) {
            if (!confirm('确定要永久删除此备份文件吗？此操作不可撤销。')) return;
            try {
                const token = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch(`/api/admin/backup/delete/${backupId}`, {
                    method: 'DELETE',
                    headers: {
                        'X-Admin-Token': token
                    }
                });
                if (res.ok) {
                    this.backups = this.backups.filter(b => b.id !== backupId);
                    this.showNotification('success');
                    await this.fetchBackupFiles();
                } else {
                    this.showNotification('error');
                }
            } catch (err) {
                console.error("删除备份失败:", err);
                this.showNotification('error');
            }
        },

        /**
         * 导出数据
         * @param {string} format - 导出格式
         */
        async exportData(format) {
            try {
                const token = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const url = `/api/admin/backup/export/${format}?token=${encodeURIComponent(token)}`;
                window.open(url, '_blank');
            } catch (err) {
                console.error("导出数据失败:", err);
                this.showNotification('error');
            }
        },

        /**
         * 执行数据恢复
         */
        async executeRestore() {
            if (!this.canExecuteRestore) return;
            this.restoreInProgress = true;
            this.restoreProgress = 0;
            try {
                const token = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const backupId = this.selectedRestoreBackup;
                const res = await fetch('/api/admin/backup/restore', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-Admin-Token': token
                    },
                    body: JSON.stringify({
                        backupId: backupId,
                        createPreRestoreBackup: this.createPreRestoreBackup,
                        verifyBeforeRestore: this.verifyBeforeRestore
                    })
                });
                if (res.ok) {
                    this.restoreProgress = 100;
                    this.showNotification('success');
                    setTimeout(() => {
                        this.restoreInProgress = false;
                        this.restoreProgress = 0;
                        this.selectedRestoreBackup = '';
                        this.confirmRestore = false;
                    }, 1000);
                } else {
                    this.showNotification('error');
                    this.restoreInProgress = false;
                }
            } catch (err) {
                console.error("恢复备份失败:", err);
                this.showNotification('error');
                this.restoreInProgress = false;
            }
        },

        /**
         * 保存备份配置
         */
        async saveBackupConfig() {
            try {
                const token = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/admin/backup/config', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-Admin-Token': token
                    },
                    body: JSON.stringify(this.config)
                });
                if (res.ok) {
                    this.showNotification('success');
                } else {
                    this.showNotification('error');
                }
            } catch (err) {
                console.error("保存配置失败:", err);
                this.showNotification('error');
            }
        },

        /**
         * 重置备份配置
         */
        async resetBackupConfig() {
            if (!confirm("确定要恢复所有参数至系统预设值吗？")) return;
            try {
                const token = document.cookie.split('; ').find(row => row.startsWith('slite_admin_token='))?.split('=')[1] || '';
                const res = await fetch('/api/admin/backup/config/reset', {
                    method: 'POST',
                    headers: {
                        'X-Admin-Token': token
                    }
                });
                if (res.ok) {
                    const data = await res.json();
                    this.config = data;
                    this.showNotification('success');
                } else {
                    this.showNotification('error');
                }
            } catch (err) {
                console.error("重置配置失败:", err);
                this.showNotification('error');
            }
        },

        /**
         * 格式化日期时间
         * @param {string} isoString - ISO格式日期字符串
         * @returns {string} 格式化后的日期时间
         */
        formatDateTime(isoString) {
            if (!isoString) return '--';
            try {
                const date = new Date(isoString);
                return `${date.getFullYear()}-${(date.getMonth()+1).toString().padStart(2, '0')}-${date.getDate().toString().padStart(2, '0')} ${date.getHours().toString().padStart(2, '0')}:${date.getMinutes().toString().padStart(2, '0')}`;
            } catch (err) {
                return isoString;
            }
        },

        /**
         * 格式化文件大小
         * @param {number} bytes - 文件大小（字节）
         * @returns {string} 格式化后的文件大小
         */
        formatFileSize(bytes) {
            if (!bytes) return '0 Bytes';
            const k = 1024;
            const sizes = ['Bytes', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        }
    },
    mounted() {
        this.fetchBackupConfig();
        this.fetchBackupFiles();
        this.fetchDatabaseStatus();
    }
};

window.BackupManagerComponent = BackupManagerComponent;