# SilentOpen 前端

Vue 3 + TypeScript + Vite + shadcn-vue，作为 Wails 的 WebView 界面，通过 `wailsjs` 绑定调用 Go 的 `processes.Service`。

## 目录

```text
src/
├── main.ts                     # 挂载唯一的页面组件
├── components/ui/              # shadcn-vue 原语：唯一的通用 UI 层
├── lib/
│   ├── utils.ts                # cn（shadcn 依赖）
│   ├── format.ts               # 与业务无关的纯函数：时间、运行时长、IP:端口
│   └── __tests__/
└── features/
    └── processes/              # 「进程查看」垂直功能包：对外只暴露 ProcessViewer
        ├── ProcessViewer.vue   # 入口组件（无 props / 无 emit）：页面主干与状态生命周期
        ├── ProcessList.vue     # 列表大块：桌面表格 + 移动卡片（含展开状态）
        ├── ProcessDetails.vue  # 详情大块（唯一入参：process）
        ├── TerminateDialog.vue # 结束确认弹窗
        ├── service.ts          # 唯一 import @wailsjs/* 的文件
        ├── rules.ts            # 该业务的规则（纯函数）
        ├── useProcesses.ts     # 包内共享状态与动作
        └── __tests__/
```

别名：`@` → `src`，`@wailsjs` → `wailsjs`（见 `vite.config.ts` 与 `tsconfig.json`）。

## 状态流

`ProcessViewer.vue` 是包内唯一入口：挂载时调用 `useProcesses().start()`（注册窗口 `focus` 刷新 + 首次采集），卸载时调用 `stop()`（移除监听、丢弃迟到的结果）。包内组件都从 `useProcesses()` 读同一份状态，因此**组件之间不传 props、不发 emit**：

- `service.ts`：唯一接触生成代码的文件，导出调用契约 `ProcessServiceApi`、实现 `processService` 与前端数据契约（`ProcessInfo` / `ProcessSnapshot`）。换 Wails 绑定结构只改这里。
- `useProcesses.ts`：该功能的全部状态与动作——采集与刷新合并、图标代次与缓存、搜索与工作目录筛选、结束流程。`createProcessStore(api)` 可注入假实现。
- `rules.ts`：业务规则——图标缓存键、结束禁用原因、搜索匹配字段。
- `src/lib/format.ts`：与业务无关的格式化，可被其他功能复用。

## 测试

```sh
npm test           # vitest run：状态、规则、格式化与组件测试
npm run test:watch
npm run typecheck  # vue-tsc --noEmit
npm run build      # 类型检查 + vite build
```

约定：

- 用 `createProcessStore(假 ProcessServiceApi)` 直接测状态，用 `vi.mock('@/features/processes/service')` 让组件测试拿到假后端；两种情况都不需要 Wails 运行时。
- 包内共享实例在同一个测试文件内跨用例存在，因此每个用例先用 `beforeEach` 把它恢复到初始状态（清筛选、关弹窗）。
- 弹窗内容会 Teleport 到 `document.body`：断言用 `document.body`，并在 `afterEach` 里 `unmount()` 已挂载的实例，不要直接清空 `body`。
- 环境提示：若 `npm install` 报 `EPERM ... _cacache`，说明 npm 缓存目录不可写，修好权限（`sudo chown -R "$(id -u):$(id -g)" ~/.npm`）或换一个可写缓存目录后重试。

## 生成的绑定

`wailsjs/` 由 `wails build` / `wails dev` 生成，**不要手改**。Go 侧绑定面变化后运行 `wails build -s`（只生成绑定并编译 Go），并删除生成器不会自动清理的旧目录（例如从 `go/main/App` 迁到 `go/processes/Service` 后的 `go/main/`）。

## 加新功能

新增 `src/features/<功能>/`：一个对外无 props 的入口组件，加上该功能自己的 `service.ts` / `useProcesses.ts` / `rules.ts`；在 `src/main.ts` 挂载入口组件。通用原语只用 `src/components/ui/*`，业务规则留在功能包内，不要在功能之间互相 import。

当前没有 `App.vue`：单功能阶段它只是一个纯转发层。等出现第二个功能域、确实需要页面级组合时再引入。
