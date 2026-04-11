package configllm

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewLLMHTTPClient_UsesClonedTransportAndIdleTimeout(t *testing.T) {
	c := NewLLMHTTPClient(30 * time.Second)
	if c.Timeout != 30*time.Second {
		t.Fatalf("Timeout: got %v", c.Timeout)
	}
	rt, ok := c.Transport.(retryRoundTripper)
	if !ok {
		t.Fatalf("Transport type %T, want retryRoundTripper", c.Transport)
	}
	tr, ok := rt.base.(*http.Transport)
	if !ok {
		t.Fatalf("wrapped transport type %T, want *http.Transport", rt.base)
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
	if rt.maxRetries != llmMaxNetworkRetries {
		t.Fatalf("maxRetries=%d", rt.maxRetries)
	}
}

func TestRetryRoundTripper_RetriesNetworkErrors(t *testing.T) {
	var calls int
	rt := retryRoundTripper{
		maxRetries: 2,
		backoffs:   []time.Duration{0, 0},
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls < 3 {
				return nil, temporaryNetErr{err: errors.New("dial tcp timeout")}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	req, err := http.NewRequest(http.MethodPost, "https://example.com", bytes.NewReader([]byte(`{"x":1}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if calls != 3 {
		t.Fatalf("calls=%d want 3", calls)
	}
}

func TestRetryRoundTripper_DoesNotRetryHTTPResponse(t *testing.T) {
	var calls int
	rt := retryRoundTripper{
		maxRetries: 2,
		backoffs:   []time.Duration{0, 0},
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader("bad gateway")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	req, err := http.NewRequest(http.MethodPost, "https://example.com", bytes.NewReader([]byte(`{"x":1}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if calls != 1 {
		t.Fatalf("calls=%d want 1", calls)
	}
}

func TestRetryRoundTripper_RetriesUnexpectedEOF(t *testing.T) {
	var calls int
	rt := retryRoundTripper{
		maxRetries: 2,
		backoffs:   []time.Duration{0, 0},
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			if calls < 3 {
				return nil, io.ErrUnexpectedEOF
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	req, err := http.NewRequest(http.MethodPost, "https://example.com", bytes.NewReader([]byte(`{"x":1}`)))
	if err != nil {
		t.Fatal(err)
	}
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if calls != 3 {
		t.Fatalf("calls=%d want 3", calls)
	}
}

func TestRetryRoundTripper_DoesNotRetryNonReplayableBody(t *testing.T) {
	var calls int
	rt := retryRoundTripper{
		maxRetries: 2,
		backoffs:   []time.Duration{0, 0},
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			calls++
			return nil, temporaryNetErr{err: errors.New("network down")}
		}),
	}
	req, err := http.NewRequest(http.MethodPost, "https://example.com", io.NopCloser(strings.NewReader("x")))
	if err != nil {
		t.Fatal(err)
	}
	req.GetBody = nil
	_, err = rt.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Fatalf("calls=%d want 1", calls)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type temporaryNetErr struct{ err error }

func (e temporaryNetErr) Error() string   { return e.err.Error() }
func (e temporaryNetErr) Timeout() bool   { return true }
func (e temporaryNetErr) Temporary() bool { return true }
