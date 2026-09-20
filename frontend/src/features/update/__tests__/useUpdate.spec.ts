import { describe, expect, it, vi } from 'vitest'
import { createUpdateStore } from '@/features/update/useUpdate'
import type { UpdateResult, UpdateServiceApi } from '@/features/update/service'

type Deferred<T> = { promise: Promise<T>; resolve: (value: T) => void; reject: (reason?: unknown) => void }

function defer<T>(): Deferred<T> {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((yes, no) => { resolve = yes; reject = no })
  return { promise, resolve, reject }
}

const tick = () => new Promise(resolve => setImmediate(resolve))

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

/** 假 service：检查请求由测试决定何时兑现，因此可以验证进行中的状态。 */
function createFakeApi() {
  const checks: (Deferred<UpdateResult> & { force: boolean })[] = []
  const api = {
    check: vi.fn((force: boolean) => {
      const request = Object.assign(defer<UpdateResult>(), { force })
      checks.push(request)
      return request.promise
    }),
    skip: vi.fn(async () => {}),
    openPage: vi.fn(async () => {}),
    currentVersion: vi.fn(async () => '1.0.0'),
  } satisfies UpdateServiceApi
  return { api, checks }
}

describe('启动时的静默检查', () => {
  it('发现新版本时把横幅需要的字段都准备好', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    checks[0].resolve(available())
    await tick()

    expect(store.visible.value).toBe(true)
    expect(store.latestLabel.value).toBe('v1.2.0')
    expect(store.notes.value).toBe('修好了两个问题。')
    expect(store.assetName.value).toBe('SilentOpen-1.2.0-macos-universal.zip')
    // 静默检查不产生结论文字，横幅自己会说明。
    expect(store.message.value).toBe('')
  })

  it('失败时不打扰用户', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    checks[0].reject(new Error('无法连接更新服务器'))
    await tick()

    expect(store.error.value).toBe('')
    expect(store.result.value).toBeNull()
    expect(store.checking.value).toBe(false)
  })

  it('取回版本号供界面展示', async () => {
    const { api } = createFakeApi()
    api.currentVersion.mockResolvedValue('v1.2.3')
    const store = createUpdateStore(api)
    store.start()
    await tick()

    expect(store.version.value).toBe('1.2.3')
  })

  it('重复调用 start 不会重复检查', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    store.start()
    expect(checks).toHaveLength(1)
  })

  it('进行中的检查不会被再次触发', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    void store.check(true)
    await tick()
    expect(checks).toHaveLength(1)
  })

  it('卸载后丢弃迟到的结果，并允许重新挂载再查一次', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    store.stop()
    expect(store.checking.value).toBe(false)

    checks[0].resolve(available())
    await tick()
    expect(store.result.value).toBeNull()

    store.start()
    expect(checks).toHaveLength(2)
  })
})

describe('用户主动检查', () => {
  it('失败时报出后端的中文提示，并可以关闭', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    checks[0].resolve(result())
    await tick()

    void store.check(true)
    checks[1].reject(new Error('无法连接更新服务器'))
    await tick()

    expect(store.error.value).toBe('无法连接更新服务器')
    store.clearError()
    expect(store.error.value).toBe('')
  })

  it('没有新版本时给出结论', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    checks[0].resolve(result())
    await tick()

    void store.check(true)
    checks[1].resolve(result({ latestVersion: '1.0.0' }))
    await tick()

    expect(store.message.value).toContain('已是最新版本')
    expect(store.visible.value).toBe(false)
  })

  it('本地开发构建如实说明不比较版本', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    checks[0].resolve(result())
    await tick()

    void store.check(true)
    checks[1].resolve(result({ currentVersion: 'dev', latestVersion: '1.2.0' }))
    await tick()

    expect(store.message.value).toContain('开发构建')
    expect(store.visible.value).toBe(false)
  })

  it('清掉上一次的结论再重新检查', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    checks[0].resolve(result())
    await tick()

    void store.check(true)
    checks[1].resolve(result({ latestVersion: '1.0.0' }))
    await tick()
    expect(store.message.value).not.toBe('')

    void store.check(true)
    expect(store.message.value).toBe('')
  })
})

describe('用户的选择', () => {
  it('忽略此版本会写回后端并收起横幅', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    checks[0].resolve(available())
    await tick()

    await store.skip()
    expect(api.skip).toHaveBeenCalledWith('1.2.0')
    expect(store.visible.value).toBe(false)
    expect(store.result.value?.updateAvailable).toBe(true)
  })

  it('没有新版本时忽略按钮不起作用', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    checks[0].resolve(result({ latestVersion: '1.0.0' }))
    await tick()

    await store.skip()
    expect(api.skip).not.toHaveBeenCalled()
  })

  it('忽略失败时报错', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    api.skip.mockRejectedValue(new Error('无法保存更新设置'))
    store.start()
    checks[0].resolve(available())
    await tick()

    await store.skip()
    expect(store.error.value).toBe('无法保存更新设置')
  })

  it('稍后提醒只在本次运行生效，出现另一个版本时重新提示', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    checks[0].resolve(available())
    await tick()
    store.dismiss()
    expect(store.visible.value).toBe(false)

    // 同一个版本再查一次不会把横幅叫回来。
    void store.check(true)
    checks[1].resolve(available())
    await tick()
    expect(store.visible.value).toBe(false)

    // 换了版本则重新提示。
    void store.check(true)
    checks[2].resolve(available({ latestVersion: '1.3.0' }))
    await tick()
    expect(store.visible.value).toBe(true)
    expect(store.latestLabel.value).toBe('v1.3.0')
  })
})

describe('打开下载页', () => {
  it('把发布页地址交给后端', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    store.start()
    checks[0].resolve(available())
    await tick()

    await store.openPage()
    expect(api.openPage).toHaveBeenCalledWith('https://github.com/example/SilentOpen/releases/tag/v1.2.0')
  })

  it('没有地址时什么都不做', async () => {
    const { api } = createFakeApi()
    const store = createUpdateStore(api)
    await store.openPage()
    expect(api.openPage).not.toHaveBeenCalled()
  })

  it('后端拒绝打开时报错', async () => {
    const { api, checks } = createFakeApi()
    const store = createUpdateStore(api)
    api.openPage.mockRejectedValue(new Error('只允许打开 GitHub 上的链接'))
    store.start()
    checks[0].resolve(available())
    await tick()

    await store.openPage()
    expect(store.error.value).toBe('只允许打开 GitHub 上的链接')
  })
})
