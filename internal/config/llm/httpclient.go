package configllm

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const llmMaxNetworkRetries = 2

// NewLLMHTTPClient returns an *http.Client for OpenAI-compatible chat APIs.
//
// It uses a dedicated [http.Transport] with defaults aligned to [http.DefaultTransport]
// (proxy from environment, HTTP/2 when available, idle timeout). It retries only network/transport
// errors up to 2 times (3 total attempts) when the request body can be replayed.
//
// timeout is passed to [http.Client.Timeout]; zero means no overall request timeout.
func NewLLMHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	base := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Timeout: timeout,
		Transport: retryRoundTripper{
			base:       base,
			maxRetries: llmMaxNetworkRetries,
			backoffs:   []time.Duration{300 * time.Millisecond, 1 * time.Second},
		},
	}
}

type retryRoundTripper struct {
	base       http.RoundTripper
	maxRetries int
	backoffs   []time.Duration
}

func (rt retryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := rt.base
	if base == nil {
		base = http.DefaultTransport
	}
	attemptReq := req
	for attempt := 0; ; attempt++ {
		resp, err := base.RoundTrip(attemptReq)
		if err == nil {
			return resp, nil
		}
		if attempt >= rt.maxRetries || !shouldRetryNetworkError(err) || !requestReplayable(req) {
			return nil, err
		}
		if waitErr := sleepWithContext(req.Context(), rt.backoff(attempt)); waitErr != nil {
			return nil, waitErr
		}
		cloned, cloneErr := cloneRequestForRetry(req)
		if cloneErr != nil {
			return nil, err
		}
		attemptReq = cloned
	}
}

func (rt retryRoundTripper) backoff(attempt int) time.Duration {
	if attempt < 0 || len(rt.backoffs) == 0 {
		return 0
	}
	if attempt >= len(rt.backoffs) {
		return rt.backoffs[len(rt.backoffs)-1]
	}
	return rt.backoffs[attempt]
}

func requestReplayable(req *http.Request) bool {
	if req == nil {
		return false
	}
	return req.Body == nil || req.GetBody != nil
}

func cloneRequestForRetry(req *http.Request) (*http.Request, error) {
	clone := req.Clone(req.Context())
	if req.Body == nil {
		return clone, nil
	}
	if req.GetBody == nil {
		return nil, errors.New("request body is not replayable")
	}
	body, err := req.GetBody()
	if err != nil {
		return nil, err
	}
	clone.Body = body
	return clone, nil
}

func shouldRetryNetworkError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) || errors.Is(err, net.ErrClosed) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unexpected eof") || msg == "eof"
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
