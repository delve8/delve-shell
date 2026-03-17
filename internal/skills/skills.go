package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"delve-shell/internal/config"
	"delve-shell/internal/git"
	"gopkg.in/yaml.v3"
)

// SkillMeta is the minimal skill metadata from SKILL.md front matter (name, description required by spec).
type SkillMeta struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	// Optional extensions (defaults used when empty)
	Summary          string `yaml:"summary,omitempty"`
	RiskLevel        string `yaml:"risk_level,omitempty"`
	AlwaysConfirm    bool   `yaml:"always_confirm,omitempty"`
	Scope            string `yaml:"scope,omitempty"`             // local, remote, both
	RemoteUploadDir  string `yaml:"remote_upload_dir,omitempty"` // default /tmp/
	// LocalName is the directory name under ~/.delve-shell/skills (used for /skill and /config del-skill commands).
	LocalName string `yaml:"-"`
}

// List returns skill names (subdirs of SkillsDir that contain SKILL.md). Order is undefined.
func List() ([]SkillMeta, error) {
	root := config.SkillsDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []SkillMeta
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "" || name[0] == '.' {
			continue
		}
		skillPath := filepath.Join(root, name)
		skillMD := filepath.Join(skillPath, "SKILL.md")
		if _, err := os.Stat(skillMD); err != nil {
			continue
		}
		meta, err := LoadSKILL(skillPath)
		if err != nil {
			continue
		}
		meta.LocalName = name
		if meta.Name == "" {
			meta.Name = name
		}
		out = append(out, *meta)
	}
	return out, nil
}

