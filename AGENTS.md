# common-agent 编码规范

通用 Agent 基座（Go 核心 + React 前端）。本文档为各 AI 代理提供编码指引。

## 构建与测试

```bash
make                    # 先 check，再前端 production + 后端
make lint               # go vet + eslint --max-warnings 0 + prettier format:check
make check-standards     # 文件/函数/参数/Go doc，规则见「计数」
make check              # lint + check-standards
make test               # go test ./... + pnpm test（不含 integration tag）
make test-integration   # go test -tags=integration ./test/integration/...
make build              # 必须依赖 check
make release            # 必须先 check
```

`golangci-lint run` 只在 CI 跑（gocyclo ≤ 15、嵌套 > 4 失败），不进 `make lint`。过闸门不得放宽阈值、删测试、eslint-disable / `nolint`、跳过 hook。无 `go generate` / `go install` / Docker。

- **Go**：`go mod` 管依赖，禁止手改 `go.mod` 版本号
- **前端**：`pnpm add` / `pnpm add -D`，禁止手改 `package.json` 版本号

## 架构约束（最高优先级）

- **单入口 Agent Loop**：CLI / HTTP / gRPC 等所有路径必须调用同一个 `loop.Run` 函数，禁止在 handler 里复制 loop 逻辑
- **Tool 统一抽象**：一切能力通过 `Tool` 接口暴露，禁止在 loop 内 hardcode 业务逻辑
- **Agent 驱动差异**：提示词、工具白名单、权限、模型配置来自 Agent YAML，禁止在 core 写 `switch agent` 分支
- **事件流输出**：对外只有 `event.Event`。类型：`text_delta` / `reasoning_delta` / `tool_start` / `tool_result` / `tool_approval_required` / `compaction` / `sub_agent_start` / `sub_agent_end` / `error` / `done`
- **事件是契约**：本仓库消费的事件类型必须在同一变更里同步 HTTP SSE、gRPC convert、TUI、前端 SSE 分发、`sdk/ts`、`sdk/python`。漏一处不得合并；commit 注明 breaking
- **依赖方向**：`cmd → server → core`。core 不得依赖 server / cmd。HTTP 解析与编码在 `server/http`，业务在 `core/service`；handler 不调用 Provider、不复制 loop

## 目录与包布局

```
cmd/                  # 入口，仅 wiring
core/{agent,config,hook,knowledge,service,skill,store,telemetry,audit,auth,i18n,kanban,tui,version}
server/{http,grpc}
frontend/             # React SPA（Vite + Tailwind + shadcn）
sdk/{ts,python}
```

本仓库负责配置、知识库、Skill 文件、Agent 文件、观测和界面。不得存在 `core/billing`、`core/tenant`、`core/workflow`（除非用户另开产品任务）。

| 层 | 包 | 职责 |
|----|----|------|
| 入口 | `cmd/` | CLI wiring；默认命令启动 Web UI（`web`；`serve` 为别名） |
| 传输 | `server/http/` | HTTP/SSE：解析请求 → 调用 `loop.Run` → 流式事件 |
| 业务 | `core/service/` | HTTP/gRPC 共用业务；handler 不调 Provider、不复制 loop |
| 核心 | `core/config/` | 用户配置 `~/.common-agent/`（settings、providers、`CredentialStore`、wizard） |

- 包名小写、短、无下划线（`loop` 而非 `agent_loop`）；目录名即包名
- 每个目录一个包；禁止 `util`、`common`、`helper`、`misc` 包或文件（含 `handler_misc.go`，必须按资源拆并改名）
- 跨包共享类型放语义明确的包（如 `event`、`agent`）

## 关键约定

- **配置目录**：`~/.common-agent/` — `state.db`（设置、供应商、凭证、会话等）、`agents/*.yaml`（Agent 定义）。禁止提交 credentials 或 `.env`。升级时会一次性把旧的 `config.yaml` / `providers.yaml` / `credentials.yaml` / `mcp.yaml` 迁入 SQLite。
- **凭证**：`config.NewCredentialStore(home)` 创建线程安全 store（读 `state.db`），将其 `Lookup` 传给 catalog；包级旧函数已废弃
- **工具命名**：snake_case，全局唯一（如 `read_file`）。通过 `registry.Register()` 显式注册，禁止 `init()` 魔法注册
- **Provider 解析**：`cmd` 层按名从 catalog 解析并创建具体 `provider.Provider`。业务层只依赖接口

