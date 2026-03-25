# 统一输入生命周期架构设计（草案）

本文记录下一阶段的输入主链路重构思路。目标不是继续在现有 `SubmitChan` / `Slash*Chan` 之上做局部修补，而是建立一套新的统一输入框架，再由新模块逐步接管旧路径。

## 1. 设计目标

- 普通问题与 slash 命令进入同一套输入生命周期框架。
- `slash` 不再被视为“另一套系统”，而是统一输入框架中的一种提交类型。
- `/q`、`Ctrl-C`、`Esc` 等系统控制行为单列为 control 类型，不与普通业务命令混写。
- UI、Controller、Bus、AI runner、slash handler 的边界按生命周期重新定义，避免继续按历史 channel 分叉。
- 优先建设新模块，不在旧模块内部做长周期、小步缝补式迁移。

## 2. 生命周期模型

统一输入框架分四个阶段：

1. `PreInput`
2. `Submit`
3. `Process`
4. `Output`

### 2.1 PreInput

负责输入尚未提交前的即时交互与候选准备。

普通问题：

- 回显输入字符
- 维护输入框状态

slash：

- 候选生成
- 选中索引维护
- fill-only 行为
- 输入期提示与局部 overlay 准备

说明：

- `PreInput` 是 UI 交互阶段，不应直接承载业务副作用。
- slash 的“输入期特殊行为”与“提交后的业务处理”必须分开。

### 2.2 Submit

用户按下 Enter 后，统一形成一个结构化提交对象 `InputSubmission`。

这里的统一，指统一提交协议，而不是把所有输入重新压扁成一个 `string`。

### 2.3 Process

根据 `InputSubmission.Kind` 进入不同处理器：

- `chat`：进入 AI 对话、tool 调用、HIL、审批、敏感确认等流程
- `slash`：进入 slash 命令处理器，执行如打开对话框、切换状态、配置修改、remote 操作等
- `control`：执行 cancel / quit / interrupt 等系统级控制

### 2.4 Output

处理阶段不直接拼 UI 细节，而是返回统一结果，再由输出层转换为 UI 可消费结果：

- 文本消息
- 状态变化
- overlay 开关
- 执行结果
- 审批卡
- 错误提示

## 3. 核心判断

### 3.1 统一的是生命周期，不是旧 channel

当前项目中的核心问题，不是“普通问题与 slash 是否都叫提交”，而是它们是否真的进入同一条生命周期主链。

因此下一阶段不建议继续扩展：

- `SubmitChan`
- `SlashRequestChan`
- `SlashTraceChan`

而应建立新的统一提交模型，由新模型承担输入主链路。

补充：

- 2026-03-25 当前实现已移除旧的 `SlashSubmitChan -> controller -> TUI relay` 回路。
- 2026-03-26 当前实现中，主输入框的 `Enter` 与 `Esc` 已统一先进入 lifecycle：
  - 普通输入经 `inputpreflight.Engine.OnEnter -> chatproc`
  - slash 早路径经 `inputpreflight.PlanSlashEnter -> slashproc`
  - `Esc` / `Ctrl-C` / `/q` 经 `controlproc`
- `Esc` 的优先级（overlay -> pre-input -> cancel processing）现在由 `controlproc` 统一解析，UI 不再在按键分支里分别判断。
- 仍未完全收敛的点：slash 业务执行本身短期仍通过 `slashproc` 适配到 UI 本地 registry/handler。

### 3.2 slash 的执行权短期仍可留在专用处理器

统一输入框架不等于要求 slash 业务立刻迁入当前 `controller`。

推荐顺序：

1. 先统一提交模型与处理框架
2. 再决定 slash 业务执行权长期放置位置

这样可以先解决主路径分叉问题，再处理 slash 执行位置问题。

### 3.3 control 命令单列

下列行为不视为普通 slash 业务命令：

- `/q`
- `Ctrl-C`
- `Esc`

它们属于 runtime control，应作为单独的 `control` submission 处理。

### 3.4 `Esc` 作为首选取消入口

建议将 `Esc` 作为取消当前交互或处理中 AI 请求的首选入口。

原因：

- 它在语义上更接近“中断当前状态”，而不是业务命令
- 它比输入命令再取消更快
- 它可以把取消行为从 slash 命令集合中剥离出来

兼容策略：

- `Esc` 为主入口
- UI 在 AI 处理中给出显式提示，例如 `Press Esc to cancel`

建议优先级：

