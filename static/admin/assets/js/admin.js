/**
 * Admin.js - 管理后台核心控制器
 * 负责系统配置分发、状态管理及子组件调度
 * 集成模块：Site, Basic, Nodes, Network, Warning, Security, Logs, About, Backup
 */

const { createApp, ref, computed, onMounted, watch } = Vue;

const MINIMAL_BORDER = 'border: 1px solid #000000';

/**
 * 组件占位模板
 * 当特定功能的组件脚本未加载时，显示此提示信息
 */
const PLACEHOLDER_TEMPLATE = (name) => `
    <div class="p-20 text-center font-black opacity-30 uppercase tracking-widest m-10" style="${MINIMAL_BORDER}">
        ${name} System Under Development
    </div>
`;

createApp({
    /*
     * 组件注册
     * 动态加载各功能模块组件，未加载时显示占位模板
     */
    components: {
        'site': typeof SiteComponent !== 'undefined' ? SiteComponent : { template: PLACEHOLDER_TEMPLATE('Site') },
        'basic': typeof BasicComponent !== 'undefined' ? BasicComponent : { template: PLACEHOLDER_TEMPLATE('Basic') },
        'nodes': typeof NodesComponent !== 'undefined' ? NodesComponent : { template: PLACEHOLDER_TEMPLATE('Nodes') },
        'network': typeof NetworkComponent !== 'undefined' ? NetworkComponent : { template: PLACEHOLDER_TEMPLATE('Network') },
        'alert': typeof AlertComponent !== 'undefined' ? AlertComponent : { template: PLACEHOLDER_TEMPLATE('Alert') },
        'security': typeof SecurityComponent !== 'undefined' ? SecurityComponent : { template: PLACEHOLDER_TEMPLATE('Security') },
        'backup': typeof BackupManagerComponent !== 'undefined' ? BackupManagerComponent : { template: PLACEHOLDER_TEMPLATE('Backup') },
        'logs': typeof LogsComponent !== 'undefined' ? LogsComponent : { template: PLACEHOLDER_TEMPLATE('Logs') },
        'about': typeof AboutComponent !== 'undefined' ? AboutComponent : { template: PLACEHOLDER_TEMPLATE('About') }
    },

    setup() {
        const currentTab = ref('site');
        const isSaving = ref(false);
        const saveSuccess = ref(false);
        const adminToken = ref(localStorage.getItem('slite_admin_token') || '');
        
        /*
         * 网站全局视觉配置状态
         * 对应SiteConfig表，存储网站基本信息
         */
        const siteConfig = ref({
            id: 1,
            site_logo: "",
            site_name: "SLITE HUB",
            site_slogan: "分布式实时监控 system",
            site_footer: "© 2026 SLITE SYSTEM",
            version: "v1.0.2"
        });

        /*
         * 系统基础运行配置状态
         * 整合来自不同API端点的配置数据
         * 包含主题配置但主题功能已由theme-switcher.js统一管理
         */
        const basicConfig = ref({
            id: 1,
            show_docker: true,
            show_net_delay: true,
            show_heatmap: true,
            node_refresh_interval: 5,
            task_show_count: 8,
            max_log_count: 100,
            view_auth_enabled: false,
            view_auth_key: "",
            admin_auth_key: "",
            admin_theme: "default",
            monitor_theme: "default",
            theme_mode: "fixed"
        });

        /*
         * 备份系统配置状态
         * 存储数据库备份相关配置参数
         */
        const backupConfig = ref({
            id: 1,
            auto_backup_enabled: false,
            backup_strategy: 'weekly_full_inc',
            backup_retention_days: 30,
            backup_encryption_enabled: true,
            backup_encryption_algorithm: 'AES-256-GCM',
            backup_verify_integrity: true,
            backup_max_count: 10,
            backup_schedule_time: '03:00',
            backup_notification_enabled: true,
            backup_compression_enabled: true
        });

        /*
         * 权限检查函数
         * 验证管理员Token，无效时重定向到登录页
         */
        const checkAuth = () => {
            if (!adminToken.value) {
                window.location.replace('/admin/login');
                return false;
            }
            return true;
        };

        /*
         * 鉴权请求封装函数
         * 自动附加管理员Token到请求头
         */
        const authorizedFetch = async (url, options = {}) => {
            const headers = {
                ...options.headers,
                'Authorization': `Bearer ${adminToken.value}`,
                'X-Admin-Token': adminToken.value
            };
            const res = await fetch(url, { ...options, headers });
            if (res.status === 401) {
                localStorage.removeItem('slite_admin_token');
                window.location.replace('/admin/login');
                throw new Error('Unauthorized');
            }
            return res;
        };

        /*
         * 处理基础设置组件的保存事件
         * 接收子组件数据并触发保存操作
         */
        const handleSaveBasic = (basicData) => {
            handleAction('saveSettings', basicData);
        };

        /*
         * 数据初始化函数
         * 并行加载所有系统配置数据
         */
        const initData = async () => {
            if (!checkAuth()) return;
            try {
                const [resSite, resAdminConfig, resSettings, resBackup] = await Promise.all([
                    authorizedFetch('/api/admin/config'),
                    authorizedFetch('/api/admin/config'),
                    authorizedFetch('/api/config/settings'),
                    authorizedFetch('/api/admin/backup/config')
                ]);

                if (resSite.ok) {
                    const data = await resSite.json();
                    Object.assign(siteConfig.value, data);
                }

                let mergedBasic = {};
                if (resAdminConfig.ok) {
                    const adminData = await resAdminConfig.json();
                    mergedBasic = { ...mergedBasic, ...adminData };
                }
                if (resSettings.ok) {
                    const settingsData = await resSettings.json();
                    mergedBasic = { ...mergedBasic, ...settingsData };
                }
                
                Object.assign(basicConfig.value, mergedBasic);

                if (resBackup.ok) {
                    const backupData = await resBackup.json();
                    Object.assign(backupConfig.value, backupData);
                }
                
            } catch (err) {
                if (err.message !== 'Unauthorized') {
                }
            }
        };

        /*
         * 数据保存逻辑处理函数
         * 根据保存类型分发到不同的API端点
         */
        const handleAction = async (type, payloadFromComponent = null) => {
            if (!checkAuth()) return;
            isSaving.value = true;
            saveSuccess.value = false;

            let url, payload;
            
            switch(type) {
                case 'saveConfig':
                    url = '/api/admin/config';
                    payload = siteConfig.value;
                    break;
                case 'saveSettings':
                    url = '/api/admin/settings';
                    payload = payloadFromComponent || basicConfig.value;
                    break;
                case 'saveBackupConfig':
                    url = '/api/admin/backup/config';
                    payload = backupConfig.value;
                    break;
                default:
                    isSaving.value = false;
                    return;
            }

            try {
                const res = await authorizedFetch(url, {
                    method: 'POST',
                    headers: { 
                        'Content-Type': 'application/json',
                        'X-Admin-Token': adminToken.value
                    },
                    body: JSON.stringify(payload)
                });
                
                if (res.ok) {
                    saveSuccess.value = true;
                    
                    if (type === 'saveSettings' && payloadFromComponent) {
                        Object.assign(basicConfig.value, payloadFromComponent);
                    }
                    
                    setTimeout(() => { 
                        saveSuccess.value = false; 
                    }, 2000);
                } else {
                    await res.text();
                }
            } catch (err) {
                if (err.message !== 'Unauthorized') {
                }
            } finally {
                isSaving.value = false;
            }
        };

        /*
         * 备份相关操作处理函数
         * 支持创建、恢复、删除、验证备份等操作
         */
        const handleBackupAction = async (action, data = {}) => {
            if (!checkAuth()) return false;
            
            try {
                let url = '';
                let options = {
                    method: 'POST',
                    headers: { 
                        'Content-Type': 'application/json',
                        'X-Admin-Token': adminToken.value
                    },
                    body: JSON.stringify(data)
                };

                switch(action) {
                    case 'createBackup': url = '/api/admin/backup/create'; break;
                    case 'restoreBackup': url = '/api/admin/backup/restore'; break;
                    case 'deleteBackup': url = '/api/admin/backup/delete'; options.method = 'DELETE'; break;
                    case 'verifyBackup': url = '/api/admin/backup/verify'; break;
                    case 'exportData': url = '/api/admin/backup/export'; break;
                    default: return false;
                }

                const res = await authorizedFetch(url, options);
                return res.ok;
            } catch (err) {
                return false;
            }
        };

        /*
         * 注销处理函数
         * 清除本地Token并重定向到登录页
         */
        const handleLogout = () => {
            localStorage.removeItem('slite_admin_token');
            window.location.replace('/admin/login');
        };

        /*
         * 组件挂载生命周期
         * 初始化数据和设置监听器
         */
        onMounted(() => {
            initData();
            
            watch(
                () => [basicConfig.value.admin_theme, basicConfig.value.theme_mode],
                () => {
                },
                { deep: true }
            );
        });

        /*
         * 侧边栏菜单项配置
         * 定义管理后台的所有功能模块
         */
        const menuItems = [
            { id: 'site', name: '网站设置' },
            { id: 'basic', name: '基本设置' },
            { id: 'nodes', name: '节点管理' },
            { id: 'network', name: '网络检测' },
            { id: 'alert', name: '预警通知' },
            { id: 'security', name: '安全设置' },
            { id: 'backup', name: '备份管理' },
            { id: 'logs', name: '日志记录' },
            { id: 'about', name: '关于系统' }
        ];

        /*
         * 计算属性：所有传递给子组件的属性
         * 集中管理子组件所需的全部数据和方法
         */
        const allProps = computed(() => ({
            site_config: siteConfig.value,
            basic_config: basicConfig.value,
            backup_config: backupConfig.value,
            is_saving: isSaving.value,
            save_success: saveSuccess.value,
            admin_token: adminToken.value,
            handle_backup_action: handleBackupAction,
            handle_action: handleAction,
            handle_save_basic: handleSaveBasic
        }));

        return {
            currentTab,
            menuItems,
            allProps,
            handleAction,
            handleSaveBasic,
            handleBackupAction,
            handleLogout,
            basicConfig,
            getTabTitle: () => menuItems.find(m => m.id === currentTab.value)?.name
        };
    }
}).mount('#app');