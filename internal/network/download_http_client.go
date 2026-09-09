package network

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

type downloadTransportKey struct {
	forceHTTP1 bool
	timeoutNs  int64
}

var (
	downloadTransportMu sync.Mutex
	downloadTransports  = map[downloadTransportKey]*http.Transport{}
)

func sharedDownloadTransport(headerTimeout time.Duration, forceHTTP1 bool) *http.Transport {
	if headerTimeout <= 0 {
		headerTimeout = 10 * time.Second
	}
	key := downloadTransportKey{forceHTTP1: forceHTTP1, timeoutNs: headerTimeout.Nanoseconds()}

	downloadTransportMu.Lock()
	defer downloadTransportMu.Unlock()
	if tr, ok := downloadTransports[key]; ok {
		return tr
	}

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = headerTimeout
	if forceHTTP1 {
		tr.ForceAttemptHTTP2 = false
	}
	downloadTransports[key] = tr
	return tr
}

// NewDownloadHTTPClient returns an http.Client suitable for large/binary downloads.
// It uses a cookie jar so redirects that set cookies behave closer to browsers.
//
// Transports are reused across calls with the same header timeout and HTTP version
// so TLS sessions and HTTP/2 connections can be pooled. A new Transport per request
// (via Clone) forces a full handshake on every font file — the dominant cost when
// installing families like Roboto that are published as many small TTF URLs.
func NewDownloadHTTPClient(headerTimeout time.Duration, forceHTTP1 bool, onRedirect func(from *url.URL, to *url.URL, viaCount int)) *http.Client {
	tr := sharedDownloadTransport(headerTimeout, forceHTTP1)

	jar, _ := cookiejar.New(nil)

	client := &http.Client{
		Transport: tr,
		Jar:       jar,
	}

	if onRedirect != nil {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if len(via) == 0 {
				return http.ErrUseLastResponse
			}
			last := via[len(via)-1]
			if last != nil && last.URL != nil && req != nil && req.URL != nil {
				onRedirect(last.URL, req.URL, len(via))
			}
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return nil
		}
	}

	return client
}

// ShouldRetryGoDownloadStatus reports whether the Go downloader should retry the request
// for transient upstream failures.
func ShouldRetryGoDownloadStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests: // 429
		return true
	case http.StatusBadGateway: // 502
		return true
	case http.StatusServiceUnavailable: // 503
		return true
	default:
		return false
	}
}

// FormatHTTPHeadersForDebug returns a compact, stable header dump for debug logs.
func FormatHTTPHeadersForDebug(h http.Header) string {
	if h == nil {
		return ""
	}
	keys := []string{
		"location",
		"content-type",
		"cache-control",
		"x-amzn-waf-action",
		"retry-after",
	}
	var b strings.Builder
	for _, k := range keys {
		if v := h.Get(k); v != "" {
			if b.Len() > 0 {
				b.WriteString(", ")
			}
			b.WriteString(k)
			b.WriteString("=")
			b.WriteString(v)
		}
	}
	return b.String()
}
