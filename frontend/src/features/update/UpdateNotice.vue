<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { X } from '@lucide/vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { useUpdate } from './useUpdate'

const {
  visible, latestLabel, notes, assetName, checking, error, message,
  skip, dismiss, clearError, openPage, start, stop,
} = useUpdate()

onMounted(start)
onBeforeUnmount(stop)
</script>

<template>
  <!--
    三块各自独立：一次失败的检查不该把已经查到的版本藏起来；而"有新版本"与"已是最新"
    本来就是互斥的两种结论，所以结论块只在既没有横幅也没有错误时出现。
  -->
  <Alert v-if="visible" class="shrink-0">
    <AlertTitle>有新版本 {{ latestLabel }}</AlertTitle>
    <AlertDescription>
      <p v-if="notes" class="line-clamp-4 break-words whitespace-pre-line">{{ notes }}</p>
      <p v-if="assetName" class="break-all text-muted-foreground">匹配本机下载：{{ assetName }}</p>
      <span class="flex flex-wrap items-center gap-2 pt-1.5">
        <Button size="sm" :disabled="checking" @click="openPage">打开下载页</Button>
        <Button size="sm" variant="ghost" :disabled="checking" @click="skip">忽略此版本</Button>
        <Button size="icon" variant="ghost" class="size-7" aria-label="稍后提醒" title="稍后提醒" @click="dismiss"><X class="size-4" aria-hidden="true" /></Button>
      </span>
    </AlertDescription>
  </Alert>

  <Alert v-if="error" variant="destructive" class="shrink-0">
    <AlertTitle>检查更新失败</AlertTitle>
    <AlertDescription class="flex items-start gap-2">
      <span class="min-w-0 flex-1 break-all">{{ error }}</span>
      <Button size="icon" variant="ghost" class="size-7 shrink-0" aria-label="关闭提示" title="关闭提示" @click="clearError"><X class="size-4" aria-hidden="true" /></Button>
    </AlertDescription>
  </Alert>

  <Alert v-if="!visible && !error && message" role="status" class="shrink-0">
    <AlertDescription class="break-all">{{ message }}</AlertDescription>
  </Alert>
</template>
