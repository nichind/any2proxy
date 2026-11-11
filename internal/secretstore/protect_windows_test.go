//go:build windows

package secretstore

import (
	"bytes"
	"testing"
)

func TestProtectHidesPayload(t *testing.T) {
	secret := "tok" + "en=secret"
	input := []byte("https://example.com/sub?" + secret)
	encoded, err := Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(secret)) {
		t.Fatal("encoded payload contains the plaintext")
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, input) {
		t.Fatal("DPAPI round trip mismatch")
	}
}
