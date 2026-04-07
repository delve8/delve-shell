package config

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// readOnlyCLIPolicyBody is the YAML value under each commands map key (name is the key, not a field).
type readOnlyCLIPolicyBody struct {
	Global *GlobalSpec `yaml:"global,omitempty"`
	Root   *RootSpec   `yaml:"root,omitempty"`
}

// UnmarshalYAML decodes allowlist root: version + commands map; each map key becomes [ReadOnlyCLIPolicy.Name].
func (ld *LoadedAllowlist) UnmarshalYAML(n *yaml.Node) error {
	var aux struct {
		Version  int                              `yaml:"version"`
		Commands map[string]readOnlyCLIPolicyBody `yaml:"commands"`
	}
	if err := n.Decode(&aux); err != nil {
		return err
	}
	if aux.Commands == nil {
		return fmt.Errorf("allowlist: missing commands")
	}
	ld.Version = aux.Version
	ld.Commands = make(map[string]ReadOnlyCLIPolicy, len(aux.Commands))
	for k, v := range aux.Commands {
		if strings.TrimSpace(k) == "" {
			return fmt.Errorf("allowlist: empty commands map key")
		}
		ld.Commands[k] = ReadOnlyCLIPolicy{Name: k, Global: v.Global, Root: v.Root}
	}
	return nil
}

// MarshalYAML encodes version and commands as a mapping with lexicographically sorted keys for stable diffs.
func (ld *LoadedAllowlist) MarshalYAML() (interface{}, error) {
	if ld == nil {
		return nil, nil
	}
	keys := make([]string, 0, len(ld.Commands))
	for k := range ld.Commands {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	cmdNode := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	for _, k := range keys {
		p := ld.Commands[k]
		body := readOnlyCLIPolicyBody{Global: p.Global, Root: p.Root}
		b, err := yaml.Marshal(body)
		if err != nil {
			return nil, err
		}
		var doc yaml.Node
		if err := yaml.Unmarshal(b, &doc); err != nil {
			return nil, err
		}
		if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 {
			return nil, fmt.Errorf("allowlist: internal yaml encode commands[%q]", k)
		}
		kn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: k}
		cmdNode.Content = append(cmdNode.Content, kn, doc.Content[0])
	}

	vn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(ld.Version)}
	root := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	root.Content = append(root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "version"}, vn,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "commands"}, cmdNode,
	)
	return root, nil
}
