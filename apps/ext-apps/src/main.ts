
/**
 * Sim-Hub 外部扩展注册入口 (Development Mode)
 * 该脚本由主应用动态加载，用于自述其提供的视图、预览器及动作。
 */

const registerPlugins = () => {
    const target = (window as any).SimHub || (window.parent as any).SimHub

    if (!target) {
        console.warn('[PluginHost] SimHub API not found. Registration failed.')
        return
    }

    // 1. 注册自定义视图
    console.log('[PluginHost] Registering view: demo-view')
    target.registerView({
        key: 'demo-view',
        label: '测试视图',
        icon: 'Picture',
        path: 'External:http://localhost:30031/demo-view/'
    })

    // 2. 注册自定义预览器 (Viewer)
    console.log('[PluginHost] Registering viewer: demo-preview')
    target.registerViewer({
        key: 'demo-preview',
        label: '深度质检',
        path: 'External:http://localhost:30031/demo-preview/'
    })

    console.log('[PluginHost] All plugins registered successfully.')
}

// 立即执行
registerPlugins()

export default registerPlugins
