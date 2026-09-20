<script setup lang="ts">
import { computed } from 'vue'
import { terminationDisabledReason } from './rules'
import type { ProcessInfo } from './service'

const props = defineProps<{ process: ProcessInfo }>()
const disabledReason = computed(() => terminationDisabledReason(props.process))
</script>

<template>
  <dl class="min-w-0 space-y-3 text-xs">
    <div><dt class="mb-1 text-muted-foreground">父进程</dt><dd class="break-all">{{ process.parentName || '未知' }} <span class="font-mono text-muted-foreground">（PID {{ process.ppid ?? '未知' }}）</span></dd></div>
    <div><dt class="mb-1 text-muted-foreground">工作目录</dt><dd class="break-all font-mono select-text">{{ process.cwd || '未知' }}</dd></div>
    <div><dt class="mb-1 text-muted-foreground">完整启动命令</dt><dd><pre class="whitespace-pre-wrap break-all font-mono select-text">{{ process.command || '未知' }}</pre></dd></div>
    <div v-if="disabledReason"><dt class="mb-1 text-muted-foreground">结束进程不可用</dt><dd class="break-all">{{ disabledReason }}</dd></div>
  </dl>
</template>