## Agent YAML 参考

加载、校验工具引用、文件监听在 `core/agent`。文件放在 `agents/{id}.yaml`。

## 可维护性约束

| 指标 | Go | 前端 |
|------|-----|------|
| 单文件行数 | ≤ 500 行（测试 ≤ 600） | ≤ 400 行（测试 ≤ 600） |
| 单函数行数 | ≤ 80 行 | ≤ 60 行（含组件与 hooks） |
| 函数参数数 | ≤ 5 个（超过用 Config struct 封装） | ≤ 5 个（超过用 options object） |
| 嵌套深度 | ≤ 4 层（超过用早返回或提取子函数） | ≤ 4 层 |
| 圈复杂度 | ≤ 15 | ≤ 15 |

数字不可改，超标只拆代码。嵌套与圈复杂度由 CI golangci（Go）与 eslint `complexity` / `max-depth`（前端）强制。

**计数**（`check-standards.py` 必须按此实现）：

- **文件**：物理行，含空行与注释。Go 生产 `*.go`（非 `_test.go`）；Go 测试 `*_test.go`。前端生产 `frontend/src/**/*.{ts,tsx}` 且路径不含 `.test.`；前端测试文件名含 `.test.`。
- **函数**：从签名行到匹配的最后一个 `}`，含首尾。Go 凡 `func`（含方法、测试）均 ≤ 80。前端一律 ≤ 60：`function` / `async function` / `const name = (` 或 `async (` 后接箭头 / `export function` / `export const name = (`；`React.forwardRef` 内函数同样计。不算函数：`const X = (expr)` 且不是箭头（如 `(a - b) / 2`）。
- **参数**：每个形参计 1，含 `ctx`；方法接收者不计；`...T` 计 1；必须解析多行签名；`a, b string` 为 2 个。
- **Go 导出文档**：每个导出 `func` / `type` / `var` / `const` 上一行必须是以该名称开头的 doc comment。`const (` 块内每个导出标识符各自一条，禁止一条注释罩多个。
- **体积豁免**：`proto/`、`vendor/`、`node_modules/`、`frontend/src/components/ui/`。`ui/` 仍过 eslint / `any` / class / Prettier；禁止业务逻辑、调项目 API、项目专用文案（token 类名除外）、把业务组件放进去躲行数。

## Go 编码规范

- **命名**：导出 PascalCase，未导出 camelCase，文件 snake_case，工具名 snake_case
- **Import**：三组分隔 — 标准库 / 外部依赖 / 内部包
- **错误处理**：`(T, error)` + `fmt.Errorf("context: %w", err)` 包装；早返回；禁止 panic 处理可预期错误
- **Context**：I/O/LLM 第一参数，不存 struct，支持 cancellation
- **接口**：小接口 + `NewXxx(deps...)` 注入；避免 `init()` 全局注册
- **注释**：导出符号必须有 doc comment（以名称开头的简短注释）；只写「为什么」，禁止复述代码与分区横幅（`// ───`）
- **日志**：`log/slog` 结构化日志，统一字段 `session_id` / `agent` / `tool` / `turn`
- 新函数参数已超过 5 个则必须带 Config，不得先写后改

## 前端编码规范

- **格式**：无分号、双引号、2 空格缩进、尾逗号 ES5（Prettier 强制）
- **组件**：函数组件 + hooks，禁止 class 组件，禁止 `any` 类型。组件按函数计行数（≤ 60）。`frontend/src/components/ui/` 豁免体积检查，仍须过 eslint / Prettier，且禁止写入业务逻辑
- **样式**：Tailwind utility-first，用 `cn()` 合并类名，禁止内联 style（动态计算值如进度百分比、背景图 URL 除外）；颜色必须用 index.css 设计 token（primary / muted-foreground / success / warning / chart-*），禁止散装色板类
- **页面原语**：页面必须经由 `components/shared/` 的 PageHeader / PageShell / PageMain / EmptyState / ListRow 组装，禁止手写页头与空态拷贝
- **页面 vs 弹窗**：确认/危险/短交互用 Dialog；表单/详情/多步走独立路由页面
- **IME**：表单与聊天的 Enter 提交必须先过 `isComposingEvent`，避免输入法确认候选词时误提交
- **命名**：组件文件 kebab-case，工具函数 camelCase，hooks 以 `use` 前缀
- **TypeScript**：strict mode，对象用 `interface`，联合用 `type`，优先 `as const` 替代 `enum`
- **React Refresh**：组件文件只导出组件；CVA `*Variants` 不要导出（除非确有外部消费者）；Context Provider 可与其 hook 同文件，hook 名必须列入 `frontend/eslint.config.js` 的 `allowExportNames`
- **工作区会话**：只在 `location.search` 变化时从 URL 恢复会话；写入 URL 的 effect 不得订阅 `location.search`；非流式滚到底用 `behavior: "auto"`；自动滚动 `resetKey` 必须是真实 `sessionId`
- **Shell**：已登录布局只挂载一次侧栏与顶栏，子路由只换主栏。禁止每个 route 复制 `AppLayout`。`useAgentManager` 只在 Shell 层一份

