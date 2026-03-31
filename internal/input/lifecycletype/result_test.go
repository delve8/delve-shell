package inputlifecycletype

import (
	"errors"
	"testing"
)

func TestConsumedResultCopiesOutputs(t *testing.T) {
	evs := []OutputEvent{{Kind: OutputStatusChange, Status: &StatusPayload{Key: "running"}}}
	got := ConsumedResult(evs...)
	evs[0].Status.Key = "mutated"

	if !got.Consumed {
		t.Fatal("ConsumedResult should mark result as consumed")
	}
	if got.Err != nil {
		t.Fatal("ConsumedResult should not set error")
	}
	if got.Outputs[0].Status.Key != "running" {
		t.Fatalf("expected copied outputs to remain stable, got %q", got.Outputs[0].Status.Key)
	}
}

func TestErrorResult(t *testing.T) {
	wantErr := errors.New("boom")
	got := ErrorResult(wantErr)

	if !got.Consumed {
		t.Fatal("ErrorResult should mark result as consumed")
	}
	if !errors.Is(got.Err, wantErr) {
		t.Fatal("ErrorResult should preserve the original error")
	}
}
