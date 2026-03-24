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
| `RegisterOverlayCloseHook` | **任意** overlay 关闭时复位业务字段（Esc / 程序化 close）；生产侧由 `ui.ApplyOverlayCloseFeatureResets` 统一注册一次 |
| `RegisterMessageProvider` | `tea.Msg` 的优先处理（在 `ui` 默认 switch 之前） |

### 2.3 与 `update_slash_*` 的关系

- `dispatchSlashExact` / `dispatchSlashPrefix` 遍历 registry；**具体命令实现不在 `update_slash_exact_entries.go` 里堆业务**（剩余多为通用 `/config` 等）。
- `handleSlashSelectedFallback` 只跑 `slashSelectedProviders`，内置硬编码应趋近于零（当前 `/run` 在 `internal/run`，`/skill` 在 `internal/skill`）。

---

## 3. 硬性约束与已知的「不能简单 import」

### 3.1 `internal/ui` 测试包 vs `internal/run` 的 **import cycle**

- **事实**：`internal/run` → `internal/ui`；若 `internal/ui` 的 `*_test.go` 再 `_ import "delve-shell/internal/run"`，则形成 **`ui`（测试）→ `run` → `ui`**，Go 禁止。
- **后果**：`/run` 的 `RegisterSlashSelectedProvider`（及同类）在 **`feature_registry_test.go` 的 `init()` 里必须镜像一份**，并与 `ui.SlashRunUsageOption` 常量保持语义一致。
- **后续若要去重**：需要 **第三包**（仅承载「注册表 + 弱类型回调」、或把 `Model` 换成接口）打破环；工作量大，属 P2。

### 3.2 `feature_registry_test.go` 的定位

- **目的**：`go test ./internal/ui` 时不 import `remote` / `session` / `configllm` / `skill` / `run`（避免循环或过重依赖）。
- **内容**：测试用 `registerSlashExact`、MessageProvider 替身、`RegisterSlashOptionsProvider`（sessions）、`RegisterSlashSelectedProvider`（skill、/run 镜像）、**`RegisterTitleBarFragmentProvider`**（remote 标题镜像）。
- **Overlay 复位**：`internal/ui/overlay_close_feature_reset.go` 中 **`ApplyOverlayCloseFeatureResets`** + `init()` 单次 `RegisterOverlayCloseHook`；新增业务 overlay 字段时只改该函数（及对应 overlay 打开逻辑），**不再**在 remote/skill/configllm 各写一份 hook。

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

3. **`update_slash_exact_entries.go` 审计**（已核对）  
   - 现状条目均为壳层：`/config`、`/config show`、取消与重载、`/q`/`/sh`、allowlist 更新；无进一步下沉到 feature 包的必要；与 `RegisterSlashExact` 分工一致。

### P2 — 中风险、牵涉 Model

4. **`internal/ui/model.go` 字段分包**（阶段性已落地）  
   - 已完成：`ConfigLLM`、`RemoteAuth`、`AddRemote`、`AddSkill`、`UpdateSkill`、`PathCompletion`、`RunCompletion`、`Context`、`Interaction`、`Overlay`、`Layout`、`Startup`、`Approval` 收敛为嵌套状态结构；宿主通信端口收敛为 `UIPorts`。  
   - 结论：**状态仍留在 `ui.Model`**，feature 包通过 `Register*` + 约定字段协作。  
   - **Bubble Tea 约束**：子 `textinput.Model` 的 `Update` 仍在 `ui` 或 `overlay_key` 链路上，后续继续拆分时别破坏 `tea.Model` 更新顺序。

5. **打破 `ui` 测试与 `run` 的循环（可选）**  
   - 例如：`package slashreg` 仅含 `[]func()` 与弱类型注册——易脏，**仅当测试重复成为明显负担时再做**。

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
| 2025-03-24 | P2：`Model` 再收敛 `Layout`/`Startup`/`Approval`；新增 `hasPendingApproval`、`contentWidth`、`OpenOverlay`、`CloseOverlayVisual` 等 helper 并替换重复逻辑 |
| 2025-03-24 | P2：`Model` 状态分组（`ConfigLLM`/`RemoteAuth`/`AddRemote`/`AddSkill`/`UpdateSkill`/`PathCompletion`）+ `UIPorts` |
| 2025-03-24 | 集中 overlay 关闭复位：`ApplyOverlayCloseFeatureResets`（移除 remote/skill/configllm 分散 hook） |
| 2025-03-24 | P1：`RegisterTitleBarFragmentProvider` + `view_approval_card.go`；交接文档 §4/§5 同步 |
| （待填） | 初版：registry、overlay close hook、view 文件拆分、`SlashRunUsageOption`、import 循环说明 |
