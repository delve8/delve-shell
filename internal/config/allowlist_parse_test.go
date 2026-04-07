package config

import (
	"strings"
	"testing"
)

func TestKubectlGlobalSurvivesParseAndEncode(t *testing.T) {
	ld := DefaultLoadedAllowlist()
	k, ok := ld.Commands["kubectl"]
	if !ok {
		t.Fatal("missing kubectl")
	}
	if k.Global == nil || !k.Global.Flags.IsAllowList() {
		t.Fatalf("kubectl global allow-list lost after load: global=%v", k.Global)
	}
	data, err := EncodeAllowlistYAML(ld)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "all-namespaces") {
		i := strings.Index(string(data), "\n  kubectl:\n")
		excerpt := string(data)
		if i >= 0 && i+600 < len(excerpt) {
			excerpt = excerpt[i : i+600]
		}
		t.Fatalf("encoded allowlist missing kubectl global flag; kubectl excerpt:\n%s", excerpt)
	}
}

func TestValidateLoadedAllowlist_globalMustForbidden(t *testing.T) {
	ld := &LoadedAllowlist{
		Version: AllowlistSchemaVersion,
		Commands: map[string]ReadOnlyCLIPolicy{
			"bad": {
				Global: &GlobalSpec{
					Flags: NewFlagAllow([]AllowedOption{{Short: "x"}}).WithMust([]AllowedOption{{Short: "x"}}),
				},
			},
		},
	}
	if err := ValidateLoadedAllowlist(ld); err == nil {
		t.Fatal("expected error when global.flags has must")
	}
}

func TestValidateLoadedAllowlist_mustImpliesAllowWithoutDuplicateAllowRow(t *testing.T) {
	ld := &LoadedAllowlist{
		Version: AllowlistSchemaVersion,
		Commands: map[string]ReadOnlyCLIPolicy{
			"ok": {
				Root: &RootSpec{
					Flags: NewFlagAllow([]AllowedOption{{Short: "x"}}).WithMust([]AllowedOption{{Short: "y"}}),
				},
			},
		},
	}
	if err := ValidateLoadedAllowlist(ld); err != nil {
		t.Fatalf("must-only-in-addition-to-allow should validate: %v", err)
	}
}

func TestValidateLoadedAllowlist_mustOnlyMappingValidates(t *testing.T) {
	const y = `version: 2
commands:
  demo:
    root:
      flags:
        must:
          - short: z
            value: none
      operands: any
`
	ld, err := ParseAllowlistYAML([]byte(y))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateLoadedAllowlist(ld); err != nil {
		t.Fatal(err)
	}
}

func TestParseAllowlistYAML_roundTrip(t *testing.T) {
	def := DefaultLoadedAllowlist()
	if err := ValidateLoadedAllowlist(def); err != nil {
		t.Fatal(err)
	}
	data, err := EncodeAllowlistYAML(def)
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseAllowlistYAML(data)
	if err != nil {
		t.Fatal(err)
	}
	if back.Version != AllowlistSchemaVersion {
		t.Fatalf("version got %d want %d", back.Version, AllowlistSchemaVersion)
	}
	if len(back.Commands) != len(def.Commands) {
		t.Fatalf("commands len got %d want %d", len(back.Commands), len(def.Commands))
	}
}

func TestParseAllowlistYAML_rejectsWrongVersion(t *testing.T) {
	const y = `version: 1
commands:
  pwd: {}
`
	if _, err := ParseAllowlistYAML([]byte(y)); err == nil {
		t.Fatal("expected error for version != 2")
	}
}

func TestParseAllowlistYAML_missingCommands(t *testing.T) {
	const y = `version: 2
`
	if _, err := ParseAllowlistYAML([]byte(y)); err == nil {
		t.Fatal("expected error for missing commands")
	}
}

func TestParseAllowlistYAML_omittedGlobalRoot_defaults(t *testing.T) {
	const y = `version: 2
commands:
  pwd: {}
`
	ld, err := ParseAllowlistYAML([]byte(y))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateLoadedAllowlist(ld); err != nil {
		t.Fatal(err)
	}
	p := ld.Commands["pwd"]
	if p.Global != nil || p.Root != nil {
		t.Fatalf("expected omitted global/root to stay nil, got global=%v root=%v", p.Global, p.Root)
	}
	g := p.EffectiveGlobal()
	if !g.Flags.IsNone() {
		t.Fatalf("EffectiveGlobal flags: want none, got %#v", g.Flags)
	}
	r := p.EffectiveRoot()
	if !r.Flags.IsAny() || !r.Operands.IsAny() {
		t.Fatalf("EffectiveRoot: want flags any + operands any, got flags=%#v operands=%#v", r.Flags, r.Operands)
	}
	if !p.PermissiveVarArgs() {
		t.Fatal("PermissiveVarArgs should be true for default simple tool")
	}
}
