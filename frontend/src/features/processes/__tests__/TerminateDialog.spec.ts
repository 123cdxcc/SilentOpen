import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import TerminateDialog from '../TerminateDialog.vue'
import { useProcesses } from '../useProcesses'
import type { ProcessInfo, ProcessSnapshot } from '../service'

const fake = vi.hoisted(() => ({ list: vi.fn(), icon: vi.fn(), terminate: vi.fn() }))

vi.mock('@/features/processes/service', () => ({ processService: fake }))

const store = useProcesses()

const process = (pid: number, startedAt?: number, overrides: Partial<ProcessInfo> = {}): ProcessInfo => ({
  pid,
  ppid: 1,
  parentName: 'launchd',
  name: 'node',
  project: 'demo',
  cwd: '/demo',
  command: 'node server',
  listeners: [{ ip: '127.0.0.1', port: 3000 }],
  startedAt,
  ...overrides,
})

const snapshot = (...processes: ProcessInfo[]): ProcessSnapshot => ({
  processes,
  collectedAt: Date.UTC(2024, 0, 2, 3, 4, 5),
  warnings: [],
})

const bodyText = () => document.body.textContent ?? ''
const buttonByText = (text: string) =>
  Array.from(document.body.querySelectorAll('button')).find(button => button.textContent?.trim() === text)

/** 弹窗内容会 Teleport 到 body，因此统一在 afterEach 卸载，而不是清空 body。 */
const wrappers: { unmount: () => void }[] = []

function mountDialog() {
  const wrapper = mount(TerminateDialog)
  wrappers.push(wrapper)
  return wrapper
}

beforeEach(async () => {
  fake.list.mockReset().mockResolvedValue(snapshot(process(100, 1)))
  fake.icon.mockReset().mockResolvedValue('')
  fake.terminate.mockReset().mockResolvedValue(undefined)
  store.setDialogOpen(false)
  store.terminationError.value = ''
  store.terminationMessage.value = ''
  await store.refresh()
})

afterEach(() => {
  wrappers.splice(0).forEach(wrapper => wrapper.unmount())
})

describe('TerminateDialog', () => {
  it('stays closed until a target is requested', () => {
    mountDialog()
    expect(bodyText()).not.toContain('结束进程？')
  })

  it('shows the requested target', async () => {
    store.requestTermination(process(100, 1))
    mountDialog()
    await nextTick()
    expect(bodyText()).toContain('结束进程？')
    expect(bodyText()).toContain('node')
    expect(bodyText()).toContain('100')
  })

  it('confirms through the shared store and closes', async () => {
    store.requestTermination(process(100, 1))
    mountDialog()
    await nextTick()

    buttonByText('确认结束')?.click()
    await flushPromises()

    expect(fake.terminate).toHaveBeenCalledWith(100, 1)
    expect(store.dialogOpen.value).toBe(false)
    expect(store.terminationTarget.value).toBeNull()
    expect(store.terminationMessage.value).toContain('PID 100')
  })

  it('cannot be closed or resubmitted while busy', async () => {
    let finish: () => void = () => {}
    fake.terminate.mockImplementation(() => new Promise<void>(resolve => { finish = resolve }))
    store.requestTermination(process(100, 1))
    mountDialog()
    await nextTick()

    buttonByText('确认结束')?.click()
    await nextTick()

    expect(bodyText()).toContain('正在结束…')
    expect(buttonByText('正在结束…')?.hasAttribute('disabled')).toBe(true)
    expect(buttonByText('取消')?.hasAttribute('disabled')).toBe(true)

    store.setDialogOpen(false)
    expect(store.dialogOpen.value).toBe(true)

    finish()
    await flushPromises()
    expect(store.terminating.value).toBe(false)
  })

  it('keeps the dialog open and shows the failure', async () => {
    fake.terminate.mockRejectedValue(new Error('权限不足，无法结束进程'))
    store.requestTermination(process(100, 1))
    mountDialog()
    await nextTick()

    buttonByText('确认结束')?.click()
    await flushPromises()

    expect(bodyText()).toContain('结束请求失败')
    expect(bodyText()).toContain('权限不足，无法结束进程')
    expect(store.dialogOpen.value).toBe(true)
  })
})
