//go:build cgo

package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/nichind/any2proxy/internal/service"
	"github.com/nichind/any2proxy/internal/version"
)

var (
	handles sync.Map
	nextID  atomic.Uintptr
)

type callError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type callResult struct {
	Result any        `json:"result,omitempty"`
	Error  *callError `json:"error,omitempty"`
}

type parseParams struct {
	Body   string `json:"body"`
	Format string `json:"format"`
}

type compileParams struct {
	Body   string `json:"body"`
	Format string `json:"format"`
	Target string `json:"target"`
	Port   uint16 `json:"port"`
	Strict bool   `json:"strict"`
}

type runParams struct {
	Body   string `json:"body"`
	Format string `json:"format"`
	Target string `json:"target"`
	Binary string `json:"binary"`
	Port   uint16 `json:"port"`
	Strict bool   `json:"strict"`
}

type stopParams struct {
	SessionID string `json:"session_id"`
}

//export a2p_version
func a2p_version() *C.char {
	return C.CString(version.Version)
}

//export a2p_new
func a2p_new() C.uintptr_t {
	id := nextID.Add(1)
	handles.Store(id, service.New())
	return C.uintptr_t(id)
}

//export a2p_call
func a2p_call(handle C.uintptr_t, method *C.char, params *C.char) *C.char {
	value, ok := handles.Load(uintptr(handle))
	if !ok {
		return C.CString(`{"error":{"code":"invalid_input","message":"invalid handle"}}`)
	}
	svc := value.(*service.Service)
	name := C.GoString(method)
	body := C.GoString(params)
	result, err := dispatch(context.Background(), svc, name, body)
	if err != nil {
		return encodeResult(callResult{Error: err})
	}
	return encodeResult(callResult{Result: result})
}

//export a2p_free
func a2p_free(value *C.char) {
	C.free(unsafe.Pointer(value))
}

//export a2p_close
func a2p_close(handle C.uintptr_t) {
	value, ok := handles.LoadAndDelete(uintptr(handle))
	if !ok {
		return
	}
	value.(*service.Service).Close()
}

func dispatch(ctx context.Context, svc *service.Service, method, body string) (any, *callError) {
	switch method {
	case "version":
		return map[string]string{"version": version.Version}, nil
	case "capabilities":
		return svc.Capabilities(), nil
	case "parse":
		var params parseParams
		if err := json.Unmarshal([]byte(body), &params); err != nil {
			return nil, &callError{Code: "invalid_input", Message: err.Error()}
		}
		profile, err := svc.Parse(params.Body, params.Format)
		if err != nil {
			return nil, &callError{Code: "invalid_input", Message: err.Error()}
		}
		return profile, nil
	case "compile":
		var params compileParams
		if err := json.Unmarshal([]byte(body), &params); err != nil {
			return nil, &callError{Code: "invalid_input", Message: err.Error()}
		}
		result, err := svc.Compile(params.Body, params.Format, params.Target, params.Port, params.Strict)
		if err != nil {
			return nil, &callError{Code: "invalid_input", Message: err.Error()}
		}
		return result, nil
	case "run":
		var params runParams
		if err := json.Unmarshal([]byte(body), &params); err != nil {
			return nil, &callError{Code: "invalid_input", Message: err.Error()}
		}
		result, err := svc.Run(ctx, params.Body, params.Format, params.Target, params.Binary, params.Port, params.Strict)
		if err != nil {
			return nil, &callError{Code: "spawn_failed", Message: err.Error()}
		}
		return result, nil
	case "stop":
		var params stopParams
		if err := json.Unmarshal([]byte(body), &params); err != nil {
			return nil, &callError{Code: "invalid_input", Message: err.Error()}
		}
		if err := svc.Stop(params.SessionID); err != nil {
			return nil, &callError{Code: "invalid_input", Message: err.Error()}
		}
		return map[string]bool{"stopped": true}, nil
	default:
		return nil, &callError{Code: "unsupported", Message: "unknown method " + method}
	}
}

func encodeResult(result callResult) *C.char {
	data, err := json.Marshal(result)
	if err != nil {
		return C.CString(`{"error":{"code":"internal","message":"encode error"}}`)
	}
	return C.CString(string(data))
}

func main() {}
