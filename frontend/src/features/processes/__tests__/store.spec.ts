import { afterEach, describe, expect, it, vi } from 'vitest'
import { createProcessStore, type ProcessStore } from '@/features/processes/useProcesses'
import type { ProcessInfo, ProcessServiceApi, ProcessSnapshot } from '@/features/processes/service'

type Deferred<T> = { promise: Promise<T>; resolve: (value: T) => void; reject: (reason?: unknown) => void }

function defer<T>(): Deferred<T> {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no })
  return { promise, resolve, reject }
}

const tick = () => new Promise(resolve => setImmediate(resolve))

const process = (pid: number, startedAt?: number, overrides: Partial<ProcessInfo> = {}): ProcessInfo => ({
  pid,
  ppid: 1,
  parentName: 'launchd',
  name: 'node',
  project: 'demo',
  cwd: '/demo',
  command: 'node server',
  listeners: [{ ip: '127.0.0.1', port: 3000 }, { ip: '2001:db8::1', port: 3000 }],
  startedAt,
  ...overrides,
})

const snapshot = (...processes: ProcessInfo[]): ProcessSnapshot => ({ processes, collectedAt: Date.now(), warnings: [] })

type IconRequest = Deferred<string> & { pid: number; startedAt: number }

function createFakeApi() {
  const lists: Deferred<ProcessSnapshot>[] = []
  const icons: IconRequest[] = []
  let terminate: (pid: number, startedAt: number) => Promise<void> = async () => {}
  const api: ProcessServiceApi = {
    list: () => {
      const request = defer<ProcessSnapshot>()
      lists.push(request)
      return request.promise
    },
    icon: (pid, startedAt) => {
      const request = Object.assign(defer<string>(), { pid, startedAt })
      icons.push(request)
      return request.promise
    },
    terminate: (pid, startedAt) => terminate(pid, startedAt),
  }
  return { api, lists, icons, setTerminate: (fn: typeof terminate) => { terminate = fn } }
}

const stores: ProcessStore[] = []

function newStore(api: ProcessServiceApi) {
  const store = createProcessStore(api)
  stores.push(store)
  return store
}

afterEach(() => {
  stores.splice(0).forEach(store => store.stop())
  vi.restoreAllMocks()
})

describe('collection, focus refresh and icons', () => {
  it('fetches once, coalesces focus refreshes, and keeps icons async', async () => {
    const { api, lists, icons } = createFakeApi()
    const store = newStore(api)
    const add = vi.spyOn(window, 'addEventListener')

    store.start()
    expect(lists).toHaveLength(1)

    window.dispatchEvent(new Event('focus'))
    window.dispatchEvent(new Event('focus'))
    expect(lists).toHaveLength(1)

    lists[0].resolve(snapshot(process(100, 1)))
    await tick()
    expect(lists).toHaveLength(2)
    expect(add.mock.calls.filter(([type]) => type === 'focus')).toHaveLength(1)

    lists[1].resolve(snapshot(process(100, 1), process(200, undefined)))
    await tick()
    expect(store.loading.value).toBe(false)
    expect(icons.map(({ pid, startedAt }) => [pid, startedAt])).toEqual([[100, 1], [100, 1]])

    icons[0].resolve('stale')
    await tick()
    expect(store.iconFor(process(100, 1))).toBe('')

    icons[1].resolve('data:image/png;base64,valid')
    await tick()
    expect(store.iconFor(process(100, 1))).toBe('data:image/png;base64,valid')
  })

  it('reuses icons by identity and prunes them when a PID is reused', async () => {
    const { api, lists, icons } = createFakeApi()
    const store = newStore(api)

    store.start()
    lists[0].resolve(snapshot(process(100, 1)))
    await tick()
    icons[0].resolve('data:image/png;base64,valid')
    await tick()
    expect(store.iconFor(process(100, 1))).toBe('data:image/png;base64,valid')

    window.dispatchEvent(new Event('focus'))
    lists[1].resolve(snapshot(process(100, 1)))
    await tick()
    expect(icons).toHaveLength(1)
    expect(store.iconFor(process(100, 1))).toBe('data:image/png;base64,valid')

    window.dispatchEvent(new Event('focus'))
    lists[2].resolve(snapshot(process(100, 2)))
    await tick()
    expect(store.iconFor(process(100, 1))).toBe('')
    expect(icons).toHaveLength(2)
    expect(icons[1]).toMatchObject({ pid: 100, startedAt: 2 })
  })

  it('removes the focus listener on stop and discards late results', async () => {
    const { api, lists, icons } = createFakeApi()
    const store = newStore(api)
    const add = vi.spyOn(window, 'addEventListener')
    const remove = vi.spyOn(window, 'removeEventListener')

    store.start()
    lists[0].resolve(snapshot(process(100, 1)))
    await tick()
    expect(icons).toHaveLength(1)

    store.stop()
    expect(remove).toHaveBeenCalledWith('focus', expect.any(Function))

    const before = lists.length
    window.dispatchEvent(new Event('focus'))
    expect(lists).toHaveLength(before)

    icons[0].resolve('late')
    await tick()
    expect(store.iconFor(process(100, 1))).toBe('')
    expect(add.mock.calls.filter(([type]) => type === 'focus')).toHaveLength(1)
  })

  it('keeps the last snapshot when a refresh fails', async () => {
    const { api, lists } = createFakeApi()
    const store = newStore(api)

    store.start()
    lists[0].resolve(snapshot(process(100, 1)))
    await tick()

    window.dispatchEvent(new Event('focus'))
    lists[1].reject(new Error('读取监听端口失败'))
    await tick()
    expect(store.error.value).toBe('读取监听端口失败')
    expect(store.allProcesses.value.map(p => p.pid)).toEqual([100])
  })

  it('never polls after the initial load', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'setInterval', 'clearTimeout', 'clearInterval'] })
    try {
      const { api, lists } = createFakeApi()
      const store = newStore(api)
      store.start()
      lists[0].resolve(snapshot(process(100, 1)))
      await tick()
      expect(lists).toHaveLength(1)

      await vi.advanceTimersByTimeAsync(60_000)
      expect(lists).toHaveLength(1)
      expect(store.loading.value).toBe(false)
    } finally {
      vi.useRealTimers()
    }
  })
})

