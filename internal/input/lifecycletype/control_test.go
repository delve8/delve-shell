package inputlifecycletype

import "testing"

func TestResolveEscActionPriority(t *testing.T) {
	tests := []struct {
		name   string
		ctx    ControlContext
		want   ControlAction
		wantOK bool
	}{
		{
			name:   "overlay wins first",
			ctx:    ControlContext{HasActiveOverlay: true, HasPreInputState: true, WaitingForAI: true},
			want:   ControlCloseOverlay,
			wantOK: true,
		},
		{
			name:   "pre input wins before cancel",
			ctx:    ControlContext{HasPreInputState: true, WaitingForAI: true},
			want:   ControlClearPreInput,
			wantOK: true,
		},
		{
			name:   "cancel when only AI running",
			ctx:    ControlContext{WaitingForAI: true},
			want:   ControlCancelProcessing,
			wantOK: true,
		},
		{
			name:   "noop when nothing active",
			ctx:    ControlContext{},
			want:   "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ResolveEscAction(tt.ctx)
			if ok != tt.wantOK {
				t.Fatalf("ResolveEscAction() ok=%v want %v", ok, tt.wantOK)
			}
			if got != tt.want {
				t.Fatalf("ResolveEscAction()=%q want %q", got, tt.want)
			}
		})
	}
}
