import { describe, expect, it } from 'vitest'
import { hasKnownStartTime, iconKey, matchesQuery, terminationDisabledReason } from '@/features/processes/rules'
import type { ProcessInfo } from '@/features/processes/service'

const process = (overrides: Partial<ProcessInfo> = {}): ProcessInfo => ({
  pid: 100,
  ppid: 1,
  parentName: 'launchd',
  name: 'node',
  project: 'demo',
  cwd: '/demo',
  listeners: [{ ip: '127.0.0.1', port: 3000 }, { ip: '2001:db8::1', port: 3000 }],
  command: 'node server',
  startedAt: 1,
  ...overrides,
})

describe('iconKey', () => {
  it('separates identities that reuse a PID', () => {
    expect(iconKey(process())).toBe('100:1')
    expect(iconKey(process({ startedAt: 2 }))).toBe('100:2')
  })
})

describe('hasKnownStartTime', () => {
  it('accepts real timestamps only', () => {
    expect(hasKnownStartTime(process({ startedAt: 1 }))).toBe(true)
    expect(hasKnownStartTime(process({ startedAt: undefined }))).toBe(false)
    expect(hasKnownStartTime(process({ startedAt: 0 }))).toBe(false)
    expect(hasKnownStartTime(process({ startedAt: Number.NaN }))).toBe(false)
  })
})

describe('terminationDisabledReason', () => {
  it('refuses system processes', () => {
    expect(terminationDisabledReason(process({ pid: 1 }))).toContain('系统保留进程')
    expect(terminationDisabledReason(process({ pid: 0 }))).toContain('系统保留进程')
  })

  it('refuses processes with an unknown identity', () => {
    expect(terminationDisabledReason(process({ startedAt: undefined }))).toContain('启动时间未知')
  })

  it('allows a usable target', () => {
    expect(terminationDisabledReason(process())).toBe('')
  })
})

describe('matchesQuery', () => {
  it('matches pid, name, project, cwd, endpoints and command', () => {
    for (const query of ['100', 'node', 'demo', '/demo', '2001:db8::1', '127.0.0.1', '3000', 'node server']) {
      expect(matchesQuery(process(), query.toLocaleLowerCase()), query).toBe(true)
    }
  })

  it('rejects unrelated queries', () => {
    expect(matchesQuery(process(), 'postgres')).toBe(false)
  })
})
