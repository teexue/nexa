# Nexa 前端

React 19 SPA，为 nexa 基座提供 Web 控制台（对话工作区、资源管理、看板、请求日志、管理后台、设置等）。

## 技术栈

- React 19 + TypeScript（strict）+ Vite
- Tailwind CSS v4 + shadcn/ui（`src/components/ui/`）
- React Router 7、i18next、react-markdown
- 包管理：**pnpm**（禁止 npm）

## 目录结构

```
src/
├── App.tsx            # 路由表与外层组合
├── routes/            # 各页面路由组件
├── components/        # 业务组件（按域分目录，文件 kebab-case）
│   ├── shared/        # 页面级共享原语（PageHeader/PageShell/PageMain/EmptyState/ListRow）
│   └── ui/            # shadcn/ui 基础组件
├── hooks/             # 自定义 hooks（use 前缀）
├── lib/               # API 客户端与纯工具函数
├── types/             # 共享类型定义
└── main.tsx           # 入口
```

## 常用命令

```bash
pnpm dev           # 开发服务器
pnpm build         # 类型检查 + 生产构建
pnpm lint          # ESLint
pnpm typecheck     # tsc --noEmit
pnpm test          # vitest
pnpm format        # Prettier 写入（ts/tsx/css）
pnpm format:check  # Prettier 检查（CI 用）
```

## 规范

编码规范见仓库根目录 `AGENTS.md` 前端规范节。要点：

- 无分号、双引号、2 空格缩进、尾逗号 ES5（Prettier 强制）
- 函数组件 + hooks；禁止 class 组件与 `any` 类型
- Tailwind utility-first，用 `cn()` 合并类名；禁止内联 style（动态计算值除外）
- 颜色必须使用 index.css 的设计 token（primary / muted-foreground / success / warning / chart-*），禁止 blue-500 等散装色板类
- 页面必须经由 `components/shared/` 的页面原语组装：PageHeader（页头）、PageShell/PageMain（布局与滚动）、EmptyState（空态）、ListRow（列表行）；禁止手写页头/空态拷贝
- 组件文件 kebab-case，hooks 以 `use` 前缀
- 单文件 ≤ 400 行，单组件 ≤ 200 行，单函数 ≤ 60 行

## 添加 shadcn 组件

```bash
pnpm dlx shadcn@latest add button
```
