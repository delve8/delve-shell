# Host bus：路径审计（内部）

供对照 §10「单一事件链」与观测扩展；非终端用户文档。

## 进入 `hostbus.Bus` 的来源

| 来源 | 事件形态 |
|------|----------|
| `BridgeInputs` ← `SubmitChan` | `hostroute.ClassifyUserSubmit` → `KindSessionNewRequested` / `KindSessionSwitchRequested` / `KindUserChatSubmitted` |
| `BridgeInputs` ← `ConfigUpdatedChan` | `KindConfigUpdated` |
| `BridgeInputs` ← `CancelRequestChan` | `KindCancelRequested` |
| `BridgeInputs` ← `ExecDirectChan` | `KindExecDirectRequested` |
| `BridgeInputs` ← `RemoteOnChan` / `RemoteOffChan` / `RemoteAuthRespChan` | 对应 Remote `Kind*` |
| `BridgeInputs` ← `SlashRequestChan` | `KindSlashRequested`（TUI 在调用 registry handler **之前**由 `Host.RequestSlashDispatch` 写入） |
| `BridgeInputs` ← `SlashTraceChan` | `KindSlashEntered`（TUI 已成功分发 slash 后由 `Host.TraceSlashEntered` 写入） |
| `BridgeInputs` ← `SlashSubmitChan` | `KindSlashRelayToUI`（主 Enter 的 slash 经 `Host.TryRelaySlashSubmit` → 中控 → `SlashSubmitRelayMsg` 回灌 TUI） |
| `BridgeInputs` ← `AgentUIChan` | `bridgeAgentUI` → approval / sensitive / exec / unknown |
| `hostcontroller` / LLM 完成路径 | `KindLLMRunCompleted`（`PublishBlocking`） |

## 双队列与职责（`events` vs `uiMsgs`）

`hostbus.Bus` 内有两条独立队列：

| 队列 | 写入方 | 消费方 | 用途 |
|------|--------|--------|------|
| `events` | `Publish` / `PublishBlocking`（含 `BridgeInputs`） | `hostcontroller.Controller.run`（`<-bus.Events()`） | 领域事件编排（会话、LLM、远程、HIL 等） |
| `uiMsgs` | `EnqueueUI` / `EnqueueUIBlocking` | `StartUIPump` → `tea.Program.Send` | 发往 Bubble Tea 的 **tea.Msg**（含 Presenter 封装的消息） |

`uipresenter.BusSender` 将 Presenter 的出站消息写入 **`uiMsgs`（阻塞）**，与 **`events`** 解耦：中控先消费领域事件，再在 handler 内通过 Presenter 间接入队 UI 消息。

## 主 Enter：slash 中继（§10.8.1 第 2 轮）

当 `Host` 为已接线的 `*Runtime` 且 `SlashSubmitChan` 非 nil 时，**主 Enter** 路径上以 `/` 开头的行优先 **`TryRelaySlashSubmit`**：

1. TUI：`handleMainEnterCommand` → `TryRelaySlashSubmit(hostroute.SlashSubmitPayload{…})`（成功则本帧直接返回，不在本帧内执行 registry）。
2. `BridgeInputs` → `KindSlashRelayToUI`（`Event.SlashSubmit`）。
3. `Controller`：`handleSlashRelayToUI` → `Presenter.Raw(SlashSubmitRelayMsg{…})` → **`EnqueueUIBlocking`**。
4. 下一帧 TUI：`Update` 收到 `SlashSubmitRelayMsg` → **`executeMainEnterCommandNoRelay`**（与原先本地执行同逻辑，含 `RequestSlashDispatch` / `TraceSlashEntered`）。

若 `TryRelaySlashSubmit` 返回 false（`Nop()`、channel 满、未接线），**回退**为同帧 **`executeMainEnterCommandNoRelay`**，与旧行为一致。

**未走此链的路径**：仅走 `handleSlashEnterKey` 的 slash Enter（例如 overlay 选中）、以及 `/new` / `/sessions` 经 `SubmitChan` 的会话命令。

## 主对话 → LLM（典型顺序）

以下描述**稳态**下的主路径；具体函数名以代码为准。

