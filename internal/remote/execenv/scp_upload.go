package execenv

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// scpUpload sends a local file to the remote host using the classic SCP protocol (sink mode:
// remote runs "scp -t <dir>") over the existing SSH connection. This avoids SFTP and does not
// invoke the local scp binary.
func scpUpload(ctx context.Context, client *ssh.Client, localPath, remotePath string) error {
	if client == nil {
		return errors.New("ssh client is nil")
	}
	fi, err := os.Stat(localPath)
	if err != nil {
		return err
	}
	size := fi.Size()

	remotePath = filepath.ToSlash(remotePath)
	remoteDir := path.Dir(remotePath)
	base := path.Base(remotePath)
	if remoteDir == "" || base == "" || base == "." {
		return fmt.Errorf("invalid remote path: %s", remotePath)
	}
	// SCP expects a single path segment for the filename in the C line.
	if strings.Contains(base, "\n") {
		return fmt.Errorf("remote basename contains newline")
	}

	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer f.Close()

	sess, err := client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()

	stdin, err := sess.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := sess.StdoutPipe()
	if err != nil {
		return err
	}
	var stderrBuf strings.Builder
	sess.Stderr = &stderrBuf

	// Remote shell parses the line; %q matches OpenSSH-friendly quoting for paths with spaces.
	cmd := fmt.Sprintf("scp -qt %q", remoteDir)
	if err := sess.Start(cmd); err != nil {
		return fmt.Errorf("scp session start: %w", err)
	}

	errCh := make(chan error, 1)
	go func() {
		defer stdin.Close()
		header := fmt.Sprintf("C0644 %d %s\n", size, base)
		if _, err := io.WriteString(stdin, header); err != nil {
			errCh <- err
			return
		}
		if err := readSCPResponse(stdout); err != nil {
			errCh <- err
			return
		}
		if _, err := io.Copy(stdin, f); err != nil {
			errCh <- err
			return
		}
		if _, err := stdin.Write([]byte{0}); err != nil {
			errCh <- err
			return
		}
		if err := readSCPResponse(stdout); err != nil {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	var runErr error
	select {
	case <-ctx.Done():
		_ = sess.Signal(ssh.SIGINT)
		runErr = ctx.Err()
	case runErr = <-errCh:
	}

	waitErr := sess.Wait()
	if runErr != nil {
		return runErr
	}
	if waitErr != nil {
		msg := strings.TrimSpace(stderrBuf.String())
		if msg != "" {
			return fmt.Errorf("scp: %w: %s", waitErr, msg)
		}
		return fmt.Errorf("scp: %w", waitErr)
	}
	return nil
}

// readSCPResponse reads one SCP status byte from the remote (0 = ok). Non-zero means
// warning or error with a trailing message line.
func readSCPResponse(r io.Reader) error {
	var buf [1]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return err
	}
	if buf[0] == 0 {
		return nil
	}
	br := bufio.NewReader(r)
	line, err := br.ReadString('\n')
	if err != nil {
		return fmt.Errorf("scp status %d: %w", buf[0], err)
	}
	msg := strings.TrimSuffix(line, "\n")
	return fmt.Errorf("scp remote: %s", msg)
}