1. 若存在活动 overlay，`Esc` 先关闭 overlay
2. 若存在 slash 输入期候选或选择态，`Esc` 先清理输入期状态
3. 若 AI 正在处理中，`Esc` 触发 cancel processing
4. 其他情况，`Esc` 不执行额外动作

说明：

- 这个优先级属于统一输入生命周期的一部分，不应由各 slash 命令各自决定
- 该项已于 2026-03-26 落地：`Esc` 现在统一进入 `controlproc`

## 4. 新模块策略

本轮重构不建议继续围绕旧模块一点点搬迁。

推荐方案：

- 建立新的输入生命周期模块树
- 旧模块保持可运行
- 新入口先在局部路径上跑通
- 达到最小闭环后，再切换主入口

这样做的理由：

- 避免旧模块继续叠加兼容分支
- 避免在 `ui` / `controller` / `bus` 多处同时打补丁
- 让新架构的边界从第一天开始就清晰

### 4.1 2026-03-26 当前状态

本轮第一优先级已经完成的部分：

- 主输入框 `Enter` 不再区分“普通输入走 lifecycle、slash 再回落旧执行链”的双轨提交；提交统一先走 lifecycle。
- `Esc` 不再在 `ui.handleKeyMsg` 中手写 overlay / slash / cancel 分支，而是统一交给 `controlproc` 解析。
- UI 侧保留的职责收敛为：
  - 输入期状态维护
  - slash 候选与 fill-only
  - lifecycle 结果应用
  - 现有 slash registry 的本地执行适配

尚未完成的部分：

- `controller` 已不再对 `UIActionSubmission` 基于 `RawText` 再做 submit 分类；普通 chat submission 直接进入 `KindUserChatSubmitted`，`/new` 与 `/sessions` 改为显式 UI intent。
- `host/bus.InputPorts` 中旧的 `SubmitChan` 已移除；bus 当前只保留结构化 `SubmissionChan` 以及 slash 观测通道 `Slash*Chan`。
- slash 处理器的长期执行归属（继续留在 UI 适配层，还是迁往 controller/service）还未最终收口。

## 5. 建议的新模块分层

以下为建议模块，不要求一次全部落地，但建议按这个边界建设。

### 5.1 `internal/inputlifecycletype`

职责：

- 定义统一输入生命周期的核心数据结构
- 不依赖 UI、Host、AI runner 具体实现

建议放置：

- `InputSubmission`
- `SubmissionKind`
- `SubmissionSource`
- `ProcessResult`
- `OutputEvent`
- `ControlAction`

说明：

- 这是新的最小共享类型层。
- 后续 `ui`、`controller`、slash 处理器、chat 处理器都应依赖它，而不是互相直连。

### 5.2 `internal/inputlifecycle`

职责：

- 输入主链路编排
- 接收统一提交对象
- 路由到具体处理器
- 汇总处理结果并转给输出层

它是新的输入框架壳层，不承载具体业务实现。

### 5.3 `internal/inputpreflight`

职责：

- 处理 `PreInput` 阶段逻辑
- 普通输入回显
- slash 候选/选中/fill-only 计算

说明：

- 这层偏 UI 输入语义，但不直接做最终渲染。
- 当前 `slashSuggestionContext`、早路径 Enter 计算、部分输入态分支，后续可向这层收敛。

### 5.4 `internal/inputprocess/chatproc`

职责：

- 处理 `chat` 类型 submission
- 驱动 AI、tool、HIL、审批、敏感确认等流程

说明：

- 这层应被视为统一处理框架下的一个业务处理器，而不是“默认主链之外的特例”。

### 5.5 `internal/inputprocess/slashproc`

职责：

- 处理 `slash` 类型 submission
- 承接 slash 命令执行、对话框打开、校验、状态切换等

说明：

- 短期允许内部继续调用现有 registry/handler 体系
- 但对外只暴露统一处理器接口

### 5.6 `internal/inputprocess/controlproc`

职责：

- 处理 `control` 类型 submission
- 承接 cancel / quit / interrupt / esc 等系统控制行为

说明：

- 这层将 `/q`、`Ctrl-C`、`Esc` 从普通 slash 命令集合中分离出来。

### 5.7 `internal/inputoutput`

职责：

- 把 `ProcessResult` 转换为 UI 层可消费事件
- 屏蔽处理阶段与 UI 消息格式之间的耦合

说明：

- 处理器不直接输出 Bubble Tea 细节
- 输出层统一生成 transcript、overlay、status、approval 等结果

### 5.8 `internal/inputbridge`

