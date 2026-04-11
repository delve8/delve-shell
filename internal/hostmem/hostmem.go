package hostmem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"delve-shell/internal/config"
	"delve-shell/internal/remote/execenv"
)

const (
	maxStoredCommands = 8000
	maxSummaryItems   = 16
)

type Context struct {
	HostID       string `json:"host_id"`
	ProfileKey   string `json:"profile_key"`
	Alias        string `json:"alias,omitempty"`
	WeakIdentity bool   `json:"weak_identity,omitempty"`
}

func (c Context) Valid() bool {
	return strings.TrimSpace(c.HostID) != ""
}

type Memory struct {
	HostID         string                  `json:"host_id"`
	IdentitySource string                  `json:"identity_source,omitempty"`
	WeakIdentity   bool                    `json:"weak_identity,omitempty"`
	Aliases        []string                `json:"aliases,omitempty"`
	Machine        MachineMemory           `json:"machine"`
	Profiles       map[string]*ExecProfile `json:"profiles,omitempty"`
	UpdatedAt      time.Time               `json:"updated_at"`
}

type MachineMemory struct {
	OSFamily       string    `json:"os_family,omitempty"`
	Role           string    `json:"role,omitempty"`
	RoleConfidence float64   `json:"role_confidence,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	Notes          []string  `json:"notes,omitempty"`
	Evidence       []string  `json:"evidence,omitempty"`
	ObservedAt     time.Time `json:"observed_at,omitempty"`
}

type ExecProfile struct {
	User            string    `json:"user,omitempty"`
	Shell           string    `json:"shell,omitempty"`
	Available       []string  `json:"available_commands,omitempty"`
	Missing         []string  `json:"missing_commands,omitempty"`
	PackageManagers []string  `json:"package_managers,omitempty"`
	ObservedAt      time.Time `json:"observed_at,omitempty"`
}

type UpdatePatch struct {
	Role               string   `json:"role,omitempty"`
	RoleConfidence     float64  `json:"role_confidence,omitempty"`
	TagsAdd            []string `json:"tags_add,omitempty"`
	NotesAdd           []string `json:"notes_add,omitempty"`
	EvidenceAdd        []string `json:"evidence_add,omitempty"`
	AvailableAdd       []string `json:"available_commands_add,omitempty"`
	MissingAdd         []string `json:"missing_commands_add,omitempty"`
	PackageManagersAdd []string `json:"package_managers_add,omitempty"`
	OSFamily           string   `json:"os_family,omitempty"`
}

type ProbeResult struct {
	Context         Context
	IdentitySource  string
	OSFamily        string
	User            string
	Shell           string
	Available       []string
	Missing         []string
	Completion      []string
	PackageManagers []string
	ObservedAt      time.Time
}

var fileMu sync.Mutex

func filePath(hostID string) string {
	hostID = strings.TrimSpace(hostID)
	sum := sha256.Sum256([]byte(hostID))
	return filepath.Join(config.HostsDir(), fileNameFor(hostID, sum[:16]))
}

func fileNameFor(hostID string, digest []byte) string {
	prefix := "host"
	if head, _, ok := strings.Cut(strings.ToLower(strings.TrimSpace(hostID)), ":"); ok && head != "" {
		prefix = sanitizeFileLabel(head)
	}
	return prefix + "-" + hex.EncodeToString(digest) + ".json"
}

func sanitizeFileLabel(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "host"
	}
	return b.String()
}

func Load(hostID string) (*Memory, error) {
	hostID = strings.TrimSpace(hostID)
	if hostID == "" {
		return nil, fmt.Errorf("host_id is required")
	}
	data, err := os.ReadFile(filePath(hostID))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var mem Memory
	if err := json.Unmarshal(data, &mem); err != nil {
		return nil, err
	}
	if mem.Profiles == nil {
		mem.Profiles = make(map[string]*ExecProfile)
	}
	return &mem, nil
}

func Save(mem *Memory) error {
	if mem == nil {
		return fmt.Errorf("memory is nil")
	}
	hostID := strings.TrimSpace(mem.HostID)
	if hostID == "" {
		return fmt.Errorf("host_id is required")
	}
	if err := config.EnsureRootDir(); err != nil {
		return err
	}
	if mem.Profiles == nil {
		mem.Profiles = make(map[string]*ExecProfile)
	}
	mem.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		return err
	}
	path := filePath(hostID)
	tmp := path + ".tmp"
	fileMu.Lock()
	defer fileMu.Unlock()
	if err := os.WriteFile(tmp, append(data, '\n'), 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func View(ctx Context) (string, error) {
	if !ctx.Valid() {
		return "no host memory for current execution environment", nil
	}
	mem, err := Load(ctx.HostID)
	if err != nil {
		return "", err
	}
	if mem == nil {
		return "no host memory for current execution environment", nil
	}
	data, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func Update(ctx Context, patch UpdatePatch) (string, error) {
	if !ctx.Valid() {
		return "no host memory target is active", nil
	}
	mem, err := Load(ctx.HostID)
	if err != nil {
		return "", err
	}
	if mem == nil {
		mem = &Memory{
			HostID:       ctx.HostID,
			WeakIdentity: ctx.WeakIdentity,
			Profiles:     make(map[string]*ExecProfile),
		}
	}
	if strings.TrimSpace(ctx.Alias) != "" {
		mem.Aliases = mergeUnique(mem.Aliases, []string{ctx.Alias}, 32)
	}
	if patch.OSFamily != "" {
		mem.Machine.OSFamily = strings.TrimSpace(patch.OSFamily)
	}
	if patch.Role != "" {
		mem.Machine.Role = strings.TrimSpace(patch.Role)
	}
	if patch.RoleConfidence > 0 {
		if patch.RoleConfidence > 1 {
			patch.RoleConfidence = 1
		}
		mem.Machine.RoleConfidence = patch.RoleConfidence
	}
	mem.Machine.Tags = mergeUnique(mem.Machine.Tags, patch.TagsAdd, 32)
	mem.Machine.Notes = mergeUnique(mem.Machine.Notes, patch.NotesAdd, 32)
	mem.Machine.Evidence = mergeUnique(mem.Machine.Evidence, patch.EvidenceAdd, 64)
	if hasMachinePatch(patch) {
		mem.Machine.ObservedAt = time.Now().UTC()
	}
	profile := ensureProfile(mem, ctx.ProfileKey)
	profile.Available = mergeUnique(profile.Available, patch.AvailableAdd, maxStoredCommands)
	if len(patch.AvailableAdd) > 0 {
		profile.Missing = subtract(profile.Missing, patch.AvailableAdd)
	}
	profile.Missing = mergeUnique(profile.Missing, patch.MissingAdd, maxStoredCommands)
	if len(patch.MissingAdd) > 0 {
		profile.Available = subtract(profile.Available, patch.MissingAdd)
	}
	profile.PackageManagers = mergeUnique(profile.PackageManagers, patch.PackageManagersAdd, 16)
	if hasProfilePatch(patch) {
		profile.ObservedAt = time.Now().UTC()
	}
	if err := Save(mem); err != nil {
		return "", err
	}
	return Summary(mem, ctx.ProfileKey), nil
}

func SummaryForContext(ctx Context) string {
	if !ctx.Valid() {
		return ""
	}
	mem, err := Load(ctx.HostID)
	if err != nil || mem == nil {
		return ""
	}
	return Summary(mem, ctx.ProfileKey)
}

func Summary(mem *Memory, profileKey string) string {
	if mem == nil {
		return ""
	}
	var lines []string
	if !mem.UpdatedAt.IsZero() {
		lines = append(lines, "Cached: "+mem.UpdatedAt.UTC().Format(time.RFC3339))
	}
	if mem.WeakIdentity {
		lines = append(lines, "Identity: weak fallback")
	}
	if osFamily := strings.TrimSpace(mem.Machine.OSFamily); osFamily != "" {
		lines = append(lines, "OS: "+osFamily)
	}
	if role := strings.TrimSpace(mem.Machine.Role); role != "" {
		line := "Role: " + role
		if mem.Machine.RoleConfidence > 0 {
			line += fmt.Sprintf(" (confidence %.2f)", mem.Machine.RoleConfidence)
		}
		lines = append(lines, line)
	}
	if len(mem.Machine.Tags) > 0 {
		lines = append(lines, "Tags: "+strings.Join(limitList(mem.Machine.Tags, 8), ", "))
	}
	if profile := lookupProfile(mem, profileKey); profile != nil {
		if user := strings.TrimSpace(profile.User); user != "" {
			lines = append(lines, "User: "+user)
		}
		if shell := strings.TrimSpace(profile.Shell); shell != "" {
			lines = append(lines, "Shell: "+shell)
		}
		if len(profile.PackageManagers) > 0 {
			lines = append(lines, "Package managers: "+strings.Join(limitList(profile.PackageManagers, 6), ", "))
		}
		if len(profile.Available) > 0 {
			lines = append(lines, "Available commands: "+strings.Join(limitList(profile.Available, maxSummaryItems), ", "))
		}
		if len(profile.Missing) > 0 {
			lines = append(lines, "Missing commands: "+strings.Join(limitList(profile.Missing, 8), ", "))
		}
	}
	if len(mem.Machine.Notes) > 0 {
		lines = append(lines, "Notes: "+strings.Join(limitList(mem.Machine.Notes, 3), " | "))
	}
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func Probe(ctx context.Context, executor execenv.CommandExecutor, alias string) (ProbeResult, error) {
	metaOut, _, _, err := executor.Run(ctx, identityProbeScript())
	if err != nil {
		return ProbeResult{}, err
	}
	fields := parseProbeKV(metaOut)
	now := time.Now().UTC()
	osFamily := strings.TrimSpace(fields["os"])
	user := strings.TrimSpace(fields["user"])
	shell := strings.TrimSpace(fields["shell"])
	machineID := strings.TrimSpace(fields["machine_id"])
	profileKey := profileKey(user)
	ctxInfo := Context{
		Alias:        strings.TrimSpace(alias),
		ProfileKey:   profileKey,
		HostID:       hostIDFor(osFamily, machineID, user, alias),
		WeakIdentity: strings.TrimSpace(machineID) == "",
	}
	cmdOut, _, _, cmdErr := executor.Run(ctx, commandInventoryScript())
	completion := normalizeCommands(cmdOut)
	cmds, missing := importantCommandSnapshot(completion)
	pkgs := normalizeCommands(fields["pkg"])
	if cmdErr != nil {
		completion = normalizeCommands(fields["have"])
		cmds, missing = importantCommandSnapshot(completion)
	}
	return ProbeResult{
		Context:         ctxInfo,
		IdentitySource:  identitySource(osFamily, machineID),
		OSFamily:        osFamily,
		User:            user,
		Shell:           shell,
		Available:       cmds,
		Missing:         missing,
		Completion:      completion,
		PackageManagers: pkgs,
		ObservedAt:      now,
	}, nil
}

func ApplyProbe(pr ProbeResult) (Context, error) {
	mem, err := Load(pr.Context.HostID)
	if err != nil {
		return Context{}, err
	}
	if mem == nil {
		mem = &Memory{
			HostID:         pr.Context.HostID,
			IdentitySource: pr.IdentitySource,
			WeakIdentity:   pr.Context.WeakIdentity,
			Profiles:       make(map[string]*ExecProfile),
		}
	}
	if mem.IdentitySource == "" {
		mem.IdentitySource = pr.IdentitySource
	}
	mem.WeakIdentity = pr.Context.WeakIdentity
	mem.Machine.OSFamily = firstNonEmpty(pr.OSFamily, mem.Machine.OSFamily)
	if !pr.ObservedAt.IsZero() {
		mem.Machine.ObservedAt = pr.ObservedAt
	}
	if strings.TrimSpace(pr.Context.Alias) != "" {
		mem.Aliases = mergeUnique(mem.Aliases, []string{pr.Context.Alias}, 32)
	}
	profile := ensureProfile(mem, pr.Context.ProfileKey)
	profile.User = firstNonEmpty(pr.User, profile.User)
	profile.Shell = firstNonEmpty(pr.Shell, profile.Shell)
	if len(pr.Available) > 0 {
		profile.Available = normalizeCommands(strings.Join(pr.Available, "\n"))
	}
	if len(pr.Missing) > 0 {
		profile.Missing = normalizeCommands(strings.Join(pr.Missing, "\n"))
	}
	profile.Missing = subtract(profile.Missing, pr.Available)
	if len(pr.PackageManagers) > 0 {
		profile.PackageManagers = normalizeCommands(strings.Join(pr.PackageManagers, "\n"))
	}
	profile.ObservedAt = pr.ObservedAt
	if err := Save(mem); err != nil {
		return Context{}, err
	}
	return pr.Context, nil
}

func identityProbeScript() string {
	return `os="$(uname -s 2>/dev/null || printf '')"
printf 'os=%s\n' "$os"
user="$(id -un 2>/dev/null || printf '%s' "${USER:-}")"
printf 'user=%s\n' "$user"
printf 'shell=%s\n' "${SHELL:-}"
mid=""
case "$os" in
  Linux)
    if [ -r /etc/machine-id ]; then
      mid="$(tr -d '\r\n' </etc/machine-id)"
    fi
    if [ -z "$mid" ] && [ -r /var/lib/dbus/machine-id ]; then
      mid="$(tr -d '\r\n' </var/lib/dbus/machine-id)"
    fi
    if [ -z "$mid" ] && [ -r /sys/class/dmi/id/product_uuid ]; then
      mid="$(tr -d '\r\n' </sys/class/dmi/id/product_uuid)"
    fi
    ;;
  Darwin)
    if command -v ioreg >/dev/null 2>&1; then
      mid="$(ioreg -rd1 -c IOPlatformExpertDevice 2>/dev/null | awk -F '"' '/IOPlatformUUID/ {print $(NF-1); exit}')"
    fi
    ;;
  MINGW*|MSYS*|CYGWIN*|Windows_NT)
    if command -v reg >/dev/null 2>&1; then
      mid="$(reg query 'HKLM\SOFTWARE\Microsoft\Cryptography' /v MachineGuid 2>/dev/null | awk '/MachineGuid/ {print $NF; exit}')"
    fi
    ;;
