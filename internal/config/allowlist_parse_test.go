package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestParseAllowlistYAML_roundTrip(t *testing.T) {
	def := DefaultLoadedAllowlist()
	if err := ValidateLoadedAllowlist(def); err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(def)
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
