# internal/ui 重构交接：现状、设计思路与后续规划

本文供新会话接续使用：概括已落地的架构、硬性约束、推荐演进顺序与风险点。语气为内部技术记录，不面向终端用户。

**仓库内路径**：`docs/ui-refactor-handoff.md`（可提交）。若本地使用 Cursor，结构任务清单可能在 `.cursor/code-structure-tasks.md`（部分仓库中该文件被跟踪）。

---

## 1. 项目目标（与 delve-shell 原则对齐）

- **HIL**：执行路径必须经过本工具展示与确认；registry 只影响「路由与 UI」，不改变 HIL 边界。
- **可审计**：与执行相关的决策应可追溯；本文讨论的拆分以**可读性 / 模块边界**为主，若动到历史写入路径需单独评审。
- **AI 不写敏感资源**：配置、允许列表、历史文件仍不由 AI 工具直接改写；registry 是编译期 `init()` 注册，符合「人/构建产物」控制。

---

## 2. 当前架构：registry + feature 包

### 2.1 设计思路

- **`internal/ui` 偏「壳」**：`Model`、`View`、`Update` 主循环、通用 overlay 框架、slash 分发表、样式。
- **业务按域下沉**：`internal/skill`、`internal/remote`、`internal/session`、`internal/configllm`、`internal/run` 等通过 **`init()` + `ui.Register*`** 注册，避免 `ui` 直接 `import` 所有业务（除测试替身外）。
- **单一注册入口**：`internal/cli/run.go` 使用空白 import `_ "delve-shell/internal/..."` 触发各包 `init()`。

### 2.2 已存在的注册点（`internal/ui/feature_providers.go` 及邻域）

| 机制 | 用途 |
|------|------|
| `RegisterSlashExact` / `RegisterSlashPrefix` | Enter 路径上的精确 / 前缀命令 |
| `RegisterSlashOptionsProvider` | 输入 `/...` 时的下拉候选 |
| `RegisterSlashSelectedProvider` | 选中某行后 Enter、且不走 exact/prefix 时的行为（如 fill-only） |
| `RegisterOverlayKeyProvider` | overlay 激活时的按键 |
| `RegisterOverlayContentProvider` | overlay 正文渲染 |
| `RegisterOverlayCloseHook` | **任意** overlay 关闭时复位业务字段（Esc / 程序化 close）；生产侧由 feature 注册层统一挂载 |
| `RegisterMessageProvider` | `tea.Msg` 的优先处理（在 `ui` 默认 switch 之前） |

### 2.3 与 `update_slash_*` 的关系

- `dispatchSlashExact` / `dispatchSlashPrefix` 遍历 registry；**具体命令实现不在 `ui` 包内堆业务**（`/config*` 等已迁至 `internal/run`）。
- `handleSlashSelectedFallback` 只跑 `slashSelectedProviders`，内置硬编码应趋近于零（当前 `/run` 在 `internal/run`，`/skill` 在 `internal/skill`）。

---

## 3. 硬性约束与已知的「不能简单 import」

### 3.1 `internal/ui` 测试包 vs `internal/run` 的 **import cycle**

- **事实**：`internal/run` → `internal/ui`；若 `internal/ui` 的 `*_test.go` 再 `_ import "delve-shell/internal/run"`，则形成 **`ui`（测试）→ `run` → `ui`**，Go 禁止。
- **现状变化**：`ui` 行为链路测试已迁移到 `package ui_test` 黑盒（真实 feature 注册链），不再依赖 `feature_registry_*_test.go` 的镜像注册。
- **架构判断**：循环依赖应视为架构告警，不应长期靠测试镜像兜底。  
- **后续去重方向**：  
  1) 引入 **registry core**（第三包，脱离 `ui.Model` 直接依赖）；或  
  2) 将 `internal/ui` 测试逐步迁移到 `package ui_test`（减少对未导出符号耦合）。  
  当前两条可并行小步推进，优先保持行为稳定与可回滚。

### 3.2 `ui_test` 黑盒测试定位

- **目的**：以真实注册链验证 UI 行为，直接覆盖 slash/overlay 主链路，降低测试结构对 `ui` 内部未导出实现与镜像脚手架的耦合。
- **内容（现状）**：`internal/ui/model_blackbox_test.go` 使用 `package ui_test` + 空白导入 feature 包，已覆盖 `/help`、`/remote on`、`/cancel`、`/run`、`/new`、`/sessions`、startup overlay 等关键路径。
- **结果**：`feature_registry_*_test.go` 与 `main_test.go` 镜像装配层已删除。

---

## 4. `view` 层拆分（最近完成）

### 4.1 文件职责（便于定位）

