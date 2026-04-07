package config

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed allowlist_default.yaml
var defaultAllowlistYAML []byte

var (
	defaultAllowlistOnce   sync.Once
	defaultAllowlistParsed *LoadedAllowlist
	defaultAllowlistErr    error
)

func mustParseEmbeddedDefaultAllowlist() *LoadedAllowlist {
	defaultAllowlistOnce.Do(func() {
		ld, err := ParseAllowlistYAML(defaultAllowlistYAML)
		if err != nil {
			defaultAllowlistErr = err
			return
		}
		if err := ValidateLoadedAllowlist(ld); err != nil {
			defaultAllowlistErr = err
			return
		}
		ld.Commands = NormalizeReadOnlyCLIPolicies(ld.Commands)
		defaultAllowlistParsed = ld
	})
	if defaultAllowlistErr != nil {
		panic("config: builtin allowlist_default.yaml: " + defaultAllowlistErr.Error())
	}
	return defaultAllowlistParsed
}

// AllowlistSchemaVersion is the only supported on-disk allowlist.yaml schema.
const AllowlistSchemaVersion = 2

// ReadOnlyCLIPolicy is one argv0 basename and a tree of global flags, subcommands, and operands.
// On disk, the basename is the commands map key; Name mirrors that key in memory.
// Global and Root are optional in YAML: nil means defaults (see [ReadOnlyCLIPolicy.EffectiveGlobal], [ReadOnlyCLIPolicy.EffectiveRoot]).
type ReadOnlyCLIPolicy struct {
	Name   string      `yaml:"-"`
	Global *GlobalSpec `yaml:"global,omitempty"`
	Root   *RootSpec   `yaml:"root,omitempty"`
}

// GlobalSpec holds flags that appear before the first subcommand (kubectl-style globals).
type GlobalSpec struct {
	// No omitempty on Flags: FlagRule uses only unexported fields, so reflect.IsZero is always true
	// and yaml.v3 would incorrectly omit flags (e.g. kubectl global allow-list became global: {}).
	Flags FlagRule `yaml:"flags"`
}

// SubcommandMap is nested subcommands keyed by name (no `name` field in YAML).
type SubcommandMap map[string]SubcommandNode

func (m *SubcommandMap) UnmarshalYAML(n *yaml.Node) error {
	var raw map[string]SubcommandNode
	if err := n.Decode(&raw); err != nil {
		return err
	}
	assignSubcommandMapKeys(SubcommandMap(raw))
	*m = raw
	return nil
}

func assignSubcommandMapKeys(m SubcommandMap) {
	if len(m) == 0 {
		return
	}
	for k, v := range m {
		v.Name = k
		assignSubcommandMapKeys(v.Subcommands)
		m[k] = v
	}
}

// MarshalYAML encodes subcommands with sorted keys for stable diffs.
func (m SubcommandMap) MarshalYAML() (interface{}, error) {
	if len(m) == 0 {
		return nil, nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, k := range keys {
		b, err := yaml.Marshal(m[k])
		if err != nil {
			return nil, err
		}
		var doc yaml.Node
		if err := yaml.Unmarshal(b, &doc); err != nil {
			return nil, err
		}
		if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 {
			return nil, fmt.Errorf("config: internal marshal subcommand %q", k)
		}
		kn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}
		node.Content = append(node.Content, kn, doc.Content[0])
	}
	return node, nil
}

// RootSpec is the policy after globals: optional flags/operands at the root and child subcommands.
type RootSpec struct {
	// No omitempty on Flags/Operands: same IsZero issue as [GlobalSpec.Flags] for yaml.v3.
	Flags       FlagRule      `yaml:"flags"`
	Operands    OperandsRule  `yaml:"operands"`
	Subcommands SubcommandMap `yaml:"subcommands,omitempty"`
}

// SubcommandNode is a named branch; leaves use flags/operands; inner nodes add deeper subcommands.
// On disk the name is the map key under subcommands.
// Omitted flags/operands default to any/any (open subcommand); use explicit none or allow-list to tighten.
type SubcommandNode struct {
	Name        string         `yaml:"-"`
	Flags       *FlagRule      `yaml:"flags,omitempty"`
	Operands    *OperandsRule  `yaml:"operands,omitempty"`
	Subcommands SubcommandMap  `yaml:"subcommands,omitempty"`
}

