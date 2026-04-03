package configllm

import (
	"net"
	"net/http"
	"time"
)

// NewLLMHTTPClient returns an *http.Client for OpenAI-compatible chat APIs.
//
// It uses a dedicated [http.Transport] with defaults aligned to [http.DefaultTransport]
// (proxy from environment, HTTP/2 when available, idle timeout). There is no automatic
// POST retry; callers may submit again after transient failures.
//
// timeout is passed to [http.Client.Timeout]; zero means no overall request timeout.
func NewLLMHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           dialer.DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}