| 文件 | 内容 |
|------|------|
| `view.go` | `View()`、`appendSuggestedLine` |
| `view_content.go` | `buildContent()`（消息流；审批块委托 `view_approval_card.go`） |
| `view_approval_card.go` | `appendApprovalViewportContent`（敏感 / 标准审批卡文案与样式） |
| `view_slash_dropdown.go` | slash 下拉、`choiceLinesBelowInput`、`waitingLineBelowInput` |
| `view_overlay.go` | `renderOverlay`、`overlayBoxMaxWidth` |
| `view_title.go` | `titleLine`、`statusKey` |
| `view_choices.go` | 审批/敏感数字选项、`syncInputPlaceholder` |
| `view_history_lines.go` | `sessionEventsToMessages`、`maxSessionHistoryEvents` |
| `view_wrap.go` | `wrapString` |

### 4.2 设计注意点

- **`renderOverlay` 内 lipgloss 局部样式** 曾用名 `titleStyle`，与包级 **`titleStyle`（顶栏）** 冲突；已改为 `overlayTitleBarStyle`。后续新增样式禁止复用包级同名。
- **overlay 正文** 已由 `RegisterOverlayContentProvider` 聚合；`renderOverlay` 保持「画框 + 居中」即可。

---

## 5. 后续规划（建议优先级）

### P1 — 低风险、边界清晰

1. **`view_title.go` 中 Remote 展示**（已落地）  
   - `RegisterTitleBarFragmentProvider`：`internal/remote/registration.go` 注册；`feature_registry_test.go` 镜像（与 overlay hook 同理）。  
   - `view_title.go` 通过 `titleBarLeadingSegment()` 聚合；无 provider 命中时默认 `"Local"`。

2. **`buildContent` 中审批卡与 skill 行**（已落地 **B**）  
   - `view_approval_card.go`：`appendApprovalViewportContent`；`view_approval_card_test.go` / `view_title_test.go` 覆盖片段行为。  
   - 后续若要将「行列表」上移到 `hiltypes`，可与现有函数并行演进。

3. **slash 业务处理位置审计**（已核对）  
   - 现状：`ui` 仅保留分发与展示壳层；`/config*`、取消与重载、`/q`/`/sh`、allowlist 更新等处理逻辑在 `internal/run`。

### P2 — 中风险、牵涉 Model

4. **`internal/ui/model.go` 字段分包**（阶段性已落地）  
   - 已完成：`ConfigLLM`、`RemoteAuth`、`AddRemote`、`AddSkill`、`UpdateSkill`、`PathCompletion`、`RunCompletion`、`Context`、`Interaction`、`Overlay`、`Layout`、`Startup`、`Approval` 收敛为嵌套状态结构；宿主通信端口收敛为 `UIPorts`。  
   - 结论：**状态仍留在 `ui.Model`**，feature 包通过 `Register*` + 约定字段协作。  
   - **Bubble Tea 约束**：子 `textinput.Model` 的 `Update` 仍在 `ui` 或 `overlay_key` 链路上，后续继续拆分时别破坏 `tea.Model` 更新顺序。

5. **打破 `ui` 测试与 `run` 的循环（进行中）**  
   - 已做：slash exact/prefix 命令注册从 `ui` 下沉到 `internal/run`；`ui` 仅保留壳层分发与 registry API。  
   - 已做：删除 `ui` 内部 `registerSlashExact` 别名，只保留 `RegisterSlashExact`。  
   - 已做：测试镜像按领域拆分，减轻“单点大文件”维护成本。  
   - 已做（阶段 1）：抽出 `internal/slashreg` 的 `ExactRegistry` / `PrefixRegistry`，`ui` 改为适配层。  
   - 已做（阶段 2）：provider 链（options/selected/message/overlay/title/close-hook）迁移到 `slashreg.ProviderChain` 容器。  
   - 待做（结构性）：评估 `ui_test` 外部包迁移成本，逐步减少测试镜像覆盖面。

### P3 — 产品/仓库卫生

6. **`internal/e2e` 目录位置**（见结构任务清单）：纯风格，与行为无关。  
7. **`cli.Run` 再瘦身**：`run.go` 稳定后可抽 `startTUI(...)`。  
8. **`modelinfo` HTTP 超时**：行为改进，与 ui 重构正交。

---

## 6. 验证与提交约定

- 本地若 sandbox 缺模块：`GOMODCACHE="${HOME}/go/pkg/mod" go test ./...`。  
- **Commit message 使用英文**，conventional commits（仓库规则）。  
- 文档类修改：若无功能变更，可单独 `docs:` 或并入同一 PR 说明。

---

## 7. 关键文件速查

- Registry 定义：`internal/ui/feature_providers.go`（含 `TitleBarFragmentProvider`）、`slash_exact_registry.go`、`slash_prefix_registry.go`  
- Overlay 关闭：`internal/ui/update_overlay_key.go` + `internal/run/overlay_close_hook.go`  
- 黑盒测试入口：`internal/ui/model_blackbox_test.go`  
- CLI 接线：`internal/cli/run.go`（空白 import 列表）  
- Host 总线与中控：`internal/hostbus/bus.go`、`internal/hostcontroller/controller.go`  
- Host→TUI 消息门面：`internal/uipresenter/presenter.go`  
- 结构任务总表：`.cursor/code-structure-tasks.md`（若仓库跟踪该文件）