describe('filters', () => {
  it('searches endpoints, ports and identity fields', async () => {
    const { api, lists } = createFakeApi()
    const store = newStore(api)
    store.start()
    lists[0].resolve(snapshot(process(100, 1), process(200, 1)))
    await tick()

    for (const query of ['2001:db8::1', '127.0.0.1', '3000', 'node']) {
      store.search.value = query
      expect(store.filteredProcesses.value, query).toHaveLength(2)
    }
    store.search.value = 'postgres'
    expect(store.filteredProcesses.value).toHaveLength(0)
  })

  it('keeps directory options independent of the search result', async () => {
    const { api, lists } = createFakeApi()
    const store = newStore(api)
    store.start()
    lists[0].resolve(snapshot(
      process(100, 1),
      process(200, 1, { cwd: '/other', project: 'other' }),
      process(300, 1, { cwd: '', project: '' }),
    ))
    await tick()

    expect(store.directories.value).toEqual(['/demo', '/other'])
    store.search.value = 'demo'
    expect(store.filteredProcesses.value).toHaveLength(1)
    expect(store.directories.value).toEqual(['/demo', '/other'])
    expect(store.hasActiveFilter.value).toBe(true)

    store.search.value = ''
    store.directory.value = '__unknown__'
    expect(store.filteredProcesses.value.map(p => p.pid)).toEqual([300])
    expect(store.directoryLabel.value).toBe('未知工作目录')

    store.directory.value = '/other'
    expect(store.filteredProcesses.value.map(p => p.pid)).toEqual([200])
    expect(store.directoryLabel.value).toBe('/other')
    expect(store.missingDirectory.value).toBe(false)

    store.directory.value = '/gone'
    expect(store.missingDirectory.value).toBe(true)

    store.directory.value = '__all__'
    expect(store.directoryLabel.value).toBe('全部工作目录')
    expect(store.hasActiveFilter.value).toBe(false)
  })
})

describe('termination', () => {
  it('refuses unusable targets and opens the dialog for the rest', async () => {
    const { api } = createFakeApi()
    const store = newStore(api)

    store.requestTermination(process(1, 1))
    store.requestTermination(process(100, undefined))
    expect(store.dialogOpen.value).toBe(false)

    store.requestTermination(process(100, 1, { name: '' }))
    expect(store.dialogOpen.value).toBe(true)
    expect(store.terminationTarget.value).toEqual({ pid: 100, name: '未知', startedAt: 1 })

    store.setDialogOpen(false)
    expect(store.dialogOpen.value).toBe(false)
    expect(store.terminationTarget.value).toBeNull()
  })

  it('reports success, closes the dialog and refreshes', async () => {
    const { api, lists } = createFakeApi()
    const store = newStore(api)
    store.start()
    lists[0].resolve(snapshot(process(100, 1)))
    await tick()

    store.requestTermination(process(100, 1))
    await store.confirmTermination()
    expect(store.terminationMessage.value).toContain('PID 100')
    expect(store.dialogOpen.value).toBe(false)
    expect(store.terminationTarget.value).toBeNull()
    expect(lists).toHaveLength(2)
  })

  it('keeps the dialog open and shows the failure', async () => {
    const fake = createFakeApi()
    fake.setTerminate(async () => { throw new Error('权限不足，无法结束进程') })
    const store = newStore(fake.api)
    store.start()
    fake.lists[0].resolve(snapshot(process(100, 1)))
    await tick()

    store.requestTermination(process(100, 1))
    await store.confirmTermination()
    expect(store.terminationError.value).toBe('权限不足，无法结束进程')
    expect(store.dialogOpen.value).toBe(true)
    expect(store.terminating.value).toBe(false)
  })

  it('cannot be closed or resubmitted while busy', async () => {
    const fake = createFakeApi()
    const pending = defer<void>()
    fake.setTerminate(() => pending.promise)
    const store = newStore(fake.api)
    store.start()
    fake.lists[0].resolve(snapshot(process(100, 1), process(200, 1)))
    await tick()

    store.requestTermination(process(100, 1))
    const confirming = store.confirmTermination()
    expect(store.terminating.value).toBe(true)

    store.setDialogOpen(false)
    expect(store.dialogOpen.value).toBe(true)
    store.requestTermination(process(200, 1))
    expect(store.terminationTarget.value?.pid).toBe(100)

    pending.resolve()
    await confirming
    expect(store.terminating.value).toBe(false)
  })
})
