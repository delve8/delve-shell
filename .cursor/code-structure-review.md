# delve-shell 代码结构问题梳理与组织建议

本文档记录当前代码结构层面的主要问题模式（与典型 bug 关联），并给出包/文件的推荐组织方式与依赖方向约束，作为后续重构与修复的依据。本文档聚焦结构与工程化，不讨论具体功能细节。

## 背景与目标

### 背景

当前功能已接近完成，但 bug 密度较高且存在“偶发/难复现”特征。代码上出现了典型的结构性风险信号：超大函数/超大状态机、跨层直接写配置、并发与生命周期缺少统一收口、安全边界依赖启发式解析等。

### 目标

- **降低 bug 密度**：减少竞态、状态错乱、时序不一致、难复现问题。
- **明确边界与依赖方向**：让 UI/业务/基础设施分层清晰，避免循环依赖与跨层写入。
- **提高可测试性**：将核心逻辑从 TUI 事件循环中剥离出来，使单测/集成测覆盖关键路径。
- **稳住安全边界**：allowlist、敏感判定、审批流程作为安全边界，需要可解释、可预测、可审计。

## 当前结构概览（以职责视角）

### 入口与编排

- `cmd/delve-shell/main.go`：入口。
- `internal/cli/*`：Cobra 入口与主流程（目前主要在 `internal/cli/run.go`）。

### 交互层（TUI）

- `internal/ui/*`：Bubble Tea Model、View、slash 命令解析、overlay 等。
  - `internal/ui/model.go`：状态与 Update 逻辑集中，体积很大。

### Agent 与工具

- `internal/agent/*`：Runner（LLM + tools）与 tools 实现（`execute_command`、`view_context`、skills tools）。

### 安全与审计

- `internal/hil/*`：allowlist、敏感规则与匹配。
- `internal/history/*`：会话 JSONL 记录、读取、脱敏、剪枝等。

### 执行环境与外部能力

- `internal/execenv/*`：本地/远端（SSH）命令执行器抽象。
- `internal/git/*`：skill 安装/更新相关 git 操作。
- `internal/config/*`：配置路径、读写、默认值与解析。
- `internal/i18n/*`：UI 文案。

## 结构性问题清单（问题模式 → 风险/后果）

### 1) 入口编排“上帝函数化”（`internal/cli/run.go`）

现状：`runRun` 同时承担 wiring、并发编排、状态管理、远端连接、会话切换、UI 事件转发等多类职责；并使用大量 goroutine + channel 与共享状态（`currentP/session/runner/executor` 等）。

风险与后果：

- **竞态与偶发 bug**：多个 goroutine 在无统一收口的情况下读写共享变量（例如 `currentP`、`session`），容易出现“发消息到已退出的 program”“切 session 后 UI/runner 状态不同步”等。
- **生命周期不清**：后台 goroutine 缺乏统一的 `context` 退出控制；退出/重启 TUI 时容易残留工作流。
- **难测试**：主流程耦合 UI、网络（SSH）、配置 IO、历史 IO，难以构造小颗粒测试。

### 2) `internal/ui/model.go` 巨型状态机，UI 与业务强耦合

现状：`Model` 同时保存 UI 渲染状态、业务状态、以及多条业务子流程（config llm / remote auth / add-skill / update-skill 等）的字段与状态转移。

风险与后果：

- **状态爆炸**：overlay/子流程越多，`Update` 分支越复杂，常见 bug 为状态未复位、焦点/索引错误、错误消息串台。
- **难复用/难测试**：业务逻辑绑定 Bubble Tea 消息循环，单测难覆盖，回归主要依赖 e2e。

### 3) 安全边界依赖启发式 shell “半解析器”（`internal/hil/allowlist.go` 等）

现状：allowlist、管道/链式分割、重定向检测使用启发式字符串扫描与粗切分；对引号、转义、子 shell 等语法缺少完备覆盖。

风险与后果：

- **误拦/误放**：误拦影响体验；误放触及安全边界（更严重）。
- **复杂度不可控**：继续在字符串层面补丁式支持更多 shell 语法会导致维护成本与 bug 风险持续上升。

### 4) history 读取的长行风险（`bufio.Scanner` 默认 64K 限制）

现状：`history.ReadRecent` 使用 `bufio.Scanner` 逐行读 JSONL；当单行 event 过大（长输出/长回复）时可能被截断或失败。

风险与后果：