职责：

- 作为新旧路径之间的过渡适配层
- 将旧 `ui`、`host`、`controller`、registry 能力接入新输入生命周期模块
- 控制“新主链已接管到什么程度”

说明：

- 这是过渡模块，不是长期业务中心
- 一旦新主链完全接管，应尽量收缩甚至删除

## 6. 建议模块清单与职责拆分

推荐按下表建模块，而不是只按目录名讨论。

| 模块 | 核心职责 | 允许依赖 | 禁止依赖 |
|------|----------|----------|----------|
| `inputlifecycletype` | 生命周期核心类型与接口 | 无业务实现包 | `ui`、`host`、`runtime`、`run`、`remote` |
| `inputpreflight` | 输入期状态计算与 submission 形成 | `inputlifecycletype`、纯计算包 | `chatproc`、`slashproc`、`controlproc` |
| `inputlifecycle` | 路由与流程编排 | `inputlifecycletype`、processor 接口 | 具体 UI 渲染实现 |
| `inputprocess/chatproc` | chat 处理流程 | `inputlifecycletype`、AI/runtime 适配接口 | `slashview`、输入候选构建逻辑 |
| `inputprocess/slashproc` | slash 处理流程 | `inputlifecycletype`、registry/feature 适配接口 | AI runner 具体实现 |
| `inputprocess/controlproc` | control 处理流程 | `inputlifecycletype`、host/runtime 控制接口 | slash 候选逻辑、chat 流程细节 |
| `inputoutput` | 输出结果到 UI 事件的统一映射 | `inputlifecycletype`、UI 适配接口 | 业务处理器内部实现 |
| `inputbridge` | 新旧路径过渡接线 | 上述新模块 + 旧模块适配接口 | 成为新的常驻业务中心 |

## 7. 依赖方向规则

### 7.1 单向依赖

推荐依赖方向：

`inputlifecycletype` <- `inputpreflight` <- `inputlifecycle`

`inputlifecycletype` <- `inputprocess/*`

`inputlifecycletype` <- `inputoutput`

`inputlifecycle` <- `inputbridge`

说明：

- 类型层位于最底部
- 编排层只能依赖接口与类型，不应直接依赖 UI 具体绘制逻辑
- 输出层只消费结果，不回流业务决策

### 7.2 不允许的耦合

以下耦合在新设计里应明确禁止：

- `inputpreflight` 直接调 AI / HIL / remote 业务处理
- `chatproc` 直接访问 slash 候选或选中态
- `slashproc` 直接持有 Bubble Tea 输入组件
- `controlproc` 作为 slash handler 的一个分支存在
- `inputoutput` 重新做业务路由判断
- `ui` 继续根据旧 channel 决定走 chat 还是 slash

### 7.3 适配器边界

新模块接旧实现时，优先通过小接口适配，不直接穿透到旧大对象。

例如：

- chat 处理器通过 `ChatRuntime` 接口接 AI runner
- slash 处理器通过 `SlashExecutor` 接口接现有 registry
- control 处理器通过 `ControlRuntime` 接口接 cancel / quit / interrupt
- 输出层通过 `OutputSink` 接口接 UI / Presenter

## 8. 第一层核心类型包细化

`inputlifecycletype` 应先于其他新模块建设，并保持尽可能小。

建议首批文件：

- `submission.go`
- `result.go`
- `output.go`
- `control.go`
- `interfaces.go`

### 8.1 `submission.go`

职责：

- `SubmissionKind`
- `SubmissionSource`
- `InputSubmission`

后续如果需要扩展输入元数据，也优先加在这里，而不是散落到 processor 自定义参数里。

### 8.2 `result.go`

职责：

- `ProcessResult`
- 结果合并辅助函数

建议补充小型 helper，例如：

```go
func ConsumedResult(outputs ...OutputEvent) ProcessResult
func ErrorResult(err error, outputs ...OutputEvent) ProcessResult
```

这样后续处理器可以少写重复样板。

### 8.3 `output.go`

职责：

- `OutputEventKind`
- `OutputEvent`
- 与 transcript / overlay / status / approval 相关的 payload 类型

建议不要长期维持 `Payload any` 的无约束状态。

推荐逐步演进为：

- `TranscriptPayload`
- `OverlayPayload`
- `StatusPayload`
- `ApprovalPayload`
- `ErrorPayload`

### 8.4 `control.go`

职责：

- `ControlAction`
- `ControlSignal`
- `Esc` 优先级模型

建议定义：

