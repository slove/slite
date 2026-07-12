/**
 * 主题切换器 - 统一管理监控系统主题
 */

(function() {
    /*
     * 隐藏页面直到主题加载完成
     * 防止主题切换时的页面闪烁
     */
    document.documentElement.style.visibility = 'hidden';
    
    /*
     * 获取要应用的主题
     * 优先级：用户临时选择 > API配置 > 默认主题
     */
    async function getThemeToApply() {
        const userTempTheme = localStorage.getItem('user_theme_temp');
        
        if (userTempTheme === 'default' || userTempTheme === 'dark') {
            return userTempTheme;
        }
        
        try {
            const response = await fetch('/api/config/public/settings');
            if (response.ok) {
                const data = await response.json();
                return data.monitor_theme || 'default';
            }
        } catch (error) {
        }
        
        return 'default';
    }
    
    /*
     * 加载并应用主题
     * 创建CSS链接并设置页面主题属性
     */
    async function loadAndApplyTheme() {
        const theme = await getThemeToApply();
        
        document.documentElement.setAttribute('data-theme', theme);
        
        const link = document.createElement('link');
        link.id = 'theme-style';
        link.rel = 'stylesheet';
        link.href = `/static/monitor/assets/css/${theme}.css`;
        
        return new Promise((resolve) => {
            link.onload = function() {
                setTimeout(() => {
                    document.documentElement.style.visibility = 'visible';
                    document.documentElement.style.opacity = '1';
                    
                    const app = document.getElementById('app');
                    if (app) {
                        app.classList.add('app-ready');
                    }
                    
                    resolve(theme);
                }, 100);
            };
            
            link.onerror = function() {
                link.href = '/static/monitor/assets/css/default.css';
                document.documentElement.setAttribute('data-theme', 'default');
                document.documentElement.style.visibility = 'visible';
                resolve('default');
            };
            
            document.head.insertBefore(link, document.head.firstChild);
        });
    }
    
    /*
     * 页面加载时立即应用主题
     */
    loadAndApplyTheme().then(theme => {
    });
    
    /*
     * 主题切换函数
     * 用户手动切换主题时调用此函数
     * 参数themeName：要切换的主题名称，支持'default'和'dark'
     */
    window.switchTheme = function(themeName) {
        if (themeName !== 'default' && themeName !== 'dark') return;
        
        localStorage.setItem('user_theme_temp', themeName);
        
        const link = document.getElementById('theme-style');
        if (link) {
            /*
             * 创建全新的CSS链接元素
             * 添加时间戳防止浏览器缓存
             * 确保新旧链接正确替换
             */
            const newLink = document.createElement('link');
            newLink.id = 'theme-style';
            newLink.rel = 'stylesheet';
            
            /*
             * 添加时间戳参数强制重新加载CSS
             * 防止浏览器缓存导致样式不更新
             */
            const timestamp = Date.now();
            newLink.href = `/static/monitor/assets/css/${themeName}.css?t=${timestamp}`;
            
            /*
             * CSS加载完成后的处理
             * 更新主题属性并清理旧链接
             */
            newLink.onload = function() {
                document.documentElement.setAttribute('data-theme', themeName);
                
                /*
                 * 延迟移除旧链接
                 * 确保新链接完全生效后再清理
                 */
                setTimeout(() => {
                    if (link.parentNode && link !== newLink) {
                        link.remove();
                    }
                }, 100);
            };
            
            /*
             * CSS加载失败时的处理
             * 降级到默认主题
             */
            newLink.onerror = function() {
                newLink.href = `/static/monitor/assets/css/default.css?t=${Date.now()}`;
                document.documentElement.setAttribute('data-theme', 'default');
            };
            
            /*
             * 将新链接插入到文档头部
             * 新旧链接会同时存在直到新链接加载完成
             */
            document.head.appendChild(newLink);
        } else {
            /*
             * 如果没有找到现有链接
             * 直接设置主题属性
             */
            document.documentElement.setAttribute('data-theme', themeName);
        }
    };
    
    /*
     * 重置主题函数
     * 清除用户临时选择，重新使用系统配置的主题
     */
    window.resetTheme = function() {
        localStorage.removeItem('user_theme_temp');
        loadAndApplyTheme();
    };
    
    /*
     * 获取当前主题函数
     * 返回当前页面应用的主题名称
     */
    window.getCurrentTheme = function() {
        return document.documentElement.getAttribute('data-theme') || 'default';
    };
})();