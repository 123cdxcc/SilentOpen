import { CheckUpdate, GetVersion, OpenReleasePage, SkipVersion } from '@wailsjs/go/main/App'

/** 前端侧的更新检查契约：与生成代码解耦，组件和测试都用普通对象即可构造。 */
export interface UpdateResult {
  currentVersion: string
  /** 没有任何已发布版本时为空串。 */
  latestVersion: string
  updateAvailable: boolean
  /** 用户已选择不再提示 latestVersion。 */
  skipped: boolean
  notes: string
  pageUrl: string
  /** 为当前平台挑出的发布文件；没有匹配项时为空串。 */
  assetName: string
  /** 结论产生的时间，Unix 秒。 */
  checkedAt: number
}

/** 后端 `update.Checker` 的调用契约；测试可注入假实现。 */
export interface UpdateServiceApi {
  check(force: boolean): Promise<UpdateResult>
  skip(version: string): Promise<void>
  openPage(url: string): Promise<void>
  currentVersion(): Promise<string>
}

/** 唯一 import Wails 生成代码的文件：换绑定结构只改这里。 */
export const updateService: UpdateServiceApi = {
  check: force => CheckUpdate(force),
  skip: version => SkipVersion(version),
  openPage: url => OpenReleasePage(url),
  currentVersion: () => GetVersion(),
}
