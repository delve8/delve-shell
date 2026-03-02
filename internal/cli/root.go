package cli

import (
	"github.com/spf13/cobra"
)

// Execute 运行根命令并返回错误码语义的 error（非 nil 表示应 exit 1）
// 直接执行 TUI，无 run/help/completion 子命令；帮助在 TUI 内用 /help 查看。
func Execute() error {
	root := &cobra.Command{
		Use:          "delve-shell",
		Short:        "AI 辅助运维执行命令，经用户确认后再执行",
		RunE:         runRun,
		SilenceUsage: true, // 出错时只显示错误信息，不打印 Usage/Flags
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpCommand(nil)
	return root.Execute()
}
