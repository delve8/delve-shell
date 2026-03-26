package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// allowlistFile is the allowlist.yaml file structure.
type allowlistFile struct {
	Allowlist []AllowlistEntry `yaml:"allowlist"`
}

// LoadAllowlist loads allowlist from allowlist.yaml. If missing, writes default and returns it.
func LoadAllowlist() ([]AllowlistEntry, error) {
	path := AllowlistPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			def := defaultAllowlist()
			if err := EnsureRootDir(); err != nil {
				return nil, err
			}
			if err := WriteAllowlist(def); err != nil {
				return nil, err
			}
			return def, nil
		}
		return nil, err
	}
	var f allowlistFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Allowlist, nil
}

// WriteAllowlist writes the allowlist to allowlist.yaml (call after changes; EnsureRootDir before first write).
func WriteAllowlist(entries []AllowlistEntry) error {
	data, err := yaml.Marshal(allowlistFile{Allowlist: entries})
	if err != nil {
		return err
	}
	return os.WriteFile(AllowlistPath(), data, 0600)
}

// AllowlistUpdateWithDefaults merges current allowlist with built-in default: keep existing, add missing patterns. Returns number added.
func AllowlistUpdateWithDefaults() (added int, err error) {
	path := AllowlistPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := EnsureRootDir(); err != nil {
				return 0, err
			}
			def := defaultAllowlist()
			if err := WriteAllowlist(def); err != nil {
				return 0, err
			}
			return len(def), nil
		}
		return 0, err
	}
	var f allowlistFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return 0, err
	}
	have := make(map[string]bool)
	for _, e := range f.Allowlist {
		have[e.Pattern] = true
	}
	out := f.Allowlist
	for _, e := range defaultAllowlist() {
		if !have[e.Pattern] {
			out = append(out, e)
			have[e.Pattern] = true
			added++
		}
	}
	if added == 0 {
		return 0, nil
	}
	if err := WriteAllowlist(out); err != nil {
		return 0, err
	}
	return added, nil
}