// EffectiveFlags returns flags or the default (any) when omitted in YAML.
func (n SubcommandNode) EffectiveFlags() FlagRule {
	if n.Flags != nil {
		return *n.Flags
	}
	return NewFlagAny()
}

// EffectiveOperands returns operands or the default (any) when omitted in YAML.
func (n SubcommandNode) EffectiveOperands() OperandsRule {
	if n.Operands != nil {
		return *n.Operands
	}
	return NewOperandsAny()
}

// AllowedOption is a long/short option for a closed allow-list flag rule.
type AllowedOption struct {
	Short string `yaml:"short,omitempty"`
	Long  string `yaml:"long,omitempty"`
	Value string `yaml:"value,omitempty"`
}

// ValueRequired reports whether the option takes a separate token or =value.
func (g AllowedOption) ValueRequired() bool {
	return strings.EqualFold(strings.TrimSpace(g.Value), "required")
}

// FlagRule is YAML either a scalar "any"|"none" or a mapping `{ allow: [...], must: [...] }`.
// Mapping form requires at least one of allow or must. Options listed only under must are still
// consumable (must implies allow for those rows); allow lists additional optional flags. A must entry
// that already matches an allow row (same short/long where set) uses that allow row for value: rules.
// When must is non-empty, each must entry must be consumed at least once before the first operand token
// at the same matcher node, and before recursing into a child subcommand when that node has subcommands
// (see hil structured matcher). must is not valid on global.flags. Empty must means no extra requirement.
type FlagRule struct {
	any   bool
	allow []AllowedOption
	must  []AllowedOption
}

// NewFlagAny returns a rule that accepts any flag tokens.
func NewFlagAny() FlagRule { return FlagRule{any: true} }

// NewFlagNone returns a rule that rejects all flag tokens.
func NewFlagNone() FlagRule { return FlagRule{} }

// NewFlagAllow returns a closed allow-list of options.
func NewFlagAllow(opts []AllowedOption) FlagRule {
	return FlagRule{allow: append([]AllowedOption(nil), opts...)}
}

// WithMust returns a copy of f with must requirements (allow-list only; validated separately).
func (f FlagRule) WithMust(must []AllowedOption) FlagRule {
	f.must = append([]AllowedOption(nil), must...)
	return f
}

func (f FlagRule) IsAny() bool                { return f.any && len(f.allow) == 0 && len(f.must) == 0 }
func (f FlagRule) IsAllowList() bool          { return len(f.allow) > 0 || len(f.must) > 0 }
func (f FlagRule) AllowList() []AllowedOption { return f.allow }
func (f FlagRule) MustList() []AllowedOption  { return f.must }
func (f FlagRule) IsNone() bool               { return !f.any && len(f.allow) == 0 && len(f.must) == 0 }

// EffectiveConsumableAllowList is the closed set of flag rows used to parse argv at this node: explicit
// allow entries plus any must entry not already covered by an allow row (same short/long per
// [AllowedEntrySatisfiesMust]).
func (f FlagRule) EffectiveConsumableAllowList() []AllowedOption {
	if len(f.allow) == 0 && len(f.must) == 0 {
		return nil
	}
	out := append([]AllowedOption(nil), f.allow...)
outer:
	for _, m := range f.must {
		for _, a := range f.allow {
			if AllowedEntrySatisfiesMust(m, a) {
				continue outer
			}
		}
		out = append(out, m)
	}
	return out
}

// AllowedEntrySatisfiesMust is true if consuming allowEntry (from the allow list) satisfies the must
// requirement: every non-empty field in must must equal the corresponding field on allowEntry.
func AllowedEntrySatisfiesMust(must, allowEntry AllowedOption) bool {
	if must.Short == "" && must.Long == "" {
		return false
	}
	if must.Short != "" && must.Short != allowEntry.Short {
		return false
	}
	if must.Long != "" && must.Long != allowEntry.Long {
		return false
	}
	return true
}

func (f FlagRule) MarshalYAML() (interface{}, error) {
	if f.IsAny() {
		return "any", nil
	}
	if f.IsAllowList() {
		type out struct {
			Allow []AllowedOption `yaml:"allow,omitempty"`
			Must  []AllowedOption `yaml:"must,omitempty"`
		}
		return out{Allow: f.allow, Must: f.must}, nil
	}
	return "none", nil
}

