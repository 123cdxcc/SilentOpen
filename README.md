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

## 下载后无法打开

以下操作仅适用于从本仓库 Releases 下载、且你确认可信的文件。macOS 版本尚未经过 Apple 公证，Windows 版本尚未做代码签名，首次运行可能被系统安全机制拦截。

### macOS

如果提示「无法验证开发者」或「Apple 无法检查其是否包含恶意软件」：

1. 解压下载的 ZIP，将 `SilentOpen.app` 拖入「应用程序」文件夹。
2. 尝试打开一次，然后进入「系统设置 → 隐私与安全性」。
3. 在安全性区域找到 SilentOpen 的拦截提示，点击「仍要打开」，按提示确认并输入密码或使用 Touch ID。

如果提示「应用已损坏，无法打开」，先重新下载并解压；可将下载文件的 SHA-256 与同一 Release 的 `checksums.txt` 核对。确认文件完整且来源可信后，若仍被拦截，可在「终端」中移除该应用的下载隔离属性，再尝试打开：

```sh
xattr -dr com.apple.quarantine /Applications/SilentOpen.app
```

如果应用放在其他目录，请将命令中的路径替换为实际路径；路径含空格时用双引号包裹。此命令仅处理指定应用，无需关闭系统的全局安全检查。

### Windows

如果 SmartScreen 提示「Windows 已保护你的电脑」，点击「更多信息 → 仍要运行」。

如果文件属性中显示来自其他计算机的安全限制，可右键下载的 `.exe`，打开「属性 → 常规」，勾选「解除锁定」（如有），点击「应用」后重新运行。若提示缺少 WebView2 Runtime，请先安装 [Microsoft Edge WebView2 Runtime](https://developer.microsoft.com/microsoft-edge/webview2/)。

### Linux

如果解压后提示「权限不足」或 `Permission denied`，在可执行文件所在目录打开终端，添加执行权限后运行：

```sh
chmod +x ./SilentOpen
./SilentOpen
```

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
