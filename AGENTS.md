# 仓库指南

## 项目结构与模块组织

SilentOpen 是基于 Wails v2、Go、Vue 3 和 TypeScript 构建的跨平台桌面应用。

- `main.go`、`app.go` 和 `app_update.go`：桌面应用启动逻辑及供前端调用的 Go 方法。
- `pkg/process/`、`pkg/icon/`、`pkg/icondata/`、`pkg/update/`：进程检查、图标处理和更新逻辑；Go 测试与实现文件放在同一目录。
- `frontend/src/features/`：进程与更新功能。共享 UI 基础组件位于 `components/ui/`，辅助函数位于 `lib/`，图片和字体位于 `assets/`。
- `frontend/wailsjs/`：自动生成的绑定代码，请勿手动编辑。
- `build/`：应用图标和各平台打包配置；二进制产物输出到 `build/bin/`。

## 构建、测试与开发命令

使用 Go 1.23+、兼容 Vite 8 的 Node.js/npm，以及 Wails v2 CLI，并安装对应平台所需的依赖。

在仓库根目录执行：

- `wails dev`：启动桌面应用及前端开发工具。
- `wails build`：构建前端并打包桌面应用。
- `go test ./pkg/...`：运行后端包测试。
- `go test ./...`：运行包含应用层测试在内的全部 Go 测试；由于 `main.go` 会嵌入 `frontend/dist`，需先构建前端。

在 `frontend/` 目录执行：

- `npm install`：安装依赖。
- `npm run dev`：启动 Vite；桌面集成功能需要通过 Wails 运行。
- `npm test`：运行一次 Vitest；`npm run test:watch` 会监听文件变化并重新测试。
- `npm run typecheck`：检查 Vue/TypeScript 类型。
- `npm run build`：执行类型检查并生成 `dist/`。

## 代码风格与命名约定

Go 代码使用 `gofmt` 格式化，导出名称遵循首字母大写的标准约定，测试文件使用 `*_test.go` 命名。前端沿用现有风格：两空格缩进、TypeScript 字符串使用单引号、不加分号。Vue 组件使用 PascalCase 命名，组合式函数文件使用 `useFeature.ts` 命名。Wails 调用集中放在各功能的 `service.ts` 文件中。复用现有 UI 基础组件。目前未配置专用的代码检查或格式化脚本。

## 测试指南

后端使用 Go 的 `testing` 包；前端使用 Vitest，配合 Vue Test Utils 和 jsdom。前端测试文件使用 `__tests__/*.spec.ts` 命名。通过模拟服务接口隔离测试，避免依赖桌面运行环境。为行为变更补充回归测试，优先执行与改动直接相关的检查。目前未设置测试覆盖率门槛。

## 安全与工作范围

保留结束进程前的确认、PID 与启动时间身份校验，以及对系统进程和应用自身进程的保护。围绕用户最新明确提出的目标开展修改，避免无关清理，不要将自动生成的构建产物纳入变更。