esac
printf 'machine_id=%s\n' "$mid"
for p in apt apt-get yum dnf microdnf apk brew port pacman zypper; do
  if command -v "$p" >/dev/null 2>&1; then
    printf 'pkg=%s\n' "$p"
  fi
done
for c in sh bash zsh ash fish busybox awk sed grep egrep fgrep find xargs sort uniq cut head tail wc cat less column ls cp mv rm mkdir rmdir chmod chown ln readlink realpath stat date sleep timeout env printenv tee touch tar gzip gunzip zip unzip base64 openssl curl wget ssh scp rsync nc netcat sudo su systemctl journalctl service ps top free df du mount umount lsblk blkid ip ifconfig ss netstat lsof apt apt-get yum dnf microdnf apk brew port pacman zypper git make go python python3 pip pip3 node npm npx pnpm yarn java javac mvn gradle cargo rustc docker docker-compose podman nerdctl ctr crictl kubectl helm kubeadm kubelet kind minikube terraform ansible ansible-playbook aws gcloud az aliyun mysql psql redis-cli; do
  if command -v "$c" >/dev/null 2>&1; then
    printf 'have=%s\n' "$c"
  fi
done`
}

func commandInventoryScript() string {
	return `if command -v bash >/dev/null 2>&1; then
  bash -lc 'compgen -c'
