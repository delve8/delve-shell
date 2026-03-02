package hil

import "testing"

func TestContainsWriteRedirection(t *testing.T) {
	tests := []struct {
		cmd   string
		want  bool
	}{
		{"ping -c 1 x.com", false},
		{"ping -c 1 x.com > /tmp/out", true},
		{"echo hello >> log.txt", true},
		{"cat a 2> err", true},
		{"cat a 2>> err", true},
		{"echo '> not redirect'", false},
		{"echo 'foo' > f", true},
		{"echo a >= b", false},
		{"echo a => b", false},
		{"true", false},
		{"", false},
	}
	for _, tt := range tests {
		got := ContainsWriteRedirection(tt.cmd)
		if got != tt.want {
			t.Errorf("ContainsWriteRedirection(%q) = %v, want %v", tt.cmd, got, tt.want)
		}
	}
}
