import { beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import ProcessList from '../ProcessList.vue'
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
  listeners: [{ ip: '127.0.0.1', port: 3000 }, { ip: '2001:db8::1', port: 3000 }],
  startedAt,
  ...overrides,
})

const snapshot = (...processes: ProcessInfo[]): ProcessSnapshot => ({
  processes,
  collectedAt: Date.UTC(2024, 0, 2, 3, 4, 5),
  warnings: [],
})

const cardTerminateButton = (wrapper: ReturnType<typeof mount>) =>
  wrapper.findAll('button').filter(button => button.text() === '结束进程')[0]

const rowTerminateButton = (wrapper: ReturnType<typeof mount>) =>
  wrapper.findAll('button').filter(button => button.text() === '结束')[0]

beforeEach(async () => {
  fake.list.mockReset().mockResolvedValue(snapshot(process(100, 1)))
  fake.icon.mockReset().mockResolvedValue('')
  fake.terminate.mockReset().mockResolvedValue(undefined)
  store.search.value = ''
  store.directory.value = '__all__'
  store.setDialogOpen(false)
  store.terminationMessage.value = ''
  await store.refresh()
})

describe('ProcessList', () => {
  it('renders endpoints, project and timing for every process', () => {
    const wrapper = mount(ProcessList)
    const text = wrapper.text()
    expect(text).toContain('PID ↑')
    expect(text).toContain('100')
    expect(text).toContain('node')
    expect(text).toContain('demo')
    expect(text).toContain('127.0.0.1:3000')
    expect(text).toContain('[2001:db8::1]:3000')
    expect(text).toContain('运行时长')
  })

  it('shows details only while a process is expanded', async () => {
    const wrapper = mount(ProcessList)
    expect(wrapper.find('#desktop-details-100').exists()).toBe(false)

    await wrapper.get('button[aria-controls="desktop-details-100"]').trigger('click')
    expect(wrapper.find('#desktop-details-100').exists()).toBe(true)
    expect(wrapper.get('#desktop-details-100').text()).toContain('node server')

    await wrapper.get('button[aria-controls="desktop-details-100"]').trigger('click')
    expect(wrapper.find('#desktop-details-100').exists()).toBe(false)
  })

  it('expands and collapses a desktop row by clicking anywhere on it', async () => {
    const wrapper = mount(ProcessList)
    const row = wrapper.findAll('tbody tr')[0]

    await row.trigger('click')
    expect(wrapper.find('#desktop-details-100').exists()).toBe(true)

    await row.trigger('click')
    expect(wrapper.find('#desktop-details-100').exists()).toBe(false)
  })

  it('expands and collapses a mobile card by tapping it', async () => {
    const wrapper = mount(ProcessList)
    const card = wrapper.findAll('[data-slot="card"]')[0]

    await card.trigger('click')
    expect(wrapper.find('#mobile-details-100').exists()).toBe(true)

    await card.trigger('click')
    expect(wrapper.find('#mobile-details-100').exists()).toBe(false)
  })

  it('does not toggle a row or card when its terminate button is clicked', async () => {
    const wrapper = mount(ProcessList)

    await rowTerminateButton(wrapper).trigger('click')
    await cardTerminateButton(wrapper).trigger('click')

    expect(wrapper.find('#desktop-details-100').exists()).toBe(false)
    expect(wrapper.find('#mobile-details-100').exists()).toBe(false)
  })

  it('keeps a mobile card open while its details are used', async () => {
    const wrapper = mount(ProcessList)
    await wrapper.findAll('[data-slot="card"]')[0].trigger('click')

    await wrapper.get('#mobile-details-100').trigger('click')
    expect(wrapper.find('#mobile-details-100').exists()).toBe(true)
  })

  it('asks the shared store to terminate a process', async () => {
    const wrapper = mount(ProcessList)
    await cardTerminateButton(wrapper).trigger('click')
    expect(store.dialogOpen.value).toBe(true)
    expect(store.terminationTarget.value).toEqual({ pid: 100, name: 'node', startedAt: 1 })
  })

  it('disables termination for a system process and explains why', async () => {
    fake.list.mockResolvedValue(snapshot(process(1, 1)))
    await store.refresh()
    const wrapper = mount(ProcessList)

    const button = cardTerminateButton(wrapper)
    expect(button.attributes('disabled')).toBeDefined()
    expect(button.element.closest('span')?.getAttribute('title')).toContain('系统保留进程不可结束')

    await button.trigger('click')
    expect(store.dialogOpen.value).toBe(false)
  })

  it('shows the empty state when nothing matches the search', async () => {
    const wrapper = mount(ProcessList)
    store.search.value = 'postgres'
    await nextTick()
    expect(wrapper.text()).toContain('没有匹配的进程')
  })

  it('explains when no process is listening at all', async () => {
    fake.list.mockResolvedValue(snapshot())
    await store.refresh()
    const wrapper = mount(ProcessList)
    expect(wrapper.text()).toContain('暂无监听进程')
  })

  it('shows the initial loading state before the first snapshot', () => {
    store.snapshot.value = null
    const wrapper = mount(ProcessList)
    expect(wrapper.text()).toContain('正在读取本机进程…')
  })
})
