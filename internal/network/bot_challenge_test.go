package network

import (
	"net/http"
	"testing"
)

func TestIsBotChallenge(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if IsBotChallenge(nil) {
			t.Fatalf("expected false for nil response")
		}
	})

	t.Run("non-challenge", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{},
		}
		if IsBotChallenge(resp) {
			t.Fatalf("expected false for non-challenge response")
		}
	})

	t.Run("aws-waf-challenge", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusAccepted,
			Header: http.Header{
				"x-amzn-waf-action": []string{"challenge"},
			},
		}
		if !IsBotChallenge(resp) {
			t.Fatalf("expected true for AWS WAF challenge response")
		}
	})
}

