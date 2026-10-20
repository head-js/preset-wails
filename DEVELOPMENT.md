# Development

## 约束

- 目标平台：macOS（`.app`）、Windows（`.exe`）。不含 Linux。
- 当前仅提供开发模式；生产构建、资源嵌入、签名与打包另行设计。
- 前端经 Vite dev server 提供（根目录 `src/`），dev 模式不使用 `go:embed`。
- `wails3` CLI 通过 `src-wails/go.mod` 的 `tool` 指令锁定版本，禁止全局安装；Taskfile 与 dev 配置一律以 `go tool wails3` 调用。
- 构建命令只允许项目本地运行，不修改系统状态。

## 开发

```sh
pnpm install   # 根目录，安装 Vite 与 React 依赖
pnpm dev       # 统一入口：scripts/dev.mjs → src-wails 下 go tool wails3 dev -port 5173
```

## 前端

- 技术栈：React 18 + Vite 6，JSX 由 `@vitejs/plugin-react` 处理（含 react-refresh 热更新）；无路由、无 TypeScript、无 UI 库。
- 入口：`src/index.html`（`#root` 容器）→ `src/main.jsx`（`ReactDOM.createRoot` + `StrictMode`）→ `src/App.jsx`（计数按钮 + `fetch("http://127.0.0.1:20080/api/version")` 展示版本号）。
- 所有依赖（`react`、`react-dom`、`@vitejs/plugin-react`、`vite`）版本号写死，不使用范围符。

## 调试 WebView（macOS）

- **正确方式：Safari 开发菜单远程检查**。Safari → 设置 → 高级 → 勾选“在菜单栏中显示‘开发’菜单”→ 菜单栏“开发”→ 你的 Mac → 选择应用内页面（`wails://localhost/`）即可附加 Web Inspector。检查的是 Wails 应用内的 WKWebView（页面运行在 Wails 进程、请求走 `wails://` scheme 经 AssetServer），Safari 仅作为调试 UI。
- dev 构建（`DEV=true`，无 `production` tag）下 Wails 已自动对 WebView 设置 `inspectable = YES`，无需额外配置；production 构建被编译剔除，除非加 `devtools` tag。
- **菜单栏 View → Open Developer Tools 在 macOS 14 上不可用**：Wails v3（beta.32）该项走的是 WebKit 私有 `_WKInspector` API，对第三方应用静默失效（无异常、无窗口）。同理窗口右键菜单的“检查元素”也可能缺失。不要以这两条路径判断 devtools 是否可用。

## 关键约定

- 根目录是前端工作区（`package.json`、`vite.config.js`、`src/`）；`src-wails/` 是 Go 模块与 Wails 项目，Wails 命令一律以 `src-wails/` 为工作目录。
- `wails3 dev -port 5173` 显式指定端口（beta.32 默认实为 9245），同时导出 `WAILS_VITE_PORT` 与 `FRONTEND_DEVSERVER_URL=http://localhost:5173`；`vite.config.js` 优先读取 `WAILS_VITE_PORT`，独立运行时回退 5173。
- Vite 绑定 `127.0.0.1` 而非 `localhost`：macOS 上 `localhost` 会让 Vite 只监听 IPv6（`::1`），而 beta.32 的 dev 反向代理对 localhost 强制走 IPv4（tcp4），导致 `connection refused`。
- dev 执行序列见 `src-wails/build/config.yml`：blocking 编译 Go dev 版本 → background 启动 Vite（readiness 检查 `http://localhost:5173/`，60s 超时）→ primary 运行应用，退出即整体关停。后台命令改为 `go tool wails3` 后不再匹配 Wails 的默认命令识别，readiness 必须显式配置。
- 前端 `src/` 在 Go watcher 监控范围（`src-wails/`）之外，由 Vite 负责热更新，不触发 Go 重编译。
- dev 模式下 assetserver 检测 `FRONTEND_DEVSERVER_URL` 并反向代理至 Vite，`main.go` 中的空 `fs.FS` 仅作占位。
- 业务 API 是独立的 Go HTTP 服务，与 Wails 应用同进程、在 goroutine 中运行（`src-wails/server.go`）。固定监听 `0.0.0.0:20080`（tcp4，所有 IPv4 网卡），不提供端口配置，不自动换端口；`AssetOptions.Handler` 只做静态资源/开发代理，不再分流 `/api/*`。
- 前端固定请求 `http://127.0.0.1:20080`（`src/App.jsx`），本机其他客户端同用该地址，局域网设备用主机实际 IPv4 加 20080 端口。CORS 为 `Access-Control-Allow-Origin: *`，不带凭据；OPTIONS 预检只允许 `GET, HEAD, OPTIONS`，拒绝自定义请求头。
- 端口冲突即启动失败：第二实例或任何占用 20080 的程序会让应用报错退出，不识别占用者、不做单实例唤醒。排查用 `lsof -iTCP:20080 -sTCP:LISTEN`。
- 退出统一走 `application.Options.OnShutdown` 停止 HTTP 服务（`Shutdown` 5 秒超时后强制 `Close`），托盘退出、系统退出、Ctrl+C 均释放端口；关闭窗口只是隐藏，不停止服务。强制 kill（SIGKILL）由操作系统回收端口。
- 单元测试见 `src-wails/api_test.go` 与 `src-wails/server_test.go`：在 `src-wails/` 执行 `go test ./...`，生命周期并发检查用 `go test -race ./...`。
- `GOOS`/`ARCH` 必须作为 task 变量（任务名之后）传入；设为环境变量会把 `go tool wails3` 自身交叉编译。
- `src-wails/bin/` 是产物目录（gitignore）；`src-wails/build/` 是 Wails 构建配置目录（必须提交）；`.task/`、`*.syso`、`node_modules/` 已忽略。
- macOS 上 `package` 任务负责组装 `.app` bundle（`go build` 只产裸二进制），不含签名步骤；打包流程尚未适配新目录布局，不作为可用入口。
