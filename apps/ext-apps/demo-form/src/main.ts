import ScoreRaterRemoteFactory from './components/ScoreRaterRemote'
import DemoActionFactory from './components/DemoAction'

// Get the component definition by invoking the factory
// The factory expects window.Vue and window.ElementPlus to be present
const ScoreRaterRemote = ScoreRaterRemoteFactory()
const DemoActionRemote = DemoActionFactory()

// 暴露组件给主应用（通过全局注册表）
;(window as any).SimHubCustomComponents = (window as any).SimHubCustomComponents || {}
;(window as any).SimHubCustomComponents['ScoreRater'] = ScoreRaterRemote
;(window as any).SimHubCustomComponents['DemoAction'] = DemoActionRemote

// 仅用于本地开发预览的挂载逻辑
if (import.meta.env.DEV) {
    const devRoot = document.getElementById('app-demo-form')
    if (devRoot) {
         // Create a fake host environment for local dev
         import('vue').then(Vue => {
             (window as any).Vue = Vue
             import('element-plus').then(ElementPlus => {
                 (window as any).ElementPlus = ElementPlus
                 import('element-plus/dist/index.css')
                 
                 // Re-create component now that globals are ready
                 const LocalComponent = ScoreRaterRemoteFactory()
                 
                 const app = Vue.createApp(LocalComponent, { modelValue: 5 })
                 app.use(ElementPlus)
                 app.mount(devRoot)
             })
         })
    }
}
