import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import ProcessViewer from '../ProcessViewer.vue'
import { useProcesses } from '../useProcesses'
import type { ProcessInfo, ProcessSnapshot } from '../service'

const fake = vi.hoisted(() => ({ list: vi.fn(), icon: vi.fn(), terminate: vi.fn() }))

vi.mock('@/features/processes/service', () => ({ processService: fake }))

/** 包内共享实例：每个用例先把它恢复到「无筛选、无弹窗」的初始状态。 */
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
const bodyButton = (text: string) =>
  Array.from(document.body.querySelectorAll('button')).find(button => button.textContent?.trim() === text)

/** 弹窗内容会 Teleport 到 body，因此统一在 afterEach 卸载。 */
const wrappers: { unmount: () => void }[] = []

async function mountViewer() {
  const wrapper = mount(ProcessViewer)
  wrappers.push(wrapper)
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  fake.list.mockReset().mockResolvedValue(snapshot(process(100, 1), process(200, 1, { name: 'vite', project: 'vite-app', cwd: '/demo/vite' })))
  fake.icon.mockReset().mockResolvedValue('')
  fake.terminate.mockReset().mockResolvedValue(undefined)
  store.search.value = ''
  store.directory.value = '__all__'
  store.setDialogOpen(false)
  store.terminationMessage.value = ''
})

afterEach(() => {
  wrappers.splice(0).forEach(wrapper => wrapper.unmount())
})

describe('ProcessViewer', () => {
  it('loads the list on mount and shows the process count', async () => {
    const wrapper = await mountViewer()
    expect(fake.list).toHaveBeenCalled()
    expect(wrapper.get('header').text()).toMatch(/监听中的进程\s*2/)
    expect(wrapper.text()).toContain('共 2 个进程')
  })

  it('refreshes on demand', async () => {
    const wrapper = await mountViewer()
    const before = fake.list.mock.calls.length

    await wrapper.get('button[aria-label="刷新进程列表"]').trigger('click')
    await flushPromises()

    expect(fake.list.mock.calls.length).toBe(before + 1)
  })

  it('filters from the toolbar and reports the match count', async () => {
    const wrapper = await mountViewer()
    await wrapper.get('input[type="search"]').setValue('vite')

    expect(wrapper.text()).toContain('vite-app')
    expect(wrapper.text()).toContain('匹配 1 / 2 个进程')
  })

  it('reports refresh failures without dropping the last data', async () => {
    const wrapper = await mountViewer()
    fake.list.mockRejectedValue(new Error('读取监听端口失败'))

    await wrapper.get('button[aria-label="刷新进程列表"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('刷新失败，当前显示上次成功的数据')
    expect(wrapper.text()).toContain('读取监听端口失败')
    expect(wrapper.text()).toContain('共 2 个进程')
  })

  it('surfaces collection warnings', async () => {
    fake.list.mockResolvedValue({
      ...snapshot(process(100, 1)),
      warnings: ['部分监听端口无法读取所属 PID，未列入进程表。'],
    })
    const wrapper = await mountViewer()

    expect(wrapper.text()).toContain('部分信息不可读取')
    expect(wrapper.text()).toContain('部分监听端口无法读取所属 PID，未列入进程表。')
  })

  it('reports a sent termination request after confirming', async () => {
    const wrapper = await mountViewer()
    await wrapper.get('button[aria-label^="结束进程"]').trigger('click')
    await nextTick()

    bodyButton('确认结束')?.click()
    await flushPromises()

    expect(wrapper.text()).toContain('已发送结束请求：node（PID 100）')
  })
})
