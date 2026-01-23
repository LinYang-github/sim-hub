<template>
  <el-container class="app-wrapper">
    <!-- Sidebar -->
    <el-aside width="210px" class="sidebar">
      <div class="logo-area">
        <el-icon class="logo-icon"><Platform /></el-icon>
        <span class="logo-text">SimHub 平台</span>
      </div>
      
      <el-scrollbar>
        <el-menu 
          router 
          :default-active="$route.path"
          class="sidebar-menu"
        >
          <el-menu-item index="/">
            <el-icon><Odometer /></el-icon>
            <span>系统概览</span>
          </el-menu-item>
          
          <el-menu-item 
            v-for="menu in menus" 
            :key="menu.path" 
            :index="menu.path"
          >
            <el-icon v-if="menu.icon"><component :is="menu.icon" /></el-icon>
            <el-icon v-else><Files /></el-icon>
            <span>{{ menu.label }}</span>
          </el-menu-item>
        </el-menu>
      </el-scrollbar>
    </el-aside>
    
    <el-container class="main-container">
      <!-- Header -->
      <el-header class="app-header">
        <div class="header-left">
          <span class="page-title">{{ currentPageTitle }}</span>
        </div>
        
        <div class="header-right">
          <div class="theme-toggle" @click="toggleDark()">
            <el-icon v-if="isDark"><Moon /></el-icon>
            <el-icon v-else><Sunny /></el-icon>
          </div>
          
          <el-dropdown trigger="click">
            <div class="user-info">
              <el-avatar :size="28" icon="UserFilled" />
              <span class="username">admin</span>
              <el-icon><ArrowDown /></el-icon>
            </div>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item>个人中心</el-dropdown-item>
                <el-dropdown-item>修改密码</el-dropdown-item>
                <el-dropdown-item divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </el-header>
      
      <!-- Main Content -->
      <el-main class="app-main">
        <router-view v-slot="{ Component }">
          <transition name="fade-transform" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { 
  Platform, Odometer, Files, Bell, Sunny, Moon, 
  UserFilled, ArrowDown 
} from '@element-plus/icons-vue'
import { moduleManager } from './core/moduleManager'
import { useDark, useToggle } from '@vueuse/core'

const isDark = useDark()
const toggleDark = useToggle(isDark)
const route = useRoute()
const menus = moduleManager.getMenus()

const currentPageTitle = computed(() => {
  if (route.path === '/') return '工作台概览'
  const activeMenu = menus.find(m => m.path === route.path)
  return activeMenu ? activeMenu.label : '仿真资源'
})
</script>

<style>
/* Global Styles */
:root {
  --app-bg: #f5f7fa;
  --header-height: 60px;
  --sidebar-bg: #ffffff;
}

.dark {
  --app-bg: #1a1a1a;
  --sidebar-bg: #141414;
}

body {
  margin: 0;
  background-color: var(--app-bg);
}

.fade-transform-enter-active,
.fade-transform-leave-active {
  transition: all 0.3s;
}

.fade-transform-enter-from {
  opacity: 0;
  transform: translateX(-15px);
}

.fade-transform-leave-to {
  opacity: 0;
  transform: translateX(15px);
}
</style>

<style scoped lang="scss">
.app-wrapper {
  height: 100vh;
  background-color: var(--app-bg);
}

.sidebar {
  background-color: var(--sidebar-bg);
  border-right: 1px solid var(--el-border-color-lighter);
  display: flex;
  flex-direction: column;
  transition: all 0.3s;

  .logo-area {
    height: var(--header-height);
    display: flex;
    align-items: center;
    padding: 0 20px;
    gap: 12px;
    
    .logo-icon {
      font-size: 24px;
      color: var(--el-color-primary);
    }
    
    .logo-text {
      font-weight: 700;
      font-size: 18px;
      color: var(--el-text-color-primary);
    }
  }

  .sidebar-menu {
    border-right: none;
    background: transparent;
    
    :deep(.el-menu-item) {
      height: 50px;
      margin: 4px 12px;
      border-radius: 8px;
      
      &.is-active {
        background-color: var(--el-color-primary-light-9);
      }
      
      &:hover {
        background-color: var(--el-fill-color-light);
      }
    }
  }
}

.app-header {
  height: var(--header-height);
  background-color: var(--sidebar-bg);
  border-bottom: 1px solid var(--el-border-color-lighter);
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;

  .header-left {
    .page-title {
      font-size: 16px;
      font-weight: 600;
      color: var(--el-text-color-primary);
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 20px;

    .header-icon {
      font-size: 20px;
      color: var(--el-text-color-secondary);
      cursor: pointer;
      transition: color 0.2s;
      
      &:hover {
        color: var(--el-color-primary);
      }
    }

    .theme-toggle {
      cursor: pointer;
      font-size: 20px;
      display: flex;
      align-items: center;
      color: var(--el-text-color-secondary);
    }

    .user-info {
      display: flex;
      align-items: center;
      gap: 8px;
      cursor: pointer;
      
      .username {
        font-size: 14px;
        color: var(--el-text-color-primary);
        font-weight: 500;
      }
    }
  }
}

.app-main {
  padding: 20px 24px;
  overflow-x: hidden;
}
</style>