---

## 8. 开放问题（留给后续会话）

- **Overlay 互斥**：当前假设同时只有一个「业务 overlay」语义；若未来允许多层，需要重新定义 `OverlayCloseHook` 是否按栈弹出。  
- **i18n**：多处 feature 包写死 `lang := "en"`，与 `m.getLang()` 长期是否统一需产品决定。  
- **性能**：`titleLine` / `View` 每帧调用；若引入 provider 链，避免在 provider 内做 IO。

---

## 9. 变更记录（可选维护）

| 日期 | 说明 |
|------|------|
| 2026-03-25 | §10.8.1：三轮收尾计划（阶段 4～5 与阶段 3 最小闭环的分批交付与验收） |
| 2026-03-25 | §10.8：`KindSlashRequested`（handler 前）与 §10.4.1/§10.7 表述对齐；阶段 3 记为「观测加深、Submit 统一仍待」 |
| 2026-03-25 | §10.8：阶段 2/4/5 部分落地（`KindSlashEntered`、`docs/host_bus_audit.md`、`ui/widget` overlay）；阶段 3（slash 主路径上收）仍待后续 |
| 2026-03-25 | §10.8：五阶段计划中阶段 1 落地与 2–5 待做表；e2e 修复 + `wireHostStack` 抽取 |
| 2026-03-25 | 交接文档 §10.7.1：`Model` 显式 `*Runtime` 与 Remote 总线合并命名的可选彻底层；§10.7 补充「已无 Install 全局」现状说明 |
| 2026-03-25 | 交接文档 §10.7：总线目标 2 刻意未做项（Payload 分型、Kind 改名、Slash 上总线、默认观测、动态 handler）及后续演进建议 |
| 2026-03-25 | 目标 2（总线语义可追踪）：`Kind.SemanticLabel` 与 §10.4 草稿对齐；`Event.RedactedSummary`；`hostbus.WithPublishHook`；`hostcontroller` 表驱动 `hostEventHandlers` + `Options.OnEventDispatch` |
| 2026-03-25 | Host 主干切换：`internal/hostbus` + `internal/hostcontroller` 替代已删除的 `internal/cli/hostloop`；`internal/uipresenter` 作为 Host→Bubble Tea 消息的统一门面；`internal/run/host_wire.go` 替代 `hostloop_chans.go` |
| 2025-03-24 | `registry core` 两阶段：`internal/slashreg` 承接 slash exact/prefix registry 与 provider chain 容器，`ui` 保持注册 API 但不再持有底层容器实现 |
| 2025-03-24 | overlay close 业务复位下沉：`ApplyOverlayCloseFeatureResets` 从 `ui` 迁到 `internal/run` 注册层，`ui` 保留通用 hook 执行机制 |
| 2025-03-24 | `config` 业务逻辑继续下沉：删除 `internal/ui/config_handlers.go`，allowlist update/auto-run 处理迁至 `internal/run`；`ui` 仅保留壳层分发/渲染职责 |
| 2025-03-24 | `ErrLLMNotConfigured` 文案所需配置路径改为由 host 注入 `Model.Context.ConfigPath`，去除 `ui` 对 `internal/config` 的生产依赖 |
| 2025-03-24 | slash 静态候选下沉：`/help`、`/config` 等顶层与 `/config` 子命令候选由注册机制提供（`ui.RegisterRootSlashOptionProvider` + `ui.RegisterSlashOptionsProvider`），`ui/slash.go` 去除硬编码与 `/config` fallback |
| 2025-03-24 | `/run` 候选下沉：`ui/slash.go` 移除 `/run` completion 默认实现，改由 `ui.RegisterSlashOptionsProvider` 在 `internal/run` 提供 |
| 2025-03-24 | 启动 overlay 下沉：`ui` 不再硬编码 `dispatchSlashExact(\"/config llm\")`，改为 `RegisterStartupOverlayProvider`，由 `internal/configllm` 注册 |
| 2025-03-24 | slash 选中执行路径去特判：`ui` 删除 `/skill` 的硬编码 fill-only 分支，统一先走 `RegisterSlashSelectedProvider` 再做 exact/prefix 分发 |
| 2025-03-24 | `ui` 测试 mirror 去重：remote/configllm mirror 抽出共享 helper（overlay 初始化逻辑），降低重复维护成本 |
| 2025-03-24 | 启动 `ui_test` 黑盒迁移：新增 `internal/ui/model_blackbox_test.go`，直接加载真实 feature 注册链验证 slash/overlay 关键路径，减少对 mirror 的结构依赖 |
| 2025-03-24 | 黑盒迁移加速：`model_blackbox_test.go` 扩展至 cancel/run/sh/new/sessions 等真实链路，并从 `model_test.go` 移除重复的包内 slash 行为用例 |
| 2025-03-24 | 继续压缩 `package ui` 内 slash 下拉测试：将 Up/Down+Enter、`/cancel` 双 Enter、`/config update-skill` 等迁移到 `ui_test` 黑盒，进一步削减 mirror 依赖面 |
| 2025-03-24 | 测试装配显式化：`feature_registry_test.go` 从 `init` 改为 `sync.Once` 注册，并通过 `main_test.go` 的 `TestMain` 显式注入 mirror，降低隐式副作用 |
| 2025-03-24 | mirror 覆盖面继续收缩：移除 startup overlay mirror 注册（`package ui` 测试不再依赖该镜像），由 `ui_test` 黑盒覆盖启动 overlay 行为 |
| 2025-03-24 | 镜像测试层整批删除：移除 `feature_registry_*_test.go`、`main_test.go`、`config_handlers_test_helpers_test.go`，`internal/ui` 行为测试改由 `model_blackbox_test.go` 驱动真实 feature 注册链 |
| 2025-03-24 | session 历史解析下沉：删除 `ui/view_history_lines.go` 与 `ui/session_events_export.go`，`history.Event -> UI 行` 转换迁至 `internal/session/history_lines.go`，并迁移对应测试到 `internal/session/history_lines_test.go` |
| 2025-03-24 | `/run` 本地命令发现下沉：删除 `ui/run_completion.go`，PATH 扫描与本地可执行缓存迁至 `internal/run/local_commands.go`，`ui` 不再持有命令发现逻辑 |
| 2025-03-24 | 路径补全能力下沉：删除 `ui/pathcomplete.go`，新增 `internal/pathcomplete/candidates.go` 并由 `remote` 等业务包直接依赖，`ui` 不再直接访问文件系统补全细节 |
| 2025-03-24 | 审批文案组装下沉：新增 `internal/approvalview/blocks.go` 承接敏感/审批卡片文案规则，`ui/view_approval_card.go` 改为样式渲染适配层 |
| 2025-03-24 | 审批选项规则下沉：新增 `internal/approvalview/choices.go`（选项数、选项文案、输入占位符规则），`ui/view_choices.go` 改为调用适配层 |
| 2025-03-24 | 审批决策回写下沉：`update_approval.go` 中审批/敏感决策文案拼装改为调用 `approvalview.BuildDecision`，`ui` 仅保留样式映射与状态流转 |
| 2025-03-24 | 审批按键解释下沉：新增 `internal/approvalflow/choice.go`，`up/down/enter/数字键` 到决策动作的映射从 `ui` 抽离，`update_approval.go` 聚焦执行动作与状态更新 |
| 2025-03-24 | slash 可见项策略下沉：新增 `internal/slashview/filter.go`，`visibleSlashOptions` 与 `slashChosenToInputValue` 迁移到独立包，`ui/slash.go` 仅做适配调用 |
| 2025-03-24 | slash 导航规则下沉：新增 `internal/slashview/navigation.go`，`up/down` 循环选择与越界修正从 `update_main_key.go` / `update_slash_key.go` 抽离到独立包 |
| 2025-03-24 | slash 选中判定下沉：新增 `internal/slashview/selection.go`，`fill-only` 与 `selected resolve` 规则从 `update_main_key.go`、`update_slash_key.go`、`update_main_enter_command.go` 统一抽离 |
| 2025-03-24 | slash Enter 结果判定下沉：新增 `internal/slashflow/enter.go`，session switch / session none / selected resolve / unknown 的分支决策从 `update_main_enter_command.go` 抽离 |
| 2025-03-24 | slash Enter 执行路径收口：`update_main_enter_command.go` 新增 `handleSlashOutcome` 与 `resolveSelectedSlash`，主流程收敛为 “dispatch miss -> outcome evaluate -> action apply” 三段 |
| 2025-03-24 | slash 选中项提取下沉：新增 `internal/slashview/selected.go`（`SelectedByVisibleIndex`），`update_main_key.go` 与 `update_main_enter_command.go` 不再直接操作 `opts[vis[idx]]` |
| 2025-03-24 | slash 下拉构建下沉：`view_slash_dropdown.go` 的候选行布局与描述换行策略整体迁至 `internal/slashview/dropdown.go`，`ui` 仅保留样式渲染 |
| 2025-03-24 | 文本换行与 slash Enter 规则下沉：删除 `ui/view_wrap.go`（迁至 `internal/textwrap`），新增 `internal/slashflow/enter_key.go` 统一 slash Enter 动作决策，`ui` 侧只做分发执行 |
| 2025-03-24 | main 输入流继续下沉：新增 `internal/maininput`（capture/sync/main-enter/prompt），删除 `ui/update_main_key.go`，`update_keymsg.go` 改为薄编排并复用下沉逻辑 |
| 2025-03-24 | `ui` 包内测试收口：`model_test.go` 删除已被 `approvalflow`/`approvalview`/黑盒链路覆盖的重复用例，仅保留 `ui` 壳层可见性断言，降低 `ui` 目录体量与重复维护成本 |
| 2025-03-24 | 会话上下文继续去 UI 化：移除 `RuntimeContextState.CurrentSessionPath` 与 slash options provider 的 `currentSessionPath` 参数；`/sessions` 候选排除当前会话改由 `internal/session` 内部状态维护，`ui` 不再显式传递会话路径/标识 |
| 2025-03-24 | `SessionSwitchChan` 下线：删除 `ui.UIPorts.SessionSwitchChan`、`hostloop` 对应 multiplex 分支与依赖字段；`/sessions <id>` 改为提交命令后在 submit loop 内执行 session 切换 |
| 2025-03-24 | `SessionSwitchedMsg` 去 payload：移除 `SessionSwitchedMsg.Path`，host loop 改为更新 `internal/session` 内部会话状态后发送空事件，UI 不再承载/传递会话路径信息 |
| 2025-03-24 | slash 注册下沉：`/config*`、`/cancel`、`/q`、`/sh`、`/help`、`/config auto-run` 从 `ui` 迁到 `run/feature` 包；删除 `ui.registerSlashExact` 别名 |
| 2025-03-24 | `internal/ui` 测试镜像重组：`feature_registry_test.go` 拆分为 remote/configllm、skill、session、slash-exact 多文件，主文件仅做汇总 init |
| 2025-03-24 | P2：`Model` 再收敛 `Layout`/`Startup`/`Approval`；新增 `hasPendingApproval`、`contentWidth`、`OpenOverlay`、`CloseOverlayVisual` 等 helper 并替换重复逻辑 |
| 2025-03-24 | P2：`Model` 状态分组（`ConfigLLM`/`RemoteAuth`/`AddRemote`/`AddSkill`/`UpdateSkill`/`PathCompletion`）+ `UIPorts` |
| 2025-03-24 | 集中 overlay 关闭复位：`ApplyOverlayCloseFeatureResets`（移除 remote/skill/configllm 分散 hook） |
| 2025-03-24 | P1：`RegisterTitleBarFragmentProvider` + `view_approval_card.go`；交接文档 §4/§5 同步 |
| （待填） | 初版：registry、overlay close hook、view 文件拆分、`SlashRunUsageOption`、import 循环说明 |