```go
type ControlAction string

const (
    ControlCancelProcessing ControlAction = "cancel_processing"
    ControlCloseOverlay     ControlAction = "close_overlay"
    ControlClearPreInput    ControlAction = "clear_pre_input"
    ControlQuit             ControlAction = "quit"
    ControlInterrupt        ControlAction = "interrupt"
)
```

### 8.5 `interfaces.go`

职责：

- `SubmissionProcessor`
- `SubmissionRouter`
- `OutputAdapter`
- `PreInputEngine`

建议首批只放最稳定接口，不要一开始把适配器接口全部塞进类型层。

## 9. 关键数据结构建议

### 9.1 `InputSubmission`

建议字段：

```go
type SubmissionKind string

const (
    SubmissionChat    SubmissionKind = "chat"
    SubmissionSlash   SubmissionKind = "slash"
    SubmissionControl SubmissionKind = "control"
)

type SubmissionSource string

const (
    SourceMainEnter       SubmissionSource = "main_enter"
    SourceSlashEarlyEnter SubmissionSource = "slash_early_enter"
    SourceKeyboardSignal  SubmissionSource = "keyboard_signal"
    SourceProgrammatic    SubmissionSource = "programmatic"
)

type InputSubmission struct {
    Kind          SubmissionKind
    Source        SubmissionSource
    RawText       string
    InputLine     string
    SelectedIndex int
    ControlSignal ControlSignal
}
```

约束：

- `RawText` 是用户提交的标准化文本
- `InputLine` 保留输入期原始缓冲，主要给 slash 早路径使用
- `SelectedIndex` 仅对 slash 有意义；无值时统一为 `-1`
- `ControlSignal` 仅对 control submission 有意义

### 9.2 `ProcessResult`

建议字段：

```go
type ProcessResult struct {
    Outputs      []OutputEvent
    WaitingForAI bool
    Consumed      bool
    Err           error
}
```

说明：

- `Consumed` 表示本次 submission 是否已被处理器完整消费
- `Outputs` 描述输出层要做的事情
- `Err` 由输出层转换为用户可见错误，不直接在处理层拼 UI 文案

### 9.3 `OutputEvent`

建议不要直接复用现有 `tea.Msg` 作为统一输出协议。

建议抽象为：

```go
type OutputEventKind string

const (
    OutputTranscriptAppend OutputEventKind = "transcript_append"
    OutputOverlayOpen      OutputEventKind = "overlay_open"
    OutputOverlayClose     OutputEventKind = "overlay_close"
    OutputStatusChange     OutputEventKind = "status_change"
    OutputApprovalOpen     OutputEventKind = "approval_open"
    OutputErrorNotice      OutputEventKind = "error_notice"
)

type OutputEvent struct {
    Kind    OutputEventKind
    Text    string
    Payload any
}
```

## 10. 关键接口建议

### 10.1 提交路由接口

```go
type SubmissionRouter interface {
    Route(InputSubmission) (ProcessResult, error)
}
```

### 10.2 分类处理器接口

```go
type SubmissionProcessor interface {
    CanProcess(InputSubmission) bool
    Process(InputSubmission) (ProcessResult, error)
}
```

建议实现：

- `chatproc.Processor`
- `slashproc.Processor`
- `controlproc.Processor`

### 10.3 输出适配接口

```go
type OutputAdapter interface {
    Apply(ProcessResult) error
}
```

### 10.4 PreInput 接口

```go
type PreInputEngine interface {
    OnInputChanged(current string) PreInputState
    OnEnter(current string, selectedIndex int) (InputSubmission, bool)
}
```

说明：

- `OnEnter` 负责把输入态转换为统一 submission
- 这样 Enter 后不再走“chat 一套、slash 一套”的老分叉

## 11. 新旧模块映射建议

为避免“新架构只停留在抽象层”，建议先明确旧模块向新模块的收敛方向。

| 旧能力位置 | 未来主要归属 |
|-----------|--------------|
| `internal/ui/update_main_enter_command.go` | `inputpreflight` + `inputbridge` |
| `internal/ui/update_slash.go` 中的输入期 Enter 决策 | `inputpreflight` |
| `internal/ui` 中 chat/slash/control 入口分叉 | `inputlifecycle` |
| `internal/uiflow/enterflow` | 优先并入 `inputpreflight` 或成为其纯计算依赖 |
| `internal/host/controller/ui_actions.go` 中输入类动作翻译 | `inputbridge`，后续再收敛 |
| `SubmitChan` / `Slash*Chan` 的输入主链角色 | 过渡期由 `inputbridge` 承接，最终被统一 submission 入口替代 |
| 现有 slash registry 执行 | 过渡期由 `slashproc` 通过适配器调用 |

