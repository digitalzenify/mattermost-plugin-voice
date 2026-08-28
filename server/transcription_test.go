package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignAndVerifyDownloadToken(t *testing.T) {
	sig := signDownloadToken("s3cr3t", "fileid123", 1234567890)
	if sig == "" {
		t.Fatal("expected non-empty signature")
	}

	// Same inputs produce the same signature.
	if got := signDownloadToken("s3cr3t", "fileid123", 1234567890); got != sig {
		t.Errorf("signature not deterministic: got %q, want %q", got, sig)
	}

	// Any change to the inputs changes the signature.
	if got := signDownloadToken("s3cr3t", "fileid123", 1234567891); got == sig {
		t.Error("expected different exp to produce a different signature")
	}
	if got := signDownloadToken("other-secret", "fileid123", 1234567890); got == sig {
		t.Error("expected different secret to produce a different signature")
	}
}

func TestHasValidBearerSecret(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/transcriptions", nil)
	req.Header.Set("Authorization", "Bearer correct-secret")
	if !hasValidBearerSecret(req, "correct-secret") {
		t.Error("expected matching bearer secret to validate")
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/transcriptions", nil)
	req2.Header.Set("Authorization", "Bearer wrong-secret")
	if hasValidBearerSecret(req2, "correct-secret") {
		t.Error("expected mismatched bearer secret to fail")
	}

	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/transcriptions", nil)
	if hasValidBearerSecret(req3, "correct-secret") {
		t.Error("expected missing Authorization header to fail")
	}

	req4 := httptest.NewRequest(http.MethodPost, "/api/v1/transcriptions", nil)
	req4.Header.Set("Authorization", "Bearer ")
	if hasValidBearerSecret(req4, "") {
		t.Error("expected empty configured secret to never validate")
	}
}

func TestInt64FromProps(t *testing.T) {
	cases := []struct {
		in   any
		want int64
	}{
		{int64(42), 42},
		{42, 42},
		{float64(42), 42},
		{"42", 0},
		{nil, 0},
	}
	for _, tc := range cases {
		if got := int64FromProps(tc.in); got != tc.want {
			t.Errorf("int64FromProps(%v) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
