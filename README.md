# 题迹 Questrace 2.0

题迹是一款错题整理、图片录入、AI 分析与相似题复盘工具。2.0 将服务端和网页打包到电脑程序中：数据留在电脑，手机连接电脑使用。

仓库地址：[kukusuyi/Questrace](https://github.com/kukusuyi/Questrace)。本产品原名 Notebook（旧称），自本版本起改名为题迹 Questrace；旧版本的数据、设置与登录状态可以直接沿用，见下方“从 Notebook 升级”。

## 下载与使用

- **Windows**：下载并运行 `Questrace-版本-windows-x64-Setup.exe`，按向导选择安装目录，之后通过桌面或开始菜单快捷方式启动。关闭窗口或选择托盘菜单“退出并停止服务”都会退出程序并停止后台服务（最多等待约 11 秒）。在 Windows 设置的“应用”中卸载，或运行安装目录内的卸载程序；卸载时可选择是否保留用户数据，默认勾选保留；取消勾选并确认后删除当前用户 AppData 下的 Questrace 数据、程序缓存及可识别的旧版 Notebook 数据。自定义数据目录保留。
- **macOS**：解压对应芯片的 `.app` 并打开。关闭窗口进入菜单栏，退出会停止服务。
- **Linux**：解压对应架构的发行包，运行 `./start.sh`，浏览器打开输出的地址。
- **Android**：安装 APK，填写电脑显示的局域网地址（例如 `http://192.168.1.10:8080`），验证连接后登录。

无需安装或配置 MySQL、MinIO、Qdrant、Docker、Node.js 或 Go。电脑需保持运行，手机与电脑需网络互通；系统提示防火墙授权时允许局域网访问。不同平台需要下载各自的发行包。

如果早期 Windows 安装版卸载时报 `Installer integrity check has failed`，先退出程序，将修复包中的 `Uninstall Questrace.exe` 放入原安装目录，替换同名文件，再从 Windows“应用”或控制面板卸载。不要在下载目录直接运行独立卸载文件。卸载默认保留学习数据。新版构建会同时校验安装包及其内置卸载程序。

首次运行时创建管理员。桌面窗口自动带入初始化凭据；Linux 用户从启动输出复制 `setup_token`。公开注册默认关闭，管理员可在设置中添加用户或开启注册。

**没有模型密钥也可以手动录题、上传图片、管理标签、查询和打印。** 管理员在设置页添加 AI 模型服务后可使用分析；OCR 当前使用通义千问视觉模型；相似题需要配置兼容 OpenAI Embeddings 接口的模型。模型调用需要联网并由服务商计费。密钥保存在电脑的数据目录，不分发给手机。

## 数据与备份

默认数据目录：

| 平台 | 位置 |
|---|---|
| Windows | `%APPDATA%\Questrace` |
| macOS | `~/Library/Application Support/Questrace` |
| Linux | `$XDG_DATA_HOME/Questrace`，未设置时为 `~/.local/share/Questrace` |

目录包含 `questrace.db`、`files/`、`settings.json`、`questrace.log`。重装或替换程序不会覆盖数据。备份含模型密钥与登录配置，应作为私人文件保存。

可用环境变量指定数据目录：`QUESTRACE_DATA_DIR` 优先，`NOTEBOOK_DATA_DIR` 作为旧名称兼容别名保留；命令行 `--data-dir` 始终优先。

退出程序后执行（Windows 使用发行产物中的维护二进制 `questrace-server.exe`；macOS 可使用 `.app/Contents/Resources/backend/questrace-server`）：

```sh
./questrace-server --backup /path/outside-data/questrace-backup.zip
./questrace-server --restore /path/questrace-backup.zip
```

可加 `--data-dir /path/to/data` 指定目录。恢复先验证临时目录，成功后替换；恢复前数据保留在同级 `*.before-restore-*` 目录。数据目录加进程锁，不允许多个进程同时运行或在线恢复。首次 2.0 建库不导入 1.x 数据。

默认端口为 8080，被占用时自动选择空闲端口；`--port 8090` 显式指定时，冲突会报错。`--host 127.0.0.1` 可限制为本机使用。公网 HTTPS、自动更新和跨电脑同步不在 2.0 首版范围内。

## 从 Notebook 升级

更名不影响既有安装；程序会继续原地使用旧数据，不复制、不移动任何文件。

- **数据目录**：仅存在旧 `Notebook` 目录时继续使用该目录；仅存在新 `Questrace` 目录时使用新目录。两者同时存在时启动会报错并停止，避免静默选错数据，此时请保留一个，或用 `--data-dir` / `QUESTRACE_DATA_DIR` 指定。
- **数据库文件**：数据目录内只有旧 `notebook.db` 时继续原地使用；只有新 `questrace.db` 时使用新文件；两者同时存在时报错停止。
- **备份**：新备份统一使用 `questrace.db` 条目；恢复同时接受旧备份中的 `notebook.db`，并在恢复时规范化为新名称。
- **浏览器与移动端**：启动时把旧 `math-notebook:*`、`notebook:appearance:v1` 本机数据复制到对应的 `questrace:*` 键，旧键保留以便回退；登录 Cookie 写入 `questrace_session`，同时继续读取并在退出时清理旧 `notebook_session`。
- **安装标识**：Electron `appId`、Android `applicationId` 与 iOS Bundle Identifier 保持原值，确保系统仍把新版识别为同一应用、可以就地升级。这些标识是有意保留的兼容例外。
- **签名开关**：`QUESTRACE_REQUIRE_RELEASE_SIGNING` 为正式变量，`NOTEBOOK_REQUIRE_RELEASE_SIGNING` 继续作为兼容别名生效。

## 源码构建

开发者需要 Go 1.25+、Node.js 22+；构建 APK 另需 Flutter 3.44.0、JDK 17 与 Android SDK。正式流水线固定这些版本，并在 macOS runner 上执行 Flutter golden 测试，以保证图像基线和发行产物可重复。

```sh
node scripts/build.mjs
cd desktop
npm ci
npm run dist
```

`build.mjs` ：它先检查本机是否存在 Go 1.25+，再构建 Vue、嵌入网页并生成独立 Go 程序，最后复制到 Electron 的 resources。第一阶段输出位于 `dist/questrace-<版本>-<系统>-<架构>/`（例如 `dist/questrace-2.1.0-macos-arm64/`），其中的服务端程序名为 `questrace-server`；随后必须在 `desktop/` 执行 `npm run dist` 才会生成 Windows/macOS 桌面包。可用 `GOOS`/`GOARCH` 选择后端目标；桌面外壳应在对应系统打包。Linux 输出包括 `start.sh`。

若出现 `Go compiler not found`，请先安装 Go 1.25+ 并确认 `go version` 可执行。

移动端：

```sh
cd mobile/flutter_app
printf '{"flavor":"production","apiBaseUrl":""}\n' > config/app_config.json
flutter pub get
flutter build apk --release --dart-define=APP_FLAVOR=production
```

发行构建默认无服务器地址；开发可在设置页填写地址。本地未提供 `android/key.properties` 和 keystore 时仅生成测试包，密钥文件不要提交仓库。

iOS 只能在 macOS + Xcode 环境中构建。无需付费 Apple 开发者账号即可完成无签名 release 编译：

```sh
cd mobile/flutter_app
printf '{"flavor":"production","apiBaseUrl":""}\n' > config/app_config.json
flutter pub get
flutter build ios --release --no-codesign --dart-define=APP_FLAVOR=production
```

上述命令输出 `build/ios/iphoneos/Runner.app`，但无签名应用不能直接安装到 iPhone。模拟器运行无需开发者账号；使用个人 Apple ID 可以通过 Xcode 自动签名安装到自己的 iPhone，但配置描述文件通常只有短期有效期。生成可对外分发的 IPA 或上传 TestFlight/App Store 必须加入 Apple Developer Program。模拟器运行、证书选择、真机调试、归档导出和常见问题见 [iOS 编译说明](mobile/flutter_app/docs/IOS_SETUP.md)。

GitHub Actions 的 `release-v2.yml` 仅允许从 `master-v2.0` 手动预检，或由该分支提交上的 `v2.*` 标签正式发布。它构建 Windows x64、macOS arm64/x64、Linux amd64/arm64 和 Android。Windows 使用项目自签名 Authenticode 证书，macOS 使用 ad-hoc 签名且不进行 Apple 公证，Android 使用项目 release keystore。Windows 和 macOS 包不具备公开 CA/Apple 信任，首次启动会显示安全警告。

## 架构

```text
Electron 窗口 / 浏览器 / Flutter APK
                 │ HTTP /api/v1
                 ▼
       Go 单进程服务（含 Vue 网页与提示词）
          ├── SQLite：账户、题目、标签、向量与任务队列
          ├── 本地 files/：鉴权图片
          └── 可选外部 OCR / LLM / Embedding API
```

保留登录与用户数据隔离。向量生成是持久化后台任务：题目保存不等待模型；失败可重试；更换模型会排队重建。检索先过滤用户、科目和标签，再计算相似度，避免跨用户候选截断。

- `backend/`：本地后端、API、存储与任务（Go module `github.com/kukusuyi/Questrace/backend`）
- `frontend/`：桌面窗口和浏览器共享的 Vue 页面
- `desktop/`：Electron 生命周期、托盘和打包
- `mobile/flutter_app/`：Android/iOS 客户端源码（Dart 包 `questrace_flutter`）
- `design-system/questrace/`：主题与界面规范
- `scripts/`：发行构建入口
- `docs/API.md`：2.0 本地服务补充接口说明

1.x 的 MySQL、对象存储和独立服务器部署资料保留在 `master` 与 `dev-ios-adaptation` 分支。2.0 不提供 1.x 数据迁移工具。

## 检查

```sh
cd backend && go test ./...
cd frontend && npm ci && npm run build
cd mobile/flutter_app && flutter analyze && flutter test
```

开发服务端可运行 `go run ./cmd/api --data-dir /tmp/questrace-dev`，开发网页需先构建或单独运行 Vite。接口文档位于 `/docs`。

## 许可证

[CC BY-NC 4.0](LICENSE)：可分享和修改，需署名，禁止商业使用。
### 多学科分类与相似题

内置考研数学、408、高中数学／地理／生物／物理／化学，也可创建个人学科。408 按数据结构、计算机组成原理、操作系统、计算机网络分组，提供 26 个章节提示词；考研数学沿用原有章节库。其他学科使用通用分析。自动识别无法确定时保留“待分类”，可以手动选择。

错题列表支持学科、课程、章节和分类状态筛选；勾选题目后可批量复核分类。升级会自动备份数据库，旧题保留原内容并标记“历史分类待复核”。确认分类后生成新索引；分类改变时原标签、摘要保留供复核，但在复核前不参与标签匹配，向量改用题干／错解原文。

相似题始终按用户和稳定学科 ID 隔离，408 默认只查同一课程，也可选择“整个408”。未分类、历史待复核题不会参与召回；关闭标签过滤也不会关闭学科隔离。升级后的向量采用格式版本 2，旧格式不参与查询。

API 调用方通过 `GET /api/v1/subjects` 获取学科目录，使用 `subject_id`、`course_id` 和 `chapter` 传递分类；未传学科 ID 不再默认数学。自定义学科用 `POST /api/v1/subjects` 创建，批量复核使用 `POST /api/v1/wrong-questions/classification` 并携带列表返回的 `revision`。完整字段见内置 OpenAPI。旧客户端需要升级后才能确认分类和使用相似题检索。

### 学习阶段与开放学科

新注册账户及首次创建管理员时需要选择“大学”或“高中”。数据库升级会为已有账户设置“大学”，可在网页／桌面的「设置与账户 → 个人信息」或手机的「我的 → 个人信息」修改。阶段决定学科推荐与 AI 自动识别的上下文，不会改写已有错题的分类或向量。

学科目录不是封闭清单。大学阶段可添加材料科学基础、材料力学、机械原理、医学等任意专业科目；高中阶段也可添加其他课程。没有专用提示词的学科使用通用分析。AI 遇到尚未建立的学科会建议名称，用户确认添加后再归类，不会强行套用数学／408。编辑已有题目时保留其原分类，即使与当前阶段不同。

API：注册及初始化请求新增必填 `education_stage`（`university` / `highschool`）；`GET /api/v1/users/me` 返回该字段，`PUT /api/v1/users/me` 可更新当前账户阶段。学科目录返回当前阶段和每项的 `recommended` 标志；稳定学科 ID 仍用于保存和向量隔离。
