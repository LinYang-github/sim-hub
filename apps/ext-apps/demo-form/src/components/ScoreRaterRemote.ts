export default function() {
  const { h, computed } = (window as any).Vue
  const { ElRate } = (window as any).ElementPlus

  return {
    props: {
      modelValue: { type: Number, default: 0 },
      propDef: { type: Object }
    },
    emits: ['update:modelValue'],
    setup(props: any, { emit }: any) {
        
      const rateValue = computed({
        get: () => props.modelValue || 0,
        set: (val: number) => emit('update:modelValue', val || 0)
      })

      return () => h('div', { 
        style: {
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '8px',
            backgroundColor: 'var(--el-bg-color)', /* Use theme var */
            border: '1px solid var(--el-border-color)', /* Use theme var */
            padding: '4px 12px',
            borderRadius: '4px',
            width: '100%',
            height: '32px', // Standard Input Height
            boxSizing: 'border-box'
        }
      }, [
         h('div', { style: { flex: '1', minWidth: 0, display: 'flex', alignItems: 'center' } }, [
             h(ElRate, {
                 modelValue: rateValue.value,
                 'onUpdate:modelValue': (val: number) => rateValue.value = val,
                 max: 10,
                 showScore: true,
                 textColor: 'var(--el-color-warning)',
                 voidColor: 'var(--el-border-color-darker)',
                 style: { transform: 'scale(0.85)', transformOrigin: 'left center' } // Scale down slightly to fit better
             })
         ])
      ])
    }
  }
}