---

## 10. 下一阶段方向：Host Bus + Controller + UI 控件化（2026-03-25）

本节记录当前重构共识，用于后续会话直接续接。

### 10.1 目标

- 收敛现有多路 channel / setter 接线，建立单一「主机事件总线（Host Bus）」。
- 引入中控（Controller）作为编排层，统一处理输入路由：slash 命令与 AI 对话主路径。
- 将 `internal/ui` 进一步收敛为壳层：提供少量控制接口 + 可复用控件，不承载业务分支。
- 在不改变 HIL 安全边界与审计语义前提下，降低跨包耦合与路径追踪成本。

### 10.2 共识架构（高层）

1. 用户输入先进入 Host Bus（事件形式，不直接调用具体业务包）。
2. Controller 监听输入事件并路由：
   - slash 开头 -> 对应 slash handler；
   - 非 slash -> AI 流程（runner/agent）。
3. UI 通过窄接口暴露控制能力（如 header 状态、对话框显示/关闭、消息追加）。
4. UI 可复用能力抽象为控件（dialog/dropdown/selector 等），由功能模块组合调用。

### 10.3 关键边界约束（防止重构后再次失控）

- Host Bus 传递领域事件，不传样式细节与布局参数。
- Controller 只做编排与状态推进，不承载具体业务实现。
- UI 控件层不得反向依赖 `agent` / `run` / `remote` 等业务包，避免循环依赖。
- 保持 HIL 语义不变：命令执行前审批与敏感确认流程不可绕过。
- 迁移期间优先适配层方案，避免大面积目录重排导致行为回归。

