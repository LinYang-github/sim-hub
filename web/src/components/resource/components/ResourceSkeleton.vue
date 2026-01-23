<template>
  <div class="skeleton-wrapper">
    <div v-if="viewMode === 'list'" class="list-skeleton">
      <el-skeleton :rows="10" animated />
    </div>
    
    <div v-else class="card-skeleton">
      <el-row :gutter="20">
        <el-col :xs="24" :sm="12" :md="8" :lg="6" :xl="4" v-for="i in 8" :key="i">
          <el-card shadow="never" class="skeleton-card">
            <el-skeleton animated>
              <template #template>
                <el-skeleton-item variant="image" class="skeleton-image" />
                <div class="skeleton-body">
                  <el-skeleton-item variant="h3" style="width: 70%" />
                  <div class="skeleton-meta">
                    <el-skeleton-item variant="text" style="width: 30%" />
                    <el-skeleton-item variant="text" style="width: 30%" />
                  </div>
                </div>
              </template>
            </el-skeleton>
          </el-card>
        </el-col>
      </el-row>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  viewMode: string
}>()
</script>

<style scoped lang="scss">
.skeleton-wrapper {
  padding: 12px 0;
}

.skeleton-card {
  margin-bottom: 20px;
  border-radius: 12px;
  border-color: var(--el-border-color-extra-light);
}

.skeleton-image {
  height: 140px;
  width: 100%;
}

.skeleton-body {
  padding: 16px;
}

.skeleton-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 12px;
}
</style>
