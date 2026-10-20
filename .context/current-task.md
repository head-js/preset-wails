# preset-wails 服务架构调整实施方案

## 目标与已确认决策

- Wails 桌面与 Go HTTP 服务运行在同一个应用进程；HTTP Serve 在 goroutine 中运行，不使用 sidecar，不启动额外后端可执行文件。
- 业务服务写死监听 `0.0.0.0:20080`，不提供端口配置或自动换端口。该地址监听所有 IPv4 网卡，并非回环地址。
- 应用前端固定使用 `http://127.0.0.1:20080`；本机外部客户端同样使用该地址，其他设备使用应用主机的实际 IPv4 地址和 20080 端口。
- CORS 按用户澄清使用 `Access-Control-Allow-Origin: *`，不做来源白名单，不启用 Cookie 跨域凭据。
- `/api/version` 为公开只读接口，无需认证。未来敏感业务接口另加认证，CORS 不承担认证职责。
- 本次只调整服务架构，保留现有托盘、置顶、关闭窗口隐藏、macOS 重新打开窗口以及 React/Vite 热更新。生产资源嵌入与打包仍不在本次范围。

## 当前实现与差距

- `src-wails/main.go` 的 `newAPIHandler` 已使用标准 `http.ServeMux`，提供 `GET /api/version`，返回 `{"version":"0.1.0"}`。
- `newAppHandler` 把 `/api/*` 分流到业务 Handler，其余请求交给 AssetServer；尚无独立业务监听端口。
- `src/App.jsx` 使用相对路径 `fetch("/api/version")`，需要改为业务服务的绝对地址。
- `src-wails/tray.go` 已实现关闭窗口隐藏与托盘退出，服务清理必须挂到应用统一退出流程，不能放在窗口关闭事件中。
- 开发模式由 Wails 启动 Vite，并通过 AssetServer 反向代理前端页面。此链路不需要因为业务 API 独立而变动。
- 已检查锁定版本 Wails v3.0.0-beta.32 源码：支持 `application.Options.OnShutdown`；清理任务同步执行，且应用 context 在执行钩子前已取消，因此 HTTP Shutdown 必须使用独立 context。

## 结构与文件调整

### 1. 拆分业务 Handler：`src-wails/api.go`

- 将版本常量和 `newAPIHandler()` 从 `main.go` 移入该文件。
- 继续使用 `http.NewServeMux()` 与 `GET /api/version`，保持响应 JSON、Content-Type 和已有版本值。
- 未注册路径返回 404，不支持的方法由 ServeMux 返回 405；不添加 SPA 回退。
- CORS 作为最外层 middleware 包裹业务 mux，使成功、404、405 等响应均带 `Access-Control-Allow-Origin: *`。
- OPTIONS 预检返回 204，明确声明当前允许的方法 `GET, HEAD, OPTIONS`。对不支持的预检方法或请求头返回明确拒绝，不无条件反射请求头。
- 当前接口不需要自定义请求头；未来增加 JSON 写接口或 Bearer 认证时，同步扩充允许方法及 `Content-Type`、`Authorization` 等必要头。
- 不设置 `Access-Control-Allow-Credentials: true`，前端不使用 `credentials: "include"`。

### 2. 封装 HTTP 生命周期：`src-wails/server.go`

- 定义生产监听常量 `0.0.0.0:20080` 和前端访问地址常量 `http://127.0.0.1:20080`，名称区分监听地址与客户端地址。
- 定义服务封装，持有 `http.Server`、listener、Serve 完成信号、运行错误及幂等停止状态。
- 服务构造可接收监听地址用于测试；生产入口只能传固定常量，不引入环境变量或配置覆盖。
- 启动时同步执行 `net.Listen("tcp4", addr)`。成功才启动 `Server.Serve(listener)` goroutine 并返回就绪；不通过异步 `ListenAndServe` 推断启动成功。
- 配置 `ReadHeaderTimeout`（建议 5 秒）和 `IdleTimeout`（建议 60 秒）。不设置统一短 `WriteTimeout`，避免未来 SSE 被截断；各业务自行控制请求期限。
- 不包装或替换 ResponseWriter，保留标准 Handler 的流式响应与连接升级能力。
- 将 `http.ErrServerClosed` 视为正常停止；其他 Serve 错误送到应用监控路径，记录具体错误并触发应用退出，避免留下可见窗口却没有业务服务。
- 停止操作幂等，重复调用等待同一次清理完成。先以独立 `context.Background()` 派生 5 秒超时执行 `Shutdown`，超时或失败记录错误并执行 `Close`，最后等待 Serve goroutine 结束。
- `Shutdown` 负责等待普通在途请求；将来引入 SSE 时需主动取消流，WebSocket 需维护连接注册表并显式关闭。当前不实现虚构的 SSE/WebSocket 端点。

### 3. 接入应用：`src-wails/main.go`

执行顺序：

