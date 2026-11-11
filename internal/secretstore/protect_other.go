//go:build !windows

package secretstore

func Protect(data []byte) ([]byte, error) {
	return data, nil
}

func Unprotect(data []byte) ([]byte, error) {
	return data, nil
}
