import { formatListener } from '@/lib/format'
import type { ProcessInfo } from './service'

/** 图标缓存键：PID 被复用后键随之改变。 */
export function iconKey(process: ProcessInfo) {
  return `${process.pid}:${process.startedAt}`
}

export function hasKnownStartTime(process: ProcessInfo) {
  return !!process.startedAt && Number.isFinite(process.startedAt)
}

/** 返回空串表示可以结束该进程。 */
export function terminationDisabledReason(process: ProcessInfo) {
  if (process.pid <= 1) return '系统保留进程不可结束'
  if (!hasKnownStartTime(process)) return '启动时间未知，无法安全确认进程身份'
  return ''
}

/** query 需由调用方预先 trim + 小写。 */
export function matchesQuery(process: ProcessInfo, query: string) {
  return [process.pid, process.name, process.project, process.cwd, ...process.listeners.map(formatListener), process.command]
    .join(' ').toLocaleLowerCase().includes(query)
}