- **历史丢失/上下文不完整**：表现为“偶发”缺事件、摘要异常、view_context 不完整，进一步放大排障难度。

### 5) 跨层写配置与刷新路径分散（`internal/ui` 与 `internal/cli` 并行介入）

现状：UI 直接 `config.Write`、同时 CLI 依赖 `configUpdatedChan` 重建 runner/刷新 allowlist；配置属于安全与行为关键数据，但当前的写入、刷新与依赖重建路径分散在多个层。

风险与后果：

- **时序 bug**：保存后刷新未及时生效、刷新时读取到不一致状态、overlay 与实际配置不一致。
- **责任边界模糊**：后续扩展配置项时容易引入更多“写配置点”，放大回归面。

## 推荐的包与文件组织方式（分层 + 依赖方向）

以下为“目标结构”，并不要求一次性迁移完成；建议从 P0 路线逐步实现。

### 依赖方向原则

- `ui` 只能依赖 **应用层接口**（例如 `app/controller` 或 `app/service`），不直接依赖 `config/history/execenv/git` 的具体实现。
- `agent` 视为应用层的一部分（或基础设施的一部分），其 tools 不直接操纵 UI，改为输出领域事件（Domain Events）或返回结构化结果，由上层决定如何展示。
- `config/history/hil/execenv/git/i18n` 作为基础包（可被上层依赖），彼此依赖尽量单向且最小化。

### 推荐分层

#### 1) `internal/app`：应用层（可测试的业务用例）

建议新增（或演进到）以下子包：

- `internal/app/controller`：面向 UI 的“控制器”，暴露纯方法：`SubmitMessage`、`ApproveCommand`、`SwitchSession`、`RemoteOn`、`RemoteOff`、`UpdateConfig` 等。
- `internal/app/service`：核心业务用例（Use Case）。例如：
  - `ChatService`：组装历史上下文、调用 runner、处理取消。
  - `ApprovalService`：审批状态机（Run/Reject/Copy/Dismiss + SensitiveChoice），与 history 记录绑定。
  - `RemoteService`：远端连接流程（含缓存策略、超时、状态事件）。
  - `SessionService`：新建/切换会话与列表摘要。
- `internal/app/events`：定义 UI 需要订阅的事件类型（例如 `RemoteStatusChanged`、`CommandExecuted`、`ConfigReloaded` 等），替代当前跨多处的 ad-hoc channel。

目标：将“流程/状态机”从 TUI 里抽出来，保证这些服务可以不依赖 Bubble Tea 进行单测。

#### 2) `internal/ui`：纯交互层

- `Model` 尽量只保留 UI 状态与“当前正在进行的交互子状态”（例如：当前 overlay 选择、输入焦点、列表索引）。
- `Update` 收到消息后，只做：
  - 更新 UI 状态；
  - 调用 `controller` 的方法触发业务动作；
  - 渲染/展示来自 `events` 的结果。

建议拆文件（示例）：

- `internal/ui/model/model.go`：仅 Bubble Tea `Model` 基础结构与主 Update 路由。
- `internal/ui/model/overlay_*.go`：各 overlay 的 UI 状态与渲染（不写 config、不做 git/ssh）。
- `internal/ui/view/*.go`：视图构建（可按 header/viewport/footer/cards 拆）。
- `internal/ui/input/*.go`：slash completion、path completion 等纯 UI 工具。

#### 3) `internal/cli`：入口 wiring 与运行时组装

- `internal/cli/run.go` 收缩为：
  - 初始化 config/root/rules；
  - 构造依赖（controller/service/repo/executor）；
  - 启动 TUI program；
  - 在退出时释放资源。

并发与 goroutine 建议集中到少数“管理器”里（见下方 P0 路线），避免入口散落多个 `go func(){...}`。

#### 4) `internal/infra`（可选）：基础设施适配层

将当前的一些“调用外部系统”的实现收口：

- `internal/infra/ssh`：封装 SSH executor 的创建、连接、关闭、错误分类。
- `internal/infra/git`：封装 go-git 的认证与 repo 操作。
- `internal/infra/storage`：history 的读写实现细节。

目标：应用层只依赖接口（ports），infra 提供 adapters，便于测试与替换。

### 通用函数/工具函数放置建议

避免“`internal/utils` 大杂烩”。推荐按语义归属放置：