else
  oldifs="$IFS"
  IFS=:
  for d in $PATH; do
    [ -d "$d" ] || continue
    for f in "$d"/*; do
      [ -e "$f" ] || continue
      [ -x "$f" ] || continue
      [ ! -d "$f" ] || continue
      basename "$f"
    done
  done
  IFS="$oldifs"
fi`
}

func parseProbeKV(s string) map[string]string {
	out := make(map[string]string)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		if prev, ok := out[k]; ok && prev != "" {
			out[k] = prev + "\n" + v
			continue
		}
		out[k] = v
	}
	return out
}

func normalizeCommands(s string) []string {
	seen := make(map[string]struct{}, 256)
	out := make([]string, 0, 256)
	for _, line := range strings.Split(s, "\n") {
		cmd := strings.TrimSpace(line)
		if cmd == "" {
			continue
		}
		if strings.ContainsAny(cmd, " \t/") {
			continue
		}
		if _, ok := seen[cmd]; ok {
			continue
		}
		seen[cmd] = struct{}{}
		out = append(out, cmd)
		if len(out) >= maxStoredCommands {
			break
		}
	}
	sort.Strings(out)
	return out
}

func importantCommandSnapshot(commands []string) (available []string, missing []string) {
	if len(commands) == 0 {
		return nil, importantCommandList()
	}
	seen := make(map[string]struct{}, len(commands))
	for _, cmd := range commands {
		seen[cmd] = struct{}{}
	}
	available = make([]string, 0, len(commands))
	missing = make([]string, 0, len(importantCommandSet))
	for _, cmd := range importantCommandList() {
		if _, ok := seen[cmd]; ok {
			available = append(available, cmd)
			continue
		}
		missing = append(missing, cmd)
	}
	return available, missing
}

var importantCommandSet = map[string]struct{}{
	"sh": {}, "bash": {}, "zsh": {}, "ash": {}, "fish": {}, "busybox": {},
	"awk": {}, "sed": {}, "grep": {}, "egrep": {}, "fgrep": {}, "rg": {}, "fd": {}, "find": {}, "xargs": {}, "sort": {}, "uniq": {}, "cut": {}, "head": {}, "tail": {}, "wc": {}, "cat": {}, "less": {}, "column": {}, "ls": {},
	"cp": {}, "mv": {}, "rm": {}, "mkdir": {}, "rmdir": {}, "chmod": {}, "chown": {}, "ln": {}, "readlink": {}, "realpath": {}, "stat": {}, "date": {}, "sleep": {}, "timeout": {}, "env": {}, "printenv": {}, "tee": {}, "touch": {},
	"jq": {}, "yq": {}, "base64": {}, "openssl": {},
	"tar": {}, "gzip": {}, "gunzip": {}, "zip": {}, "unzip": {},
	"curl": {}, "wget": {}, "ssh": {}, "scp": {}, "rsync": {}, "nc": {}, "netcat": {},
	"sudo": {}, "su": {}, "systemctl": {}, "journalctl": {}, "service": {}, "ps": {}, "top": {}, "free": {}, "df": {}, "du": {}, "mount": {}, "umount": {}, "lsblk": {}, "blkid": {}, "ip": {}, "ifconfig": {}, "ss": {}, "netstat": {}, "lsof": {},
	"apt": {}, "apt-get": {}, "yum": {}, "dnf": {}, "microdnf": {}, "apk": {}, "brew": {}, "port": {}, "pacman": {}, "zypper": {},
	"git": {}, "make": {}, "go": {}, "python": {}, "python3": {}, "pip": {}, "pip3": {}, "node": {}, "npm": {}, "npx": {}, "pnpm": {}, "yarn": {}, "java": {}, "javac": {}, "mvn": {}, "gradle": {}, "cargo": {}, "rustc": {},
	"docker": {}, "docker-compose": {}, "podman": {}, "nerdctl": {}, "ctr": {}, "crictl": {}, "kubectl": {}, "helm": {}, "kubeadm": {}, "kubelet": {}, "kind": {}, "minikube": {},
	"terraform": {}, "ansible": {}, "ansible-playbook": {}, "aws": {}, "gcloud": {}, "az": {}, "aliyun": {},
	"mysql": {}, "psql": {}, "redis-cli": {},
}

func importantCommandList() []string {
	out := make([]string, 0, len(importantCommandSet))
	for cmd := range importantCommandSet {
		out = append(out, cmd)
	}
	sort.Strings(out)
	return out
}

func hostIDFor(osFamily, machineID, user, alias string) string {
	osFamily = strings.ToLower(strings.TrimSpace(osFamily))
	machineID = strings.TrimSpace(machineID)
	if machineID != "" {
		return firstNonEmpty(osFamily, "unknown") + ":machine-id:" + machineID
	}
	weak := strings.ToLower(strings.TrimSpace(user)) + "|" + strings.ToLower(strings.TrimSpace(alias)) + "|" + osFamily
	sum := sha256.Sum256([]byte(weak))
	return "weak:" + hex.EncodeToString(sum[:8])
}

func identitySource(osFamily, machineID string) string {
	if strings.TrimSpace(machineID) == "" {
		return "weak"
	}
	switch strings.ToLower(strings.TrimSpace(osFamily)) {
	case "linux":
		return "/etc/machine-id"
	case "darwin":
		return "IOPlatformUUID"
	case "windows_nt", "msys", "mingw", "cygwin":
		return "MachineGuid"
	default:
		return "machine-id"
	}
}

func profileKey(user string) string {
	user = strings.TrimSpace(strings.ToLower(user))
	if user == "" {
		return "default"
	}
	return user
}

func ensureProfile(mem *Memory, key string) *ExecProfile {
	if mem.Profiles == nil {
		mem.Profiles = make(map[string]*ExecProfile)
	}
	key = strings.TrimSpace(key)
	if key == "" {
		key = "default"
	}
	profile := mem.Profiles[key]
	if profile == nil {
		profile = &ExecProfile{}
		mem.Profiles[key] = profile
	}
	return profile
}

func lookupProfile(mem *Memory, key string) *ExecProfile {
	if mem == nil || len(mem.Profiles) == 0 {
		return nil
	}
	key = strings.TrimSpace(key)
	if key != "" {
		if profile := mem.Profiles[key]; profile != nil {
			return profile
		}
	}
	return mem.Profiles["default"]
}

func mergeUnique(dst, add []string, max int) []string {
	seen := make(map[string]struct{}, len(dst)+len(add))
	out := make([]string, 0, len(dst)+len(add))
	push := func(items []string) {
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			out = append(out, item)
			if max > 0 && len(out) >= max {
				return
			}
		}
	}
	push(dst)
	if max == 0 || len(out) < max {
		push(add)
	}
	sort.Strings(out)
	if max > 0 && len(out) > max {
		out = out[:max]
	}
	return out
}

func subtract(src, remove []string) []string {
	if len(src) == 0 || len(remove) == 0 {
		return src
	}
	rm := make(map[string]struct{}, len(remove))
	for _, item := range remove {
		item = strings.TrimSpace(item)
		if item != "" {
			rm[item] = struct{}{}
		}
	}
	out := src[:0]
	for _, item := range src {
		if _, ok := rm[item]; ok {
			continue
		}
		out = append(out, item)
	}
	return out
}

func limitList(items []string, max int) []string {
	if len(items) <= max {
		return items
	}
	return items[:max]
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func hasMachinePatch(p UpdatePatch) bool {
	return p.Role != "" || p.RoleConfidence > 0 || p.OSFamily != "" || len(p.TagsAdd) > 0 || len(p.NotesAdd) > 0 || len(p.EvidenceAdd) > 0
}

func hasProfilePatch(p UpdatePatch) bool {
	return len(p.AvailableAdd) > 0 || len(p.MissingAdd) > 0 || len(p.PackageManagersAdd) > 0
}