func (f *FlagRule) UnmarshalYAML(n *yaml.Node) error {
	*f = FlagRule{}
	switch n.Kind {
	case yaml.ScalarNode:
		var s string
		if err := n.Decode(&s); err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(s)) {
		case "any":
			f.any = true
		case "none", "":
		default:
			return fmt.Errorf("flags: want any or none, got %q", s)
		}
		return nil
	case yaml.MappingNode:
		var aux struct {
			Allow []AllowedOption `yaml:"allow"`
			Must  []AllowedOption `yaml:"must,omitempty"`
		}
		if err := n.Decode(&aux); err != nil {
			return err
		}
		if len(aux.Allow) == 0 && len(aux.Must) == 0 {
			return fmt.Errorf("flags: mapping requires at least one of allow or must")
		}
		f.allow = aux.Allow
		f.must = aux.Must
		return nil
	default:
		return fmt.Errorf("flags: expected scalar or mapping")
	}
}

// OperandsRule is a YAML scalar "any" or "none"; omitted means none.
type OperandsRule struct {
	any bool
}

func NewOperandsAny() OperandsRule  { return OperandsRule{any: true} }
func NewOperandsNone() OperandsRule { return OperandsRule{} }

func (o OperandsRule) IsAny() bool { return o.any }

func (o OperandsRule) MarshalYAML() (interface{}, error) {
	if o.any {
		return "any", nil
	}
	return "none", nil
}

func (o *OperandsRule) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("operands: expected scalar")
	}
	var s string
	if err := n.Decode(&s); err != nil {
		return err
	}
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "none":
		o.any = false
	case "any":
		o.any = true
	default:
		return fmt.Errorf("operands: want any or none, got %q", s)
	}
	return nil
}

// LoadedAllowlist is the in-memory form of allowlist.yaml (schema v2: commands is a map keyed by argv0 basename).
type LoadedAllowlist struct {
	Version  int                          `yaml:"version"`
	Commands map[string]ReadOnlyCLIPolicy `yaml:"commands"`
}

// EffectiveGlobal returns global policy, or the default (flags: none) when Global is omitted.
func (p ReadOnlyCLIPolicy) EffectiveGlobal() GlobalSpec {
	if p.Global != nil {
		return *p.Global
	}
	return GlobalSpec{Flags: NewFlagNone()}
}

// EffectiveRoot returns root policy, or the default (flags: any, operands: any) when Root is omitted.
func (p ReadOnlyCLIPolicy) EffectiveRoot() RootSpec {
	if p.Root != nil {
		return *p.Root
	}
	return RootSpec{
		Flags:    NewFlagAny(),
		Operands: NewOperandsAny(),
	}
}

// DefaultLoadedAllowlist returns a deep copy of the builtin default (from [allowlist_default.yaml]).
func DefaultLoadedAllowlist() *LoadedAllowlist {
	base := mustParseEmbeddedDefaultAllowlist()
	return &LoadedAllowlist{
		Version:  base.Version,
		Commands: NormalizeReadOnlyCLIPolicies(base.Commands),
	}
}

// PermissiveVarArgs reports whether argv[1:] may contain shell expansions for auto-approve (simple tools only).
func (p ReadOnlyCLIPolicy) PermissiveVarArgs() bool {
	// env can invoke another program; expansions in operands must not bypass HIL.
	if p.Name == "env" {
		return false
	}
	g := p.EffectiveGlobal()
	r := p.EffectiveRoot()
	if !g.Flags.IsNone() {
		return false
	}
	if len(r.Subcommands) > 0 {
		return false
	}
	if !r.Flags.IsAny() || !r.Operands.IsAny() {
		return false
	}
	return true
}

