/**
 * 后台主题切换器 - 简单可用版本
 */
(function() {
    function applyTheme() {
        const themeLink = document.getElementById('theme-style');
        
        if (!themeLink) {
            const link = document.createElement('link');
            link.id = 'theme-style';
            link.rel = 'stylesheet';
            document.head.appendChild(link);
            return applyTheme();
        }
        
        fetch('/api/config/public/settings')
            .then(res => res.json())
            .then(data => {
                const adminTheme = data.admin_theme || 'default';
                const themeMode = data.theme_mode || 'fixed';
                
                let finalTheme = adminTheme;
                
                if (themeMode === 'system') {
                    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
                    finalTheme = prefersDark ? 'dark' : 'default';
                }
                
                if (finalTheme !== 'default' && finalTheme !== 'dark') {
                    finalTheme = 'default';
                }
                
                themeLink.href = `/static/admin/assets/css/${finalTheme}.css`;
                document.documentElement.setAttribute('data-theme', finalTheme);
            })
            .catch(() => {
                themeLink.href = `/static/admin/assets/css/default.css`;
                document.documentElement.setAttribute('data-theme', 'default');
            });
    };
    
    // 立即执行，不隐藏页面
    applyTheme();
    
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    mediaQuery.addEventListener('change', applyTheme);
})();