import { GetProcessIcon, GetProcesses, TerminateProcess } from '@wailsjs/go/main/App'

/** 前端侧的数据契约：与生成代码解耦，组件和测试都用普通对象即可构造。 */
export interface Listener {
  ip: string
  port: number
}

export interface ProcessInfo {
  pid: number
  ppid?: number
  parentName: string
  name: string
  project: string
  cwd: string
  listeners: Listener[]
  command: string
  startedAt?: number
}

export interface ProcessSnapshot {
  processes: ProcessInfo[]
  collectedAt: number
  warnings: string[]
}

/** 后端 `processes.Service` 的调用契约；测试可注入假实现。 */
export interface ProcessServiceApi {
  list(): Promise<ProcessSnapshot>
  icon(pid: number, startedAt: number): Promise<string>
  terminate(pid: number, startedAt: number): Promise<void>
}

/** 唯一 import Wails 生成代码的文件：换绑定结构只改这里。 */
export const processService: ProcessServiceApi = {
  list: () => GetProcesses(),
  icon: (pid, startedAt) => GetProcessIcon(pid, startedAt),
  terminate: (pid, startedAt) => TerminateProcess(pid, startedAt),
}
