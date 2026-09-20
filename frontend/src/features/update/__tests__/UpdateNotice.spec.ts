import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import UpdateNotice from '@/features/update/UpdateNotice.vue'
import { useUpdate } from '@/features/update/useUpdate'
import type { UpdateResult } from '@/features/update/service'

const fake = vi.hoisted(() => ({ check: vi.fn(), skip: vi.fn(), openPage: vi.fn(), currentVersion: vi.fn() }))

vi.mock('@/features/update/service', () => ({ updateService: fake }))

// 组件读取共享实例，测试之间必须把它恢复到初始状态。
const store = useUpdate()

const result = (overrides: Partial<UpdateResult> = {}): UpdateResult => ({
  currentVersion: '1.0.0',
  latestVersion: '',
  updateAvailable: false,
  skipped: false,
  notes: '',
  pageUrl: '',
  assetName: '',
  checkedAt: 0,
  ...overrides,
})

const available = (overrides: Partial<UpdateResult> = {}) => result({
  latestVersion: '1.2.0',
  updateAvailable: true,
  notes: '修好了两个问题。',
  pageUrl: 'https://github.com/example/SilentOpen/releases/tag/v1.2.0',
  assetName: 'SilentOpen-1.2.0-macos-universal.zip',
  ...overrides,
})

/** 按可见文字找按钮；找不到时给出可读的失败信息，而不是空指针。 */
function buttonByText(wrapper: VueWrapper, text: string) {
  const button = wrapper.findAll('button').find(candidate => candidate.text().trim() === text)
  if (!button) throw new Error(`没有找到按钮：${text}`)
  return button
}

function buttonByLabel(wrapper: VueWrapper, label: string) {
  const button = wrapper.find(`button[aria-label="${label}"]`)
  if (!button.exists()) throw new Error(`没有找到按钮：${label}`)
  return button
}

const wrappers: VueWrapper[] = []

async function mountNotice() {
  const wrapper = mount(UpdateNotice)
  wrappers.push(wrapper)
  await flushPromises()
  return wrapper
}

beforeEach(() => {
  fake.check.mockReset().mockResolvedValue(available())
  fake.skip.mockReset().mockResolvedValue(undefined)
  fake.openPage.mockReset().mockResolvedValue(undefined)
  fake.currentVersion.mockReset().mockResolvedValue('1.0.0')
  store.stop()
  store.result.value = null
  store.error.value = ''
  store.message.value = ''
  store.checking.value = false
  store.dismissed.value = false
  store.version.value = ''
})

afterEach(() => {
  while (wrappers.length) wrappers.pop()!.unmount()
  vi.useRealTimers()
})

describe('有新版本时', () => {
  it('显示版本、说明和匹配本机的下载文件', async () => {
    const wrapper = await mountNotice()

    expect(wrapper.text()).toContain('有新版本 v1.2.0')
    expect(wrapper.text()).toContain('修好了两个问题。')
    expect(wrapper.text()).toContain('SilentOpen-1.2.0-macos-universal.zip')
    expect(buttonByText(wrapper, '打开下载页')).toBeTruthy()
    expect(buttonByText(wrapper, '忽略此版本')).toBeTruthy()
  })

  it('打开下载页调用后端', async () => {
    const wrapper = await mountNotice()

    await buttonByText(wrapper, '打开下载页').trigger('click')
    expect(fake.openPage).toHaveBeenCalledWith('https://github.com/example/SilentOpen/releases/tag/v1.2.0')
  })

  it('忽略此版本后横幅消失', async () => {
    const wrapper = await mountNotice()

    await buttonByText(wrapper, '忽略此版本').trigger('click')
    await flushPromises()

    expect(fake.skip).toHaveBeenCalledWith('1.2.0')
    expect(wrapper.text()).not.toContain('有新版本')
  })

  it('稍后提醒后横幅消失', async () => {
    const wrapper = await mountNotice()

    await buttonByLabel(wrapper, '稍后提醒').trigger('click')
    await flushPromises()

    expect(wrapper.text()).not.toContain('有新版本')
  })

  it('没有匹配本机的文件时只提示发布页', async () => {
    fake.check.mockResolvedValue(available({ assetName: '' }))
    const wrapper = await mountNotice()

    expect(wrapper.text()).not.toContain('匹配本机下载')
    expect(buttonByText(wrapper, '打开下载页')).toBeTruthy()
  })
})

describe('没有新版本时', () => {
  it('什么都不显示', async () => {
    fake.check.mockResolvedValue(result({ latestVersion: '1.0.0' }))
    const wrapper = await mountNotice()

    expect(wrapper.text()).toBe('')
  })

  it('主动检查的结论显示三秒后不再占位，再次检查重新计时', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    fake.check.mockResolvedValue(result({ latestVersion: '1.0.0' }))
    const wrapper = await mountNotice()
    await store.check(true)
    await flushPromises()

    expect(wrapper.text()).toContain('当前已是最新版本。')
    await vi.advanceTimersByTimeAsync(2000)
    await store.check(true)
    await vi.advanceTimersByTimeAsync(1000)
    expect(wrapper.text()).toContain('当前已是最新版本。')

    await vi.advanceTimersByTimeAsync(2000)
    expect(wrapper.find('[role="status"]').exists()).toBe(false)
    expect(wrapper.text()).toBe('')
  })

  it('卸载时清除临时结论和计时器', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    fake.check.mockResolvedValue(result({ latestVersion: '1.0.0' }))
    await mountNotice()
    await store.check(true)
    expect(store.message.value).not.toBe('')

    wrappers.pop()!.unmount()
    expect(store.message.value).toBe('')
    expect(vi.getTimerCount()).toBe(0)
  })

  it('启动检查失败时不打扰用户', async () => {
    fake.check.mockRejectedValue(new Error('无法连接更新服务器'))
    const wrapper = await mountNotice()

    expect(wrapper.text()).toBe('')
  })
})

describe('主动检查失败时', () => {
  it('显示原因，关闭后消失', async () => {
    const wrapper = await mountNotice()
    store.error.value = '无法连接更新服务器'
    await flushPromises()

    expect(wrapper.text()).toContain('检查更新失败')
    expect(wrapper.text()).toContain('无法连接更新服务器')

    await buttonByLabel(wrapper, '关闭提示').trigger('click')
    await flushPromises()
    expect(wrapper.text()).not.toContain('检查更新失败')
  })

  it('不会把已经查到的版本藏起来', async () => {
    const wrapper = await mountNotice()
    store.error.value = '无法连接更新服务器'
    await flushPromises()

    expect(wrapper.text()).toContain('有新版本 v1.2.0')
    expect(wrapper.text()).toContain('检查更新失败')
  })
})
