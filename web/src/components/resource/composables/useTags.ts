import { ref, computed, Ref } from 'vue'
import request from '../../../core/utils/request'
import { ElMessage } from 'element-plus'
import type { Resource } from '../../../core/types/resource'

export function useTags(
  resources: Ref<Resource[]>, 
  onSuccess: () => void,
  currentResource?: Ref<Resource | null>
) {
  const tagDialogVisible = ref(false)
  const tagLoading = ref(false)
  const editingTags = ref<string[]>([])
  const currentResourceId = ref('')

  const existingTags = computed(() => {
    const tags = new Set<string>()
    resources.value.forEach(s => {
      if (s.tags) s.tags.forEach(t => tags.add(t))
    })
    return Array.from(tags)
  })

  const openTagEditor = (row: Resource) => {
    currentResourceId.value = row.id
    editingTags.value = [...(row.tags || [])]
    tagDialogVisible.value = true
  }

  const saveTags = async () => {
    tagLoading.value = true
    try {
      await request.patch(`/api/v1/resources/${currentResourceId.value}/tags`, {
        tags: editingTags.value
      })
      ElMessage.success('标签更新成功')
      
      // 同步更新详情抽屉中的引用
      if (currentResource?.value && currentResource.value.id === currentResourceId.value) {
        currentResource.value.tags = [...editingTags.value]
      }
      
      tagDialogVisible.value = false
      onSuccess()
    } catch (e: any) {
    } finally {
      tagLoading.value = false
    }
  }

  return {
    tagDialogVisible,
    tagLoading,
    editingTags,
    currentResourceId,
    existingTags,
    openTagEditor,
    saveTags
  }
}
