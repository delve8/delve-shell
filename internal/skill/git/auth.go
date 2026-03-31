package git

import (
	"bufio"
	"bytes"
	"net/url"
	"os"
	"os/exec"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// authFromURL returns an AuthMethod for the given repo URL when possible.
// HTTPS: 1) env (GITHUB_TOKEN, GITLAB_TOKEN, GIT_TOKEN); 2) git credential fill (uses credential.helper).
// SSH: SSH agent (ssh-add). Returns nil when no credentials available.
func authFromURL(repoURL string) transport.AuthMethod {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return nil
	}
	if strings.HasPrefix(repoURL, "https://") || strings.HasPrefix(repoURL, "http://") {
		if auth := tokenFromEnv(repoURL); auth != nil {
			return auth
		}
		if auth := credentialFromHelper(repoURL); auth != nil {
			return auth
		}
		return nil
	}
	auth, err := ssh.NewSSHAgentAuth("git")
	if err != nil {
		return nil
	}
	return auth
}

func tokenFromEnv(repoURL string) transport.AuthMethod {
	token := ""
	if strings.Contains(repoURL, "github.com") {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" && strings.Contains(repoURL, "gitlab.com") {
		token = os.Getenv("GITLAB_TOKEN")
	}
	if token == "" {
		token = os.Getenv("GIT_TOKEN")
	}
	if token == "" {
		token = os.Getenv("DELVE_SHELL_GIT_TOKEN")
	}
	if token == "" {
		return nil
	}
	return &http.BasicAuth{Username: "git", Password: token}
}

// credentialFromHelper runs "git credential fill" with protocol/host/path from repoURL and returns BasicAuth if the helper returns username and password.
func credentialFromHelper(repoURL string) transport.AuthMethod {
	u, err := url.Parse(repoURL)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return nil
	}
	host := u.Host
	if host == "" {
		return nil
	}
	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	if path != "" {
		path = path + ".git"
	}
	var stdin bytes.Buffer
	stdin.WriteString("protocol=" + u.Scheme + "\n")
	stdin.WriteString("host=" + host + "\n")
	if path != "" {
		stdin.WriteString("path=" + path + "\n")
	}
	stdin.WriteString("\n")
	cmd := exec.Command("git", "credential", "fill")
	cmd.Stdin = &stdin
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return nil
	}
	var username, password string
	sc := bufio.NewScanner(bytes.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		if line == "" {
			break
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			switch k {
			case "username":
				username = v
			case "password":
				password = v
			}
		}
	}
	if username == "" || password == "" {
		return nil
	}
	return &http.BasicAuth{Username: username, Password: password}
}
