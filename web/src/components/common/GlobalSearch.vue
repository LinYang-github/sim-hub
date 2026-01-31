<template>
  <el-dialog
    v-model="visible"
    :show-close="false"
    width="680px"
    class="spotlight-search"
    destroy-on-close
    @opened="handleOpened"
  >
    <div class="search-wrapper">
      <div class="search-input-area">
        <el-icon class="search-icon"><Search /></el-icon>
        <input 
          ref="inputRef"
          v-model="query" 
          placeholder="搜索资源、模型、想定..." 
          @input="handleInput"
          @keydown.down.prevent="moveNext"
          @keydown.up.prevent="movePrev"
          @keydown.enter="handleSelect"
          @keydown.esc="visible = false"
        />
        <div class="search-hint">ESC 退出</div>
      </div>

      <div class="search-results" v-if="query || (recentItems.length > 0)">
        <div v-if="loading" class="search-loading">
          <el-icon class="is-loading"><Loading /></el-icon> 正在搜索...
        </div>

        <template v-else-if="results.length > 0">
           <div 
             v-for="(item, index) in results" 
             :key="item.id"
             class="result-item"
             :class="{ active: index === activeIndex }"
             @mouseenter="activeIndex = index"
             @click="handleSelect"
           >
             <div class="item-icon">
                <el-icon><component :is="item.icon || 'Files'" /></el-icon>
             </div>
             <div class="item-body">
                <div class="item-name">{{ item.name }}</div>
                <div class="item-meta">
                   <span class="type-tag">{{ item.typeName }}</span>
                   <span class="divider">·</span>
                   <span class="date">{{ formatDate(item.createdAt) }}</span>
                   
                   <template v-if="item.latest_version?.file_size">
                       <span class="divider">·</span>
                       <span class="size">{{ formatSize(item.latest_version.file_size) }}</span>
                   </template>

                   <span v-if="!isNameMatch(item, query) && !item.highlights?.content" class="match-badge">
                       <el-icon><Document /></el-icon> 内容匹配
                   </span>
                </div>

                <!-- 高亮片段展示 -->
                <div v-if="item.highlights?.content" class="search-snippet" v-html="'... ' + item.highlights.content[0] + ' ...'"></div>
             </div>
             <div class="item-action">
                <el-icon><Right /></el-icon>
             </div>
           </div>
        </template>

        <div v-else-if="query && !loading" class="empty-results">
            未找到与 "{{ query }}" 相关的资源
        </div>

        <div v-else-if="!query && recentItems.length > 0" class="recent-section">
            <div class="section-title">最近访问</div>
            <div 
             v-for="(item, index) in recentItems" 
             :key="item.id"
             class="result-item"
             :class="{ active: index === activeIndex }"
             @mouseenter="activeIndex = index"
             @click="handleSelect"
           >
             <div class="item-icon">
                <el-icon><component :is="item.icon || 'Files'" /></el-icon>
             </div>
             <div class="item-body">
                <div class="item-name">{{ item.name }}</div>
                <div class="item-meta">{{ item.typeName }}</div>
             </div>
           </div>
        </div>
      </div>

      <div class="search-footer">
        <div class="footer-item">
          <span class="key-cap">↵</span> 确认
        </div>
        <div class="footer-item">
          <span class="key-cap">↑</span> <span class="key-cap">↓</span> 切换
        </div>
      </div>
    </div>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, markRaw } from 'vue'
import { Search, Loading, Right, Files, Document } from '@element-plus/icons-vue'
import request from '../../core/utils/request'
import { moduleManager } from '../../core/moduleManager'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'

const visible = ref(false)
const query = ref('')
const loading = ref(false)
const inputRef = ref<HTMLInputElement | null>(null)
const activeIndex = ref(0)
const results = ref<any[]>([])
const recentItems = ref<any[]>([])
const router = useRouter()

const open = () => {
    visible.value = true
    query.value = ''
    results.value = []
    activeIndex.value = 0
}

defineExpose({ open })

const handleOpened = () => {
    inputRef.value?.focus()
}

let searchTimer: any = null
const handleInput = () => {
    activeIndex.value = 0
    if (!query.value.trim()) {
        results.value = []
        return
    }

    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(performSearch, 300)
}

const performSearch = async () => {
    loading.value = true
    try {
        // Search all types
        const res = await request.get<any>('/api/v1/resources', {
            params: { query: query.value, size: 8 }
        })
        
        results.value = (res.items || []).map((item: any) => {
            const typeConfig = moduleManager.getActiveModules().value.find(t => t.key === item.type_key)
            const icon = typeConfig?.icon || 'Files'
            return {
                ...item,
                typeName: typeConfig?.typeName || item.type_key,
                icon: typeof icon === 'string' ? icon : markRaw(icon)
            }
        })
    } catch (e) {
        console.error(e)
    } finally {
        loading.value = false
    }
}

const moveNext = () => {
    const list = results.value.length > 0 ? results.value : recentItems.value
    if (activeIndex.value < list.length - 1) {
        activeIndex.value++
    }
}

const movePrev = () => {
    if (activeIndex.value > 0) {
        activeIndex.value--
    }
}

const handleSelect = () => {
    const list = results.value.length > 0 ? results.value : recentItems.value
    const selected = list[activeIndex.value]
    if (selected) {
        visible.value = false
        // Navigation: if it's a specific resource, we might want to go to its list and open details
        // For now, let's just go to the list
        router.push(`/res/${selected.type_key}`)
        
        // Add to recent (simplified)
        addToRecent(selected)
    }
}