### 10.4 建议的最小事件集合（初稿）

- `UserSubmitted{text}`
- `SlashRequested{text}`
- `AIRequested{text}`
- `ApprovalRequested{command,...}`
- `ApprovalResolved{choice}`
- `SensitiveConfirmationRequested{command}`
- `SensitiveConfirmationResolved{choice}`
- `CommandExecuted{command,result,...}`
- `RemoteStatusChanged{active,label}`
- `ConfigReloaded{}`

注：命名可调整，但事件职责应保持「业务语义优先，UI 表现后置」。

#### 10.4.1 稳定 Kind 与 §10.4 语义标签（已落地）

总线 **`hostbus.Kind`** 的字符串值保持稳定（测试与日志）；架构草稿里的英文名称由 **`Kind.SemanticLabel()`** 给出，便于对照与追踪：

| `Kind` 常量 | `SemanticLabel()` |
|-------------|-------------------|
| `KindSessionNewRequested` | `SessionNewRequested` |
| `KindSessionSwitchRequested` | `SessionSwitchRequested` |
| `KindUserChatSubmitted` | `AIRequested`（非 slash 主对话进入 LLM 路径） |
| `KindConfigUpdated` | `ConfigReloaded` |
| `KindCancelRequested` | `CancelRequested` |
| `KindExecDirectRequested` | `ExecDirectRequested` |
| `KindRemoteOnRequested` | `RemoteConnectRequested` |
| `KindRemoteOffRequested` | `RemoteDisconnectRequested` |
| `KindRemoteAuthResponseSubmitted` | `RemoteAuthAnswerSubmitted` |
| `KindApprovalRequested` | `ApprovalRequested` |
| `KindSensitiveConfirmationRequested` | `SensitiveConfirmationRequested` |
| `KindAgentExecEvent` | `CommandExecuted` |
| `KindAgentUnknown` | `AgentUIPassthrough` |
| `KindLLMRunCompleted` | `LLMRunCompleted` |
| `KindSlashRequested` | `SlashRequested`（TUI 调用 registry handler **之前**，不经 `SubmitChan`） |
| `KindSlashEntered` | `SlashEntered`（handler 已成功返回后） |

