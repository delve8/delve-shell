package history

import "testing"

func TestSwitchSessionIDFromSlashLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in     string
		wantID string
		wantOK bool
	}{
		{"/history abc", "abc", true},
		{"/history  abc  rest of desc", "abc", true},
		{"/history\tid2", "id2", true},
		{"/history", "", false},
		{"/history ", "", false},
		{"/historybook", "", false},
		{"  /history x ", "x", true},
		{"/history abc123 [Current]", "abc123", true},
		{"/history  id2  [Current]  extra", "id2", true},
	}
	for _, tt := range tests {
		got, ok := SwitchSessionIDFromSlashLine(tt.in)
		if ok != tt.wantOK || got != tt.wantID {
			t.Fatalf("SwitchSessionIDFromSlashLine(%q) = (%q, %v); want (%q, %v)", tt.in, got, ok, tt.wantID, tt.wantOK)
		}
	}
}
