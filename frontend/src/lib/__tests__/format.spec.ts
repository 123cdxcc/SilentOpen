import { describe, expect, it } from 'vitest'
import { formatClockTime, formatListener, formatStartedAt, formatUptime } from '@/lib/format'

describe('formatListener', () => {
  it('renders IPv4 endpoints as-is', () => {
    expect(formatListener({ ip: '127.0.0.1', port: 3000 })).toBe('127.0.0.1:3000')
  })

  it('brackets IPv6 endpoints', () => {
    expect(formatListener({ ip: '::', port: 3000 })).toBe('[::]:3000')
  })

  it('marks a missing host as unknown', () => {
    expect(formatListener({ ip: '', port: 3000 })).toBe('未知 IP:3000')
  })
})

describe('formatStartedAt', () => {
  it('marks missing timestamps as unknown', () => {
    expect(formatStartedAt(null)).toBe('未知')
    expect(formatStartedAt(undefined)).toBe('未知')
  })

  it('formats a known timestamp', () => {
    expect(formatStartedAt(Date.UTC(2024, 0, 2, 3, 4, 5))).not.toBe('未知')
  })
})

describe('formatUptime', () => {
  const now = Date.UTC(2024, 0, 2, 3, 4, 5)

  it('marks missing timestamps as unknown', () => {
    expect(formatUptime(null, now)).toBe('未知')
  })

  it('buckets seconds, minutes, hours and days', () => {
    expect(formatUptime(now - 5_000, now)).toBe('5 秒')
    expect(formatUptime(now - 5 * 60_000, now)).toBe('5 分钟')
    expect(formatUptime(now - 3 * 3_600_000 - 10 * 60_000, now)).toBe('3 小时 10 分')
    expect(formatUptime(now - 2 * 86_400_000 - 3 * 3_600_000, now)).toBe('2 天 3 小时')
  })

  it('never reports negative uptime', () => {
    expect(formatUptime(now + 10_000, now)).toBe('0 秒')
  })
})

describe('formatClockTime', () => {
  it('formats a clock time', () => {
    expect(formatClockTime(Date.UTC(2024, 0, 2, 3, 4, 5))).toMatch(/\d{2}:\d{2}:\d{2}/)
  })
})
