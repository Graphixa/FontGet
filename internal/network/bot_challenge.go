package network

import (
	"net/http"
	"strings"
)

// IsBotChallenge returns true when the HTTP response looks like a WAF/bot challenge
// rather than a normal download response.
//
// This is intentionally generic: it should not mention or depend on any particular
// upstream source. Extend it carefully as new challenge patterns are observed.
func IsBotChallenge(resp *http.Response) bool {
	if resp == nil {
		return false
	}

	// Observed AWS WAF signal: 202 + x-amzn-waf-action: challenge
	// Use case-insensitive header lookup: synthetic http.Response values may use non-canonical map keys.
	if resp.StatusCode == http.StatusAccepted && strings.EqualFold(HeaderValueFold(resp.Header, "x-amzn-waf-action"), "challenge") {
		return true
	}

	return false
}

// HeaderValueFold returns the first header value for a key, matching key case-insensitively.
func HeaderValueFold(h http.Header, key string) string {
	if h == nil {
		return ""
	}
	for k, vv := range h {
		if strings.EqualFold(k, key) && len(vv) > 0 {
			return vv[0]
		}
	}
	return ""
}