说明：

- 映射的重点是“新主链接管职责”，不是逐文件机械搬家
- 如果某个旧模块只剩适配价值，不应继续向里加业务

## 12. 模块责任边界

### 12.1 UI

UI 只负责：

- 输入态采集
- 候选展示
- 输出事件渲染
- 向用户展示控制提示，如 AI 处理中显示 `Press Esc to cancel`

UI 不负责：

- 决定 slash 和 chat 走哪条旧 channel
- 直接承担业务处理编排

### 12.2 Controller

Controller 后续应转型为：

- 处理统一 submission 的编排入口
- 或者作为新输入框架的一部分被替代/包裹

Controller 不应继续扩大为“所有路径的大 switch”。

### 12.3 slash 处理器

slash 处理器负责：

- slash 业务命令执行
- slash 校验
- slash 相关对话框控制

但不负责输入期候选渲染。

### 12.4 chat 处理器

chat 处理器负责：

- AI 请求
- tool 调用
- HIL
- 审批与敏感确认协作

### 12.5 output 层

output 层负责：

- 统一结果到 UI 事件的转换

不负责重新做业务判断。

## 13. 推荐落地顺序

### 第 1 步：文档与类型先行

- 固化本设计文档
- 建立 `inputlifecycletype` 包
- 先定义统一数据结构与接口

### 第 2 步：建设新入口，不碰旧业务实现

- 建立 `inputlifecycle`、`inputpreflight`
- 先让 Enter 产出 `InputSubmission`
- 旧 slash/chat 处理逻辑暂时通过适配器接入

### 第 3 步：建设分类处理器

- 建立 `chatproc`
- 建立 `slashproc`
- 建立 `controlproc`

### 第 4 步：建设输出适配层

- 引入 `ProcessResult -> OutputEvent -> UI`
- 降低处理器对 Bubble Tea 细节的直接依赖

### 第 5 步：切换主入口

- 新入口覆盖主 Enter
- 再清理旧 `SubmitChan` / `Slash*Chan` 路径

说明：

- 真正删除旧路径，应发生在新主链跑通并由测试覆盖之后
- 不建议在旧链路里持续增加“为了兼容新模型”的临时判断

## 14. 实施阶段与任务拆分

### 阶段 A：类型与约束落地

目标：

- 落地 `inputlifecycletype`
- 固化接口和依赖规则

完成标准：

- 核心类型包存在
- 依赖方向在代码里可见
- 不引入业务实现

### 阶段 B：PreInput 与统一 submission 成形

目标：

- 建立 `inputpreflight`
- 让 Enter 统一产出 `InputSubmission`

完成标准：

- 主 Enter 不再直接决定走 chat 还是 slash 老路径
- `Esc` 的优先级规则有单一实现入口

### 阶段 C：分类处理器接通

目标：

- 建立 `chatproc`
- 建立 `slashproc`
- 建立 `controlproc`

完成标准：

- 三类 submission 都能进入统一处理器接口
- `Esc`、`/q`、`Ctrl-C` 共用 control 主链

### 阶段 D：输出适配统一

目标：

- 建立 `inputoutput`
- 统一 `ProcessResult -> UI`

完成标准：

- 处理器不再直接拼大量 UI 细节
- transcript / overlay / approval / status 由输出层统一映射

### 阶段 E：新入口切主

目标：

- `inputbridge` 接通新主链
- 旧输入路径退化为兼容层

完成标准：

- 主 Enter 走统一 submission 框架
- 旧 `SubmitChan` / `Slash*Chan` 不再是输入主链的事实源

## 15. 暂不做的事

当前阶段先不做：

- 把全部 slash registry 立即迁出当前体系
- 重写全部 UI 渲染
- 一步到位替换 bus 事件模型
- 先做大规模实现，再回头补设计

## 16. 当前结论

下一阶段最重要的不是继续讨论“slash 算不算特殊命令”，而是把输入主链路正式定义为统一生命周期：

- `PreInput`
- `Submit`
- `Process`
- `Output`

在这个框架下：

- 普通问题是 `chat submission`
- slash 命令是 `slash submission`
- `/q`、`Ctrl-C`、`Esc` 是 `control submission`

后续模块设计、接口抽象、单元测试与编码，都应围绕这个统一模型展开。
