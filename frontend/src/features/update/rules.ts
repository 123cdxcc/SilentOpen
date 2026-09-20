import type { UpdateResult } from './service'

/**
 * 后端把无法解析的版本号原样上报：本地开发构建是 "dev"，此时它不做版本比较，
 * 因此永远不会报告有新版本。后端不额外传"是否参与比较"的标志，前端用同一条规则
 * 决定怎么向用户解释。
 */
export function isReleaseVersion(version: string): boolean {
  return /^\d+(\.\d+){0,2}(-[0-9A-Za-z.-]+)?$/.test(version.trim())
}

/** 版本号用于展示时统一去掉可能存在的 "v" 前缀，由界面自己决定加不加。 */
export function versionLabel(version: string): string {
  return version.trim().replace(/^[vV]/, '')
}

/**
 * 用户主动检查完成后给出的结论。发现新版本时返回空串——那种情况由横幅本身表达，
 * 不需要额外的文字。
 */
export function checkSummary(result: UpdateResult): string {
  if (!isReleaseVersion(result.currentVersion)) return '当前是本地开发构建，不参与版本比较。'
  if (!result.latestVersion) return 'GitHub 上还没有已发布的版本。'
  return result.updateAvailable ? '' : '当前已是最新版本。'
}
