package tools

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"delve-shell/internal/execenv"
)

// syncSkillScriptsToRemote ensures that the local scriptsDir contents are present on the remote host under remoteScriptsDir.
// It compares remote and local file contents and only updates when they differ. No tar/gzip or extra tools are required;
// all operations use basic sh + mkdir + cat.
func syncSkillScriptsToRemote(ctx context.Context, executor execenv.CommandExecutor, scriptsDir, remoteScriptsDir string) error {
	if scriptsDir == "" || remoteScriptsDir == "" {
		return nil
	}
	if _, ok := executor.(*execenv.SSHExecutor); !ok {
		// Local executor: nothing to sync.
		return nil
	}
	info, err := os.Stat(scriptsDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	// Ensure remote root directory exists.
	if _, _, _, err := executor.Run(ctx, "sh -c "+quoteForSh("mkdir -p "+remoteScriptsDir)); err != nil {
		return err
	}
	return filepath.WalkDir(scriptsDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(scriptsDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "" || rel == "." {
			return nil
		}
		localData, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		remoteFile := remoteScriptsDir + "/" + rel
		remoteDir := remoteScriptsDir
		if idx := strings.LastIndex(remoteFile, "/"); idx > 0 {
			remoteDir = remoteFile[:idx]
		}
		// Read remote content if file exists.
		readCmd := "if [ -f " + quoteForSh(remoteFile) + " ]; then cat " + quoteForSh(remoteFile) + "; fi"
		remoteOut, _, _, _ := executor.Run(ctx, "sh -c "+quoteForSh(readCmd))
		if remoteOut == string(localData) {
			return nil
		}
		// Create parent dir and upload file via here-doc.
		uploadBuilder := &strings.Builder{}
		uploadBuilder.WriteString("mkdir -p ")
		uploadBuilder.WriteString(quoteForSh(remoteDir))
		uploadBuilder.WriteString(" && cat > ")
		uploadBuilder.WriteString(quoteForSh(remoteFile))
		// Use a delimiter that is very unlikely to appear in scripts.
		delimiter := "EOF_DELVE_SKILL"
		uploadBuilder.WriteString(" << '")
		uploadBuilder.WriteString(delimiter)
		uploadBuilder.WriteString("'\n")
		uploadBuilder.Write(localData)
		if !strings.HasSuffix(uploadBuilder.String(), "\n") {
			uploadBuilder.WriteString("\n")
		}
		uploadBuilder.WriteString(delimiter)
		uploadBuilder.WriteString("\n")
		if _, _, _, err := executor.Run(ctx, "sh -c "+quoteForSh(uploadBuilder.String())); err != nil {
			return err
		}
		return nil
	})
}

// quoteForSh wraps s in single quotes and escapes single quotes as '\”.
func quoteForSh(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
