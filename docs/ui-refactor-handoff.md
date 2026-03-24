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
- **后果**：`/run` 的 `RegisterSlashSelectedProvider`（及同类）在 **`feature_registry_test.go` 的 `init()` 里必须镜像一份**，并与 `ui.SlashRunUsageOption` 常量保持语义一致。
- **架构判断**：循环依赖应视为架构告警，不应长期靠测试镜像兜底。  
- **后续去重方向**：  
  1) 引入 **registry core**（第三包，脱离 `ui.Model` 直接依赖）；或  
  2) 将 `internal/ui` 测试逐步迁移到 `package ui_test`（减少对未导出符号耦合）。  
  当前两条可并行小步推进，优先保持行为稳定与可回滚。

### 3.2 `feature_registry_test.go` 的定位

- **目的**：`go test ./internal/ui` 时不 import `remote` / `session` / `configllm` / `skill` / `run`（避免循环或过重依赖）。
- **内容（现状）**：测试镜像已从单文件拆分为按领域文件：  
  - `feature_registry_remote_configllm_test.go`  
  - `feature_registry_skill_test.go`  
  - `feature_registry_session_test.go`  
  - `feature_registry_slash_exact_test.go`  
  `feature_registry_test.go` 保留汇总入口（init 调度），降低单文件耦合。
- **Overlay 复位**：已迁至 `internal/run` 注册侧统一挂载 `RegisterOverlayCloseHook`；`ui` 仅执行 hook，不持有业务复位实现。`internal/ui` 测试通过 mirror 保持隔离。

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
- Overlay 关闭：`internal/ui/update_overlay_key.go` + `internal/ui/overlay_close_feature_reset.go`  
- 测试替身：`internal/ui/feature_registry_test.go`  
- CLI 接线：`internal/cli/run.go`（空白 import 列表）  
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
| 2025-03-24 | slash 注册下沉：`/config*`、`/cancel`、`/q`、`/sh`、`/help`、`/config auto-run` 从 `ui` 迁到 `run/feature` 包；删除 `ui.registerSlashExact` 别名 |
| 2025-03-24 | `internal/ui` 测试镜像重组：`feature_registry_test.go` 拆分为 remote/configllm、skill、session、slash-exact 多文件，主文件仅做汇总 init |
| 2025-03-24 | P2：`Model` 再收敛 `Layout`/`Startup`/`Approval`；新增 `hasPendingApproval`、`contentWidth`、`OpenOverlay`、`CloseOverlayVisual` 等 helper 并替换重复逻辑 |
| 2025-03-24 | P2：`Model` 状态分组（`ConfigLLM`/`RemoteAuth`/`AddRemote`/`AddSkill`/`UpdateSkill`/`PathCompletion`）+ `UIPorts` |
| 2025-03-24 | 集中 overlay 关闭复位：`ApplyOverlayCloseFeatureResets`（移除 remote/skill/configllm 分散 hook） |
| 2025-03-24 | P1：`RegisterTitleBarFragmentProvider` + `view_approval_card.go`；交接文档 §4/§5 同步 |
| （待填） | 初版：registry、overlay close hook、view 文件拆分、`SlashRunUsageOption`、import 循环说明 |
