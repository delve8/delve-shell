package tools

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"delve-shell/internal/remote/execenv"
)

// syncSkillScriptsToRemote ensures that the local scriptsDir contents are present on the remote host under remoteScriptsDir.
// Files are sent with the SCP protocol (remote scp -t) over the existing golang.org/x/crypto/ssh connection — not SFTP and not the local scp binary.
func syncSkillScriptsToRemote(ctx context.Context, executor execenv.CommandExecutor, scriptsDir, remoteScriptsDir string) error {
	if scriptsDir == "" || remoteScriptsDir == "" {
		return nil
	}
	sshExec, ok := executor.(*execenv.SSHExecutor)
	if !ok {
		return nil
	}
	info, err := os.Stat(scriptsDir)
	if err != nil || !info.IsDir() {
		return nil
	}
	if _, _, _, err := sshExec.Run(ctx, "sh -c "+quoteForSh("mkdir -p "+remoteScriptsDir)); err != nil {
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
		remoteFile := remoteScriptsDir + "/" + rel
		remoteDir := remoteScriptsDir
		if idx := strings.LastIndex(remoteFile, "/"); idx > 0 {
			remoteDir = remoteFile[:idx]
		}
		if _, _, _, err := sshExec.Run(ctx, "sh -c "+quoteForSh("mkdir -p "+remoteDir)); err != nil {
			return err
		}
		return sshExec.CopyLocalFileToRemote(ctx, path, remoteFile)
	})
}

// quoteForSh wraps s in single quotes and escapes single quotes as '\”.
func quoteForSh(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