const addToRecent = (item: any) => {
    recentItems.value = [item, ...recentItems.value.filter(i => i.id !== item.id)].slice(0, 5)
}

const formatDate = (date: string) => dayjs(date).format('YYYY-MM-DD')
const formatSize = (bytes: number) => {
    if (!bytes) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

const isNameMatch = (item: any, q: string) => {
    if (!q) return true
    const lowerQ = q.toLowerCase()
    return item.name.toLowerCase().includes(lowerQ)
}

// Global shortcut
const handleKeyDown = (e: KeyboardEvent) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault()
        open()
    }
}

const handleGlobalOpen = () => open()

onMounted(() => {
    window.addEventListener('keydown', handleKeyDown)
    window.addEventListener('open-global-search', handleGlobalOpen)
})

onUnmounted(() => {
    window.removeEventListener('keydown', handleKeyDown)
    window.removeEventListener('open-global-search', handleGlobalOpen)
})
</script>

<style lang="scss">
.spotlight-search {
    .el-dialog__header {
        display: none;
    }
    .el-dialog__body {
        padding: 0 !important;
        background: transparent;
    }
    background: transparent !important;
    box-shadow: none !important;
    margin-top: 15vh !important;
}

.search-wrapper {
    background: var(--el-bg-color);
    border-radius: 12px;
    box-shadow: 0 20px 50px rgba(0,0,0,0.3);
    overflow: hidden;
    border: 1px solid var(--el-border-color-lighter);
}

.search-input-area {
    display: flex;
    align-items: center;
    padding: 16px 20px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    gap: 12px;

    .search-icon {
        font-size: 20px;
        color: var(--el-text-color-secondary);
    }

    input {
        flex: 1;
        border: none;
        background: transparent;
        font-size: 18px;
        color: var(--el-text-color-primary);
        outline: none;
        &::placeholder {
            color: var(--el-text-color-placeholder);
        }
    }

    .search-hint {
        font-size: 12px;
        color: var(--el-text-color-secondary);
        background: var(--el-fill-color-light);
        padding: 2px 6px;
        border-radius: 4px;
        border: 1px solid var(--el-border-color-lighter);
    }
}

.search-results {
    max-height: 400px;
    overflow-y: auto;
    padding: 8px;

    .search-loading, .empty-results {
        padding: 40px;
        text-align: center;
        color: var(--el-text-color-secondary);
        font-size: 14px;
    }

    .recent-section {
        .section-title {
            padding: 8px 12px;
            font-size: 12px;
            font-weight: 600;
            color: var(--el-text-color-secondary);
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }
    }

    .result-item {
        display: flex;
        align-items: center;
        padding: 10px 12px;
        border-radius: 8px;
        gap: 14px;
        cursor: pointer;
        transition: all 0.2s;

        &.active {
            background: var(--el-color-primary-light-9);
            .item-action {
                opacity: 1;
                transform: translateX(0);
            }
        }

        .item-icon {
            width: 36px;
            height: 36px;
            background: var(--el-fill-color-light);
            border-radius: 8px;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 18px;
            color: var(--el-color-primary);
        }

        .item-body {
            flex: 1;
            .item-name {
                font-size: 14px;
                font-weight: 500;
                color: var(--el-text-color-primary);
            }
            .item-meta {
                font-size: 12px;
                color: var(--el-text-color-secondary);
                margin-top: 2px;
                display: flex;
                align-items: center;
                gap: 6px;

                .type-tag {
                    color: var(--el-color-primary);
                    font-weight: 500;
                }
                .divider { opacity: 0.5; }
                
                .match-badge {
                    display: inline-flex;
                    align-items: center;
                    gap: 2px;
                    background: var(--el-color-success-light-9);
                    color: var(--el-color-success);
                    font-size: 10px;
                    padding: 0 4px;
                    border-radius: 4px;
                    margin-left: 6px;
                    border: 1px solid var(--el-color-success-light-5);
                    height: 18px;
                }
            }
            
            .search-snippet {
                font-size: 12px;
                color: var(--el-text-color-secondary);
                margin-top: 4px;
                line-height: 1.4;
                display: -webkit-box;
                -webkit-line-clamp: 2;
                -webkit-box-orient: vertical;
                overflow: hidden;
                
                em {
                    color: var(--el-color-primary);
                    font-style: normal;
                    font-weight: bold;
                    background: var(--el-color-primary-light-9);
                    padding: 0 2px;
                    border-radius: 2px;
                }
            }
        }

        .item-action {
            opacity: 0;
            transform: translateX(-5px);
            transition: all 0.2s;
            color: var(--el-text-color-secondary);
        }
    }
}

.search-footer {
    padding: 12px 20px;
    background: var(--el-fill-color-lighter);
    display: flex;
    gap: 16px;
    border-top: 1px solid var(--el-border-color-lighter);

    .footer-item {
        font-size: 12px;
        color: var(--el-text-color-secondary);
        display: flex;
        align-items: center;
        gap: 6px;

        .key-cap {
            background: var(--el-bg-color);
            border: 1px solid var(--el-border-color);
            box-shadow: 0 1px 0 var(--el-border-color);
            padding: 0 4px;
            border-radius: 3px;
            font-family: monospace;
            min-width: 14px;
            text-align: center;
        }
    }
}

/* Dark mode overrides if needed */
.dark .search-wrapper {
    box-shadow: 0 20px 50px rgba(0,0,0,0.6);
}
</style>