// ValidateLoadedAllowlist returns an error if the document is unusable.
func ValidateLoadedAllowlist(ld *LoadedAllowlist) error {
	if ld == nil {
		return fmt.Errorf("allowlist is nil")
	}
	if ld.Version != AllowlistSchemaVersion {
		return fmt.Errorf("allowlist version %d, want %d", ld.Version, AllowlistSchemaVersion)
	}
	if len(ld.Commands) == 0 {
		return fmt.Errorf("allowlist commands is empty")
	}
	for k, p := range ld.Commands {
		if strings.TrimSpace(k) == "" {
			return fmt.Errorf("commands: empty map key")
		}
		name := p.Name
		if strings.TrimSpace(name) == "" {
			name = k
		}
		if name != k {
			return fmt.Errorf("commands map key %q: policy name %q must match key", k, name)
		}
		g := p.EffectiveGlobal()
		r := p.EffectiveRoot()
		if err := validateFlagRule(g.Flags, "commands["+name+"].global.flags"); err != nil {
			return err
		}
		if len(g.Flags.MustList()) > 0 {
			return fmt.Errorf("commands[%s].global.flags: must is not allowed on global flags", name)
		}
		if err := validateFlagRule(r.Flags, "commands["+name+"].root.flags"); err != nil {
			return err
		}
		if err := validateSubcommands(name, r.Subcommands); err != nil {
			return err
		}
	}
	return nil
}

func validateFlagRule(f FlagRule, ctx string) error {
	if f.IsAllowList() {
		for _, o := range f.allow {
			if o.Short == "" && o.Long == "" {
				return fmt.Errorf("%s: allow entry needs short or long", ctx)
			}
		}
	}
	if len(f.MustList()) > 0 {
		if !f.IsAllowList() {
			return fmt.Errorf("%s: must is only valid with flags allow-list form", ctx)
		}
		for i, m := range f.MustList() {
			if m.Short == "" && m.Long == "" {
				return fmt.Errorf("%s: must[%d] needs short or long", ctx, i)
			}
		}
	}
	return nil
}

func validateSubcommands(policyName string, subs SubcommandMap) error {
	for k, s := range subs {
		if strings.TrimSpace(k) == "" {
			return fmt.Errorf("commands[%s]: empty subcommand map key", policyName)
		}
		name := s.Name
		if strings.TrimSpace(name) == "" {
			name = k
		}
		if name != k {
			return fmt.Errorf("commands[%s]: subcommand map key %q does not match name %q", policyName, k, name)
		}
		ctx := "commands[" + policyName + "].subcommand[" + k + "].flags"
		if err := validateFlagRule(s.EffectiveFlags(), ctx); err != nil {
			return err
		}
		if err := validateSubcommands(policyName+"/"+k, s.Subcommands); err != nil {
			return err
		}
	}
	return nil
}

// NormalizeReadOnlyCLIPolicies returns a deep copy with subcommands sorted by name at each level.
func NormalizeReadOnlyCLIPolicies(in map[string]ReadOnlyCLIPolicy) map[string]ReadOnlyCLIPolicy {
	if in == nil {
		return nil
	}
	out := make(map[string]ReadOnlyCLIPolicy, len(in))
	for k, p := range in {
		p.Name = k
		out[k] = cloneReadOnlyCLIPolicy(p)
	}
	return out
}

func cloneReadOnlyCLIPolicy(p ReadOnlyCLIPolicy) ReadOnlyCLIPolicy {
	var out ReadOnlyCLIPolicy
	out.Name = p.Name
	if p.Global != nil {
		g := *p.Global
		out.Global = &g
	}
	if p.Root != nil {
		r := *p.Root
		r.Subcommands = cloneSubcommands(r.Subcommands)
		out.Root = &r
	}
	return out
}

func cloneSubcommands(subs SubcommandMap) SubcommandMap {
	if len(subs) == 0 {
		return nil
	}
	out := make(SubcommandMap, len(subs))
	for k, v := range subs {
		v.Name = k
		if v.Flags != nil {
			f := *v.Flags
			v.Flags = &f
		}
		if v.Operands != nil {
			o := *v.Operands
			v.Operands = &o
		}
		v.Subcommands = cloneSubcommands(v.Subcommands)
		out[k] = v
	}
	return out
}

// KubectlReadOnlyCLIPolicyForTest returns the built-in kubectl policy (normalized).
func KubectlReadOnlyCLIPolicyForTest() ReadOnlyCLIPolicy {
	base := mustParseEmbeddedDefaultAllowlist()
	p, ok := base.Commands["kubectl"]
	if !ok {
		panic("config: allowlist_default.yaml: missing kubectl policy")
	}
	if p.Root != nil {
		r := *p.Root
		r.Subcommands = cloneSubcommands(r.Subcommands)
		p.Root = &r
	}
	return p
}