**`SlashRequested` 与 submit 路径**：主对话 `SubmitChan` 仍只承载 `/new`、`/sessions …` 与普通 LLM 文本；其余 slash 不经 `SubmitChan`，但在执行前后分别产生 `KindSlashRequested` / `KindSlashEntered`（见 `InputPorts.SlashRequestChan` / `SlashTraceChan`）。

**可追踪摘要**：**`hostbus.Event.RedactedSummary()`** 生成单行脱敏描述（不含远程认证密码等），供日志 / 自定义 metrics 使用。

**观测钩子**：

- **`hostbus.New(cap, hostbus.WithPublishHook(fn))`**：`fn(e, accepted)`，`Publish` 队列满时 `accepted == false`。
- **`hostcontroller.Options.OnEventDispatch`**：中控从总线取出事件后、执行 handler 表之前调用（可在此打结构化日志或 metrics）。

**中控分发**：**`hostEventHandlers`**（`internal/hostcontroller/event_dispatch.go`）表驱动各 `Kind`，未知 `Kind` 忽略（与原先无 `default` 的 `switch` 一致）。

### 10.5 渐进迁移顺序（低风险）

1. 定义 Bus 接口与事件类型，先接入适配层，不改现有主逻辑。
2. 将「用户输入 -> slash/AI 路由」迁移到 Controller。
3. 将审批、敏感确认、远程切换等异步流程迁移到 Bus 事件链。
4. 最后抽 UI 控件并替换分散实现（dialog/dropdown 等）。

### 10.5.1 已落地（实现快照）

- `cli.Run` 创建 `hostbus.Bus` 与 `hostbus.InputPorts`，`runnermgr` 的 `UIEvents` 接入 `ports.AgentUIChan`；`hostwiring.BindSendPorts` 将通道挂到 `hostapp.Runtime`，同一 `*Runtime` 作为 **`ui.Model.Host`** 注入 TUI（allowlist getter、remote 镜像、ConfigLLM 首次 layout 等仍在 `Runtime` 上）。
- `hostcontroller.Controller` 单 goroutine 消费 Bus 事件；LLM 运行在独立 goroutine 中，完成时投递 `KindLLMRunCompleted`，避免阻塞导致 `/cancel` 失效。
- `uipresenter.Presenter` 封装发往 TUI 的 `tea.Msg`，`hostcontroller` 不再散落 `ui.*` 结构体字面量（`DispatchAgentUI` 统一映射 Agent 侧 payload）。
- `internal/cli/hostloop` 包已删除（原 multiplex / submit / agent_ui 等逻辑已迁入 controller + uipresenter）。

### 10.6 完成判据（可验证）

- `internal/cli/run.go` 中全局 setter / 多路 channel 接线数量显著下降。
- 从用户输入到执行回显的主路径可由单一事件链追踪（而非跨多处分支猜测）。
- UI 层新增功能优先通过控件组合，不再在 `update_*` 中扩散重复绘制逻辑。
- 现有 e2e/关键黑盒路径通过（slash、AI、审批、敏感确认、remote）。

### 10.8 五阶段计划执行状态（2026-03-25）

与「先 e2e 打底 → slash 上收 → 总线收口 → UI 控件化」对应的五轮中，本轮仓库内**已落地**与**仍待后续 PR**如下：

| 阶段 | 目标 | 状态 |
|------|------|------|
| **1** | e2e 可验证、不因错误假设长时间无输出 | **已做**：`interactive` 补充 `_ "internal/run"`、`_ "internal/remote"`（与 `session` 并列），真实二进制具备 slash 注册；`cases` 期望与 `KeyConfigHint` 对齐；`ReadUntil`/`ReadUntilAny` 按墙钟截止收紧读片段时间并识别 `os.ErrDeadlineExceeded`；`internal/e2e/README.md` 写明 `-timeout` 与排障。 |
| **2** | slash 与总线/中控衔接（试点） | **已做（观测路径）**：`KindSlashRequested`（handler 前）+ `KindSlashEntered`（成功后）；`SlashRequestChan` / `SlashTraceChan`；`Host.RequestSlashDispatch` / `TraceSlashEntered`；`hostcontroller` 占位 handler；语义标签与 `RedactedSummary` 已覆盖。解析与执行仍在 TUI/registry。 |
| **3** | slash 主路径迁入 Controller | **部分做（观测加深）**：总线已可见「尝试→成功」对；**未做**：经 `SubmitChan` 统一路由、Controller 驱动 `tea.Cmd`、结构化 payload（含下拉选中），见 `docs/host_bus_audit.md`。 |
| **4** | 审批/敏感/远程等待总线链审计 | **部分做**：`docs/host_bus_audit.md` 为只读路径表；审批/敏感/远程事件链此前已在目标 2 落地，此处为对照清单。 |
| **5** | UI 控件化（dialog/dropdown） | **部分做**：居中 modal 的 lipgloss 布局抽至 `internal/ui/widget`（`RenderCenteredModal`），`view_overlay.go` 调用；dropdown/dialog 组件化仍待后续。 |

