package doctor

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"

	"github.com/nichind/any2proxy/internal/subscription"
)

type Status string

const (
	OK   Status = "ok"
	Warn Status = "warn"
	Fail Status = "fail"
)

type Result struct {
	Name   string `json:"name"`
	Status Status `json:"status"`
	Detail string `json:"detail"`
}

type Options struct {
	Core string
}

func Run(opts Options) []Result {
	results := []Result{{
		Name:   "platform",
		Status: OK,
		Detail: runtime.GOOS + "/" + runtime.GOARCH,
	}}

	results = append(results, coreResult(opts.Core))
	results = append(results, portResult())
	results = append(results, subscriptionResult())
	results = append(results, homeResult())
	return results
}

func coreResult(core string) Result {
	if core != "" {
		if _, err := exec.LookPath(core); err != nil {
			if _, statErr := os.Stat(core); statErr != nil {
				return Result{Name: "core", Status: Fail, Detail: core + " not found"}
			}
		}
		return Result{Name: "core", Status: OK, Detail: core}
	}
	var found []string
	for _, name := range []string{"sing-box", "mihomo", "xray"} {
		if _, err := exec.LookPath(name); err == nil {
			found = append(found, name)
		}
	}
	if len(found) == 0 {
		return Result{Name: "core", Status: Warn, Detail: "none in PATH; use --binary or A2P_CORE"}
	}
	return Result{Name: "core", Status: OK, Detail: fmt.Sprint(found)}
}

func portResult() Result {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return Result{Name: "port", Status: Fail, Detail: err.Error()}
	}
	addr := listener.Addr().String()
	_ = listener.Close()
	return Result{Name: "port", Status: OK, Detail: addr}
}

func subscriptionResult() Result {
	store, err := subscription.Load("")
	if err != nil {
		return Result{Name: "subscriptions", Status: Fail, Detail: err.Error()}
	}
	return Result{Name: "subscriptions", Status: OK, Detail: fmt.Sprintf("%d", len(store.List()))}
}

func homeResult() Result {
	dir, err := os.UserCacheDir()
	if err != nil {
		return Result{Name: "cache", Status: Warn, Detail: "no user cache dir"}
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return Result{Name: "cache", Status: Warn, Detail: dir + " missing; will be created"}
	}
	probe, err := os.CreateTemp(dir, ".a2p-doctor-*")
	if err != nil {
		return Result{Name: "cache", Status: Fail, Detail: dir + " not writable"}
	}
	name := probe.Name()
	_ = probe.Close()
	_ = os.Remove(name)
	return Result{Name: "cache", Status: OK, Detail: dir}
}
