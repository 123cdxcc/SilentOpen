<script setup lang="ts">
import { LoaderCircle } from '@lucide/vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { AlertDialog, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { formatStartedAt } from '@/lib/format'
import { useProcesses } from './useProcesses'

const { terminationTarget, dialogOpen, terminating, terminationError, setDialogOpen, confirmTermination } = useProcesses()
</script>

<template>
  <AlertDialog :open="dialogOpen" @update:open="setDialogOpen">
    <AlertDialogContent class="max-h-[calc(100dvh-2rem)] overflow-y-auto overscroll-contain">
      <AlertDialogHeader><AlertDialogTitle>结束进程？</AlertDialogTitle><AlertDialogDescription>结束后服务会中断；Windows 会直接终止进程。</AlertDialogDescription></AlertDialogHeader>
      <dl v-if="terminationTarget" class="grid min-w-0 grid-cols-[auto_minmax(0,1fr)] gap-x-4 gap-y-2 text-sm"><dt class="text-muted-foreground">进程</dt><dd class="break-all">{{ terminationTarget.name }}</dd><dt class="text-muted-foreground">PID</dt><dd class="font-mono">{{ terminationTarget.pid }}</dd><dt class="text-muted-foreground">启动时间</dt><dd>{{ formatStartedAt(terminationTarget.startedAt) }}</dd></dl>
      <Alert v-if="terminationError" variant="destructive"><AlertTitle>结束请求失败</AlertTitle><AlertDescription class="break-all">{{ terminationError }}</AlertDescription></Alert>
      <AlertDialogFooter><AlertDialogCancel :disabled="terminating">取消</AlertDialogCancel><Button variant="destructive" :disabled="terminating" @click="confirmTermination"><LoaderCircle v-if="terminating" class="size-4 animate-spin" aria-hidden="true" />{{ terminating ? '正在结束…' : '确认结束' }}</Button></AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>
