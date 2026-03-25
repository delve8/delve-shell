# ADR 0001：Slash 结构化提交载荷（契约，未接线）

## 状态

已采纳（**仅文档与类型契约**）；运行时仍走既有 TUI 内分发与 `SlashRequestChan` / `SlashTraceChan` 观测路径。

## 背景

- `SubmitChan` 当前载荷为 **`string`**；`hostroute.ClassifyUserSubmit` 仅区分 `/new`、`/sessions …` 与其余（后者映射为 **LLM 路径** 的 `KindUserChatSubmitted`）。
- 主 Enter 路径上，slash 依赖 **下拉选中索引**、`slashflow` / `maininput.PlanMainEnter` 等上下文；若仅提交 TrimSpace 字符串，**无法**无损重建「用户选中的候选行」与歧义消解结果。
- §10.8.1 第 2 轮需要一种**不与此冲突**的扩展点，以便将来将「意图」送达 Controller 再回灌 TUI。

## 决策

1. **不**将通用 slash 行塞入现有 `SubmitChan` 的「字符串-only」路径并指望 `ClassifyUserSubmit` 单独消化；**不**改变 `/new`、`/sessions …` 在 `ClassifyUserSubmit` 中的语义，除非单独里程碑与迁移说明。
2. 将来接线时，使用 **`hostroute.SlashSubmitPayload`**（见 `internal/hostroute/slash_submit_contract.go`）作为**候选**结构化单元；实际传输可以是：
   - 独立 channel（例如 `chan SlashSubmitPayload`）由 `BridgeInputs` 映射到既有或新增 `Kind`，或
   - 扩展 `hostbus.Event` 字段（需评估 `RedactedSummary` 与兼容）。
3. Controller 在收到该事件后的职责限于 **编排**（例如 `EnqueueUIBlocking` 携带专用 `tea.Msg`）；**registry 执行仍留在 TUI** 直至进一步里程碑。

## 后果

- 正面：索引与 `RawLine` 可同传，为「中控收到意图 → TUI 执行 `dispatchSlash*`」提供清晰形状。
- 负面：多一条路径与更多测试矩阵；需严防与 `SubmitChan` 上 `/new`、`/sessions` 重复触发。

## 相关

- `docs/host_bus_audit.md`
- `docs/ui-refactor-handoff.md` §10.8.1
