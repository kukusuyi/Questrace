## 2.1.2 — 2026-10-05

- 新增多学科识别、408 课程与章节分析，以及大学／高中学习阶段设置。
- 修复 Windows 局域网发现的主机名响应兼容性，优先使用可达的物理网卡，避免虚拟网卡地址干扰手机连接。
- 手机优先使用局域网发现返回的 IPv4 地址连接电脑，减少重复解析 .local 域名造成的失败。
- 主机名响应不可用时保留电脑发现服务；Windows 安装时尝试登记专用网络和域网络的 mDNS 防火墙规则。
- Windows 2.x 安装版支持直接升级并保留用户数据；Notebook 1.x 数据不自动迁移。

## 2.1.1

- 模型列表移至设置中加载，分析页面仅选择已配置模型；新增 32B 建议与高校学生优惠入口。
- 新 Qwen 配置默认使用 qwen3.8-flash，保留已保存模型。
- Windows/macOS 提供独立 .local 地址，手机可发现电脑并在换网后重新连接。
- 按公网 IP 选择 GitCode/GitHub 更新源，支持失败回退与双站校验发布。

# 更新日志 / Changelog

## 2.1.0 — 2026-09-26

### 中文

#### 新增

- 复习计划：自动组卷、自评、续练与历史，掌握状态随到期复习逐步变化。
- 各端增加 GitHub 正式版本更新提醒，可关闭自动检查并手动检查。
- 手机外观新增松林绿、樱雾粉、落日橙、星夜靛、石墨灰五种配色与“自定义”选项：调色盘提供 16 个常用色、拖动选色和整套浅深外观的实时预览。
- 错题支持按标签 ID 多选筛选，标签管理可直达对应题目。

#### 变更