**回归命令**：`go test ./internal/e2e/... -timeout=60s -count=1`（勿依赖默认 10m 超时判断健康）。

#### 10.8.1 三轮收尾计划（余下阶段 3～5）

以下将 **阶段 3**（slash 主路径与中控协作）、**阶段 4**（审计闭环）、**阶段 5**（可复用 UI 控件）拆成三轮；每轮可独立合并，**验收以本表与 `go test ./...`（含 e2e）为准**。

| 轮次 | 侧重点 | 交付物（建议） | 验收 |
|------|--------|----------------|------|
| **第 1 轮** | **阶段 4 收口** + **阶段 3 的设计与契约** | 更新 `docs/host_bus_audit.md`：审批 / 敏感 / 远程 / 主对话在 **Bridge → Bus → Controller → uipresenter → TUI** 上的顺序与责任边界（可表格 + 简短序列说明）；列出与 §10.6 完成判据的对照缺口（若有）。另起短节或 ADR：`SlashSubmitPayload`（或等价字段）的**形状**（含 `raw_line`、`selected_index` 等）、与 `hostroute.ClassifyUserSubmit` 的衔接、**禁止**与现有 `SubmitChan` 上 `/new`、`/sessions` 语义冲突的约定。代码侧可仅增加 **类型 + 注释**（无行为变更）或零改动。 | 文档可指导实现；`go test ./...` 通过。 |
| **第 2 轮** | **阶段 3 最小闭环** | 在 **不破坏** HIL 与 registry 的前提下，实现 **一条** 可验证的「结构化 slash 意图 → 总线或专用 channel → Controller → `EnqueueUIBlocking` → TUI `Update` 执行既有 `dispatchSlash*`」路径（例如先覆盖 **主 Enter 的某一类** 或 **仅非下拉歧义命令**）；`RequestSlashDispatch` / `TraceSlashEntered` 与新区间的关系写清（避免重复事件或文档化双写策略）。 | 黑盒或 e2e 覆盖该路径；**既有** slash 路径与 `/new`、`/sessions`、LLM 提交**行为不变**；全量测试通过。 |
| **第 3 轮** | **阶段 5 控件化** | 将 **slash 下拉**（`view_slash_dropdown.go` 与 `slashview` 的衔接处）中 **lipgloss 行/布局** 抽到 `internal/ui/widget`（或 `widget` 下子文件），`view` 仅组装数据；若篇幅允许，将 **审批/敏感底部数字选项行**（`view_choices.go` 一带）的重复样式迁入同一 widget 层。 | 视觉与行为与抽离前一致（`ui` 测试 + e2e）；`go test ./...` 通过。 |

**说明**：若第 2 轮结束后阶段 3 仍仅覆盖部分路径，应在 §10.8 主表中将阶段 3 标为 **「已做（范围：…）」** 并保留余量条目，避免与「全量 Enter 上收」混淆。

### 10.7 总线语义化（目标 2）：刻意未做项与后续计划

**现状（避免与旧方案混淆）**：`internal/hostapp` 已不再提供进程级 `Install` / 包级 `Submit` 等全局句柄；生产路径由 **`interactive.Run`** 构造 **`*hostapp.Runtime`**，经 **`hostwiring.BindSendPorts`** 接线后，以 **`hostapp.Host` 接口** 注入 **`ui.Model.Host`**。测试使用 **`hostapp.Nop()`** 或自建 `*Runtime`。若仍提「去掉全局」，所指应为下表 **§10.7.1** 中「比接口更进一步」的收口，而非尚未删除的 `Install`（已删除）。

以下条目在目标 2 落地时有意识地**未实现**，与「语义可对表、脱敏摘要、观测钩子、中控 handler 表」正交；此处记录**原因**与**若要做时的推荐顺序**，供后续里程碑引用。

