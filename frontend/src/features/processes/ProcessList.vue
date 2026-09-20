<script setup lang="ts">
import { ref } from 'vue'
import { ChevronDown, ChevronRight, LoaderCircle, SquareTerminal } from '@lucide/vue'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { formatListener, formatStartedAt, formatUptime } from '@/lib/format'
import ProcessDetails from './ProcessDetails.vue'
import { terminationDisabledReason } from './rules'
import { useProcesses } from './useProcesses'

const {
  snapshot, loading, error, search, directory, allProcesses, missingDirectory, filteredProcesses,
  now, iconFor, discardIcon, terminating, requestTermination,
} = useProcesses()

const expandedPid = ref<number | null>(null)

/** 整行/整卡都是展开热区：点击行内任意位置切换，再点同一行收起。 */
function toggleDetails(pid: number) {
  expandedPid.value = expandedPid.value === pid ? null : pid
}
</script>

<template>
  <Table v-if="filteredProcesses.length" class="hidden md:table">
    <caption class="sr-only">正在监听 TCP 端口的进程，按 PID 升序排列</caption>
    <TableHeader><TableRow><TableHead>PID ↑</TableHead><TableHead>进程</TableHead><TableHead>项目</TableHead><TableHead>监听端口</TableHead><TableHead>启动时间</TableHead><TableHead>运行时长</TableHead><TableHead class="text-right">操作</TableHead></TableRow></TableHeader>
    <TableBody>
      <template v-for="process in filteredProcesses" :key="process.pid">
        <TableRow :data-state="expandedPid === process.pid ? 'selected' : undefined" class="cursor-pointer" :aria-expanded="expandedPid === process.pid" @click="toggleDetails(process.pid)">
          <TableCell class="py-2">
            <Button variant="ghost" size="sm" class="-ml-2 gap-1 font-mono hover:bg-transparent aria-expanded:bg-transparent dark:hover:bg-transparent" :aria-expanded="expandedPid === process.pid" :aria-controls="`desktop-details-${process.pid}`" :aria-label="`${expandedPid === process.pid ? '收起' : '展开'}进程 ${process.pid} 的详情`" @click.stop="toggleDetails(process.pid)"><ChevronDown v-if="expandedPid === process.pid" class="size-3.5" aria-hidden="true" /><ChevronRight v-else class="size-3.5" aria-hidden="true" />{{ process.pid }}</Button>
          </TableCell>
          <TableCell><div class="flex items-center gap-2"><img v-if="iconFor(process)" :src="iconFor(process)" alt="" class="size-5 shrink-0 object-contain" aria-hidden="true" @error="discardIcon(process)" /><SquareTerminal v-else class="size-5 shrink-0 text-muted-foreground" aria-hidden="true" /><span class="block max-w-36 truncate font-medium" :title="process.name || '未知'">{{ process.name || '未知' }}</span></div></TableCell>
          <TableCell><span class="block max-w-40 truncate" :title="process.cwd || '未知'">{{ process.project || '未知' }}</span></TableCell>
          <TableCell><div class="flex max-w-64 flex-wrap gap-1"><Badge v-for="listener in process.listeners" :key="formatListener(listener)" variant="secondary" class="h-auto min-h-5 max-w-full whitespace-normal break-all font-mono text-xs">{{ formatListener(listener) }}</Badge></div></TableCell>
          <TableCell class="text-xs text-muted-foreground">{{ formatStartedAt(process.startedAt) }}</TableCell>
          <TableCell class="text-xs text-muted-foreground">{{ formatUptime(process.startedAt, now) }}</TableCell>
          <TableCell class="text-right"><span :title="terminationDisabledReason(process)"><Button variant="outline" size="sm" class="text-destructive hover:bg-destructive/10 hover:text-destructive" :disabled="terminating || !!terminationDisabledReason(process)" :aria-label="`结束进程 ${process.name || '未知'}（PID ${process.pid}）${terminationDisabledReason(process) ? `：${terminationDisabledReason(process)}` : ''}`" @click.stop="requestTermination(process)">结束</Button></span></TableCell>
        </TableRow>
        <TableRow v-if="expandedPid === process.pid" :id="`desktop-details-${process.pid}`" class="bg-muted/40 hover:bg-muted/40"><TableCell colspan="7" class="whitespace-normal p-4"><ProcessDetails :process="process" /></TableCell></TableRow>
      </template>
    </TableBody>
  </Table>

  <div v-if="filteredProcesses.length" class="space-y-3 p-3 md:hidden">
    <Card v-for="process in filteredProcesses" :key="process.pid" class="min-w-0 cursor-pointer gap-3 p-4 shadow-none" @click="toggleDetails(process.pid)">
      <CardHeader class="flex min-w-0 flex-row items-start justify-between gap-2 p-0">
        <div class="min-w-0"><div class="flex min-w-0 items-center gap-2"><img v-if="iconFor(process)" :src="iconFor(process)" alt="" class="size-5 shrink-0 object-contain" aria-hidden="true" @error="discardIcon(process)" /><SquareTerminal v-else class="size-5 shrink-0 text-muted-foreground" aria-hidden="true" /><CardTitle class="truncate text-base" :title="process.name || '未知'">{{ process.name || '未知' }}</CardTitle></div><p class="mt-1 truncate text-xs text-muted-foreground" :title="process.cwd || '未知'">{{ process.project || '未知项目' }}</p></div>
        <Badge variant="outline" class="shrink-0 font-mono">PID {{ process.pid }}</Badge>
      </CardHeader>
      <CardContent class="min-w-0 space-y-3 p-0">
        <div class="flex min-w-0 flex-wrap gap-1" aria-label="监听 IP 和端口"><Badge v-for="listener in process.listeners" :key="formatListener(listener)" variant="secondary" class="h-auto min-h-5 max-w-full whitespace-normal break-all font-mono">{{ formatListener(listener) }}</Badge></div>
        <dl class="grid grid-cols-[auto_minmax(0,1fr)] gap-x-3 gap-y-1 text-xs"><dt class="text-muted-foreground">启动时间</dt><dd>{{ formatStartedAt(process.startedAt) }}</dd><dt class="text-muted-foreground">运行时长</dt><dd>{{ formatUptime(process.startedAt, now) }}</dd></dl>
        <ProcessDetails v-if="expandedPid === process.pid" :id="`mobile-details-${process.pid}`" :process="process" class="border-t pt-3" @click.stop />
        <div class="flex items-center justify-between gap-2 border-t pt-3">
          <Button variant="ghost" size="sm" class="-ml-2 text-muted-foreground" :aria-expanded="expandedPid === process.pid" :aria-controls="`mobile-details-${process.pid}`" :aria-label="`${expandedPid === process.pid ? '收起' : '展开'}进程 ${process.pid} 的详情`" @click.stop="toggleDetails(process.pid)">{{ expandedPid === process.pid ? '收起详情' : '查看详情' }}<ChevronDown class="size-3.5 transition-transform" :class="{ 'rotate-180': expandedPid === process.pid }" aria-hidden="true" /></Button>
          <span :title="terminationDisabledReason(process)"><Button variant="outline" size="sm" class="text-destructive hover:bg-destructive/10 hover:text-destructive" :disabled="terminating || !!terminationDisabledReason(process)" :aria-label="`结束进程 ${process.name || '未知'}（PID ${process.pid}）`" @click.stop="requestTermination(process)">结束进程</Button></span>
        </div>
      </CardContent>
    </Card>
  </div>

  <div v-if="!filteredProcesses.length" class="flex min-h-52 flex-col items-center justify-center gap-2 px-6 py-12 text-center" role="status">
    <LoaderCircle v-if="!snapshot && loading" class="mb-2 size-6 animate-spin text-muted-foreground" aria-hidden="true" />
    <template v-if="!snapshot"><p class="text-sm font-medium">{{ error ? '暂时无法获取进程列表' : '正在读取本机进程…' }}</p><p class="text-xs text-muted-foreground">{{ error ? '可手动刷新重试，或切回应用重新读取。' : '正在查找监听 TCP 端口的服务。' }}</p></template>
    <template v-else-if="allProcesses.length === 0 && directory === '__all__' && !search.trim()"><p class="text-sm font-medium">暂无监听进程</p><p class="text-xs text-muted-foreground">当前权限范围内没有发现监听 TCP 端口的进程。</p></template>
    <template v-else><p class="text-sm font-medium">没有匹配的进程</p><p class="text-xs text-muted-foreground">{{ missingDirectory ? '所选工作目录当前没有监听进程。' : '调整搜索内容或工作目录筛选。' }}</p></template>
  </div>
</template>
