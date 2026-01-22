<template>
  <div class="common-layout">
    <el-container>
      <el-header>
        <div class="logo">SimHub 仿真资源</div>
        <el-menu mode="horizontal" router>
          <el-menu-item index="/">工作台</el-menu-item>
          <el-menu-item index="/resources">资源库</el-menu-item>
        </el-menu>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </div>
</template>

<script setup lang="ts">
</script>

<style scoped>
.logo {
  float: left;
  line-height: 60px;
  font-weight: bold;
  font-size: 20px;
  margin-right: 40px;
  color: #409EFF;
}
.el-header {
  border-bottom: 1px solid #dcdfe6;
}
</style>
