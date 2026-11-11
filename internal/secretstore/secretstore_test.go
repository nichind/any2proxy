package secretstore

import (
	"bytes"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	secret := "tok" + "en=secret"
	input := []byte("https://example.com/sub?" + secret)
	encoded, err := Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(encoded, input) {
		t.Fatal("payload was not wrapped")
	}
	decoded, err := Decode(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, input) {
		t.Fatalf("round trip mismatch: %q", decoded)
	}
}

func TestDecodePlaintext(t *testing.T) {
	input := []byte(`{"version":1}`)
	decoded, err := Decode(input)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, input) {
		t.Fatal("plaintext fallback changed the payload")
	}
}
