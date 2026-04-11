package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSkillSlashOptions_RootIncludesReservedNew(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	writeTestSkill(t, "demo", "Demo")

	opts := getSkillSlashOptions("en", "")
	if len(opts) < 4 {
		t.Fatalf("expected reserved rows plus installed skills, got %#v", opts)
	}
	if opts[0].Cmd != "/skill demo" {
		t.Fatalf("expected installed skill first, got %#v", opts[0])
	}
	if opts[1].Cmd != "/skill New" {
		t.Fatalf("expected New reserved row after installed skills, got %#v", opts[1])
	}
	if opts[2].Cmd != "/skill Remove" {
		t.Fatalf("expected Remove reserved row, got %#v", opts[2])
	}
	if opts[3].Cmd != "/skill Update" {
		t.Fatalf("expected Update reserved row, got %#v", opts[3])
	}
}

func TestGetSkillSlashOptions_ExactReservedNewStaysDistinctFromSkillNamedNew(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	writeTestSkill(t, "new", "new")

	opts := getSkillSlashOptions("en", "New")
	if len(opts) != 1 || opts[0].Cmd != "/skill New" {
		t.Fatalf("expected exact reserved row only, got %#v", opts)
	}
}

func TestGetSkillSlashOptions_LowercaseNewShowsReservedAndSkillNamedNew(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	writeTestSkill(t, "new", "new")

	opts := getSkillSlashOptions("en", "new")
	if len(opts) != 2 {
		t.Fatalf("expected reserved row plus installed skill, got %#v", opts)
	}
	if opts[0].Cmd != "/skill new" || opts[1].Cmd != "/skill New" {
		t.Fatalf("unexpected options %#v", opts)
	}
}

func TestGetSkillSlashOptions_ExactReservedRemoveAndUpdateStayDistinct(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	writeTestSkill(t, "remove", "remove")
	writeTestSkill(t, "update", "update")

	removeOpts := getSkillSlashOptions("en", "Remove")
	if len(removeOpts) != 1 || removeOpts[0].Cmd != "/skill Remove" {
		t.Fatalf("expected exact Remove reserved row only, got %#v", removeOpts)
	}

	updateOpts := getSkillSlashOptions("en", "Update")
	if len(updateOpts) != 1 || updateOpts[0].Cmd != "/skill Update" {
		t.Fatalf("expected exact Update reserved row only, got %#v", updateOpts)
	}
}

func writeTestSkill(t *testing.T, dirName, metaName string) {
	t.Helper()
	skillDir := filepath.Join(os.Getenv("DELVE_SHELL_ROOT"), "skills", dirName)
	if err := os.MkdirAll(skillDir, 0o700); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + metaName + "\ndescription: test\n---\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
