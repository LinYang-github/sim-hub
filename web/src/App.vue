<template>
  <el-container class="layout-container" style="height: 100vh">
    <el-aside width="220px" style="background-color: #fff; border-right: 1px solid #dcdfe6">
      <div class="logo">SimHub 仿真资源</div>
      <el-menu 
        router 
        :default-active="$route.path"
        style="border-right: none;"
      >
        <el-menu-item index="/">
            <el-icon><Monitor /></el-icon>
            <span>工作台</span>
        </el-menu-item>
        
        <!-- Dynamic Module Menus -->
        <el-menu-item 
          v-for="menu in menus" 
          :key="menu.path" 
          :index="menu.path"
        >
          <el-icon v-if="menu.icon"><component :is="menu.icon" /></el-icon>
          <el-icon v-else><Menu /></el-icon>
          <span>{{ menu.label }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    
    <el-container>
      <el-header style="text-align: right; font-size: 12px; background-color: #f5f7fa; border-bottom: 1px solid #e4e7ed; line-height: 60px;">
        <span style="margin-right: 20px; font-weight: bold; color: #606266;">Administrator</span>
      </el-header>
      
      <el-main style="background-color: #f0f2f5; padding: 20px;">
        <div style="background: #fff; padding: 20px; min-height: calc(100% - 40px); border-radius: 4px;">
            <router-view />
        </div>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { Monitor, Menu } from '@element-plus/icons-vue'
import { moduleManager } from './core/moduleManager'
const menus = moduleManager.getMenus()
</script>

<style scoped>
.logo {
  height: 60px;
  line-height: 60px;
  padding-left: 20px;
  font-weight: bold;
  font-size: 20px;
  color: #409EFF;
  border-bottom: 1px solid #dcdfe6;
}
</style>
