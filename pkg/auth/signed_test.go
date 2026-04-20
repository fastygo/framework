package auth

import (
	"strings"
	"testing"
)

type signedPayload struct {
	A int    `json:"a"`
	B string `json:"b"`
}

func TestSignedEncode_RoundTrip(t *testing.T) {
	want := signedPayload{A: 7, B: "hello"}
	encoded, err := SignedEncode(want, "secret-x")
	if err != nil {
		t.Fatalf("SignedEncode: %v", err)
	}
	if !strings.Contains(encoded, ".") {
		t.Fatalf("encoded value must contain a '.' separator, got %q", encoded)
	}

	var got signedPayload
	if err := SignedDecode(encoded, "secret-x", &got); err != nil {
		t.Fatalf("SignedDecode: %v", err)
	}
	if got != want {
		t.Fatalf("payload mismatch: got %+v, want %+v", got, want)
	}
}

func TestSignedEncode_MissingSecret(t *testing.T) {
	_, err := SignedEncode(signedPayload{A: 1}, "")
	if err == nil {
		t.Fatalf("SignedEncode with empty secret must return an error")
	}
	if !strings.Contains(err.Error(), "secret") {
		t.Fatalf("error should mention missing secret, got: %v", err)
	}
}

func TestSignedDecode_MissingSecret(t *testing.T) {
	encoded, err := SignedEncode(signedPayload{A: 1}, "ok")
	if err != nil {
		t.Fatalf("SignedEncode: %v", err)
	}

	var got signedPayload
	err = SignedDecode(encoded, "", &got)
	if err == nil {
		t.Fatalf("SignedDecode with empty secret must return an error")
	}
	if !strings.Contains(err.Error(), "secret") {
		t.Fatalf("error should mention missing secret, got: %v", err)
	}
}

func TestSignedDecode_NoSeparator(t *testing.T) {
	var got signedPayload
	err := SignedDecode("payload-without-signature", "secret", &got)
	if err == nil {
		t.Fatalf("SignedDecode must reject a value without a '.' separator")
	}
}

func TestSignedDecode_BadSignature(t *testing.T) {
	encoded, err := SignedEncode(signedPayload{A: 1}, "real-secret")
	if err != nil {
		t.Fatalf("SignedEncode: %v", err)
	}

	var got signedPayload
	if err := SignedDecode(encoded, "different-secret", &got); err == nil {
		t.Fatalf("SignedDecode must reject a value signed with another secret")
	}
}

func TestSignedDecode_TamperedPayload(t *testing.T) {
	encoded, err := SignedEncode(signedPayload{A: 1, B: "ok"}, "k")
	if err != nil {
		t.Fatalf("SignedEncode: %v", err)
	}

	// Flip a byte in the payload half (before the dot). The HMAC is
	// computed over the encoded payload string, so any change must
	// fail verification.
	idx := strings.IndexByte(encoded, '.')
	if idx <= 0 {
		t.Fatalf("expected '.' in encoded value, got %q", encoded)
	}
	bytesValue := []byte(encoded)
	if bytesValue[0] == 'a' {
		bytesValue[0] = 'b'
	} else {
		bytesValue[0] = 'a'
	}

	var got signedPayload
	if err := SignedDecode(string(bytesValue), "k", &got); err == nil {
		t.Fatalf("SignedDecode must reject a tampered payload")
	}
}

func TestSignedDecode_BadBase64(t *testing.T) {
	// Construct a value that has the right shape (".") and a valid
	// HMAC for the literal "***" payload, but the payload is not
	// valid base64.
	payload := "***"
	mac := computeHMAC(payload, "k")
	value := payload + "." + mac

	var got signedPayload
	err := SignedDecode(value, "k", &got)
	if err == nil {
		t.Fatalf("SignedDecode must reject a payload that is not valid base64")
	}
}

func TestRandomToken_Length(t *testing.T) {
	tests := []struct {
		name      string
		byteLen   int
		wantChars int
	}{
		{"default-on-zero", 0, 32},     // hex(16) = 32 chars
		{"default-on-negative", -5, 32}, // hex(16) = 32 chars
		{"explicit-32-bytes", 32, 64},
		{"explicit-1-byte", 1, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := RandomToken(tc.byteLen)
			if len(got) != tc.wantChars {
				t.Fatalf("len = %d, want %d (token: %q)", len(got), tc.wantChars, got)
			}
		})
	}
}

func TestRandomToken_Uniqueness(t *testing.T) {
	const samples = 256
	seen := make(map[string]struct{}, samples)
	for i := 0; i < samples; i++ {
		token := RandomToken(16)
		if _, dup := seen[token]; dup {
			t.Fatalf("duplicate token after %d samples: %q", i, token)
		}
		seen[token] = struct{}{}
	}
}
