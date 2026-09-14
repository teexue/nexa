<div align="center">
  <h1>Nexa</h1>
  <p><strong>面向生产环境的通用 Agent Runtime 基座</strong></p>
  <p>
    <a href="https://github.com/teexue/nexa/releases/latest"><img alt="Release" src="https://img.shields.io/github/v/release/teexue/nexa?label=release"/></a>
    <img alt="Go" src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go"/>
    <img alt="React" src="https://img.shields.io/badge/React-19-61DAFB?logo=react"/>
    <img alt="License" src="https://img.shields.io/badge/License-MIT-yellow"/>
  </p>
</div>

基于单一 Agent Loop 架构的自托管运行时。通过 YAML 配置定义 Agent，即可在终端、Web 界面或 API 中获得具备工具调用能力的 AI 助手。支持云端模型与本地 Ollama。

## 界面预览

对话工作区：工具调用、思考过程与 Markdown 回复。

<p align="center">
  <img src="screenshots/chat.jpg" alt="对话工作区" width="920"/>
</p>

<table>
  <tr>
    <td width="50%"><img src="screenshots/manage.jpg" alt="资源管理"/></td>
    <td width="50%"><img src="screenshots/kanban.jpg" alt="看板"/></td>
  </tr>
  <tr>
    <td align="center"><sub>资源管理 — Agent、工具、技能、知识库、提供商、MCP</sub></td>
    <td align="center"><sub>看板 — 入队执行，完成后待审核</sub></td>
  </tr>
  <tr>
    <td width="50%"><img src="screenshots/settings.jpg" alt="设置"/></td>
    <td width="50%"><img src="screenshots/admin.jpg" alt="管理后台"/></td>
  </tr>
  <tr>
    <td align="center"><sub>设置 — 外观、工作目录与运行偏好</sub></td>
    <td align="center"><sub>管理后台 — 用户、API Key 与进程指标</sub></td>
  </tr>
</table>

## 功能

- **接入方式** — 默认 Web 控制台；也可用终端对话、一次性命令行运行、HTTP SSE，以及 TypeScript / Python SDK；gRPC 需显式开启
- **Agent 配置** — 每个助手一份 YAML：系统提示词、模型、工具白名单、审批与知识库；提供商、凭证、全局 MCP 和偏好设置存在本地 SQLite。界面可复制已有 Agent 再改
- **模型提供商** — 内置 OpenAI、Anthropic、Moonshot、DeepSeek、智谱、通义、Ollama（本地与 Cloud）等预设，也可填写兼容端点；按 Agent 切换
- **工具与 MCP** — 文件读写与编辑、目录、搜索、Shell、网页抓取、查询时间；可委派子 Agent。全局 MCP 在设置里配置，Agent YAML 可按助手覆盖
- **工具审批** — 按工具名设置自动通过或拒绝，未列出的调用需人工确认后执行
- **会话与压缩** — 对话持久化、多会话切换，会话可指定工作目录；用量接近上下文窗口时自动压缩
- **看板** — 任务入队后由运行时执行，成功后进入待审核；拒绝可带反馈重跑，失败可重新入队
- **知识库** — 导入文档并做向量检索，Agent 通过工具按需搜索
- **技能** — SKILL.md 技能包（全局或按 Agent）；先提供目录，完整说明按需载入
- **用量与日志** — Token 消耗按日、模型、会话汇总；管理员可查询每次 LLM 请求。顶栏有健康状态，管理后台有进程指标
- **多用户** — 登录与角色；API Key 带权限范围
- **国际化** — 中英文界面与消息

## 快速使用

一键下载到当前目录：

macOS / Linux：

```bash
curl -fsSL https://raw.githubusercontent.com/teexue/nexa/main/scripts/install.sh | bash
./nexa
```

Windows（PowerShell）：

```powershell
irm https://raw.githubusercontent.com/teexue/nexa/main/scripts/install.ps1 | iex
.\nexa.exe
```

也可从 [Releases](https://github.com/teexue/nexa/releases/latest) 手动下载对应平台的二进制，无需安装 Go 或 Node.js。无参数启动即打开 Web 控制台（默认 `http://localhost:8080`）。浏览器里注册第一个账户，该用户会成为管理员；随后在设置里填写模型提供商与 API Key 即可对话。关掉运行窗口或进程后服务会停止。

按系统选择文件：

| 系统 | 架构 | 文件 |
|------|------|------|
| Windows | x64 | `nexa-windows-amd64.exe` |
| Windows | ARM | `nexa-windows-arm64.exe` |
| macOS | Apple 芯片 | `nexa-darwin-arm64` |
| macOS | Intel | `nexa-darwin-amd64` |
| Linux | x64 | `nexa-linux-amd64` |
| Linux | ARM | `nexa-linux-arm64` |

### Windows

双击 `.exe`。会弹出一个控制台窗口，保持打开，浏览器访问 [http://localhost:8080](http://localhost:8080)。

若 SmartScreen 拦截，选「更多信息」再「仍要运行」。

