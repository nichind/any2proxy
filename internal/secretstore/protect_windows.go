//go:build windows

package secretstore

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	crypt32       = syscall.NewLazyDLL("crypt32.dll")
	kernel32      = syscall.NewLazyDLL("kernel32.dll")
	procProtect   = crypt32.NewProc("CryptProtectData")
	procUnprotect = crypt32.NewProc("CryptUnprotectData")
	procLocalFree = kernel32.NewProc("LocalFree")
)

type dataBlob struct {
	size uint32
	data *byte
}

func Protect(data []byte) ([]byte, error) {
	return call(procProtect, data, "CryptProtectData")
}

func Unprotect(data []byte) ([]byte, error) {
	return call(procUnprotect, data, "CryptUnprotectData")
}

func call(proc *syscall.LazyProc, data []byte, name string) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	in := dataBlob{size: uint32(len(data)), data: &data[0]}
	var out dataBlob
	ok, _, err := proc.Call(
		uintptr(unsafe.Pointer(&in)),
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&out)),
	)
	if ok == 0 {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	defer localFree(out.data)
	// SAFETY: out.data points to out.size bytes allocated by the Crypt32 call.
	return append([]byte(nil), unsafe.Slice(out.data, out.size)...), nil
}

func localFree(ptr *byte) {
	if ptr == nil {
		return
	}
	_, _, _ = procLocalFree.Call(uintptr(unsafe.Pointer(ptr)))
}
