package agentctx

import (
	"context"
	"testing"
)

func TestSkillSlashTurn(t *testing.T) {
	ctx := context.Background()
	if _, ok := SkillSlashSkillName(ctx); ok {
		t.Fatal("expected no skill without context")
	}
	ctx = WithSkillSlashTurn(ctx, "  myskill  ")
	n, ok := SkillSlashSkillName(ctx)
	if !ok || n != "myskill" {
		t.Fatalf("got name=%q ok=%v", n, ok)
	}
	ctx = WithSkillSlashTurn(context.Background(), "")
	if _, ok := SkillSlashSkillName(ctx); ok {
		t.Fatal("empty name should not set marker")
	}
}
