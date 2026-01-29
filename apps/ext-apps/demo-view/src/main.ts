
// 模拟外部视图的注册脚本
// 该脚本应由主应用在启动时或通过插件机制加载

const registerDemoView = () => {
    // 尝试访问主应用的全局 API
    const target = (window as any).SimHub || (window.parent as any).SimHub

    if (target && target.registerView) {
        console.log('[DemoView] Registering view: demo-view')
        target.registerView({
            key: 'demo-view',
            label: '画廊视图', // 自定义 Label
            icon: 'Picture', // 使用 ElementPlus 图标名 (需主应用支持解析字符串)
            path: 'External:http://localhost:30031/demo-view/' // 必须以 External: 开头以触发 ExternalViewer
            // 如果需要提供组件，可以传入 Vue 组件对象
            // icon: defineAsyncComponent(...) 
        })
    } else {
        console.warn('[DemoView] SimHub API not found. Registration skipped.')
    }
}

// 立即执行注册
registerDemoView()

export default registerDemoView
