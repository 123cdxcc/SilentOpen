import { computed, ref } from 'vue'
import { hasKnownStartTime, iconKey, matchesQuery, terminationDisabledReason } from './rules'
import { processService, type ProcessInfo, type ProcessServiceApi, type ProcessSnapshot } from './service'

const ALL_DIRECTORIES = '__all__'
const UNKNOWN_DIRECTORY = '__unknown__'

/**
 * 进程查看功能的全部状态与动作。包内组件通过 useProcesses() 共享同一实例，
 * 因此组件之间不需要传递 props。createProcessStore() 供测试注入假 service。
 */
export function createProcessStore(api: ProcessServiceApi = processService) {
  const snapshot = ref<ProcessSnapshot | null>(null)
  const loading = ref(false)
  const error = ref('')
  const now = ref(Date.now())
  const processIcons = ref<Record<string, string>>({})
  const search = ref('')
  const directory = ref(ALL_DIRECTORIES)
  const terminationTarget = ref<{ pid: number; name: string; startedAt: number } | null>(null)
  const dialogOpen = ref(false)
  const terminating = ref(false)
  const terminationError = ref('')
  const terminationMessage = ref('')

  let active = true
  let started = false
  let refreshAgain = false
  let iconGeneration = 0

  const allProcesses = computed(() => [...(snapshot.value?.processes ?? [])].sort((a, b) => a.pid - b.pid))
  const directories = computed(() => [...new Set(allProcesses.value.map(process => process.cwd).filter(Boolean))].sort())
  const directoryLabel = computed(() => directory.value === ALL_DIRECTORIES ? '全部工作目录' : directory.value === UNKNOWN_DIRECTORY ? '未知工作目录' : directory.value)
  const missingDirectory = computed(() => ![ALL_DIRECTORIES, UNKNOWN_DIRECTORY].includes(directory.value) && !directories.value.includes(directory.value))
  const hasActiveFilter = computed(() => search.value.trim() !== '' || directory.value !== ALL_DIRECTORIES)
  const filteredProcesses = computed(() => {
    const query = search.value.trim().toLocaleLowerCase()
    return allProcesses.value.filter(process => {
      const matchesDirectory = directory.value === ALL_DIRECTORIES || (directory.value === UNKNOWN_DIRECTORY ? !process.cwd : process.cwd === directory.value)
      return matchesDirectory && matchesQuery(process, query)
    })
  })

  function iconFor(process: ProcessInfo) {
    return processIcons.value[iconKey(process)] ?? ''
  }

  function discardIcon(process: ProcessInfo) {
    processIcons.value[iconKey(process)] = ''
  }

  async function loadIcons(list: ProcessInfo[], generation: number) {
    for (const process of list) {
      if (!active || generation !== iconGeneration) return
      if (!hasKnownStartTime(process)) continue
      const key = iconKey(process)
      if (key in processIcons.value) continue
      let icon = ''
      try {
        icon = await api.icon(process.pid, process.startedAt!)
      } catch {
        // 进程可能在采集后退出或拒绝访问，此时保留通用图标。
      }
      if (!active || generation !== iconGeneration) return
      processIcons.value[key] = icon
    }
  }

  async function refresh() {
    if (!active) return
    if (loading.value) {
      refreshAgain = true
      return
    }
    const generation = ++iconGeneration
    loading.value = true
    try {
      const result = await api.list()
      if (!active) return
      snapshot.value = result
      now.value = Date.now()
      error.value = ''
      processIcons.value = Object.fromEntries(result.processes
        .map(iconKey).filter(key => key in processIcons.value)
        .map(key => [key, processIcons.value[key]]))
      void loadIcons(result.processes, generation)
    } catch (reason) {
      if (active) error.value = reason instanceof Error ? reason.message : String(reason)
    } finally {
      if (active) {
        loading.value = false
        if (refreshAgain) {
          refreshAgain = false
          void refresh()
        }
      }
    }
  }

  function requestTermination(process: ProcessInfo) {
    if (terminating.value || terminationDisabledReason(process)) return
    terminationTarget.value = { pid: process.pid, name: process.name || '未知', startedAt: process.startedAt! }
    terminationError.value = ''
    terminationMessage.value = ''
    dialogOpen.value = true
  }

  function setDialogOpen(open: boolean) {
    if (terminating.value) return
    dialogOpen.value = open
    if (!open) terminationTarget.value = null
  }

  async function confirmTermination() {
    const target = terminationTarget.value
    if (!target || terminating.value) return
    terminating.value = true
    terminationError.value = ''
    try {
      await api.terminate(target.pid, target.startedAt)
      if (!active) return
      terminationMessage.value = `已发送结束请求：${target.name}（PID ${target.pid}）。`
      dialogOpen.value = false
      terminationTarget.value = null
      void refresh()
    } catch (reason) {
      if (active) terminationError.value = reason instanceof Error ? reason.message : String(reason)
    } finally {
      terminating.value = false
    }
  }

  /** 由入口组件在挂载时调用：注册焦点刷新并做首次采集。 */
  function start() {
    if (started) return
    started = true
    active = true
    window.addEventListener('focus', refresh)
    void refresh()
  }

  /** 由入口组件在卸载时调用：停止刷新并丢弃迟到的结果。 */
  function stop() {
    if (!started) return
    started = false
    active = false
    iconGeneration++
    window.removeEventListener('focus', refresh)
  }

  return {
    snapshot, loading, error, now, search, directory,
    allProcesses, directories, directoryLabel, missingDirectory, hasActiveFilter, filteredProcesses,
    terminationTarget, dialogOpen, terminating, terminationError, terminationMessage,
    iconFor, discardIcon, refresh, start, stop,
    requestTermination, setDialogOpen, confirmTermination,
  }
}

export type ProcessStore = ReturnType<typeof createProcessStore>

let store: ProcessStore | null = null

/** 包内共享实例：同一功能的组件读同一份状态，互相不传 props。 */
export function useProcesses() {
  return (store ??= createProcessStore())
}
