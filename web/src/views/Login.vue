<template>
  <div class="login-container">
    <div class="login-card">
      <div class="logo-area">
        <el-icon class="logo-icon"><Platform /></el-icon>
        <span class="logo-text">SimHub 登录</span>
      </div>
      
      <el-form :model="form" @keyup.enter="handleLogin">
        <el-form-item>
          <el-input 
            v-model="form.username" 
            placeholder="用户名" 
            prefix-icon="User"
            size="large"
          />
        </el-form-item>
        <el-form-item>
          <el-input 
            v-model="form.password" 
            type="password" 
            placeholder="密码" 
            prefix-icon="Lock" 
            show-password
            size="large"
          />
        </el-form-item>
        
        <el-button 
          type="primary" 
          class="login-btn" 
          :loading="loading" 
          @click="handleLogin" 
          size="large"
        >
          立即登录
        </el-button>
      </el-form>
      
      <div class="footer-tip">
        默认账户: admin / 123456
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Platform, User, Lock } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import axios from 'axios'

const router = useRouter()
const form = ref({ username: '', password: '' })
const loading = ref(false)

const handleLogin = async () => {
  if (!form.value.username || !form.value.password) {
    ElMessage.warning('请输入用户名和密码')
    return
  }

  loading.value = true
  try {
    const res = await axios.post('/api/v1/auth/login', form.value)
    // 存储 Token
    localStorage.setItem('simhub_token', res.data.token)
    localStorage.setItem('simhub_user', form.value.username)
    
    ElMessage.success('登录成功')
    router.push('/')
  } catch (err: any) {
    ElMessage.error(err.response?.data?.error || '登录失败，请检查账号密码')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped lang="scss">
.login-container {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--app-bg);
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
}

.login-card {
  width: 360px;
  padding: 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 8px 24px rgba(0,0,0,0.1);

  .logo-area {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 12px;
    margin-bottom: 30px;
    
    .logo-icon {
      font-size: 48px;
      color: var(--el-color-primary);
    }
    
    .logo-text {
      font-size: 24px;
      font-weight: 600;
      color: #303133;
    }
  }

  .login-btn {
    width: 100%;
    margin-top: 10px;
  }

  .footer-tip {
    margin-top: 20px;
    text-align: center;
    font-size: 12px;
    color: #909399;
  }
}
</style>
