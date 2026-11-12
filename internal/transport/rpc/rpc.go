package rpc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/nichind/any2proxy/internal/service"
)

type request struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type response struct {
	ID     json.RawMessage `json:"id"`
	Result any             `json:"result,omitempty"`
	Error  *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
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

func Serve(ctx context.Context, in io.Reader, out io.Writer, svc *service.Service) error {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	encoder := json.NewEncoder(out)

	for scanner.Scan() {
		var req request
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			if err := encoder.Encode(response{Error: &rpcError{Code: "invalid_input", Message: "invalid JSON"}}); err != nil {
				return err
			}
			continue
		}
		result, rpcErr := dispatch(ctx, svc, req)
		reply := response{ID: req.ID, Result: result, Error: rpcErr}
		if err := encoder.Encode(reply); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func dispatch(ctx context.Context, svc *service.Service, req request) (any, *rpcError) {
	switch req.Method {
	case "version":
		return map[string]string{"version": svc.Capabilities()["version"].(string)}, nil
	case "capabilities":
		return svc.Capabilities(), nil
	case "parse":
		var params parseParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, &rpcError{Code: "invalid_input", Message: err.Error()}
		}
		profile, err := svc.Parse(params.Body, params.Format)
		if err != nil {
			return nil, &rpcError{Code: "invalid_input", Message: err.Error()}
		}
		return profile, nil
	case "compile":
		var params compileParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, &rpcError{Code: "invalid_input", Message: err.Error()}
		}
		result, err := svc.Compile(params.Body, params.Format, params.Target, params.Port, params.Strict)
		if err != nil {
			return nil, &rpcError{Code: "invalid_input", Message: err.Error()}
		}
		return result, nil
	case "run":
		var params runParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, &rpcError{Code: "invalid_input", Message: err.Error()}
		}
		result, err := svc.Run(ctx, params.Body, params.Format, params.Target, params.Binary, params.Port, params.Strict)
		if err != nil {
			return nil, &rpcError{Code: "spawn_failed", Message: err.Error()}
		}
		return result, nil
	case "stop":
		var params stopParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, &rpcError{Code: "invalid_input", Message: err.Error()}
		}
		if err := svc.Stop(params.SessionID); err != nil {
			return nil, &rpcError{Code: "invalid_input", Message: err.Error()}
		}
		return map[string]bool{"stopped": true}, nil
	default:
		return nil, &rpcError{Code: "unsupported", Message: fmt.Sprintf("unknown method %q", req.Method)}
	}
}