- 正式展示名由 Notebook（旧称）更改为**题迹 Questrace**，仓库迁移至 [kukusuyi/Questrace](https://github.com/kukusuyi/Questrace)。
- Go module 改为 `github.com/kukusuyi/Questrace/backend`；npm 包改为 `questrace-frontend`、`questrace-desktop`；Flutter 包改为 `questrace_flutter`。
- 服务端程序与发行产物改为 `questrace-server`、`Questrace-<版本>-<平台>`；GitHub Actions 工作流名称、并发组、临时产物与 Release 标题同步改名。
- Android 源码 namespace 改为 `com.kukusuyi.questrace.mobile`，设计系统目录改为 `design-system/questrace/`。
- 手机端强调色改为跟随外观色，不再单独设置；旧的自选强调色自动迁移为“自定义”外观。
- 手机设置分离服务与索引状态；背景图片上限提升至 10 MiB，并迁移至专用本地存储。
- 移除手机首页快捷入口及应用内图形 Logo，保留文字名称与系统应用图标。
- 公式搜索兼容空格和常见 LaTeX 排版差异。
- Windows 发行包改为带向导的 `Setup.exe` 安装程序，提供快捷方式、控制面板卸载入口和独立卸载程序。

#### 修复

- 手机读取请求遇到短暂断线或超时时最多重试一次；缩短空闲连接保留时间，避免无关设置重建网络客户端。连接地址检测同步支持有限重试。
- 修复手机连接验证锁定旧版本的问题。
- 复习组卷表单的上下输入框之间留出间距，不再贴在一起。
- 关闭 Windows 窗口时退出应用并停止后台服务，增加退出超时清理并避免启动过程中退出后重新打开窗口。
- 卸载时可选择是否保留用户数据，默认保留；明确取消勾选并确认后清理当前用户 AppData 中本应用的数据与缓存，自定义数据目录保留。
- 补齐桌面启动所需的 `branding.cjs` 和 `data-dir.cjs`，修复 macOS 交叉构建的卸载程序校验值，新增启动、退出和安装包完整性检查。
- 修复发布流水线校验 Windows 安装包内置卸载程序时的误报：Windows 构建主机上 7-Zip 只解包应用自身的可执行文件，不再因此中断发布；改为按 NSIS 头标记识别卸载器条目，并继续校验安装包与卸载器的 CRC。
- SQLite 增量升级前自动备份，保留现有题目、掌握状态和账户。

#### 兼容

- 新安装默认使用 `Questrace` 数据目录、`questrace.db`、`questrace.log` 与 `questrace-backup.zip`。
- 升级时若只存在旧 `Notebook` 数据目录或 `notebook.db`，继续原地使用，不复制或移动用户数据；新旧目录或数据库同时存在时明确报错停止。
- 新增 `QUESTRACE_DATA_DIR`，`NOTEBOOK_DATA_DIR` 作为低优先级兼容别名保留；`--data-dir` 始终优先。
- 备份统一写出 `questrace.db`；恢复同时接受旧备份中的 `notebook.db` 并规范化为新名称。
- 浏览器与 Flutter 启动时把旧 `math-notebook:*`、`notebook:appearance:v1` 数据复制到 `questrace:*` 键，旧键保留以支持回退。
- 登录 Cookie 写入 `questrace_session`，继续读取并在退出时清理旧 `notebook_session`。
- Electron `appId`、Android `applicationId` 与 iOS Bundle Identifier 保持原值，确保系统仍识别为同一应用；`NOTEBOOK_REQUIRE_RELEASE_SIGNING` 保留为兼容别名。
- HTTP API 路径与数据结构不变。

#### 升级说明

- 从 2.0.0 覆盖安装即可升级：数据目录、数据库与账户保留，SQLite 会在升级前自动备份。
- 手机外观偏好沿用 `questrace:appearance:v1` 键，旧的自选强调色迁移为“自定义”外观（偏好写入版本 2）。

### English

#### Added

- Review planner with automatic sessions, self-assessment, continuing a session and history; mastery states follow due reviews.
- Official GitHub release update notices on every client, with automatic checks that can be turned off and a manual check.
- Five more phone appearance presets (pine, rose, amber, indigo, graphite) plus a custom option with a palette picker: 16 common colours, drag selection and a live preview of the whole light and dark appearance.
- Wrong questions can be filtered by multiple tag IDs, and tag management links straight to the matching questions.

#### Changed

- The product is now shown as **题迹 Questrace**; the former name Notebook is kept only in historical and compatibility notes. The repository moved to [kukusuyi/Questrace](https://github.com/kukusuyi/Questrace).
- The Go module is `github.com/kukusuyi/Questrace/backend`; the npm packages are `questrace-frontend` and `questrace-desktop`; the Flutter package is `questrace_flutter`.
- The server binary and release assets are now `questrace-server` and `Questrace-<version>-<platform>`; workflow names, concurrency groups, temporary artifacts and release titles follow.
- The Android source namespace is `com.kukusuyi.questrace.mobile` and the design system moved to `design-system/questrace/`.
- The phone accent colour now follows the selected appearance colour instead of being configured separately; a former custom accent is migrated to the custom appearance.
- Phone settings separate service health from index status; the background image limit rises to 10 MiB and moves to dedicated local storage.
- The phone home shortcuts and the in-app graphic logo are gone; the wordmark and the system application icon remain.
- Formula search tolerates extra spaces and common LaTeX formatting differences.
- Replace the portable Windows package with a `Setup.exe` wizard, shortcuts, a Control Panel entry and a standalone uninstaller.

#### Fixed

- Phone reads that hit a brief disconnect or timeout are retried once; idle sockets are released sooner so unrelated settings changes no longer rebuild the network client, and connection checks retry the same way.
- Fixed connection validation keeping an outdated version locked in.
- The review setup form now leaves a gap between stacked input fields instead of butting them together.
- Closing the Windows window stops the application and its backend, with a shutdown deadline and protection against reopening during startup cancellation.
- Offer a keep-data choice during uninstall, selected by default. Explicit opt-out and confirmation remove this application's current-user AppData and cache; custom data directories are preserved.
- Bundle the missing `branding.cjs` and `data-dir.cjs` modules, correct uninstaller checksums in macOS cross-builds, and add startup, shutdown and package integrity checks.
- Fixed a false failure in the release check for the Windows installer's bundled uninstaller: 7-Zip only unpacks the application executables on Windows build hosts, so the check now recognises uninstaller entries by their NSIS header instead of failing, and keeps verifying the CRC of both the installer and any uninstaller it finds.
- SQLite backs up automatically before an incremental upgrade, keeping existing questions, mastery states and accounts.

#### Compatibility

- Fresh installations use the `Questrace` data directory, `questrace.db`, `questrace.log` and `questrace-backup.zip`.
- An installation that only has the former `Notebook` directory or `notebook.db` keeps using it in place; when both exist, startup stops instead of guessing.
- `QUESTRACE_DATA_DIR` is the current environment variable with `NOTEBOOK_DATA_DIR` as a lower-priority alias; `--data-dir` always wins.
- Backups always store `questrace.db`; restore still accepts `notebook.db` from older archives and normalises the name.
- Browsers and the Flutter app copy former `math-notebook:*` and `notebook:appearance:v1` values to the `questrace:*` keys on startup and keep the old keys for rollback.
- The session cookie is `questrace_session`; the former `notebook_session` is still read and cleared on logout.
- The Electron `appId`, Android `applicationId` and iOS bundle identifier keep their former values so the system still recognises an update as the same application; `NOTEBOOK_REQUIRE_RELEASE_SIGNING` stays accepted.
- HTTP API paths and payloads are unchanged.

#### Upgrade notes

- Installing 2.1.0 over 2.0.0 keeps the data directory, database and accounts; SQLite backs up before migrating.
- Phone appearance preferences stay under `questrace:appearance:v1`; a former custom accent becomes the custom appearance (preferences are written as version 2).

## 2.0.0 — 2026-09-23

> 该版本发布时产品名为 Notebook（旧称）；自本版本之后的版本更名为「题迹 Questrace」。下方条目保留发布时的原始名称。
>
> This version was published as Notebook, the former product name; later versions are named 题迹 Questrace. The entries below keep the name used at release time.

### 中文

#### 新增

- 本地优先的桌面运行方式，将 Vue 界面、Go 服务、SQLite 数据库和本地文件存储打包在一起。
- Windows x64、macOS Intel 与 Apple Silicon、Linux amd64 与 arm64，以及 Android 发行包。
- 首次启动管理员设置、本地用户管理、备份恢复、外观预设、草稿恢复和双题 A4 导出。
- 由本地管理员配置的可选 OCR、LLM 和 Embedding 服务。

#### 变更

- Notebook 2.0 不再需要 MySQL、MinIO 和 Qdrant。
- 图片、模型设置、题目和向量均保存在 Notebook 本地数据目录。
- 移动客户端连接同一可达网络中的 Notebook 2.0 桌面端或 Linux 服务。

#### 升级说明

- Notebook 2.0 不会自动导入 Notebook 1.x 的数据库或对象存储文件。现有 1.x 安装请继续使用 `master` 或 `dev-ios-adaptation` 分支。
- 替换安装前请备份完整的 Notebook 数据目录；备份包含私人数据和模型凭据。
- 仓库保留 iOS 源码和编译说明，但 v2.0.0 不发布 IPA 或 TestFlight 构建。
- Windows 包未附带 Authenticode 签名，macOS 包使用未经过 Apple 公证的 ad-hoc 签名；用户首次启动时需要明确允许这些程序运行。项目没有公开可信的 CA 证书，可用签名材料只有项目自签名证书，因此无法提供受信任的发布者签名。

### English

#### Added

- Local-first desktop runtime with the Vue application, Go service, SQLite database and local file storage bundled together.
- Windows x64, macOS Intel and Apple Silicon, Linux amd64 and arm64, and Android release packages.
- First-run administrator setup, local user management, backups and restore, appearance presets, draft recovery, and two-question A4 export.
- Optional OCR, LLM and embedding providers configured by the local administrator.

#### Changed

- MySQL, MinIO and Qdrant are no longer required for Notebook 2.0.
- Images, model settings, questions and vectors are stored in the local Notebook data directory.
- Mobile clients connect to a Notebook 2.0 desktop or Linux service on the same reachable network.

#### Upgrade notes

- Notebook 2.0 does not automatically import Notebook 1.x databases or object-storage files. Continue using the `master` or `dev-ios-adaptation` branch for existing 1.x installations.
- Back up the entire Notebook data directory before replacing an installation. The backup contains private data and model credentials.
- iOS source code remains in the repository, but v2.0.0 does not publish an IPA or TestFlight build.
- Windows packages ship without an Authenticode signature, and macOS packages use ad-hoc signing without Apple notarization. Users must explicitly trust or allow these packages on first launch. The project has no publicly trusted CA certificate and its only signing material is a self-signed certificate, so no trusted-publisher signature is available.
