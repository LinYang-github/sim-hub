export default function() {
  const { ElMessageBox, ElMessage, ElNotification } = (window as any).ElementPlus

  return {
    // 这是一个无 UI 的功能组件，只负责逻辑
    render() { return null },
    setup(_: any, { expose }: any) {
      
      const execute = (resource: any) => {
          console.log('Executing action for', resource)
          
          ElMessageBox.confirm(
            `确认审批通过资源 "${resource.name}" (v${resource.latest_version?.version_num || 1}) 吗？\n这是来自远程扩展的操作演示。`,
            '扩展操作：资源审批',
            {
              confirmButtonText: '通过',
              cancelButtonText: '驳回',
              type: 'info',
              draggable: true,
            }
          )
            .then(() => {
              ElNotification({
                title: '审批通过',
                message: `资源ID: ${resource.id} 已标记为通过状态`,
                type: 'success',
                position: 'bottom-right',
              })
            })
            .catch(() => {
              ElMessage.info('已取消操作')
            })
      }

      // 暴露方法给父组件调用
      expose({ execute })

      return {}
    }
  }
}
