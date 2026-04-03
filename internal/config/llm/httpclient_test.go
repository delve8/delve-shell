package configllm

import (
	"net/http"
	"testing"
	"time"
)

func TestNewLLMHTTPClient_UsesClonedTransportAndIdleTimeout(t *testing.T) {
	c := NewLLMHTTPClient(30 * time.Second)
	if c.Timeout != 30*time.Second {
		t.Fatalf("Timeout: got %v", c.Timeout)
	}
	tr, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("Transport type %T, want *http.Transport", c.Transport)
	}
	if tr.DisableKeepAlives {
		t.Fatal("expected keep-alive (DisableKeepAlives false)")
	}
	if tr.IdleConnTimeout != 90*time.Second {
		t.Fatalf("IdleConnTimeout: got %v", tr.IdleConnTimeout)
	}
	if !tr.ForceAttemptHTTP2 {
		t.Fatal("expected ForceAttemptHTTP2 true")
	}
	if tr.DialContext == nil {
		t.Fatal("expected DialContext set")
	}
}
