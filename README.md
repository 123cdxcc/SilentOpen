<img src="build/appicon.png" alt="SilentOpen" width="120" />

# SilentOpen

> 打开就能看到谁占着端口、从哪起来的，顺手清掉。

SilentOpen 是一个跨平台的桌面应用，列出本机正在监听端口的进程，展示详情，并可在确认后结束单个进程。支持 macOS、Windows 与 Linux。

## 背景

用 agent 写代码时，它拉起的后台进程常常不会自己退出，下次开工端口还被占着。它很少主动说这件事，而是换个端口继续跑，问题就这么被绕过去。

SilentOpen 把这部分信息摆到台面上：谁在监听、从哪个目录启动、要不要关掉。

## 功能

- 打开即显示正在监听端口的进程
- 列表展示 PID、进程、项目、端口与启动时间，展开可查看完整启动命令、工作目录和父进程
- 支持按进程搜索、按工作目录筛选
- 打开和切回窗口时自动刷新，也可手动刷新
- 能取到所属应用图标时，在进程名旁显示
- 确认后可结束单个进程；操作前会重新核对目标，系统进程和自身进程不会被动

## 技术栈

| 层 | 选型 |
| --- | --- |
| 桌面框架 | [Wails](https://wails.io) v2 |
| 后端 | Go |
| 前端 | Vue 3 · TypeScript · [shadcn-vue](https://www.shadcn-vue.com) · Tailwind CSS |

## 说明

- 「项目」取进程工作目录的末级目录名，仅用于辅助判断，不能证明进程由谁启动。
- 权限受限或进程已退出时，部分信息可能显示「未知」。
- 只列出监听端口的 TCP 服务，不含其他网络连接。

## 开发

开发、构建与仓库约定见 [AGENTS.md](AGENTS.md)。

## 打包与发布

[Build and Release](.github/workflows/release.yml) 使用 GitHub Actions 分别构建三个平台：

| 平台 | Release 附件 |
| --- | --- |
| macOS（Intel / Apple Silicon） | `SilentOpen-1.2.3-macos-universal.zip`，解压后得到 `.app` |
| Windows x64 | `SilentOpen-1.2.3-windows-amd64.exe` |
| Linux x64 | `SilentOpen-1.2.3-linux-amd64.tar.gz`，解压后运行 `SilentOpen` |

在 GitHub 的 Actions 页面手动运行 **Build and Release**，可从 Artifacts 下载三个平台的开发包；手动运行不会发布 Release，版本为 `dev`，不提示更新。

正式发布时，将工作流提交到 GitHub 仓库后，给需要发布的提交打标签并推送：

```sh
git tag v1.2.3
git push origin v1.2.3
```

标签格式为 `v主版本.次版本.修订号`，也支持 `v1.2.3-rc.1` 等预发布标签。三平台全部构建成功后，工作流上传附件和 `checksums.txt`，自动生成发布说明并发布 Release。带预发布后缀的标签会标记为 Pre-release，不会成为正式更新。

构建会注入标签版本及当前 GitHub 仓库地址，附件命名与现有更新器保持一致。应用通过该仓库的最新正式 Release 检查更新，展示发布说明并打开 Release 页面供用户下载；目前不自动替换安装文件。无需额外配置 Token，发布任务使用自带的 `GITHUB_TOKEN` 和 `contents: write` 权限；仓库或组织策略需允许此权限。更新源仓库需要公开，客户端不携带 GitHub 凭据。

每个新版本使用新标签；工作流不会覆盖已发布的同名 Release。如果上传中断并留下草稿，可删除该草稿后重新运行失败任务。

平台要求：Linux 包基于 Ubuntu 24.04 构建，需要 GTK 3 和 WebKitGTK 4.1（Ubuntu 可安装 `libgtk-3-0t64 libwebkit2gtk-4.1-0`）；Windows 需要 WebView2 Runtime。macOS 包仅做本地签名，未使用 Apple 开发者证书或公证；Windows 包也未做代码签名，首次打开可能需要在系统安全提示中允许运行。

## 开源协议

SilentOpen 采用 GNU General Public License v3.0（`GPL-3.0-only`）许可，完整条款见 [LICENSE](LICENSE)。第三方依赖及资源仍遵循各自的许可协议。
