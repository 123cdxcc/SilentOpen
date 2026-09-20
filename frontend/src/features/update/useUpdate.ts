import { computed, ref } from 'vue'
import { checkSummary, versionLabel } from './rules'
import { updateService, type UpdateResult, type UpdateServiceApi } from './service'

function describe(reason: unknown) {
  return reason instanceof Error ? reason.message : String(reason)
}

/**
 * 更新提示功能的全部状态与动作。包内组件通过 useUpdate() 共享同一实例，因此组件
 * 之间不需要传递 props。createUpdateStore() 供测试注入假 service。
 */
export function createUpdateStore(api: UpdateServiceApi = updateService) {
  const result = ref<UpdateResult | null>(null)
  const checking = ref(false)
  const error = ref('')
  const message = ref('')
  const version = ref('')
  const dismissed = ref(false)

  let active = true
  let started = false
  let messageTimer: ReturnType<typeof setTimeout> | undefined

  const visible = computed(() => Boolean(result.value?.updateAvailable) && !result.value?.skipped && !dismissed.value)
  const latestLabel = computed(() => (result.value?.latestVersion ? `v${result.value.latestVersion}` : ''))
  const notes = computed(() => (result.value?.notes ?? '').trim())
  const assetName = computed(() => result.value?.assetName ?? '')

  function clearMessage() {
    clearTimeout(messageTimer)
    messageTimer = undefined
    message.value = ''
  }

  /**
   * 查询更新。force 为假时用于启动后的静默检查：失败不打扰用户，只在下一次主动
   * 检查时报告；force 为真时用于用户点击，任何结果都要有反馈。
   */
  async function check(force = false) {
    if (checking.value) return
    checking.value = true
    if (force) {
      error.value = ''
      clearMessage()
    }
    try {
      const answer = await api.check(force)
      if (!active) return
      // 发现的是另一个版本时，之前"稍后提醒"的选择不再适用。
      if (answer.latestVersion !== result.value?.latestVersion) dismissed.value = false
      result.value = answer
      if (force) {
        message.value = checkSummary(answer)
        if (message.value) messageTimer = setTimeout(clearMessage, 3000)
      }
    } catch (reason) {
      if (active && force) error.value = describe(reason)
    } finally {
      if (active) checking.value = false
    }
  }

  /** 取回当前版本号，只用于展示；取不到就不显示。 */
  async function loadVersion() {
    try {
      const current = await api.currentVersion()
      if (active) version.value = versionLabel(current)
    } catch {
      // 版本号取不到不影响任何功能。
    }
  }

  /** 不再提示这个版本。后端会持久化，重启后依然生效。 */
  async function skip() {
    const target = result.value
    if (!target?.updateAvailable || !target.latestVersion) return
    try {
      await api.skip(target.latestVersion)
      if (!active) return
      result.value = { ...target, skipped: true }
      clearMessage()
    } catch (reason) {
      if (active) error.value = describe(reason)
    }
  }

  /** 本次运行先不提示；下次启动仍会提示。 */
  function dismiss() {
    dismissed.value = true
  }

  function clearError() {
    error.value = ''
  }

  /** 打开发布页，交给系统浏览器；后端只接受 GitHub 上的 https 链接。 */
  async function openPage() {
    const url = result.value?.pageUrl
    if (!url) return
    try {
      await api.openPage(url)
    } catch (reason) {
      if (active) error.value = describe(reason)
    }
  }

  /** 由入口组件在挂载时调用：取回版本号并静默检查一次。 */
  function start() {
    if (started) return
    started = true
    active = true
    void loadVersion()
    void check(false)
  }

  /** 由入口组件在卸载时调用：丢弃迟到的结果，并允许重新挂载后再查一次。 */
  function stop() {
    clearMessage()
    if (!started) return
    started = false
    active = false
    checking.value = false
  }

  return {
    result, checking, error, message, version, dismissed,
    visible, latestLabel, notes, assetName,
    check, skip, dismiss, clearError, openPage, start, stop,
  }
}

export type UpdateStore = ReturnType<typeof createUpdateStore>

let store: UpdateStore | null = null

/** 包内共享实例：同一功能的组件读同一份状态，互相不传 props。 */
export function useUpdate() {
  return (store ??= createUpdateStore())
}