1. 启动固定地址的 HTTP 服务；监听失败输出包含 `0.0.0.0:20080` 和底层原因的明确错误，非零退出，不创建正常工作窗口。
2. 记录监听地址及本机 API 访问地址，作为就绪反馈。固定地址无需新增 Wails binding 或动态发现接口。
3. 构造 Wails 应用，将 AssetOptions.Handler 改为纯 `application.AssetFileServerFS(emptyFS{})`，删除 `newAppHandler` 和业务路径分流。
4. 在 `application.Options.OnShutdown` 中同步调用 HTTP 停止逻辑；同时在应用运行函数中 defer 相同停止操作，覆盖 `app.Run()` 失败或提前返回。
5. 创建原有窗口、安装原有托盘，运行应用。监听异常监控需在应用可安全退出后接入，并覆盖启动至 Run 的边界错误，不能丢失早到的 Serve 错误。
6. 将启动/运行封装为返回 error 的函数，最外层再记录错误并退出；不要在有待执行 defer 的函数内调用 `log.Fatal`，以免跳过清理。

重复启动的策略是依赖固定端口独占：第二实例明确报端口不可用并退出；不自动切端口、不将未知占用者当成已有应用、不引入单实例唤醒机制。端口冲突本身不需要识别占用进程。

### 4. 保留窗口与托盘：`src-wails/tray.go`

- 关闭窗口继续 Cancel + Hide，不停止 HTTP 服务。
- 托盘退出继续调用 `app.Quit()`，由统一 OnShutdown 清理 HTTP 服务。
- 系统正常退出、快捷键退出与现有 Wails 信号处理也必须走统一清理；检查实际平台行为，不仅验证托盘路径。
- 如新增服务错误通知涉及跨 goroutine 状态，使用 channel/同步原语，避免引入数据竞争。
- 强制终止（如 SIGKILL）不承诺优雅清理，操作系统会回收端口。

### 5. 前端与开发说明

- `src/App.jsx` 将请求地址改为 `http://127.0.0.1:20080/api/version`，保留现有 HTTP 错误显示。
- 无需 Vite API proxy，也不改变 Vite 5173 端口、绑定地址和 Wails FRONTEND_DEVSERVER_URL。
- 在实际 macOS WebView 验证 `wails://localhost` 页面向 HTTP 服务请求的行为；CORS * 不等于绕过 WebView 的网络或混合内容策略，如遇平台限制需按实际错误修正平台设置。
- 更新 `DEVELOPMENT.md`：移除 API 同源、无独立端口的旧说明；补充固定监听地址、客户端地址、CORS、退出释放端口、端口冲突排查及测试命令。删除或修正目前指向不存在的 `main_test.go` 的说明。

## 实施顺序

1. 拆出 API mux 与 CORS middleware，完成接口与预检行为。
2. 实现可测试的 HTTP 服务启动、停止与错误传递。
3. 接入 main 的启动顺序、统一退出钩子与错误收敛，移除 AssetServer 业务分流。
4. 修改前端请求地址，更新开发文档。
5. 执行自动测试和 macOS 开发模式手工验收；Windows 行为需在 Windows 环境验收，不能以 macOS 结果代替。

## 自动验证

- Handler 测试：版本响应 JSON 与 Content-Type；未知路径 404；不支持方法 405；上述响应的 CORS *；合法预检 204；非法预检方法/头被拒绝；无 Allow-Credentials。
- 生命周期测试：使用 `127.0.0.1:0` 真正监听临时端口并请求版本；占用地址时启动返回错误；Shutdown 等待在途请求；重复停止安全；停止后同地址可重新监听。
- 增加超时清理测试以验证 Shutdown 超时后强制 Close；通过可注入的短清理期限避免测试固定等待 5 秒。
- 测试独立 HTTP 服务应尽量不依赖 GUI 初始化。
- 在 `src-wails/` 执行 `go test ./...`，必要时执行 `go test -race ./...` 检查生命周期并发；前端使用本地 Vite 构建验证 JSX。只报告实际执行结果。

## 手工验收清单

- `pnpm dev` 启动后，窗口显示版本号；`curl -i http://127.0.0.1:20080/api/version` 得到 200 和版本 JSON。
- 使用 `Origin: https://example.com` 请求，确认 `Access-Control-Allow-Origin: *`；OPTIONS 预检符合约定。
- 在独立浏览器打开 Vite 页面同样能取版本；修改 React 页面仍有热更新，修改 Go 仍可重启并重新绑定 20080。
- 关闭窗口或托盘隐藏后，curl 持续成功；重新显示窗口后页面正常。
- 托盘退出、系统正常退出、开发模式 Ctrl+C 后端口释放，再启动无残留冲突。
- 第二实例或其他程序预占 20080 时，启动明确失败，不出现正常运行的空壳窗口，原有服务不受影响。
- 局域网客户端通过主机 IPv4:20080 请求同一接口；网络/防火墙条件不满足时如实记录，不把 CORS 当作网络连通性控制。
- 检查进程关系：业务 HTTP 服务属于 Wails 应用进程；开发期 Node/Vite/Wails CLI 是既有开发工具，不新增后端子进程。

## 完成标准

上述代码与文档变更落地，自动验证通过，并记录实际平台手工验收结果及未验证项。本文件仅为实施方案，本轮不执行架构代码修改。