// LoadSKILL reads SKILL.md in skillDir and parses YAML front matter. Returns nil meta fields with defaults.
func LoadSKILL(skillDir string) (*SkillMeta, error) {
	path := filepath.Join(skillDir, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	body := string(data)
	const delim = "---"
	start := strings.Index(body, delim)
	if start < 0 {
		return &SkillMeta{}, nil
	}
	start += len(delim)
	end := strings.Index(body[start:], delim)
	if end < 0 {
		return &SkillMeta{}, nil
	}
	yamlBlock := strings.TrimSpace(body[start : start+end])
	var meta SkillMeta
	if err := yaml.Unmarshal([]byte(yamlBlock), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

// ReadSKILLContent returns the full content of SKILL.md (front matter + Markdown body) for the AI to learn usage, params, and examples.
func ReadSKILLContent(skillDir string) (string, error) {
	path := filepath.Join(skillDir, "SKILL.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ScriptsDir returns the scripts subdir for a skill (skillDir/scripts).
func ScriptsDir(skillDir string) string {
	return filepath.Join(skillDir, "scripts")
}

// ListScripts returns executable script names in skillDir/scripts (base names only). Prefers .sh; includes other non-dir files.
func ListScripts(skillDir string) ([]string, error) {
	dir := ScriptsDir(skillDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "" || name[0] == '.' {
			continue
		}
		names = append(names, name)
	}
	return names, nil
}

// SkillDir returns the full path for a skill by name (SkillsDir/name). Caller should check SKILL.md exists.
func SkillDir(name string) string {
	return filepath.Join(config.SkillsDir(), name)
}

// ScriptPath validates that scriptName is a valid script under skillDir/scripts (no path traversal) and returns the full path.
// Returns error if the file does not exist or is outside scripts/.
func ScriptPath(skillDir, scriptName string) (string, error) {
	if scriptName == "" || strings.Contains(scriptName, "/") || strings.Contains(scriptName, "..") {
		return "", os.ErrNotExist
	}
	scriptsDir := ScriptsDir(skillDir)
	full := filepath.Join(scriptsDir, scriptName)
	rel, err := filepath.Rel(scriptsDir, full)
	if err != nil || strings.HasPrefix(rel, "..") || rel == ".." {
		return "", os.ErrNotExist
	}
	if _, err := os.Stat(full); err != nil {
		return "", err
	}
	return full, nil
}

// BuildCommand builds the shell command to run scriptName from skillDir with args: cd scripts && bash scriptName args...
// Uses bash so scripts do not need execute permission. skillDir is the skill root (e.g. ~/.delve-shell/skills/foo).
func BuildCommand(skillDir, scriptName string, args []string) (string, error) {
	scriptsDir := ScriptsDir(skillDir)
	abs, err := filepath.Abs(scriptsDir)
	if err != nil {
		return "", err
	}
	cdPart := "cd " + quoteForSh(abs) + " && bash " + quoteForSh(scriptName)
	if len(args) == 0 {
		return cdPart, nil
	}
	var b strings.Builder
	b.WriteString(cdPart)
	for _, a := range args {
		b.WriteString(" ")
		b.WriteString(quoteForSh(a))
	}
	return b.String(), nil
}

// quoteForSh wraps s in single quotes and escapes single quotes as '\''.
func quoteForSh(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// skillDiscoveryDirs are conventional locations to look for SKILL.md (relative to repo root).
// Order matters: root first, then skills/, then common agent-style dirs.
var skillDiscoveryDirs = []string{
	".",
	"skills",
	"skills/.curated",
	"skills/.experimental",
	"skills/.system",
	".agents/skills",
	".agent/skills",
	".claude/skills",
}

var skipDiscoveryDirs = map[string]bool{
	"node_modules": true, ".git": true, "dist": true, "build": true, "__pycache__": true,
}

func hasSKILLMd(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "SKILL.md"))
	return err == nil
}

// findSkillDirsRecursive returns dirs under root that contain SKILL.md, up to maxDepth.
func findSkillDirsRecursive(root string, depth, maxDepth int) ([]string, error) {
	if depth > maxDepth {
		return nil, nil
	}
	var out []string
	if hasSKILLMd(root) {
		out = append(out, root)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() || skipDiscoveryDirs[e.Name()] {
			continue
		}
		sub, err := findSkillDirsRecursive(filepath.Join(root, e.Name()), depth+1, maxDepth)
		if err != nil {
			return nil, err
		}
		out = append(out, sub...)
	}
	return out, nil
}

// DiscoverSkillDir finds a single skill directory inside repoRoot (e.g. a cloned repo).
// Returns the relative path from repoRoot (e.g. "skills/foo" or "." for repo root).
// ErrNotFound if none found; error if multiple (hint to use explicit path).
var (
	ErrSkillDirNotFound  = fmt.Errorf("no skill directory found (no SKILL.md in conventional locations or below)")
	ErrSkillDirAmbiguous = fmt.Errorf("multiple skill directories found; specify path explicitly (e.g. /config add-skill <url> [ref] <path>)")
)

func DiscoverSkillDir(repoRoot string) (subpath string, err error) {
	// 1) Conventional locations first
	candidates, err := findSkillDirsInConventional(repoRoot)
	if err != nil {
		return "", err
	}
	if len(candidates) == 1 {
		return normalizeRel(candidates[0], repoRoot), nil
	}
	if len(candidates) > 1 {
		return "", ErrSkillDirAmbiguous
	}
	// 2) Recursive search
	all, err := findSkillDirsRecursive(repoRoot, 0, 5)
	if err != nil {
		return "", err
	}
	// Normalize to relative paths and dedupe
	seen := make(map[string]bool)
	var list []string
	for _, d := range all {
		rel := normalizeRel(d, repoRoot)
		if !seen[rel] {
			seen[rel] = true
			list = append(list, rel)
		}
	}
	if len(list) == 0 {
		return "", ErrSkillDirNotFound
	}
	if len(list) > 1 {
		return "", ErrSkillDirAmbiguous
	}
	return list[0], nil
}

func findSkillDirsInConventional(repoRoot string) ([]string, error) {
	var out []string
	for _, rel := range skillDiscoveryDirs {
		dir := filepath.Join(repoRoot, rel)
		if hasSKILLMd(dir) {
			out = append(out, dir)
		}
	}
	return out, nil
}

func normalizeRel(abs, repoRoot string) string {
	rel, err := filepath.Rel(repoRoot, abs)
	if err != nil {
		return "."
	}
	rel = filepath.ToSlash(rel)
	if rel == "." || rel == "" {
		return "."
	}
	return rel
}

// copyDir copies src into dst (dst is created). Contents of src go directly under dst.
func copyDir(dst, src string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			if err := copyDir(dstPath, srcPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

// safeSkillName returns true if name is safe for a skill directory (alphanumeric, hyphen, underscore).
func safeSkillName(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if r != '-' && r != '_' && !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}

// NormalizeGitURL converts "owner/repo" to "https://github.com/owner/repo.git"; leaves full URLs as-is (appends .git if missing).
func NormalizeGitURL(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	if strings.Contains(url, "://") || strings.Contains(url, "@") {
		if !strings.HasSuffix(url, ".git") {
			url = url + ".git"
		}
		return url
	}
	// owner/repo or owner/repo.git
	if !strings.HasSuffix(url, ".git") {
		url = url + ".git"
	}
	return "https://github.com/" + url
}

// NameFromGitURL derives a skill name from a git URL (last path component, strip .git).
func NameFromGitURL(url string) string {
	url = strings.TrimSpace(url)
	if url == "" {
		return ""
	}
	if idx := strings.LastIndex(url, "/"); idx >= 0 && idx < len(url)-1 {
		url = url[idx+1:]
	}
	url = strings.TrimSuffix(url, ".git")
	return safeNameFromSegment(url)
}

// nameFromPath returns the last path component for use as local skill name (e.g. "skills/foo" -> "foo").
func nameFromPath(path string) string {
	path = strings.TrimSpace(filepath.ToSlash(path))
	if path == "" || path == "." {
		return ""
	}
	if idx := strings.LastIndex(path, "/"); idx >= 0 && idx < len(path)-1 {
		path = path[idx+1:]
	}
	return safeNameFromSegment(path)
}

// safeNameFromSegment keeps only safe chars and truncates to 64 (shared by NameFromGitURL and nameFromPath).
func safeNameFromSegment(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	s = b.String()
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

// SkillSource is the git source for an installed skill (for manifest and upgrade).
type SkillSource struct {
	URL         string `json:"url"`
	Ref         string `json:"ref,omitempty"`   // branch or tag requested at install
	CommitID    string `json:"commit_id,omitempty"` // resolved commit at install (ref may move)
	Path        string `json:"path,omitempty"`      // subpath within repo (e.g. "skills/foo"); empty means repo root or unknown
	InstalledAt string `json:"installed_at"`   // RFC3339
}

type skillsManifest struct {
	Skills map[string]SkillSource `json:"skills"`
}

var manifestMu sync.Mutex

func loadManifest() (skillsManifest, error) {
	manifestMu.Lock()
	defer manifestMu.Unlock()
	path := config.SkillsManifestPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return skillsManifest{Skills: make(map[string]SkillSource)}, nil
		}
		return skillsManifest{}, err
	}
	var m skillsManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return skillsManifest{}, err
	}
	if m.Skills == nil {
		m.Skills = make(map[string]SkillSource)
	}
	return m, nil
}

func saveManifest(m skillsManifest) error {
	manifestMu.Lock()
	defer manifestMu.Unlock()
	path := config.SkillsManifestPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// GetSkillSource returns the git source for a skill if it was installed from git. ok is false for manually added skills.
// commitID is the commit at install time; ref is the branch/tag requested (ref may have moved since).
// path is the subpath within the repo (e.g. "skills/foo"); empty means repo root or unknown.
func GetSkillSource(name string) (url, ref, commitID, path, installedAt string, ok bool) {
	m, err := loadManifest()
	if err != nil {
		return "", "", "", "", "", false
	}
	s, ok := m.Skills[name]
	if !ok {
		return "", "", "", "", "", false
	}
	return s.URL, s.Ref, s.CommitID, s.Path, s.InstalledAt, true
}

// InstallFromGit fetches the repo at url (ref optional branch/tag) and writes the skill into ~/.delve-shell/skills/<name> (no .git dir).
// path: optional subpath within the repo (e.g. "skills/foo"); empty means auto-discover (conventional dirs then recursive).
// name defaults to the last path component when path is set (e.g. "skills/foo" -> "foo"), otherwise NameFromGitURL(url).
// Returns the installed skill name. If dest already exists, returns error.
// Records url/ref/commit_id in skills/manifest.json for upgrade and display.
func InstallFromGit(url, ref, name, path string) (finalName string, err error) {
	url = NormalizeGitURL(url)
	if url == "" {
		return "", os.ErrInvalid
	}
	path = strings.TrimSpace(strings.Trim(path, "/"))
	if name == "" {
		if path != "" && path != "." {
			name = nameFromPath(path)
		}
		if name == "" {
			name = NameFromGitURL(url)
		}
	}
	name = strings.TrimSpace(name)
	if !safeSkillName(name) {
		return "", os.ErrInvalid
	}
	// Load manifest early so we can detect same-source installs even when local name differs.
	m, manifestErr := loadManifest()
	if manifestErr != nil {
		// Non-fatal: continue without duplicate-source hints.
		m = skillsManifest{Skills: make(map[string]SkillSource)}
	}
	// Look for an existing skill with the same source (url + path).
	normalizedPath := path
	var existingSourceName string
	for skillName, src := range m.Skills {
		if src.URL != url {
			continue
		}
		srcPath := strings.TrimSpace(strings.Trim(src.Path, "/"))
		if srcPath == normalizedPath {
			existingSourceName = skillName
			break
		}
	}

	dest := filepath.Join(config.SkillsDir(), name)
	if _, statErr := os.Stat(dest); statErr == nil {
		// Local name already taken. If it's the same source, tell user the existing name; otherwise it's just a name conflict.
		if existingSourceName == name {
			return "", fmt.Errorf("skill source already installed as '%s'", name)
		}
		if existingSourceName != "" {
			return "", fmt.Errorf("skill source already installed as '%s'; choose another local name", existingSourceName)
		}
		return "", os.ErrExist
	}
	if err := os.MkdirAll(config.SkillsDir(), 0700); err != nil {
		return "", err
	}
	ref = strings.TrimSpace(ref)
	ctx := context.Background()
	var commitID string
	var sourcePath string
	if path != "" {
		// Explicit path: fetch only that subtree to dest
		commitID, err = git.FetchRepoTree(ctx, url, ref, dest, path)
		if err != nil {
			_ = os.RemoveAll(dest)
			return "", fmt.Errorf("%w", err)
		}
		sourcePath = path
	} else {
		// Auto-discover: fetch full repo to temp, find single skill dir, copy to dest
		tmpDir, tmpErr := os.MkdirTemp("", "delve-shell-skill-*")
		if tmpErr != nil {
			return "", tmpErr
		}
		defer os.RemoveAll(tmpDir)
		commitID, err = git.FetchRepoTree(ctx, url, ref, tmpDir, "")
		if err != nil {
			return "", fmt.Errorf("%w", err)
		}
		subpath, discoverErr := DiscoverSkillDir(tmpDir)
		if discoverErr != nil {
			return "", discoverErr
		}
		sourcePath = subpath
		src := filepath.Join(tmpDir, filepath.FromSlash(subpath))
		if err := copyDir(dest, src); err != nil {
			_ = os.RemoveAll(dest)
			return "", err
		}
	}
	installedAt := time.Now().UTC().Format(time.RFC3339)
	m.Skills[name] = SkillSource{URL: url, Ref: ref, CommitID: commitID, Path: sourcePath, InstalledAt: installedAt}
	_ = saveManifest(m)
	appendSkillAudit("install", url, ref, name)
	return name, nil
}

// Remove deletes a skill directory. name must be safe (no path traversal).
// Removes the entry from skills/manifest.json if present.
func Remove(name string) error {
	name = strings.TrimSpace(name)
	if !safeSkillName(name) {
		return os.ErrInvalid
	}
	dir := filepath.Join(config.SkillsDir(), name)
	rel, err := filepath.Rel(config.SkillsDir(), dir)
	if err != nil || strings.HasPrefix(rel, "..") {
		return os.ErrInvalid
	}
	if _, err := os.Stat(dir); err != nil {
		return err
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	m, err := loadManifest()
	if err == nil {
		delete(m.Skills, name)
		_ = saveManifest(m)
	}
	appendSkillAudit("remove", "", "", name)
	return nil
}

var skillAuditMu sync.Mutex

func appendSkillAudit(op, url, ref, name string) {
	skillAuditMu.Lock()
	defer skillAuditMu.Unlock()
	path := config.SkillAuditPath()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	ev := map[string]string{
		"time": time.Now().UTC().Format(time.RFC3339),
		"op":   op,
		"name": name,
	}
	if url != "" {
		ev["url"] = url
	}
	if ref != "" {
		ev["ref"] = ref
	}
	line, _ := json.Marshal(ev)
	_, _ = f.Write(append(line, '\n'))
}