// defaultAllowlist is the built-in default: read-only commands; each Pattern is a regex.
// Single-command patterns use (^|\s)word(\s|$) so the word is only matched as command name:
// left side must be start or space (not -word option); right side must be space or end (not word-xxx).
func defaultAllowlist() []AllowlistEntry {
	return []AllowlistEntry{
		// dirs and paths
		{Pattern: `(^|\s)pwd(\s|$)`},
		{Pattern: `(^|\s)ls(\s|$)`},
		{Pattern: `(^|\s)dir(\s|$)`}, // some envs alias
		// user and env
		{Pattern: `(^|\s)whoami(\s|$)`},
		{Pattern: `(^|\s)id(\s|$)`},
		{Pattern: `(^|\s)env(\s|$)`},
		{Pattern: `(^|\s)printenv(\s|$)`},
		// system info
		{Pattern: `(^|\s)uname(\s|$)`},
		{Pattern: `(^|\s)hostname(\s|$)`},
		{Pattern: `(^|\s)date(\s|$)`},
		// command lookup
		{Pattern: `(^|\s)which(\s|$)`},
		{Pattern: `(^|\s)whereis(\s|$)`},
		{Pattern: `(^|\s)type(\s|$)`},
		// read-only file view (cat/head/tail/less/more read-only; cat can read any file)
		{Pattern: `(^|\s)cat(\s|$)`},
		{Pattern: `(^|\s)head(\s|$)`},
		{Pattern: `(^|\s)tail(\s|$)`},
		{Pattern: `(^|\s)less(\s|$)`},
		{Pattern: `(^|\s)more(\s|$)`},
		// file info and stats
		{Pattern: `(^|\s)file(\s|$)`},
		{Pattern: `(^|\s)stat(\s|$)`},
		{Pattern: `(^|\s)wc(\s|$)`},
		// checksum and encoding (read-only)
		{Pattern: `(^|\s)md5sum(\s|$)`},
		{Pattern: `(^|\s)sha256sum(\s|$)`},
		{Pattern: `(^|\s)sha1sum(\s|$)`},
		{Pattern: `(^|\s)shasum(\s|$)`}, // macOS
		{Pattern: `(^|\s)base64(\s|$)`},
		{Pattern: `(^|\s)cksum(\s|$)`},
		// find: common read-only usage only (-name/-type/-maxdepth), no -exec/-delete
		{Pattern: `find\s+\S+(\s+-(name|type|maxdepth|iname)\s+\S+)*\s*$`},
		// grep/egrep/fgrep: read-only search
		{Pattern: `(^|\s)grep(\s|$)`},
		{Pattern: `(^|\s)egrep(\s|$)`},
		{Pattern: `(^|\s)fgrep(\s|$)`},
		// output and pipes (read-only)
		{Pattern: `(^|\s)echo(\s|$)`},
		{Pattern: `(^|\s)printf(\s|$)`},
		// text compare and process (read-only, no file write)
		{Pattern: `(^|\s)diff(\s|$)`},
		{Pattern: `(^|\s)cmp(\s|$)`},
		{Pattern: `(^|\s)cut(\s|$)`},
		{Pattern: `(^|\s)tr(\s|$)`},
		{Pattern: `(^|\s)uniq(\s|$)`},
		{Pattern: `(^|\s)nl(\s|$)`},
		{Pattern: `(^|\s)column(\s|$)`},
		{Pattern: `(^|\s)od(\s|$)`},
		{Pattern: `(^|\s)xxd(\s|$)`},
		{Pattern: `(^|\s)hexdump(\s|$)`},
		// decompress to stdout (read-only)
		{Pattern: `(^|\s)zcat(\s|$)`},
		{Pattern: `(^|\s)bzcat(\s|$)`},
		{Pattern: `(^|\s)xzcat(\s|$)`},
		// process and system resources (read-only)
		{Pattern: `(^|\s)ps(\s|$)`},
		{Pattern: `(^|\s)uptime(\s|$)`},
		{Pattern: `(^|\s)df(\s|$)`},
		{Pattern: `(^|\s)du(\s|$)`},
		{Pattern: `(^|\s)free(\s|$)`},
		{Pattern: `(^|\s)lsblk(\s|$)`},
		// user and permissions (read-only)
		{Pattern: `(^|\s)groups(\s|$)`},
		{Pattern: `(^|\s)getent(\s|$)`},
		{Pattern: `(^|\s)locale(\s|$)`},
		// network read-only (DNS, connectivity)
		{Pattern: `(^|\s)ping(\s|$)`},
		{Pattern: `(^|\s)nslookup(\s|$)`},
		{Pattern: `(^|\s)dig(\s|$)`},
		{Pattern: `(^|\s)host(\s|$)`},
		// other read-only
		{Pattern: `(^|\s)true(\s|$)`},
		{Pattern: `(^|\s)false(\s|$)`},
		{Pattern: `(^|\s)seq(\s|$)`},
		{Pattern: `(^|\s)sleep(\s|$)`},
		// kubectl read-only subcommands
		{Pattern: `kubectl\s+get\s`},
		{Pattern: `kubectl\s+describe\s`},
		{Pattern: `kubectl\s+logs\s`},
		{Pattern: `kubectl\s+top\s`},
		{Pattern: `kubectl\s+explain\s`},
		{Pattern: `kubectl\s+api-resources`},
		{Pattern: `kubectl\s+api-versions`},
		{Pattern: `kubectl\s+cluster-info(?!\s+dump)`}, // view read-only; dump writes so excluded
		{Pattern: `kubectl\s+config\s+view`},
		{Pattern: `kubectl\s+version`},
		{Pattern: `kubectl\s+auth\s+can-i`},
		{Pattern: `kubectl\s+auth\s+whoami`},
		{Pattern: `kubectl\s+rollout\s+status`},
		{Pattern: `kubectl\s+diff\s`},
		{Pattern: `kubectl\s+.*--help`},
		// git read-only commands
		{Pattern: `git\s+status\s`},
		{Pattern: `git\s+status\s*$`},
		{Pattern: `git\s+diff\s`},
		{Pattern: `git\s+diff\s*$`},
		{Pattern: `git\s+log\s`},
		{Pattern: `git\s+log\s*$`},
		{Pattern: `git\s+show\s`},
		{Pattern: `git\s+show\s*$`},
		{Pattern: `git\s+branch(?:\s+-(?:a|v|r)|\s+--(?:list|show-current))?(?:\s|$)`},
		{Pattern: `git\s+tag(?:\s+-(?:l|list)|\s+--list)(?:\s|$)`},
		{Pattern: `git\s+tag\s*$`},
		{Pattern: `git\s+remote(?:\s+-(?:v)|\s+show)(?:\s|$)`},
		{Pattern: `git\s+config\s+(?:--get|--list|--get-all)(?:\s|$)`},
		{Pattern: `git\s+rev-parse(?:\s|$)`},
		{Pattern: `git\s+describe(?:\s|$)`},
		{Pattern: `git\s+stash\s+list(?:\s|$)`},
		{Pattern: `git\s+reflog\s`},
		{Pattern: `git\s+reflog\s*$`},
		{Pattern: `git\s+blame\s`},
		{Pattern: `git\s+ls-files\s`},
		{Pattern: `git\s+ls-tree\s`},
		{Pattern: `git\s+cat-file\s`},
		{Pattern: `git\s+for-each-ref\s`},
		{Pattern: `git\s+symbolic-ref\s`},
		{Pattern: `git\s+help\s`},
		{Pattern: `git\s+version\s*$`},
		{Pattern: `git\s+--help`},
		// other CLI help
		{Pattern: `docker\s+.*--help`},
		{Pattern: `(^|\s)--help(\s|$)`}, // most GNU tools: command --help
	}
}
