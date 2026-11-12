package any2proxy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
)

type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader

	mu   sync.Mutex
	next int
}

type Node struct {
	Tag      string            `json:"tag"`
	Protocol string            `json:"protocol"`
	Server   string            `json:"server"`
	Port     uint16            `json:"port"`
	Account  string            `json:"account,omitempty"`
	Extra    map[string]string `json:"extra,omitempty"`
}

type Warning struct {
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

type Profile struct {
	Nodes    []Node    `json:"nodes"`
	Warnings []Warning `json:"warnings,omitempty"`
}

type CompileResult struct {
	Config string          `json:"config"`
	Report json.RawMessage `json:"report"`
}

type RunResult struct {
	SessionID string `json:"session_id"`
	Endpoint  string `json:"endpoint"`
}

func Start(binary string) (*Client, error) {
	if binary == "" {
		binary = "a2pd"
	}
	return StartCommand(binary)
}

func StartCommand(name string, args ...string) (*Client, error) {
	cmd := exec.Command(name, args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}
	return &Client{cmd: cmd, stdin: stdin, stdout: bufio.NewReader(stdout)}, nil
}

func (c *Client) Call(ctx context.Context, method string, params, result any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.next++
	request := map[string]any{"id": c.next, "method": method, "params": params}
	data, err := json.Marshal(request)
	if err != nil {
		return err
	}
	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return err
	}
	line, err := c.stdout.ReadBytes('\n')
	if err != nil {
		return err
	}
	var reply struct {
		ID     int             `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(line, &reply); err != nil {
		return err
	}
	if reply.Error != nil {
		return fmt.Errorf("%s: %s", reply.Error.Code, reply.Error.Message)
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(reply.Result, result)
}

func (c *Client) Version(ctx context.Context) (string, error) {
	var result struct {
		Version string `json:"version"`
	}
	if err := c.Call(ctx, "version", nil, &result); err != nil {
		return "", err
	}
	return result.Version, nil
}

func (c *Client) Parse(ctx context.Context, body, format string) (Profile, error) {
	var profile Profile
	err := c.Call(ctx, "parse", map[string]string{"body": body, "format": format}, &profile)
	return profile, err
}

func (c *Client) Compile(ctx context.Context, body, format, target string, port uint16, strict bool) (CompileResult, error) {
	var result CompileResult
	err := c.Call(ctx, "compile", map[string]any{
		"body": body, "format": format, "target": target, "port": port, "strict": strict,
	}, &result)
	return result, err
}

func (c *Client) Run(ctx context.Context, body, format, target, binary string, port uint16, strict bool) (RunResult, error) {
	var result RunResult
	err := c.Call(ctx, "run", map[string]any{
		"body": body, "format": format, "target": target, "binary": binary, "port": port, "strict": strict,
	}, &result)
	return result, err
}

func (c *Client) Stop(ctx context.Context, sessionID string) error {
	return c.Call(ctx, "stop", map[string]string{"session_id": sessionID}, nil)
}

func (c *Client) Close() error {
	_ = c.stdin.Close()
	return c.cmd.Wait()
}