1. TUI：`Host.Submit(text)` → `SubmitChan`（仅当 UI 决定走主对话提交；普通文本、非 slash 专线路径）。
2. `BridgeInputs`：`ClassifyUserSubmit` → 若非 `/new` / `/sessions …`，则 `PublishBlocking(KindUserChatSubmitted)`。
3. `Controller`：`handleUserChat` → 启动/续跑 runner 侧 LLM 工作流。
4. LLM 结束后：`PublishBlocking(KindLLMRunCompleted)`（在独立 goroutine 中投递，避免阻塞中控循环）。
5. `Controller`：`handleLLMRunCompleted` → `uipresenter` → `AgentReply` 等 → **`EnqueueUIBlocking`** → `StartUIPump` → TUI `Update`。

**对照 §10.6**：主路径可由 **Bus 事件 Kind + Controller handler + Presenter 方法** 追踪；TUI 内仍有输入编辑与 slash 专线路径（见下节）。

## Agent HIL：审批 / 敏感确认 / 工具执行回显

1. **runner/agent** 将负载写入 **`AgentUIChan`**（`runnermgr` 选项中的 `UIEvents`）。
2. `BridgeInputs`：`bridgeAgentUI` → `KindApprovalRequested` / `KindSensitiveConfirmationRequested` / `KindAgentExecEvent` / `KindAgentUnknown`。
3. `Controller`：表驱动 handler → `uipresenter.ShowApproval` / `ShowSensitiveConfirmation` / `CommandExecutedFromTool` / `handleAgentUI`。
4. 上述方法经 **`BusSender.Send` → `EnqueueUIBlocking`** → `StartUIPump` → TUI。

**责任边界**：HIL 请求体经总线事件进入中控，再转为 **tea.Msg**；不在 Bus 上传 lipgloss 样式或布局参数。

## 远程：连接 / 断开 / 认证应答

1. TUI / feature：`Host.PublishRemoteOnTarget` / `PublishRemoteOff` / `PublishRemoteAuthResponse` → 对应 `InputPorts` channel。
2. `BridgeInputs` → `KindRemoteOnRequested` / `KindRemoteOffRequested` / `KindRemoteAuthResponseSubmitted`。
3. `Controller`：`handleRemoteOn` / `handleRemoteOff` / `handleRemoteAuthResp`（内部可再调 `uipresenter` 更新顶栏、overlay 等）。

远程 **SSH 目标字符串** 与 **认证应答** 的敏感字段仍须遵守 **`Event.RedactedSummary`** 的脱敏约定，禁止在观测层打印秘密。

## 仍仅在 TUI 内、不经「Submit→总线→LLM」主链的部分

- **Slash 解析与执行**：`ui` registry；主 Enter 的 slash 分支不调用 `SubmitChan`（除 `/new`、`/sessions` 等已走 Submit 的特例）。总线侧另有 **`KindSlashRequested` / `KindSlashEntered`**（专用 channel，仅观测）。
- **样式与 overlay 绘制**：不经过 Bus。

## 与 §10.6 完成判据的对照

| 判据 | 现状 |
|------|------|
| `cli.Run` 全局 setter / 多路接线减少 | 生产路径经 `hostwiring` + `hostapp.Runtime`；见 `interactive/host_stack.go`。 |
| 主路径可追踪 | 主对话与 Agent HIL 见上表；slash 执行仍主要在 TUI，总线侧为观测事件。 |
| UI 新增能力优先控件组合 | 进行中（§10.8 阶段 5）；overlay 已部分抽至 `internal/ui/widget`。 |
| e2e / 黑盒通过 | `go test ./internal/e2e/...`、 `internal/ui` 黑盒测试需保持绿。 |

## 后续可收紧点

- 将 `KindSlashRequested` / `KindSlashEntered` 与 metrics / 结构化日志在 `Options.OnEventDispatch` 中关联（可度量 handler 失败：有 Request 无 Entered）。
- 若 slash 编排迁入 Controller，再引入显式路由表并保持 UI 仅提交 `tea.Cmd`；结构化载荷见 `docs/adr/0001-slash-submit-payload.md` 与 `internal/hostroute/slash_submit_contract.go`。
