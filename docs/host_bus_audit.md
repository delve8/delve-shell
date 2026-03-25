# Host bus：路径审计（内部）

供对照 §10「单一事件链」与观测扩展；非终端用户文档。

## 进入 `hostbus.Bus` 的来源

| 来源 | 事件形态 |
|------|----------|
| `BridgeInputs` ← `SubmitChan` | `hostroute` 分类 → `KindSessionNewRequested` / `KindSessionSwitchRequested` / `KindUserChatSubmitted` |
| `BridgeInputs` ← `ConfigUpdatedChan` | `KindConfigUpdated` |
| `BridgeInputs` ← `CancelRequestChan` | `KindCancelRequested` |
| `BridgeInputs` ← `ExecDirectChan` | `KindExecDirectRequested` |
| `BridgeInputs` ← `RemoteOnChan` / `RemoteOffChan` / `RemoteAuthRespChan` | 对应 Remote `Kind*` |
| `BridgeInputs` ← `SlashRequestChan` | `KindSlashRequested`（TUI 在调用 registry handler **之前**由 `Host.RequestSlashDispatch` 写入） |
| `BridgeInputs` ← `SlashTraceChan` | `KindSlashEntered`（TUI 已成功分发 slash 后由 `Host.TraceSlashEntered` 写入） |
| `BridgeInputs` ← `AgentUIChan` | `bridgeAgentUI` → approval / sensitive / exec / unknown |
| `hostcontroller` / LLM 完成路径 | `KindLLMRunCompleted`（`PublishBlocking`） |

## 仍仅在 TUI 内、不经总线的部分

- **Slash 解析与执行**：`ui` registry；总线接收 **`KindSlashRequested`（尝试前）** 与 **`KindSlashEntered`（成功后）** 用于追踪与后续扩展。
- **样式与 overlay 绘制**：不经过 Bus。

## 后续可收紧点

- 将 `KindSlashRequested` / `KindSlashEntered` 与 metrics / 结构化日志在 `Options.OnEventDispatch` 中关联（可度量 handler 失败：有 Request 无 Entered）。
- 若 slash 编排迁入 Controller，再引入显式路由表并保持 UI 仅提交 `tea.Cmd`。