| 项 | 未做原因 | 后续怎么做（建议） |
|----|----------|-------------------|
| **统一 `Payload` / 按 Kind 拆独立事件结构** | 属于数据模型重塑，需改动所有 `Publish`、`BridgeInputs`、controller 消费点，回归面大；当前扁平 `Event` + `RedactedSummary` 已满足可追踪与日志安全。 | 单独立项（例如 `refactor(hostbus): typed event payloads`）：先定义 `EventV2` 或 `Payload` 接口与适配函数；按 Kind 分批迁移（先只读路径或测试双写），最后一刀切换 `Bus` 泛型/联合类型并删旧字段。 |
| **将 `Kind` 的字符串常量改为与 §10.4 草稿字面一致** | 现有字符串可能已被外部日志、监控或约定依赖；改名是破坏性变更。 | 保持 wire 值不变；若强需求对齐，仅增加**文档别名表**或监控侧映射；若必须改字符串，应版本化总线或长期保留旧值 `const` 作为兼容别名。 |
| **`Slash` 经主对话 `SubmitChan` 与 LLM 同级分类** | 当前除 `/new`、`/sessions …` 外，slash 不经 `SubmitChan`（避免丢失下拉选中索引等上下文）。`KindSlashRequested` / `KindSlashEntered` 仅经专用 channel 观测。 | 若统一 Enter→Submit：在 `hostroute` 引入结构化提交（或独立 `SlashSubmit` channel），`BridgeInputs` 映射到 `KindSlashRequested` 并由 Controller 回灌 TUI 执行 `tea.Cmd`。 |
| **生产路径默认接入 slog / metrics / trace** | 观测后端与采样策略属运维与产品决策；库内写死易产生噪音、性能与隐私风险（即便有脱敏摘要，字段策略仍需谨慎）。 | 在 `interactive.Run`（或配置层）按 **flag / 环境变量** 装配 `hostbus.WithPublishHook` 与 `Options.OnEventDispatch`；日志字段统一走 **`Event.RedactedSummary()`**，敏感路径禁止直接打印整 struct。 |
| **运行时动态注册 controller handler（插件式）** | 当前 handler 均在 `hostcontroller` 包内，`hostEventHandlers` 静态表已满足可读性与单测；动态注册引入顺序、重复注册与测试隔离成本。 | 若出现「第三方扩展命令」或「测试注入 mock handler」等硬需求：再抽象 `RegisterKind(Kind, Handler)`（需 mutex + 启动期冻结或只读 map），并文档化优先级规则。 |

**与 §10.5 的衔接**：§10.5 第 2～4 步（slash 路由上收、更多异步流程事件化、UI 控件化）推进时，应复查上表各行是否仍适用；**Submit 路径上的 slash 分类** 与 **Payload 分型** 宜在「路由上收」方案确定后再动，避免两次大改。

#### 10.7.1 更彻底一层（可选，未排期）

与目标 1/2 已交付能力相比，下列工作属于**进一步收紧类型与领域边界**，非当前缺陷；仅当团队希望「类型上只认一种生产实现」或「总线命名与运维语义完全同源」时再立项。

| 项 | 当前状态 | 未做或停留在此的原因 | 若以后要做（建议） |
|----|----------|----------------------|-------------------|
| **`ui.Model` 显式持有 `*hostapp.Runtime`（或等价具体句柄）** | `Model` 字段为 **`Host hostapp.Host`**；生产传入 `*Runtime`，测试传入 **`Nop()`**。调用链上已无「零参数取全局 Host」。 | 接口保留 **测试替身** 与 **将来第二实现** 的空间；把 Model 绑死到 `*Runtime` 会扩大 `ui` 对 `hostapp` 具体类型的耦合，且 `Nop` 路径需另设字段或可空指针。 | **方案 A**：维持接口，仅文档约定「生产必为 `*Runtime`」。**方案 B**：增加 `func (m Model) Runtime() *Runtime`（`Host` 类型断言，非 Runtime 返回 nil）供确需具体类型的调用方使用。**方案 C**：`NewModel` 重载为 `NewModelWithRuntime`，测试仍用 `NewModel(..., Nop())`；避免在 feature 包中扩散类型断言。 |
| **总线 Remote：合并 on/off/auth 等为更贴近领域的 Kind 命名** | 仍为 **`KindRemoteOnRequested` / `KindRemoteOffRequested` / `KindRemoteAuthResponseSubmitted`** 三路；`BridgeInputs` 与 `InputPorts` 仍对应三条 channel。 | 属于 **产品语义与可观测粒度** 取舍：合并后 trace 更粗、单事件字段更胖；拆分则与现有 executor/overlay 交互阶段一一对应，改动成本相对低。 | 先固定 **领域状态机**（连接中 / 已连接 / 断开 / 等待认证）再命名事件；在 **Bridge** 层将多 channel 收敛为 **一种 `Kind` + 判别子段**（或 `RemotePhase`），`hostcontroller` 单 handler 内分支；**`SemanticLabel` / `RedactedSummary`** 同步扩展，避免日志语义倒退。 |

**说明**：「单包集中」指 **`hostapp` 仍集中实现 Host 行为**，这是刻意边界，不等于进程级全局；若未来要拆包，需单独做依赖倒置（例如 `Host` 接口下移到更小模块），与上表两行独立。
