<script setup lang="ts">
import { onBeforeUnmount, onMounted } from 'vue'
import { Download, RefreshCw, Search } from '@lucide/vue'
import appIcon from '@/assets/appicon.png'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { formatClockTime } from '@/lib/format'
import UpdateNotice from '@/features/update/UpdateNotice.vue'
import { useUpdate } from '@/features/update/useUpdate'
import ProcessList from './ProcessList.vue'
import TerminateDialog from './TerminateDialog.vue'
import { useProcesses } from './useProcesses'

const {
  snapshot, loading, error, search, directory, allProcesses, directories, directoryLabel,
  missingDirectory, hasActiveFilter, filteredProcesses, terminationMessage, refresh, start, stop,
} = useProcesses()

// 更新提示与进程列表各有一份共享状态，入口组件只负责把入口按钮摆出来。
const { checking: checkingUpdate, version, check: checkUpdate } = useUpdate()

onMounted(start)
onBeforeUnmount(stop)
</script>

<template>
  <main class="flex h-full min-h-0 min-w-0 flex-col gap-4 overflow-hidden bg-background p-3 text-foreground sm:p-5">
    <header class="flex shrink-0 items-center justify-between gap-3">
      <div class="flex min-w-0 items-center gap-3">
        <img :src="appIcon" alt="" class="size-9 shrink-0 rounded-lg object-contain" aria-hidden="true" />
        <div class="min-w-0">
          <h1 class="text-lg font-semibold tracking-tight">SilentOpen</h1>
          <p class="flex items-center gap-2 text-xs text-muted-foreground">监听中的进程 <Badge variant="secondary" class="px-1.5 py-0 text-xs">{{ allProcesses.length }}</Badge></p>
        </div>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <Button variant="ghost" size="sm" :disabled="checkingUpdate" :title="version ? `当前版本 v${version}，点击检查更新` : '检查更新'" @click="checkUpdate(true)">
          <Download class="size-4" :class="{ 'animate-pulse': checkingUpdate }" aria-hidden="true" />
          <span class="hidden sm:inline">{{ checkingUpdate ? '检查中…' : '检查更新' }}</span>
        </Button>
        <span class="flex shrink-0 items-center gap-1.5 text-xs text-muted-foreground"><span class="size-1.5 rounded-full" :class="error ? 'bg-destructive' : 'bg-emerald-500'" aria-hidden="true"></span>切回应用时刷新</span>
      </div>
    </header>

    <UpdateNotice />

    <section class="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded-xl border bg-card" aria-label="本机监听进程">
      <div class="flex shrink-0 flex-col gap-2 border-b p-3 sm:flex-row sm:items-center">
        <div class="relative min-w-0 flex-1">
          <Search class="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-muted-foreground" aria-hidden="true" />
          <Input v-model="search" type="search" class="pl-9" aria-label="搜索进程、项目、IP、端口或命令" placeholder="搜索 PID、项目、IP、端口或命令…" autocomplete="off" spellcheck="false" />
        </div>
        <div class="flex min-w-0 gap-2 sm:w-[360px]">
          <Select v-model="directory">
            <SelectTrigger class="min-w-0 flex-1" aria-label="筛选工作目录" :title="directoryLabel"><SelectValue class="truncate">{{ directoryLabel }}</SelectValue></SelectTrigger>
            <SelectContent position="popper" align="start" class="max-w-[calc(100vw-2rem)]">
              <SelectItem value="__all__">全部工作目录</SelectItem>
              <SelectItem value="__unknown__">未知工作目录</SelectItem>
              <SelectItem v-if="missingDirectory" :value="directory" :title="directory"><span class="break-all">{{ directory }}（已无进程）</span></SelectItem>
              <SelectItem v-for="cwd in directories" :key="cwd" :value="cwd" :title="cwd"><span class="break-all">{{ cwd }}</span></SelectItem>
            </SelectContent>
          </Select>
          <Button variant="outline" size="icon" class="shrink-0" :disabled="loading" :aria-label="loading ? '正在刷新进程列表' : '刷新进程列表'" :title="loading ? '正在刷新进程列表' : '刷新进程列表'" @click="refresh"><RefreshCw class="size-4" :class="{ 'animate-spin': loading }" aria-hidden="true" /></Button>
        </div>
      </div>

      <div v-if="error || snapshot?.warnings.length || terminationMessage" class="max-h-36 shrink-0 space-y-2 overflow-auto overscroll-contain border-b p-3">
        <Alert v-if="error" variant="destructive"><AlertTitle>刷新失败{{ snapshot ? '，当前显示上次成功的数据' : '' }}</AlertTitle><AlertDescription class="break-all">{{ error }}</AlertDescription></Alert>
        <Alert v-if="snapshot?.warnings.length" role="status"><AlertTitle>部分信息不可读取</AlertTitle><AlertDescription><p v-for="warning in snapshot.warnings" :key="warning" class="break-all">{{ warning }}</p></AlertDescription></Alert>
        <Alert v-if="terminationMessage" role="status"><AlertDescription class="break-all">{{ terminationMessage }}</AlertDescription></Alert>
      </div>

      <div class="min-h-0 min-w-0 flex-1 overflow-auto overscroll-contain" :aria-busy="loading" aria-label="进程列表">
        <ProcessList />
      </div>

      <footer class="flex shrink-0 flex-wrap justify-between gap-x-3 gap-y-1 border-t px-3 py-2 text-xs text-muted-foreground"><span>{{ hasActiveFilter ? `匹配 ${filteredProcesses.length} / ${allProcesses.length} 个进程` : `共 ${allProcesses.length} 个进程` }}</span><span>{{ snapshot ? `更新于 ${formatClockTime(snapshot.collectedAt)}` : '尚未更新' }}</span></footer>
    </section>

    <TerminateDialog />
  </main>
</template>
