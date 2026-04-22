<template>
  <div class="card floating">
    <div class="card-title">
      <h2>版本信息</h2>
    </div>
    <div class="card-content">
      <p v-if="loading">正在加载版本信息...</p>
      <p v-else-if="error" class="error">{{ error }}</p>
      <div v-else>
        <p><strong>构建时间:</strong> {{ version }}</p>
        <p><strong>部署状态:</strong> <span class="success-text">已更新</span></p>
        <p style="margin-top: 1em; font-size: 0.9em; color: #666;">
          注意：如果构建时间未变化，请确保在部署脚本中执行了编译步骤。
        </p>
      </div>
    </div>
    <div class="card-action">
      <button class="button button--flat" @click="$router.back()">返回</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue';
import { fetchURL } from '@/api/utils';

const version = ref('');
const loading = ref(true);
const error = ref('');

onMounted(async () => {
  try {
    const res = await fetchURL('/api/version', {});
    if (res.ok) {
      version.value = await res.text();
    } else {
      error.value = '无法获取版本信息 (HTTP ' + res.status + ')';
    }
  } catch (e) {
    error.value = '请求版本接口失败';
    console.error(e);
  } finally {
    loading.value = false;
  }
});
</script>

<style scoped>
.success-text {
  color: #4caf50;
  font-weight: bold;
}
.error {
  color: #f44336;
}
</style>
