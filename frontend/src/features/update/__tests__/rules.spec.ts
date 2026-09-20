import { describe, expect, it } from 'vitest'
import { checkSummary, isReleaseVersion, versionLabel } from '@/features/update/rules'
import type { UpdateResult } from '@/features/update/service'

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

describe('isReleaseVersion', () => {
  it.each(['1.0.0', '0.1', '2', '1.2.3-rc.1', ' 1.2.3 '])('接受 %s', version => {
    expect(isReleaseVersion(version)).toBe(true)
  })

  it.each(['', 'dev', 'v1.2.3', 'nightly', '1.2.3.4', 'release-1'])('拒绝 %s', version => {
    expect(isReleaseVersion(version)).toBe(false)
  })
})

describe('versionLabel', () => {
  it.each([['1.2.3', '1.2.3'], ['v1.2.3', '1.2.3'], ['V1.2.3', '1.2.3'], [' dev ', 'dev']])('%s → %s', (input, want) => {
    expect(versionLabel(input)).toBe(want)
  })
})

describe('checkSummary', () => {
  it('本地开发构建不比较版本', () => {
    expect(checkSummary(result({ currentVersion: 'dev', latestVersion: '1.2.0' }))).toContain('开发构建')
  })

  it('仓库还没有发布版本时如实说明', () => {
    expect(checkSummary(result())).toContain('还没有已发布的版本')
  })

  it('已是最新版本', () => {
    expect(checkSummary(result({ currentVersion: '1.2.0', latestVersion: '1.2.0' }))).toContain('已是最新版本')
  })

  it('发现新版本时不额外说话，交给横幅表达', () => {
    expect(checkSummary(result({ currentVersion: '1.0.0', latestVersion: '1.2.0', updateAvailable: true }))).toBe('')
  })
})