- **字符串与格式化**：优先放在使用方包内（私有函数）；只有被多个包共享且稳定的，才抽到小包，例如 `internal/text`（wrap、truncate、width 计算）。
- **并发与生命周期**：集中到 `internal/app/runtime` 或 `internal/app/events`（例如 event bus、ctx 管理、singleflight/worker）。
- **命令解析与安全判断**：统一放在 `internal/hil` 或新增 `internal/shellspec`（明确支持的命令语法范围），不要散落在 `agent/tools` 与 `cli` 中各写一套。

## 重构路线图（按优先级，偏“结构先行”）

### P0：收口并发与状态，降低偶发 bug

- **引入统一 `context` 生命周期**：所有后台 goroutine 必须可被 cancel；退出时确保无残留。
- **将 `runRun` 拆成 3 个管理器**（可以先不新增目录，先在 `internal/cli` 或 `internal/app` 演进）：
  - `RunnerManager`：唯一负责创建/重建 runner（配置/allowlist/敏感规则变化触发）。
  - `ExecutorManager`：唯一负责 local/remote 切换、凭据缓存策略、连接结果事件。
  - `SessionManager`：唯一负责 session 的创建/切换/关闭与 history 追加。
- **把“发 UI 消息”集中化**：避免多处 goroutine 直接 `currentP.Send`；由单点转发，降低竞态面。

#### P0 当前落地状态（已完成）

以下内容用于保证文档与当前代码保持一致（截至本次重构）：

- **目录骨架已建立**：已创建 `internal/app/runtime/{sessionmgr,runnermgr,executormgr}`、`internal/ui/{model,view,adapter}`、`internal/infra/*` 等目录，便于逐步迁移。
- **已实现并接入 3 个 manager**（落在 `internal/app/runtime/*`）：
  - `sessionmgr.Manager`：统一 session new/switch/close；退出时关闭“当前 session”，避免切换后句柄泄漏。
  - `runnermgr.Manager`：统一 runner 构建/缓存/失效；在 session 切换、config reload、auto-run 变化时集中失效。
  - `executormgr.Manager`：统一当前 executor（local/ssh）与远端凭据缓存；`/remote off` 统一 close SSH + 清缓存。
- **`/remote on` 连接流程已收口到 `executormgr`**：
  - CLI 侧只负责解析 target/label/identityFile，并将 manager 返回结果转换成 UI 消息。
  - `executormgr.Connect(...)` 负责：缓存凭据优先、配置 key 自动尝试、plain SSH、失败后生成 `RemoteAuthPromptMsg`。
- **单测补齐**：
  - 新增 `internal/app/runtime/executormgr/executormgr_test.go`，通过注入 SSH factories 的方式覆盖核心分支（不依赖真实网络/SSH）。
- **构建/测试网络约束说明**：
  - 在当前环境中，建议在执行 Go 测试前运行 `source ~/proxy.sh` 以启用代理，否则 module 下载可能长时间阻塞。
  - 可用命令示例：
    - 编译验证：`source ~/proxy.sh && go test ./... -count=1 -run TestNonExistent`
    - 关键单测：`source ~/proxy.sh && go test ./internal/hil ./internal/history ./internal/ui ./internal/app/runtime/executormgr -count=1`
    - e2e（短模式）：`source ~/proxy.sh && go test ./internal/e2e -count=1 -short`

注：`internal/e2e` 的 PTY/TUI 用例对环境敏感（与终端、PTY、运行时资源等有关），在部分环境可能出现卡住的情况；短模式用于提供基础回归护栏，完整 e2e 建议在稳定环境中运行。

#### P0 补充落地：UI 消息串行化（已完成）

- `internal/cli/run.go` 中新增 UI 消息队列（`uiMsgChan`）与单点转发 goroutine：所有后台 goroutine 只写队列，由单点负责 `p.Send`。
- 目的：进一步降低并发 `Send` 的偶发问题（虽已用原子指针避免 data race，但并发 `Send` 仍可能导致难复现 UI 行为差异）。

### P1：拆 UI 状态机，提升可维护性

- **提取应用层 controller/service**：将 config 保存、remote 连接、skill 安装等业务动作挪出 `ui`。
- **overlay 子流程模块化**：按 overlay 拆文件与子状态，减少 `model.go` 巨型 Update。

#### P1 当前落地：配置与 skill 的应用层服务（已完成）

- `internal/app/service/configsvc`（已完成）
  - 将 `/config llm` 的 config 读写与连通性校验从 `internal/ui` 迁出。
  - 保留 base_url 自动补 `/v1` 的行为，并可注入 tester 以便单测。
  - 单测：`internal/app/service/configsvc/llm_test.go`
