const dateFormat = new Intl.DateTimeFormat('zh-CN', {
  year: 'numeric', month: '2-digit', day: '2-digit',
  hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false,
})
const timeFormat = new Intl.DateTimeFormat('zh-CN', {
  hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false,
})

/** 监听地址显示：IPv6 加方括号，空 IP 记为未知。 */
export function formatListener(listener: { ip: string; port: number }) {
  const ip = listener.ip.includes(':') ? `[${listener.ip}]` : listener.ip || '未知 IP'
  return `${ip}:${listener.port}`
}

export function formatStartedAt(timestamp: number | null | undefined) {
  return timestamp == null ? '未知' : dateFormat.format(timestamp)
}

export function formatUptime(timestamp: number | null | undefined, now: number) {
  if (timestamp == null) return '未知'
  const seconds = Math.max(0, Math.floor((now - timestamp) / 1000))
  if (seconds < 60) return `${seconds} 秒`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes} 分钟`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours} 小时 ${minutes % 60} 分`
  return `${Math.floor(hours / 24)} 天 ${hours % 24} 小时`
}

export function formatClockTime(timestamp: number) {
  return timeFormat.format(timestamp)
}