## 注释

- 只写「为什么」：不变量、外部契约、非直观算法、有意偏离常规的选择
- 禁止：复述代码、分区横幅、JSX 里描述 UI 的注释、过时的历史注释
- 导出符号：Go 必须有 doc comment；前端仅在非直观的公共 API 上写 JSDoc
- 禁止用 `eslint-disable` 代替重构。若规则无法用代码消除，disable 必须紧邻违规行并写明原因

## 前端 Effect 与状态

- 禁止在 `useEffect` 里同步 `setState`。数据拉取的 `setState` 只放在 promise 回调（`.then` / `.catch` / `.finally`）
- 手动刷新：在事件处理器里设 `loading` / 清错误；effect 只调用 load
- props → 表单初始值：用路由 `key` 重置组件（见 `LoginGate`、`AgentEditorRoute`），禁止 reset effect
- 能从已有 state / props 派生的不要再存一份
- `exhaustive-deps`：先补齐依赖或抽出 `useCallback`；仅当补齐会形成更新环且已有守卫时才允许 disable

## 测试

- **Go**：表驱动优先；推荐 `testify/assert` + `testify/require`；happy path + 至少一个 error path。按场景分文件，单测文件 ≤ 600 行
- **前端**：vitest + React Testing Library；测纯函数和 hooks
- **集成测试**：`test/integration/`，`//go:build integration`；mock provider，不调真实 LLM。`make test` 不含该 tag。至少覆盖：HTTP run 走 mock 的 `loop.Run`、鉴权失败、工具审批一轮

## Git 规范

- **Commit**：Conventional Commits — `<type>(<scope>): <description>`
- **Type**：`feat` / `fix` / `docs` / `refactor` / `test` / `chore` / `perf` / `style`
- **分支**：`feat/<name>`、`fix/<name>`，短横线分隔，全小写
- **PR**：标题遵循 Conventional Commits，关联 Issue，变更聚焦单一子任务。CI 必须跑 `make check` + `make test` + `golangci-lint`

## 禁止事项

| 禁止 | 原因 |
|------|------|
| core 引入 HTTP/gRPC 框架依赖 | 违反依赖方向 |
| 创建 `utils` / `helpers` / `misc` 包或文件 | 包无明确语义 |
| loop 外直接调用 LLM Provider | 绕过权限和审计 |
| 跳过 Permission 检查执行 Tool | 安全漏洞 |
| 为单个 Agent 写 if/else 分支 | Agent 配置驱动 |
| panic 处理可预期错误 | 应返回 error |
| struct 存储 context | 违反 Go 最佳实践 |
| 前端使用 `any` 类型 | 破坏类型安全 |
| 前端 class 组件 | 项目统一函数组件 |
| 在 `useEffect` 里同步 `setState` | 额外渲染，违反 react-hooks |
| 用 `eslint-disable` 掩盖可重构的 lint | 先改代码 |
| 提交 `.env` / API Key / credentials | 安全风险 |
| 手动编辑 `go.mod` / `package.json` 版本号 | 使用包管理器 |
| 过度抽象（YAGNI） | 不需要的 interface 不提前定义 |
| 为过检查而删测试或改阈值数字 | 只拆代码 |
| 恢复 billing / tenant / workflow | 除非用户另开产品任务 |

## 变更纪律

- 单次变更聚焦一个子任务
- 仅用户明确要求时才 git commit
- 新 HTTP 路由：解析编码在 `server/http`，业务在 `core/service`
- 新页面：独立路由 + PageShell 原语 + 挂已有 Shell Outlet + `en.json` 与 `zh-CN.json`
- 新包：目录名即包名，职责一句话能说清