- `internal/app/service/skillsvc`（已完成）
  - 将 Add-skill / Update-skill / Del-skill 相关的业务入口从 `internal/ui/model.go` 抽出（UI 不再直接调用 `skills.InstallFromGit/Update/Remove`）。
  - 单测：`internal/app/service/skillsvc/skillsvc_test.go`（stub 注入方式，避免真实 git/文件系统依赖）。

- `internal/app/service/remotesvc`（已完成）
  - 将 remote 配置写入（add/update/remove）从 UI 迁出（UI 不再直接调用 `config.AddRemote/UpdateRemote/RemoveRemoteByName`）。
  - 单测：`internal/app/service/remotesvc/remotesvc_test.go`

#### P1 当前落地：拆分 `internal/ui/model.go`（已完成）

- 已将以下 overlay 的 key handling 从 `Model.Update` 中抽到独立文件，降低巨型状态机的耦合与回归面：
  - Add-skill：`internal/ui/overlay_add_skill.go`
  - Update-skill：`internal/ui/overlay_update_skill.go`

### P2：稳住安全边界（allowlist/敏感/重定向）

- **明确支持的命令形式**：建议将“自动放行”限定为可证明安全的子集；复杂 shell 语法默认走“必须审批且不自动执行”。
- **集中命令语义分析**：建立单一来源（single source of truth）的命令分析模块，避免多处重复启发式逻辑。

### P2：history 稳定性与可观测性

- 修复 JSONL 读取长行问题（替换 Scanner 或增大 buffer）。
- 对关键状态迁移（session switch / remote on/off / config reload / runner rebuild）补充结构化日志（不含敏感信息）。

#### P2 当前落地：history 长行修复（已完成）

- `internal/history.ReadRecent` 已从 `bufio.Scanner` 改为 `bufio.Reader.ReadBytes('\n')`，避免 64KB 单行上限导致的历史缺失。
- 单测：`internal/history/read_large_test.go`

## 验收与度量（建议）

- **竞态检测**：引入/常态化 `go test -race ./...`（至少在 CI 或本地定期跑）。
- **结构指标**：`internal/cli/run.go` 与 `internal/ui/model.go` 的行数与圈复杂度持续下降。
- **核心流程 e2e**：覆盖 `/config llm`、`/remote on/off`、`/sessions`、审批卡（含 sensitive 三选一）、`/run` 等路径，作为重构护栏。

## 现有文件到目标模块的映射（便于拆分与迁移）

本节将当前“混在一起”的职责映射到建议的目标模块，用于拆分时逐步迁移，避免一次性大改。

### `internal/cli/run.go`（入口编排）

建议迁移目标：

- **Session 管理**：会话创建/切换/关闭、history 追加  
  - 目标：`internal/app/service/session` 或 `internal/app/runtime/sessionmgr`
- **Runner 管理**：根据 config/rules/allowlist/sensitive 重新构建 runner（以及重载策略）  
  - 目标：`internal/app/runtime/runnermgr`（单 goroutine 串行化重建）
- **Executor 管理**：local/remote 切换、SSH 连接与凭据缓存、remote completion cache 拉取  
  - 目标：`internal/app/runtime/executormgr` + `internal/infra/ssh`（可选）
- **事件转发到 UI**：由单点将领域事件（ApprovalRequested、SensitiveConfirmationRequested、CommandExecuted、RemoteStatusChanged、ConfigReloaded、SessionSwitched）转为 Bubble Tea Msg  
  - 目标：`internal/app/events` + `internal/ui/adapter`（或 `ui` 内单文件 adapter）

拆分方式建议：先抽出 3 个 manager（仍由 `runRun` 构造并持有），再把 goroutine/chan 收敛到 manager 内部，最终 `runRun` 只保留 wiring 与启动/退出。

### `internal/ui/model.go`（TUI 状态机）

当前可观察的跨层行为：

- **UI 直接写配置并触发重载**：例如 config LLM overlay 内 `config.Write`，随后通过 `ConfigUpdatedChan` 让 CLI 侧重建 runner。  
  - 目标：改为调用 `controller.UpdateConfig(...)`，由应用层决定写入与重载，UI 只展示结果事件（成功/失败/校验结果）。
