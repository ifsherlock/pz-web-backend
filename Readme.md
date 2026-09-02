# Project Zomboid Web Configurator (PZ-Web)

一个为 [Project Zomboid Dedicated Server](https://store.steampowered.com/app/380870/Project_Zomboid_Dedicated_Server/) 设计的 Web 可视化配置管理工具。

该项目作为 [pz-docker-server](https://github.com/ifsherlock/pz-docker-server) 容器中的 Sidecar 服务运行，提供 Web 界面来管理 `servertest.ini` 和 `SandboxVars.lua`，集成模组管理、多语言支持、服务器控制和备份功能。

模板、JavaScript、CSS 和官方 favicon 都通过 Go `embed` 编译进单个面板二进制；部署时只需替换 `pz-web-backend` 并重启面板进程。

---

## ✨ 核心功能

*   **可视化配置编辑**：
    *   自动解析 `Server.ini` 和 `SandboxVars.lua`。
    *   **I18n 支持**：直接读取游戏原生翻译文件，自动显示配置项的中文/英文名称和 Tooltip；缺失翻译时保留可读的配置键名。
    *   **智能分类**：自动将几百个配置项归类（如“僵尸特性”、“物资稀有度”）。
*   **表单控件**：自动识别下拉选项（Select）和文本输入（Input）。

*   **游戏版本管理**：
    *   在“服务器配置”中选择 `public`、`42.19`、`legacy41` 或自定义 Steam 分支。
    *   `public` 当前稳定版为 **42.20.4**；服务端每次启动时由 SteamCMD 自动检查并拉取该分支最新内容。
    *   仪表盘显示服务端实际启动版本号；切换分支后点击“保存并重启”即可生效。
    *   自定义输入接受 Steam 分支名，不接受不存在的任意版本号；精确锁定历史构建需要 Steam depot manifest ID。

*   **服务端安全配置**：
    *   `Password`（玩家入服密码）、管理员账户和管理员密码集中在“服务端安全”分组末尾。
    *   管理员密码仅返回掩码，保存后点击“保存并重启”才会同步到游戏数据库和启动参数。
    *   `RCONPort` / `RCONPassword` 提供远程控制台（Remote Console）接入；不使用 RCON 时保持密码为空。

*   **模组管理器**：
    *   **创意工坊集成**：支持直接输入 Workshop ID，自动从 Steam API 获取模组名称。
    *   **智能解析**：自动处理 Workshop ID 与 Mod ID 的对应关系。
    *   **一键应用**：自动生成分号分隔的配置字符串并去重。

*   **服务器监控与控制**：
    *   实时查看 Supervisor 控制台日志。
    *   仪表盘显示 CPU、内存、运行时长和当前服务端版本。
    *   提供“重启”和“更新并重启”功能（自动触发 SteamCMD 更新）。
    *   提供面板自重启功能，方便build调试

*   **备份与恢复**：
    *   “监控维护”页面支持手动备份、按小时配置定时备份、备注、保留数量上限、列表管理、删除和恢复。
    *   备份包含 `Server`、`Saves`、`db`、`Workshop`、`panel_settings.json` 等运行数据，不包含 `/opt/pzserver` 游戏程序。
    *   恢复时会先停止游戏、覆盖数据、修正 `steam` 属主并重新启动，降低存档损坏和权限错误风险。

*   **配置解释**：
    *   沙盒参数优先使用 Build 42 官方翻译和枚举定义；缺失项补充中文说明、数值含义和单位。
    *   悬浮解释会把翻译中的 `\\n`、`<br>` 转成真实换行，服务器和沙盒页面行为一致。

*   **界面资源**：
    *   服务端配置、沙盒设置和监控维护按用途分组，密码项固定在安全分组末尾。
    *   `template/assets/pz-official-logo.png` 使用 [PZ Wiki 的 SpiffoGlobe 素材](https://pzwiki.net/w/images/b/b5/SpiffoGlobe.png)，经高质量缩放为 64×64 RGBA，并作为浏览器 favicon 提供。

*   **轻量**：
    *   基于 Go (Gin) 编写；前端使用 Alpine.js + Tailwind CSS，无 Node.js 依赖。
    *   Go `embed` 内嵌模板和静态资源，Linux amd64 构建产物可直接复制部署。

---

## ⚙️ 运行时配置

面板配置分为两类：

* **游戏原生配置**：保存到 `servertest.ini` 或 `SandboxVars.lua`，包括服务端基础、网络连接、游戏规则、世界环境、车辆、模组与工坊等分组。
* **面板虚拟配置**：保存到挂载目录的 `panel_settings.json`，包括 JVM 内存上限、Steam 游戏分支、管理员账户/密码和备份策略，不会误写入游戏 ini 文件。

版本切换规则：

1. `public` 当前为 42.20.4，并自动跟随 Steam 稳定分支的后续更新。
2. `42.19`、`legacy41` 和自定义项填写的是 Steam 分支名，不是任意版本号。
3. 保存后点击“保存并重启”；服务端启动时会执行 SteamCMD `app_update 380870 -beta <分支> validate`。
4. 面板已保存的分支优先于容器环境变量 `PZ_BRANCH`。要回到环境变量控制，清空或删除 `panel_settings.json` 中的 `game_branch`。

安全配置说明：管理员密码只以掩码返回，面板设置文件权限为 `0600`；启动时通过游戏原生 `-adminpassword` 参数传入。请勿把 `panel_settings.json`、日志或包含密码的 `.env` 提交到仓库。

备份位于数据目录的 `backups/panel`，默认每 24 小时检查一次，保留数量可设置为 1-100。备份包含 `Server`、`Saves`、`db`、`Workshop`、`options.ini`、`latestSave.ini` 和面板设置，不包含 `/opt/pzserver` 游戏程序；恢复前会停止游戏并在恢复后修正属主、重新启动。

---

## 🛠️ 开发环境搭建

可在 Windows、Linux 或 macOS 上开发；Docker 服务端生产部署建议使用原生 Linux，避免 WSL2 的 UDP 网络转发问题。

### 准备环境
确保已安装 Go 1.23+（`go.mod` 以 Go 1.23 为基线）。

```bash
# 配置 Go 国内代理 (如果你在中国)
go env -w GO111MODULE=on
go env -w GOPROXY=https://goproxy.cn,direct

# 开启开发模式 (使用 mock 路径验证文件改动)
# PowerShell: $env:DEV_MODE="true"
export DEV_MODE=true

# 启动服务
go run .
```

---

## 🧱 目录结构（DDD）

后端按模块拆分为 DDD 风格结构（核心逻辑进入 `internal/`，Web 层保持薄）：  

- `internal/application/*`：用例层（Config / I18n / Mods / Update），协调领域逻辑与基础设施
- `internal/config`：`servertest.ini` / `SandboxVars.lua` 的解析与生成（含分组推断、Lua 值格式化）
- `internal/i18n`：读取游戏翻译文件（`lua/shared/Translate`）并提供翻译查询（含资源表）
- `internal/mods`：本地 Workshop 扫描 + Steam Workshop 元信息抓取（含文件缓存）
- `internal/system/update`：GitHub Release 更新检查（checker）
- `internal/infra/*`：副作用与系统依赖（路径推断 / 进程与文件操作等）
- `internal/legacy`：历史兼容入口（Deprecated，仅为重构期间过渡保留）
- `template/`：前端模板与静态资源（见下一节）

---

## 🧩 Template 结构（模块化）

`index.html` 仅保留入口，主体按模块拆分：

- `template/index.html`：入口（引入脚本、挂载 Alpine `x-data="app()"`）
- `template/app_body.html`：页面组装（仅引用各模块模板）
- `template/partials/*`：通用 UI 片段（导航栏 / Tabs / Toast / Mobile Dock）
- `template/modules/*`：业务模块 UI（Server / Sandbox / Monitor / Mod Modal）
- `template/assets/app/main.js`：前端主逻辑（从原内联脚本迁移）
- `template/assets/pz-official-logo.png`：官方 Project Zomboid favicon

说明：
- 模板加载使用 `template_loader.go` 遍历 `template/` 下所有 `.html`，避免 `ParseFS` 不支持 `**` 递归通配符的问题。
- `main.go` 使用 `//go:embed template`，因此生产环境不需要单独复制 `template/` 目录。

---

## 📦 构建与部署

本地开发构建：

```bash
go build -o pz-web-backend .
```

在 Windows 上为 Docker 容器构建 Linux amd64 二进制：

```powershell
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build -o pz-web-backend-linux-amd64 .
```

将二进制复制到容器挂载目录 `data/web-backend/pz-web-backend`，然后重启 `webconfig`。面板监听容器内 `:10888`，由 Nginx 负责 Basic Auth、HTTP/HTTPS 和 `/filebrowser/` 反向代理。

---

## 🧪 单元测试

```bash
go test ./...
```

---

## 🧷 DEV_MODE / Mock 数据

开发模式会优先使用 `testdata/`（Go 约定的测试夹具目录）：

- `testdata/mock_zomboid`：最小化的配置文件夹具
- `testdata/mock_media`：最小化的翻译文件夹具（仅覆盖测试/演示所需的少量 Key）

如果你希望在开发模式下看到完整翻译与资源（体积较大），使用：

- `testdata/mock_zomboid_full`
- `testdata/mock_media_full`

本地 Workshop 扫描（`/api/mods`）在开发模式默认读取：

- `testdata/pzserver/steamapps/...`（结构对齐 `/opt/pzserver/steamapps/...`）
