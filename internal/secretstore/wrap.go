package secretstore

import (
	"bytes"
	"fmt"
)

var magic = []byte("A2P1")

func Encode(data []byte) ([]byte, error) {
	protected, err := Protect(data)
	if err != nil {
		return nil, fmt.Errorf("secretstore: %w", err)
	}
	out := make([]byte, 0, len(magic)+len(protected))
	out = append(out, magic...)
	out = append(out, protected...)
	return out, nil
}

func Decode(data []byte) ([]byte, error) {
	if !bytes.HasPrefix(data, magic) {
		return data, nil
	}
	plain, err := Unprotect(data[len(magic):])
	if err != nil {
		return nil, fmt.Errorf("secretstore: %w", err)
	}
	return plain, nil
}