- **Remote auth / add-remote / skill 安装更新 等流程在 UI 内完成**：UI 同时承担“交互状态机”和“业务决策/调用外部能力”。  
  - 目标：UI 仅做交互状态机；业务动作（git、ssh、config 写入）迁移到应用层 service，由事件回推 UI。

拆分文件建议（先拆 UI 文件再迁业务，降低一次性变更风险）：

- `internal/ui/model/model.go`：主 Model 与 `Update` 顶层路由（只保留少量状态）
- `internal/ui/model/approval.go`：Pending/PendingSensitive 的键盘处理与渲染数据准备
- `internal/ui/model/overlay_config_llm.go`：Config LLM overlay 的 UI 状态与输入校验（不直接写 config）
- `internal/ui/model/overlay_remote_auth.go`：RemoteAuth overlay 的 UI 状态
- `internal/ui/model/overlay_skill_install.go` / `overlay_skill_update.go`：技能安装/更新 overlay UI 状态

### `internal/agent/tools.go`（工具语义与审计/安全边界）

当前特点：

- `execute_command` 同时承担：allowlist 判定、审批交互、敏感判定、执行、history 记录、结果返还策略（done vs full output）。  

建议迁移目标：

- 将 **审批状态机** 与 **history 记录策略** 抽出为应用层可复用组件（例如 `ApprovalService`），tool 仅负责与 LLM tool 协议对接与调用应用层接口。
- 将 **命令语义分析**（重定向/链式/管道/敏感路径）收敛为单一模块，避免 UI/CLI/agent 各自演进出不一致规则。

### `internal/history/read.go`（稳定性）

- `ReadRecent` 使用 `bufio.Scanner` 存在长行风险。建议尽早改造（P2）或至少配置更大 buffer（临时缓解）。该问题容易造成“上下文偶发缺失”，排障成本高。

## 目标目录树草案（结构参考）

以下仅为推荐组织方式示例，命名可按团队偏好调整，核心是分层与依赖方向。

```text
cmd/delve-shell/
  main.go

internal/cli/
  root.go
  run.go                 # 入口 wiring：构造 app + 启动 TUI

internal/app/
  controller/
    controller.go        # UI 侧调用入口（Submit/Approve/Remote/Config/Session）
  service/
    configsvc/
    skillsvc/
    chat/
    approval/
    remote/
    session/
  events/
    types.go             # 领域事件定义（UI 订阅）
    bus.go               # 事件总线/订阅（实现可先简单 channel）
  runtime/
    runnermgr/
    executormgr/
    sessionmgr/

internal/ui/
  model/
    model.go
    approval.go
    slash.go
    overlay_config_llm.go
    overlay_remote_auth.go
    overlay_skill_install.go
    overlay_skill_update.go
  view/
    view.go
    styles.go
  adapter/
    events_to_msg.go     # 将 app/events 转为 tea.Msg（单向）

internal/agent/
  runner.go
  tools.go               # tool 协议适配层，核心逻辑尽量下沉到 app/service

internal/hil/
  allowlist.go
  sensitive.go
  shellspec/             # 可选：明确支持的命令子集与解析/分类

internal/history/
  session.go
  read.go
  redact.go
  prune.go

internal/config/
  config.go
  paths.go

internal/execenv/
  execenv.go
  ssh.go

internal/git/
  auth.go
  fetch.go

internal/skills/
  skills.go

internal/i18n/
  i18n.go
```

---

## 任务清单（持续更新，目标：全部完成）

说明：本清单用于将本文档中的改造项变成可执行任务，并记录状态。每推进一阶段都更新该表。

### P0（并发与状态收口）

- [x] 引入统一 lifecycle（stop/ctx）并让后台 goroutine 可退出
- [x] 抽出 `sessionmgr` / `runnermgr` / `executormgr` 并接入 `internal/cli/run.go`
- [x] `/remote on` 连接流程收口到 `executormgr`
- [x] UI 消息串行化（单点 `p.Send`）

### P1（把业务动作迁出 UI）

- [x] `configsvc`：/config llm 读写与 check 迁出 UI，并补单测
- [x] `skillsvc`：skill install/update/remove 入口迁出 UI，并补单测
- [x] `remotesvc`：remote config 的 add/update/remove 入口迁出 UI，并补单测
- [x] 拆分 `internal/ui/model.go`（按 overlay/approval/remote/skill 等拆文件）并保持测试通过

### P2（稳定性）

- [x] `history.ReadRecent` 支持长行，并补单测

