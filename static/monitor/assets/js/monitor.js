/**
 * monitor.js - 监控页面核心控制器
 * 支持动态主题加载、实时数据同步和图表展示
 */

const { createApp, ref, reactive, onMounted, computed, nextTick, onBeforeUnmount } = Vue;

createApp({
    setup() {
        const viewToken = ref(localStorage.getItem('slite_view_token') || '');
        const wsStatus = ref('<span class="status-dot-yellow"></span>正在连接同步服务...');
        const servers = ref([]);
        const groups = ref([]);
        let socket = null;
        let reconnectTimer = null;
        const loginKey = ref('');
        
        const isLoggedIn = computed(() => {
            if (!siteConfig.view_auth_enabled) return true;
            return !!localStorage.getItem('slite_view_token') && viewToken.value !== '';
        });
        
        const hasAdminToken = computed(() => {
            return !!localStorage.getItem('slite_admin_token');
        });

        const showThemeDropdown = ref(false);
        const currentTheme = ref('default');
        let themeDropdownClickHandler = null;
        
        const colorPalette = [
            '#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#06b6d4', '#14b8a6',
            '#6366f1', '#f43f5e', '#10b981', '#fbbf24', '#2dd4bf', '#a855f7', '#fb7185', '#4ade80'
        ];

        const siteConfig = reactive({
            id: 1,
            site_logo: '',
            site_name: 'SLITE MONITORING',
            site_slogan: 'LOADING...',
            site_footer: '© 2026 SLITE MONITORING',
            version: '0.1.0',
            view_auth_enabled: false,
            total_stats_days: null
        });

        const globalSettings = reactive({
            show_cpu: true,
            show_mem: true,
            show_disk: true,
            show_docker: true,
            show_net_delay: true,
            show_heatmap: true,
            show_task_list: true,
            show_online_rate: true,
            node_refresh_interval: 5,
            max_node_display: 0
        });

        const charts = {};
        const detailVersions = reactive({});
        const extendedHistoryCache = reactive({});
        const timeRangeOptions = {
            1: '1小时', 12: '12小时', 24: '24小时', 72: '3天', 168: '7天', 720: '30天'
        };
        const availableTimeRanges = reactive({});
        const isLoading = ref(true);

        /*
         * 显示加载指示器
         */
        const showLoadingIndicator = () => {
            const indicator = document.getElementById('loading-indicator');
            if (indicator) {
                indicator.classList.remove('hidden');
                indicator.style.display = 'flex';
            }
        };

        /*
         * 隐藏加载指示器并显示应用
         */
        const hideLoadingAndShowApp = () => {
            if (typeof window.hideLoadingIndicator === 'function') {
                window.hideLoadingIndicator();
            }
            
            const indicator = document.getElementById('loading-indicator');
            if (indicator) {
                indicator.classList.add('hidden');
                indicator.style.display = 'none';
            }
            
            const app = document.getElementById('app');
            if (app) {
                app.classList.add('app-ready');
                app.style.visibility = 'visible';
                app.style.opacity = 1;
                app.style.transition = 'opacity 0.3s ease';
            }
            
            isLoading.value = false;
        };

        /*
         * 更新下拉菜单主题样式
         */
        const updateDropdownTheme = () => {
            const dropdown = document.querySelector('.theme-switcher-dropdown');
            if (!dropdown) return;
            
            if (currentTheme.value === 'dark') {
                dropdown.classList.add('theme-dark');
                dropdown.classList.remove('theme-default');
            } else {
                dropdown.classList.remove('theme-dark');
                dropdown.classList.add('theme-default');
            }
        };

        /*
         * 切换主题函数
         */
        const switchTheme = (themeName) => {
            if (!themeName || !['default', 'dark'].includes(themeName)) {
                showThemeDropdown.value = false;
                return;
            }
            
            showThemeDropdown.value = false;
            
            if (window.switchTheme) {
                window.switchTheme(themeName);
                currentTheme.value = themeName;
                updateDropdownTheme();
                
                setTimeout(() => {
                    Object.values(charts).forEach(chart => {
                        if (chart && typeof chart.update === 'function') {
                            chart.update('none');
                        }
                    });
                }, 100);
            }
        };

        /*
         * 切换主题下拉框显示状态
         */
        const toggleThemeDropdown = (e) => {
            if (e) {
                e.stopPropagation();
                e.preventDefault();
            }
            
            showThemeDropdown.value = !showThemeDropdown.value;
            
            if (showThemeDropdown.value) {
                if (themeDropdownClickHandler) {
                    document.removeEventListener('click', themeDropdownClickHandler);
                }
                
                themeDropdownClickHandler = (clickEvent) => {
                    const target = clickEvent.target;
                    if (!target.closest('.theme-switcher-wrapper')) {
                        showThemeDropdown.value = false;
                        document.removeEventListener('click', themeDropdownClickHandler);
                        themeDropdownClickHandler = null;
                    }
                };
                
                setTimeout(() => {
                    document.addEventListener('click', themeDropdownClickHandler);
                }, 100);
                
                nextTick(() => {
                    updateDropdownTheme();
                });
            } else {
                if (themeDropdownClickHandler) {
                    document.removeEventListener('click', themeDropdownClickHandler);
                    themeDropdownClickHandler = null;
                }
            }
        };

        /*
         * 阻止下拉菜单事件传播
         */
        const stopDropdownEventPropagation = (e) => {
            if (e) {
                e.stopPropagation();
                e.preventDefault();
            }
        };

        /*
         * 带认证的Fetch请求
         */
        const authorizedFetch = async (url, options = {}) => {
            const token = localStorage.getItem('slite_view_token') || '';
            const headers = {
                ...options.headers,
                'Content-Type': 'application/json',
                'X-View-Token': token
            };

            try {
                const response = await fetch(url, { ...options, headers });
                if (response && response.status === 401) {
                    localStorage.removeItem('slite_view_token');
                    localStorage.removeItem('slite_monitor_temp_theme');
                    document.cookie = "slite_view_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 UTC;";
                    window.location.href = '/monitor/login.html';
                    return null;
                }
                return response;
            } catch (err) {
                console.warn('authorizedFetch error:', err);
                return null;
            }
        };

        /*
         * 根据分组ID获取分组名称
         */
        const getGroupNameById = (groupId) => {
            if (!groupId) return '';
            const group = groups.value.find(g => g.id === groupId);
            return group ? group.name : `分组${groupId}`;
        };

        /*
         * 初始化WebSocket连接
         */
        const initWebSocket = () => {
            if (socket) { 
                try { socket.close(); } catch(e) {}
                socket = null;
            }

            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}/ws/stats`;

            socket = new WebSocket(wsUrl);

            socket.onopen = () => {
                wsStatus.value = '<span class="status-dot-green">■</span>实时同步中';
                if (reconnectTimer) {
                    clearTimeout(reconnectTimer);
                    reconnectTimer = null;
                }
            };

            socket.onmessage = (event) => {
                try {
                    const data = JSON.parse(event.data);
                    
                    if (data.groups && Array.isArray(data.groups)) {
                        groups.value = data.groups;
                    }
                    
                    if (data.nodes && Array.isArray(data.nodes)) {
                        handleServerData(data.nodes);
                    } else if (Array.isArray(data)) {
                        handleServerData(data);
                    }
                } catch (e) {
                    // 忽略解析错误
                }
            };

            socket.onclose = () => {
                wsStatus.value = '<span class="status-dot-red">■</span> 连接已断开，尝试重连...';
                if (!reconnectTimer) {
                    reconnectTimer = setTimeout(initWebSocket, 3000);
                }
            };

            socket.onerror = (err) => {
                socket.close();
            };
        };

        /*
         * 稳定更新任务数据
         */
        const updateTasksStably = (server, newTasks) => {
            if (!Array.isArray(newTasks)) {
                newTasks = [];
            }
            if (!server.tasks) {
                server.tasks = [];
            }

            if (server.tasks.length === 0) {
                server.tasks = newTasks.map(task => ({
                    ...task,
                    _sortKey: task.id || task.name || Math.random().toString(36).substr(2, 9)
                }));
                return;
            }

            const existingTasksMap = new Map();
            server.tasks.forEach(task => {
                existingTasksMap.set(task.id, task);
            });

            const newTasksMap = new Map();
            newTasks.forEach(task => {
                newTasksMap.set(task.id, {
                    ...task,
                    _sortKey: existingTasksMap.has(task.id) 
                        ? existingTasksMap.get(task.id)._sortKey
                        : task.id || task.name || Math.random().toString(36).substr(2, 9)
                });
            });

            // 删除不再存在的任务
            server.tasks = server.tasks.filter(task => newTasksMap.has(task.id));

            // 更新已有任务属性
            server.tasks.forEach(task => {
                const newTaskData = newTasksMap.get(task.id);
                if (newTaskData) {
                    task.name = newTaskData.name;
                    task.history = newTaskData.history || [];
                    task.loss_rate = newTaskData.loss_rate;
                    task.delay = newTaskData.delay;
                    task.status = newTaskData.status;
                }
            });

            // 添加新增任务
            newTasks.forEach(newTask => {
                if (!existingTasksMap.has(newTask.id)) {
                    server.tasks.push(newTasksMap.get(newTask.id));
                }
            });

            // 排序
            server.tasks.sort((a, b) => {
                if (a.sort_index !== undefined && b.sort_index !== undefined) {
                    return a.sort_index - b.sort_index;
                }
                return a._sortKey.localeCompare(b._sortKey);
            });
        };

        /*
         * 处理服务器数据更新
         */
        const handleServerData = (serverData) => {
            serverData.forEach(newItem => {
                if (newItem.is_visible === false) {
                    const index = servers.value.findIndex(s => s.id === newItem.id);
                    if (index !== -1) {
                        servers.value.splice(index, 1);
                    }
                    return;
                }

                let server = servers.value.find(s => s.id === newItem.id);
                
                if (!server) {
                    server = reactive({
                        ...newItem,
                        _showDetails: false,
                        _activeTaskId: null,
                        _selectedTimeRange: 0,
                        total_stats_days: newItem.total_stats_days || 0,
                        tasks: []
                    });
                    servers.value.push(server);
                } else {
                    if (newItem.total_stats_days && newItem.total_stats_days > 0) {
                        server.total_stats_days = newItem.total_stats_days;
                    }
                }

                if (newItem.total_stats_days && newItem.total_stats_days > 0) {
                    siteConfig.total_stats_days = newItem.total_stats_days;
                }

                server.online = newItem.online;
                server.group_id = newItem.group_id;
                server.group_name = getGroupNameById(newItem.group_id);
                server.name = newItem.name;
                server.public_ip = newItem.ip || newItem.public_ip;
                server.location = newItem.location;
                server.region = newItem.location;
                server.os = newItem.os;
                
                server.cpu = safePercent(newItem.cpu);
                server.mem = safePercent(newItem.mem);
                server.disk = safePercent(newItem.disk);
                
                const rawUptimeRate = typeof newItem.uptime_rate === 'number' ? newItem.uptime_rate : (newItem.online ? 100 : 0);
                server.uptime_rate = safePercent(rawUptimeRate);

                server.up = newItem.up || 0;
                server.down = newItem.down || 0;
                server.net_in = newItem.net_in;
                server.net_out = newItem.net_out;
                server.uptime = newItem.uptime;
                server.load = newItem.load;
                server.process_count = newItem.process_count || 0;
                server.month_up = newItem.month_up || 0;
                server.month_down = newItem.month_down || 0;
                server.offline_total = newItem.offline_total || '无离线记录';
                server.sort_index = newItem.sort_index || 0;
                
                const finalRenderDays = server.total_stats_days || siteConfig.total_stats_days || 1;
                server.uptime_heat_map = getFullHeatMap(newItem.uptime_heat_map, newItem.online, finalRenderDays);

                // 处理 Docker
                if (newItem.docker && newItem.docker.container_list) {
                    const sortedContainers = newItem.docker.container_list
                        .sort((a, b) => a.name.localeCompare(b.name))
                        .map(c => ({
                            ...c,
                            cpu_usage: c.cpu_usage ? parseFloat(c.cpu_usage).toFixed(1) : '0.0'
                        }));
                    server.docker = { ...newItem.docker, container_list: sortedContainers };
                } else {
                    // 如果没有 docker 数据，保留已有或置空
                    if (!server.docker) server.docker = null;
                }

                // 处理任务
                if (newItem.tasks) {
                    const processedTasks = newItem.tasks.map(t => {
                        const cacheKey = `${server.id}-${t.id}`;
                        let history = Array.isArray(t.history) ? t.history : [];

                        if (server._selectedTimeRange === 0) {
                            if (!extendedHistoryCache[cacheKey]) {
                                extendedHistoryCache[cacheKey] = history;
                            } else {
                                const longHistory = extendedHistoryCache[cacheKey];
                                const newestPoint = history[history.length - 1];

                                if (newestPoint !== undefined && newestPoint !== longHistory[longHistory.length - 1]) {
                                    longHistory.push(newestPoint);
                                    if (longHistory.length > 300) longHistory.shift();
                                }
                                history = [...longHistory];
                            }
                        } else {
                            const rangeKey = `${server.id}-${t.id}-${server._selectedTimeRange}`;
                            if (extendedHistoryCache[rangeKey]) {
                                const cached = extendedHistoryCache[rangeKey];
                                history = cached.values || cached.history || [];
                            }
                        }
                        return { ...t, history: history };
                    });

                    updateTasksStably(server, processedTasks);
                }

                // 自动更新图表（如果详情展开且版本一致）
                if (server._showDetails && detailVersions[server.id]) {
                    const version = detailVersions[server.id];
                    nextTick(() => {
                        if (server._showDetails && detailVersions[server.id] === version) {
                            updateDelayChart(server, false, 'none');
                        }
                    });
                }
            });
        };

        /*
         * 安全的百分比转换
         */
        const safePercent = (val) => {
            const n = parseFloat(val);
            return isNaN(n) ? 0 : Math.min(100, Math.max(0, n));
        };

        /*
         * 获取任务时间范围数据
         */
        const fetchTaskTimeRange = async (serverId, taskId) => {
            try {
                const res = await authorizedFetch(`/api/task/history/timerange?node_id=${serverId}&task_id=${taskId}`);
                if (res && res.ok) {
                    const data = await res.json();
                    const cacheKey = `${serverId}-${taskId}`;
                    availableTimeRanges[cacheKey] = data.max_hours || 1;
                    return data.max_hours || 1;
                }
            } catch (err) {
                // ignore
            }
            return 1;
        };

        /*
         * 获取可用时间选项
         */
        const getAvailableTimeOptions = (maxHours) => {
            const options = [{ hours: 0, label: '最新' }];
            const timeRangeKeys = [1, 12, 24, 72, 168, 720];
            for (const hours of timeRangeKeys) {
                if (hours <= maxHours) {
                    options.push({ hours, label: timeRangeOptions[hours] });
                }
            }
            return options;
        };

        /*
         * 获取扩展历史数据
         */
        const fetchTaskExtendedHistory = async (serverId, taskId, hours = 1) => {
            try {
                const res = await authorizedFetch(`/api/task/history?node_id=${serverId}&task_id=${taskId}&hours=${hours}`);
                if (res && res.ok) {
                    const data = await res.json();
                    const cacheKey = `${serverId}-${taskId}-${hours}`;
                    extendedHistoryCache[cacheKey] = data;

                    const server = servers.value.find(s => s.id === serverId);
                    if (server && server._showDetails && server._selectedTimeRange === hours) {
                        nextTick(() => updateDelayChart(server, false, 'default'));
                    }
                }
            } catch (err) {
                // ignore
            }
        };

        /*
         * 初始化应用
         */
        const initApp = async () => {
            try {
                const loadTimeout = setTimeout(() => {
                    if (isLoading.value) {
                        hideLoadingAndShowApp();
                    }
                }, 5000);
                
                showLoadingIndicator();
                
                const resPub = await fetch('/api/config/public');
                if (resPub && resPub.ok) {
                    const publicData = await resPub.json();
                    Object.assign(siteConfig, publicData);
                }

                if (siteConfig.view_auth_enabled && !localStorage.getItem('slite_view_token')) {
                    window.location.href = '/monitor/login.html';
                    clearTimeout(loadTimeout);
                    return;
                }

                const resSett = await authorizedFetch('/api/config/settings');
                if (resSett && resSett.ok) {
                    const settingsData = await resSett.json();
                    Object.assign(globalSettings, settingsData);
                    initWebSocket();
                } else {
                    // 即使设置接口失败，也尝试启动 WebSocket（使用默认设置）
                    initWebSocket();
                }
                
                if (window.getCurrentTheme) {
                    currentTheme.value = window.getCurrentTheme();
                }
                
                clearTimeout(loadTimeout);
                hideLoadingAndShowApp();
                
            } catch (err) {
                console.error('Init error:', err);
                wsStatus.value = '<span class="status-dot-red"></span> 配置连接失败';
                hideLoadingAndShowApp();
            }
        };

        /*
         * 处理登录请求
         */
        const handleLogin = async () => {
            try {
                const response = await fetch('/monitor/login', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ key: loginKey.value })
                });
                
                if (response && response.ok) {
                    const data = await response.json();
                    localStorage.setItem('slite_view_token', data.token);
                    viewToken.value = data.token;
                    window.location.reload();
                } else {
                    alert('验证密钥错误');
                }
            } catch (err) {
                alert('登录请求失败');
            }
        };

        /*
         * 计算排序后的服务器列表
         */
        const sortedServers = computed(() => {
            let list = [...servers.value].sort((a, b) => {
                if (a.sort_index !== b.sort_index) {
                    return a.sort_index - b.sort_index;
                }
                return (a.name || '').localeCompare(b.name || '');
            });

            if (globalSettings.max_node_display > 0) {
                list = list.slice(0, globalSettings.max_node_display);
            }
            return list;
        });

        /*
         * 计算总上传字节数
         */
        const totalUploadBytes = computed(() => servers.value.reduce((acc, s) => acc + (s.month_up || 0), 0));

        /*
         * 计算总下载字节数
         */
        const totalDownloadBytes = computed(() => servers.value.reduce((acc, s) => acc + (s.month_down || 0), 0));

        /*
         * 根据百分比获取资源进度条样式类
         */
        const getResourceProgressClass = (percent) => {
            const val = parseFloat(percent);
            if (val >= 90) return 'progress-danger';
            if (val >= 75) return 'progress-warning';
            return 'progress-normal';
        };

        /*
         * 格式化百分比显示
         */
        const formatPercent = (val) => {
            const n = parseFloat(val);
            return isNaN(n) ? '0.0' : n.toFixed(1);
        };

        /*
         * 格式化网速显示
         */
        const formatSpeed = (kbps) => {
            if (!kbps || kbps <= 0) return '0 KB/s';
            if (kbps < 1024) return kbps.toFixed(1) + ' KB/s';
            return (kbps / 1024).toFixed(2) + ' MB/s';
        };

        /*
         * 格式化流量单位
         */
        const formatTrafficUnit = (bytes) => {
            if (!bytes || bytes <= 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        };

        /*
         * 格式化在线时长（简略版）
         */
        const formatUptimeShort = (seconds) => {
            if (!seconds) return '-';
            const d = Math.floor(seconds / (3600 * 24));
            if (d > 0) return `${d}天`;
            const h = Math.floor(seconds / 3600);
            return h > 0 ? `${h}小时` : '最近启动';
        };

        /*
         * 格式化系统负载显示
         */
        const formatLoad = (load) => {
            if (!load) return '0.00 0.00 0.00';
            if (Array.isArray(load)) return load.map(v => (typeof v === 'number' ? v.toFixed(2) : v)).join(' ');
            return load;
        };

        /*
         * 计算丢包率
         */
        const calculateLossRate = (history) => {
            if (!history || history.length === 0) return "0.0";
            const lossCount = history.filter(v => v <= 0).length;
            return ((lossCount / history.length) * 100).toFixed(1);
        };

        /*
         * 生成完整的在线率热力图数据
         */
        const getFullHeatMap = (dbHeatmap, currentOnline, serverTotalDays = null) => {
            const totalDays = parseInt(serverTotalDays || siteConfig.total_stats_days || 1);
            const result = [];
            if (isNaN(totalDays) || totalDays <= 0) return result;

            const maxDisplayDays = 30;
            const displayDays = Math.min(totalDays, maxDisplayDays);
            
            const now = new Date();
            const dataMap = {};
            if (Array.isArray(dbHeatmap)) {
                dbHeatmap.forEach(item => { dataMap[item.date] = item; });
            }

            for (let i = 0; i < displayDays; i++) {
                const d = new Date(now);
                d.setDate(now.getDate() - i);
                const dateStr = d.toISOString().split('T')[0];
                const displayDate = `${d.getMonth() + 1}-${d.getDate()}`; 
                const isToday = i === 0;

                if (dataMap[dateStr]) {
                    const day = dataMap[dateStr];
                    const rate = typeof day.uptime_rate === 'number' ? day.uptime_rate : 0;
                    let status = rate >= 100 ? 'online' : (rate <= 0 ? 'offline' : 'warning');
                    result.push({ 
                        ...day, status,
                        displayInfo: `${displayDate}\n${rate.toFixed(0)}%` 
                    });
                } else {
                    const rate = isToday ? (currentOnline ? 100 : 0) : 0;
                    const status = isToday ? (currentOnline ? 'online' : 'offline') : 'none';
                    const rateText = isToday ? (currentOnline ? '100%' : '0%') : '无数据';
                    result.push({
                        status, date: dateStr,
                        displayInfo: `${displayDate}\n${rateText}`,
                        uptime_rate: rate
                    });
                }
            }
            return result; 
        };

        /*
         * 切换服务器详情显示
         */
        const toggleDetails = async (server) => {
            const id = server.id;
            if (!detailVersions[id]) {
                detailVersions[id] = 0;
            }
            const version = ++detailVersions[id];
            const opening = !server._showDetails;
            server._showDetails = opening;

            if (!opening) {
                server._activeTaskId = null;
                server._selectedTimeRange = 0;
                if (charts[id]) {
                    try {
                        charts[id].destroy();
                    } catch(e){}
                    delete charts[id];
                }
                return;
            }

            // 展开
            server._activeTaskId = null;
            server._selectedTimeRange = 0;

            if (server.tasks && server.tasks.length) {
                for (const task of server.tasks) {
                    const key = `${id}-${task.id}`;
                    if (!availableTimeRanges[key]) {
                        await fetchTaskTimeRange(id, task.id);
                    }
                    // 如果在此期间用户收起或版本变更，则退出
                    if (detailVersions[id] !== version || !server._showDetails) {
                        return;
                    }
                }
            }

            nextTick(() => {
                if (server._showDetails && detailVersions[id] === version) {
                    updateDelayChart(server, true);
                }
            });
        };

        /*
         * 选择时间范围
         */
        const selectTimeRange = (server, hours) => {
            if (server._selectedTimeRange === hours) return;
            server._selectedTimeRange = hours;

            if (hours > 0 && server.tasks && server.tasks.length > 0) {
                server.tasks.forEach(task => {
                    const cacheKey = `${server.id}-${task.id}-${hours}`;
                    if (!extendedHistoryCache[cacheKey]) {
                        fetchTaskExtendedHistory(server.id, task.id, hours);
                    }
                });
            }
            if (server._showDetails) {
                nextTick(() => updateDelayChart(server, false, 'default'));
            }
        };

        /*
         * 获取服务器时间选项
         */
        const getServerTimeOptions = (server) => {
            if (!server.tasks || server.tasks.length === 0) return getAvailableTimeOptions(0);
            let maxHours = 0;
            for (const task of server.tasks) {
                const cacheKey = `${server.id}-${task.id}`;
                maxHours = Math.max(maxHours, availableTimeRanges[cacheKey] || 0);
            }
            return getAvailableTimeOptions(maxHours);
        };

        /*
         * 更新延迟图表
         */
        const updateDelayChart = (server, forceInit = false, updateMode = 'default') => {
            if (!server._showDetails) return;
            
            const canvas = document.getElementById(`chart-${server.id}`);
            if (!canvas) return;
            if (!server.tasks || server.tasks.length === 0) {
                // 清空画布或显示无数据
                const ctx = canvas.getContext('2d');
                ctx.clearRect(0, 0, canvas.width, canvas.height);
                return;
            }

            const selectedHours = server._selectedTimeRange || 0;
            const datasets = server.tasks.map((task, idx) => {
                const color = colorPalette[idx % colorPalette.length];
                const isFiltered = server._activeTaskId !== null;
                const isActive = server._activeTaskId === task.id;
                const isHidden = isFiltered && !isActive;

                let plotData = task.history || [];
                if (selectedHours > 0) {
                    const cacheKey = `${server.id}-${task.id}-${selectedHours}`;
                    if (extendedHistoryCache[cacheKey]) {
                        plotData = extendedHistoryCache[cacheKey].values || extendedHistoryCache[cacheKey].history || [];
                    } else {
                        plotData = [];
                    }
                }

                return {
                    label: task.name,
                    data: plotData,
                    borderColor: color,
                    borderWidth: isActive ? 2.5 : 1.2,
                    backgroundColor: isActive ? color + '33' : 'transparent',
                    fill: isActive,
                    pointRadius: 0,
                    pointHoverRadius: 4,
                    tension: 0.4,
                    hidden: isHidden
                };
            });

            const maxHistoryLen = Math.max(...datasets.map(d => d.data.length), 1);
            const now = new Date();
            const interval = globalSettings.node_refresh_interval || 5;
            
            let labels = [];
            const firstTask = server.tasks[0];
            const firstCacheKey = `${server.id}-${firstTask.id}-${selectedHours}`;
            
            if (selectedHours > 0 && extendedHistoryCache[firstCacheKey]?.labels) {
                labels = extendedHistoryCache[firstCacheKey].labels;
            } else {
                labels = Array.from({length: maxHistoryLen}, (_, i) => {
                    const pointDate = new Date(now.getTime() - (maxHistoryLen - 1 - i) * interval * 1000);
                    return pointDate.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false });
                });
            }

            if (charts[server.id] && !forceInit) {
                const chart = charts[server.id];
                chart.data.labels = labels;
                chart.data.datasets = datasets;
                chart.update(updateMode);
            } else {
                if (charts[server.id]) {
                    try { charts[server.id].destroy(); } catch(e) {}
                    delete charts[server.id];
                }
                const ctx = canvas.getContext('2d');
                charts[server.id] = new Chart(ctx, {
                    type: 'line',
                    data: { labels, datasets },
                    options: {
                        responsive: true,
                        maintainAspectRatio: false,
                        interaction: { intersect: false, mode: 'index' },
                        animation: { duration: 400 },
                        plugins: {
                            legend: { display: false },
                            tooltip: {
                                padding: 10,
                                titleFont: { size: 12 },
                                bodyFont: { family: 'monospace', size: 11 },
                                callbacks: { label: (ctx) => ` ${ctx.dataset.label}: ${ctx.parsed.y}ms` }
                            }
                        },
                        scales: {
                            y: {
                                beginAtZero: true,
                                grid: { color: 'rgba(0,0,0,0.05)', drawTicks: false },
                                ticks: {
                                    font: { size: 9, family: 'monospace' },
                                    callback: (val) => val + 'ms'
                                }
                            },
                            x: {
                                display: true,
                                grid: { display: false },
                                ticks: {
                                    font: { size: 8, family: 'monospace' },
                                    maxRotation: 0,
                                    autoSkip: true,
                                    maxTicksLimit: 8,
                                    color: '#999'
                                }
                            }
                        }
                    }
                });
            }
        };

        /*
         * 选择任务焦点
         */
        const selectTask = (server, taskId) => {
            const isUnselecting = server._activeTaskId === taskId;
            server._activeTaskId = isUnselecting ? null : taskId;
            
            if (!isUnselecting && server._selectedTimeRange > 0) {
                const cacheKey = `${server.id}-${taskId}-${server._selectedTimeRange}`;
                if (!extendedHistoryCache[cacheKey]) {
                    fetchTaskExtendedHistory(server.id, taskId, server._selectedTimeRange);
                }
            }
            if (server._showDetails) {
                nextTick(() => updateDelayChart(server, false, 'default'));
            }
        };

        /*
         * 跳转到管理面板
         */
        const toAdminPanel = () => { window.location.href = '/admin'; };

        /*
         * 处理用户登出
         */
        const handleLogout = () => {
            localStorage.removeItem('slite_view_token');
            localStorage.removeItem('slite_admin_token');
            localStorage.removeItem('slite_monitor_temp_theme');
            document.cookie = "slite_view_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 UTC;";
            document.cookie = "slite_admin_token=; path=/; expires=Thu, 01 Jan 1970 00:00:00 UTC;";
            window.location.href = '/monitor';
        };

        /*
         * 组件挂载生命周期
         */
        onMounted(async () => {
            const baseStyle = document.createElement('style');
            baseStyle.textContent = `
                #app {
                    visibility: hidden;
                    opacity: 0;
                    transition: opacity 0.3s ease;
                }
                #app.app-ready {
                    visibility: visible;
                    opacity: 1;
                }
                #loading-indicator {
                    position: fixed;
                    top: 0;
                    left: 0;
                    width: 100%;
                    height: 100%;
                    background: #fff;
                    z-index: 9999;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    flex-direction: column;
                }
                #loading-indicator.hidden {
                    display: none;
                }
                .theme-switcher-wrapper {
                    position: relative;
                    z-index: 1000;
                }
                .theme-switcher-dropdown {
                    position: absolute;
                    top: 100%;
                    right: 0;
                    min-width: 120px;
                    background: #fff;
                    border: 1px solid #e5e7eb;
                    border-radius: 4px;
                    box-shadow: 0 2px 8px rgba(0,0,0,0.1);
                    padding: 8px 0;
                    margin-top: 4px;
                    opacity: 0;
                    visibility: hidden;
                    transform: translateY(-5px);
                    transition: all 0.2s ease;
                    z-index: 1001;
                    pointer-events: none;
                }
                .theme-switcher-dropdown.show {
                    opacity: 1;
                    visibility: visible;
                    transform: translateY(0);
                    pointer-events: auto;
                }
                .theme-switcher-dropdown.theme-dark {
                    background: #1f2937;
                    border-color: #374151;
                }
                .theme-switcher-dropdown.theme-default {
                    background: #fff;
                    border-color: #e5e7eb;
                }
                .theme-switcher-item {
                    padding: 8px 16px;
                    cursor: pointer;
                    color: #333;
                    font-size: 14px;
                    transition: background-color 0.1s ease;
                }
                .theme-switcher-dropdown.theme-dark .theme-switcher-item {
                    color: #e5e7eb;
                }
                .theme-switcher-item:hover {
                    background-color: #f3f4f6;
                }
                .theme-switcher-dropdown.theme-dark .theme-switcher-item:hover {
                    background-color: #374151;
                }
                .chart-container, .task-list-container {
                    border: 1px solid #000 !important;
                    box-sizing: border-box;
                }
                .uptime-day {
                    position: relative;
                    width: 38px !important; 
                    height: 38px !important;
                    margin: 1px;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    font-size: 10px;
                    line-height: 1.1;
                    text-align: center;
                    color: #fff;
                    white-space: pre-line;
                    border-radius: 2px;
                    cursor: default;
                }
                .uptime-day.online { background-color: #10b981 !important; }
                .uptime-day.warning { background-color: #f59e0b !important; }
                .uptime-day.offline { background-color: #ef4444 !important; }
                .uptime-day.none { background-color: #e5e7eb !important; color: #9ca3af; }
                .uptime-day:hover::after { display: none !important; }
            `;
            document.head.appendChild(baseStyle);
            
            await initApp();
        });

        /*
         * 组件卸载前清理
         */
        onBeforeUnmount(() => {
            if (socket) {
                try { socket.close(); } catch(e) {}
                socket = null;
            }
            if (reconnectTimer) {
                clearTimeout(reconnectTimer);
                reconnectTimer = null;
            }
            if (themeDropdownClickHandler) {
                document.removeEventListener('click', themeDropdownClickHandler);
                themeDropdownClickHandler = null;
            }
            Object.values(charts).forEach(c => {
                if (c && typeof c.destroy === 'function') c.destroy();
            });
        });

        return {
            wsStatus, 
            servers, 
            groups,
            siteConfig, 
            globalSettings,
            sortedServers, 
            totalUploadBytes, 
            totalDownloadBytes, 
            colorPalette,
            loginKey,
            isLoggedIn,
            hasAdminToken,
            handleLogin,
            handleLogout, 
            toAdminPanel,
            getResourceProgressClass, 
            formatPercent, 
            formatSpeed, 
            formatTrafficUnit,
            formatUptimeShort, 
            formatLoad, 
            getFullHeatMap, 
            calculateLossRate,
            toggleDetails, 
            selectTask, 
            selectTimeRange, 
            getServerTimeOptions,
            isLoading,
            showThemeDropdown,
            currentTheme,
            switchTheme,
            toggleThemeDropdown,
            stopDropdownEventPropagation
        };
    }
}).mount('#app');