# ADR 0001：Slash 结构化提交载荷

## 状态

已采纳后又在后续统一输入生命周期重构中**退役**。

当前状态：

- `SlashSubmitChan` / `KindSlashRelayToUI` / `SlashSubmitRelayMsg` / `TryRelaySlashSubmit` 已从代码中移除。
- slash 提交改为直接进入统一 lifecycle，并由 `ui` 本地执行现有 slash handler。
- 本 ADR 仅保留为一次过渡性设计记录，不再是当前架构事实源。

## 背景

- `SubmitChan` 当前载荷为 **`string`**；`route.ClassifyUserSubmit` 仅区分 `/new`、`/sessions …` 与其余（后者映射为 **LLM 路径** 的 `KindUserChatSubmitted`）。
- 主 Enter 路径上，slash 依赖 **下拉选中索引**、`slashflow` / `maininput.PlanMainEnter` 等上下文；若仅提交 TrimSpace 字符串，**无法**无损重建「用户选中的候选行」与歧义消解结果。
- §10.8.1 第 2 轮需要一种**不与此冲突**的扩展点，以便将来将「意图」送达 Controller 再回灌 TUI。

## 决策

1. **不**将通用 slash 行塞入现有 `SubmitChan` 的「字符串-only」路径并指望 `ClassifyUserSubmit` 单独消化；**不**改变 `/new`、`/sessions …` 在 `ClassifyUserSubmit` 中的语义，除非单独里程碑与迁移说明。
2. 当时采用结构化载荷保留 `RawLine` / `SelectedIndex` / `InputLine`，避免 slash 上下文在字符串 submit 路径上丢失。
3. 当前这一目标已由统一 `InputSubmission` 模型替代，不再依赖单独的 slash relay payload。

## 后果

- 正面：在当时的过渡阶段，索引与 `RawLine` 可以同传，避免 slash 语义被压扁。
- 负面：额外引入了一条 controller 回灌链路与测试矩阵；这也是后续统一 lifecycle 时被删除的直接原因之一。

## 相关

- `docs/host_bus_audit.md`（含 Enter 中继与 **slash 候选列表单一入口** §10.8.2 第 2 轮）
- `docs/ui-refactor-handoff.md` §10.8.1 / §10.8.2
