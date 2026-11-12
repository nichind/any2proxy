package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/nichind/any2proxy/internal/cores"
	"github.com/nichind/any2proxy/internal/doctor"
	"github.com/nichind/any2proxy/internal/emit/links"
	"github.com/nichind/any2proxy/internal/emit/mihomo"
	"github.com/nichind/any2proxy/internal/emit/singbox"
	"github.com/nichind/any2proxy/internal/emit/xray"
	"github.com/nichind/any2proxy/internal/health"
	"github.com/nichind/any2proxy/internal/ingest/clash"
	"github.com/nichind/any2proxy/internal/ingest/fetch"
	"github.com/nichind/any2proxy/internal/ingest/plain"
	"github.com/nichind/any2proxy/internal/ir"
	"github.com/nichind/any2proxy/internal/subscription"
	"github.com/nichind/any2proxy/internal/supervisor"
	"github.com/nichind/any2proxy/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		printHelp()
		return 0
	}

	switch args[0] {
	case "version", "--version", "-v":
		fmt.Printf("a2p %s\n", version.Version)
		return 0
	case "help", "--help", "-h":
		printHelp()
		return 0
	case "parse":
		return parseCommand(args[1:])
	case "compile":
		return compileCommand(args[1:])
	case "links":
		return linksCommand(args[1:])
	case "test":
		return testCommand(args[1:])
	case "run":
		return runCommand(args[1:])
	case "sub":
		return subCommand(args[1:])
	case "cores":
		return coresCommand(args[1:])
	case "doctor":
		return doctorCommand(args[1:])
	case "check":
		return checkCommand(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q; hint: a2p help\n", args[0])
		return 2
	}
}

func parseCommand(args []string) int {
	fs := flag.NewFlagSet("parse", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	noCache := fs.Bool("no-cache", false, "skip conditional requests")
	offline := fs.Bool("offline", false, "use the cache only")
	allowPrivate := fs.Bool("allow-private", false, "allow private subscription hosts")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: parse needs one file, URL or -; hint: a2p parse ./links.txt")
		return 2
	}

	profile, err := readProfile(fs.Arg(0), fetch.Options{
		NoCache:      *noCache,
		Offline:      *offline,
		AllowPrivate: *allowPrivate,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot parse %s; hint: check the file and its format\n", displaySource(fs.Arg(0)))
		return 3
	}

	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(profile); err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot write output; hint: check stdout")
		return 9
	}
	return 0
}

func testCommand(args []string) int {
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	binary := fs.String("binary", os.Getenv("A2P_CORE"), "core binary")
	coreID := fs.String("core", "", "installed core id")
	to := fs.String("to", "sing-box", "target format")
	targetURL := fs.String("url", "https://www.gstatic.com/generate_204", "URL to fetch through each proxy")
	timeout := fs.Duration("timeout", 8*time.Second, "per-node timeout")
	limit := fs.Int("limit", 0, "test only the first N nodes")
	concurrency := fs.Int("concurrency", 1, "parallel node tests")
	strict := fs.Bool("strict", false, "fail on unsupported nodes")
	jsonOut := fs.Bool("json", false, "print JSON")
	noCache := fs.Bool("no-cache", false, "skip conditional requests")
	offline := fs.Bool("offline", false, "use the cache only")
	allowPrivate := fs.Bool("allow-private", false, "allow private subscription hosts")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: test needs one file, URL or -; hint: a2p test ./config.yaml")
		return 2
	}
	if *to != "sing-box" && *to != "mihomo" && *to != "xray" {
		fmt.Fprintf(os.Stderr, "error: unsupported target %q\n", *to)
		return 4
	}
	profile, err := readProfile(fs.Arg(0), fetch.Options{
		NoCache: *noCache, Offline: *offline, AllowPrivate: *allowPrivate,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot parse %s; hint: check the file and its format\n", displaySource(fs.Arg(0)))
		return 3
	}
	if len(profile.Nodes) == 0 {
		fmt.Fprintln(os.Stderr, "error: no nodes")
		return 3
	}

	if *binary == "" {
		lookup := *to
		if *coreID != "" {
			lookup = *coreID
		}
		if store, err := cores.Open(); err == nil {
			if entry, err := store.Get(lookup); err == nil {
				*binary = entry.Path
			}
		}
	}
	if *binary == "" {
		fmt.Fprintln(os.Stderr, "error: no core binary; hint: a2p cores install <id> or --binary")
		return 5
	}

	nodes := profile.Nodes
	if *limit > 0 && *limit < len(nodes) {
		nodes = nodes[:*limit]
	}

	type result struct {
		Tag       string `json:"tag"`
		OK        bool   `json:"ok"`
		Status    int    `json:"status,omitempty"`
		LatencyMS int64  `json:"latency_ms,omitempty"`
		Error     string `json:"error,omitempty"`
	}
	results := make([]result, len(nodes))
	ctx := context.Background()
	workers := *concurrency
	if workers < 1 {
		workers = 1
	}
	if workers > 16 {
		workers = 16
	}
	if *to == "mihomo" && workers > 1 {
		fmt.Fprintln(os.Stderr, "warning: mihomo is tested sequentially; --concurrency is ignored")
		workers = 1
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, node := range nodes {
		wg.Add(1)
		go func(index int, item ir.Node) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			started, err := startNode(ctx, item, *to, *binary, *strict)
			if err != nil {
				results[index] = result{Tag: item.Tag, Error: err.Error()}
				return
			}
			status, latency, err := health.ThroughProxy(ctx, started.Endpoint.URL, *targetURL, *timeout)
			_ = started.Session.Stop()
			_ = os.Remove(started.Temp)
			if err != nil {
				results[index] = result{Tag: item.Tag, Error: err.Error()}
				return
			}
			results[index] = result{Tag: item.Tag, OK: true, Status: status, LatencyMS: latency.Milliseconds()}
		}(i, node)
	}
	wg.Wait()

	if *jsonOut {
		_ = json.NewEncoder(os.Stdout).Encode(struct {
			Results []result `json:"results"`
		}{Results: results})
	} else {
		for _, item := range results {
			if item.OK {
				fmt.Printf("ok\t%s\t%d\t%dms\n", item.Tag, item.Status, item.LatencyMS)
			} else {
				fmt.Printf("fail\t%s\t%s\n", item.Tag, item.Error)
			}
		}
	}

	for _, item := range results {
		if !item.OK {
			return 1
		}
	}
	return 0
}

func linksCommand(args []string) int {
	fs := flag.NewFlagSet("links", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "print JSON")
	noCache := fs.Bool("no-cache", false, "skip conditional requests")
	offline := fs.Bool("offline", false, "use the cache only")
	allowPrivate := fs.Bool("allow-private", false, "allow private subscription hosts")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: links needs one file, URL or -; hint: a2p links ./config.yaml")
		return 2
	}
	profile, err := readProfile(fs.Arg(0), fetch.Options{
		NoCache: *noCache, Offline: *offline, AllowPrivate: *allowPrivate,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot parse %s; hint: check the file and its format\n", displaySource(fs.Arg(0)))
		return 3
	}

	lines, dropped := links.Emit(profile)
	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "error: no supported proxy links")
		return 3
	}
	if *jsonOut {
		if err := json.NewEncoder(os.Stdout).Encode(struct {
			Links   []string `json:"links"`
			Dropped []string `json:"dropped,omitempty"`
		}{Links: lines, Dropped: dropped}); err != nil {
			return 9
		}
		return 0
	}
	for _, line := range lines {
		fmt.Println(line)
	}
	if len(dropped) > 0 {
		fmt.Fprintf(os.Stderr, "dropped %d unsupported nodes\n", len(dropped))
	}
	return 0
}

func compileCommand(args []string) int {
	fs := flag.NewFlagSet("compile", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	to := fs.String("to", "sing-box", "target format")
	port := fs.Uint("port", 1080, "local socks port for mihomo")
	strict := fs.Bool("strict", false, "fail on unsupported nodes")
	report := fs.Bool("report", false, "print the fidelity report to stderr")
	noCache := fs.Bool("no-cache", false, "skip conditional requests")
	offline := fs.Bool("offline", false, "use the cache only")
	allowPrivate := fs.Bool("allow-private", false, "allow private subscription hosts")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: compile needs one file or -; hint: a2p compile ./links.txt")
		return 2
	}
	if *to != "sing-box" && *to != "mihomo" && *to != "xray" {
		fmt.Fprintf(os.Stderr, "error: unsupported target %q; hint: use --to sing-box, mihomo or xray\n", *to)
		return 4
	}

	profile, err := readProfile(fs.Arg(0), fetch.Options{
		NoCache:      *noCache,
		Offline:      *offline,
		AllowPrivate: *allowPrivate,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot parse %s; hint: check the file and its format\n", displaySource(fs.Arg(0)))
		return 3
	}

	var (
		data []byte
		rep  ir.CapabilityReport
	)
	if *to == "mihomo" {
		data, rep, err = mihomo.Emit(profile, mihomo.Options{Port: uint16(*port), Strict: *strict})
	} else if *to == "xray" {
		data, rep, err = xray.Emit(profile, xray.Options{Port: uint16(*port), Strict: *strict})
	} else {
		data, rep, err = singbox.Emit(profile, *strict)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v; hint: use --to sing-box, --to mihomo or remove unsupported nodes\n", err)
		return 4
	}
	if *report {
		if err := json.NewEncoder(os.Stderr).Encode(rep); err != nil {
			return 9
		}
	}
	if _, err := os.Stdout.Write(append(data, '\n')); err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot write output; hint: check stdout")
		return 9
	}
	return 0
}

func runCommand(args []string) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	binary := fs.String("binary", os.Getenv("A2P_CORE"), "core binary")
	coreID := fs.String("core", "", "installed core id")
	to := fs.String("to", "sing-box", "target format")
	mode := fs.String("mode", "single", "single, auto or pool")
	strict := fs.Bool("strict", false, "fail on unsupported nodes")
	dryRun := fs.Bool("dry-run", false, "print the core config and exit")
	jsonOut := fs.Bool("json", false, "print endpoints as JSON")
	report := fs.Bool("report", false, "print the fidelity report to stderr")
	noCache := fs.Bool("no-cache", false, "skip conditional requests")
	offline := fs.Bool("offline", false, "use the cache only")
	allowPrivate := fs.Bool("allow-private", false, "allow private subscription hosts")
	duration := fs.Duration("duration", 0, "stop after this duration")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: run needs one file or -; hint: a2p run ./links.txt")
		return 2
	}
	if *to != "sing-box" && *to != "mihomo" && *to != "xray" {
		fmt.Fprintf(os.Stderr, "error: unsupported target %q; hint: use --to sing-box, mihomo or xray\n", *to)
		return 4
	}
	if *mode != "single" && *mode != "auto" && *mode != "pool" && *mode != "balance" {
		fmt.Fprintf(os.Stderr, "error: unsupported mode %q; hint: use --mode single, auto, pool or balance\n", *mode)
		return 2
	}

	profile, err := readProfile(fs.Arg(0), fetch.Options{
		NoCache:      *noCache,
		Offline:      *offline,
		AllowPrivate: *allowPrivate,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot parse %s; hint: check the file and its format\n", displaySource(fs.Arg(0)))
		return 3
	}
	if len(profile.Nodes) == 0 {
		fmt.Fprintln(os.Stderr, "error: no nodes; hint: check the input file")
		return 3
	}

	if *dryRun {
		return dryRunOutput(profile, *to, *mode, *strict, *report)
	}
	if *binary == "" {
		lookup := *to
		if *coreID != "" {
			lookup = *coreID
		}
		if store, err := cores.Open(); err == nil {
			if entry, err := store.Get(lookup); err == nil {
				*binary = entry.Path
			}
		}
	}
	if *binary == "" {
		fmt.Fprintln(os.Stderr, "error: no core binary; hint: a2p cores install <id>, --binary or A2P_CORE")
		return 5
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if *mode == "balance" {
		return runBalance(ctx, profile, *to, *binary, *strict, *report, *jsonOut, *duration)
	}
	if *mode == "auto" {
		return runAuto(ctx, profile, *to, *binary, *strict, *report, *jsonOut, *duration)
	}

	sessions := make([]*supervisor.Session, 0, len(profile.Nodes))
	endpoints := make([]runEndpoint, 0, len(profile.Nodes))
	temps := make([]string, 0, len(profile.Nodes))
	defer func() {
		for _, session := range sessions {
			_ = session.Stop()
		}
		for _, path := range temps {
			_ = os.Remove(path)
		}
	}()

	for _, node := range profile.Nodes {
		if *mode != "pool" && len(sessions) > 0 {
			break
		}

		lease, err := supervisor.AcquireRetry(3)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: cannot lease a port; hint: retry or free a port")
			return 8
		}
		data, rep, err := emitOne(*to, node, lease.Port, *strict)
		if err != nil {
			if *mode == "pool" || *mode == "auto" {
				fmt.Fprintf(os.Stderr, "warning: %s: %v\n", node.Tag, err)
				continue
			}
			fmt.Fprintf(os.Stderr, "error: %v; hint: remove unsupported nodes\n", err)
			return 4
		}
		if *report {
			if err := json.NewEncoder(os.Stderr).Encode(rep); err != nil {
				return 9
			}
		}
		path, err := writeTemp(data, *to)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: cannot create a temp config; hint: check TMPDIR")
			return 9
		}
		temps = append(temps, path)

		session, err := supervisor.Start(ctx, supervisor.Config{
			Command:      coreCommand(*binary, *to, path),
			Lease:        &lease,
			ReadyTimeout: 10 * time.Second,
			StopTimeout:  5 * time.Second,
		})
		if err != nil {
			if *mode == "pool" || *mode == "auto" {
				fmt.Fprintf(os.Stderr, "warning: %s: core did not start\n", node.Tag)
				continue
			}
			fmt.Fprintln(os.Stderr, "error: core did not start; hint: check --binary and its config")
			return 5
		}

		sessions = append(sessions, session)
		endpoints = append(endpoints, runEndpoint{Tag: node.Tag, Core: *to, URL: session.Endpoint()})
	}
	if len(sessions) == 0 {
		fmt.Fprintln(os.Stderr, "error: no core started; hint: check the nodes and --binary")
		return 5
	}

	printEndpoints(endpoints, *mode, *jsonOut)

	done := make(chan struct{}, len(sessions))
	for _, session := range sessions {
		go func(session *supervisor.Session) {
			<-session.Done()
			done <- struct{}{}
		}(session)
	}
	var timer <-chan time.Time
	if *duration > 0 {
		timer = time.After(*duration)
	}
	select {
	case <-ctx.Done():
		return 130
	case <-done:
		fmt.Fprintln(os.Stderr, "error: core exited; hint: check the binary and its config")
		return 5
	case <-timer:
		return 0
	}
}

type startedNode struct {
	Session  *supervisor.Session
	Endpoint runEndpoint
	Temp     string
	Report   ir.CapabilityReport
}

func runBalance(ctx context.Context, profile ir.Profile, to, binary string, strict, report, jsonOut bool, duration time.Duration) int {
	if to == "xray" {
		fmt.Fprintln(os.Stderr, "error: xray has no balancer support; hint: use --to mihomo or sing-box")
		return 4
	}
	lease, err := supervisor.AcquireRetry(3)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot lease a port")
		return 8
	}
	data, rep, err := emitProfile(to, profile, lease.Port, strict)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 4
	}
	if report {
		_ = json.NewEncoder(os.Stderr).Encode(rep)
	}
	path, err := writeTemp(data, to)
	if err != nil {
		return 9
	}
	defer os.Remove(path)

	session, err := supervisor.Start(ctx, supervisor.Config{
		Command:      coreCommand(binary, to, path),
		Dir:          filepath.Dir(path),
		Lease:        &lease,
		ReadyTimeout: 10 * time.Second,
		StopTimeout:  5 * time.Second,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: core did not start; hint: check --binary and the groups")
		return 5
	}
	defer session.Stop()
	printEndpoints([]runEndpoint{{Tag: "balance", Core: to, URL: session.Endpoint()}}, "single", jsonOut)

	var timer <-chan time.Time
	if duration > 0 {
		timer = time.After(duration)
	}
	select {
	case <-ctx.Done():
		return 130
	case <-session.Done():
		fmt.Fprintln(os.Stderr, "error: core exited")
		return 5
	case <-timer:
		return 0
	}
}

func emitProfile(to string, profile ir.Profile, port uint16, strict bool) ([]byte, ir.CapabilityReport, error) {
	switch to {
	case "mihomo":
		return mihomo.Emit(profile, mihomo.Options{Port: port, Strict: strict})
	case "sing-box":
		return singbox.EmitWithOptions(profile, singbox.Options{Strict: strict, SocksPort: port})
	default:
		return nil, ir.CapabilityReport{}, fmt.Errorf("unsupported target %q", to)
	}
}

func runAuto(ctx context.Context, profile ir.Profile, to, binary string, strict, report, jsonOut bool, duration time.Duration) int {
	sessions := make([]*supervisor.Session, 0, len(profile.Nodes))
	temps := make([]string, 0, len(profile.Nodes))
	defer func() {
		for _, session := range sessions {
			_ = session.Stop()
		}
		for _, path := range temps {
			_ = os.Remove(path)
		}
	}()

	var timer <-chan time.Time
	if duration > 0 {
		timer = time.After(duration)
	}

	index := 0
	for index < len(profile.Nodes) {
		select {
		case <-ctx.Done():
			return 130
		default:
		}

		var (
			started startedNode
			lastErr error
		)
		for index < len(profile.Nodes) {
			node := profile.Nodes[index]
			index++
			started, lastErr = startNode(ctx, node, to, binary, strict)
			if lastErr == nil {
				break
			}
			fmt.Fprintf(os.Stderr, "warning: %s: %v\n", node.Tag, lastErr)
		}
		if lastErr != nil {
			fmt.Fprintln(os.Stderr, "error: no core started; hint: check the nodes and --binary")
			return 5
		}

		sessions = append(sessions, started.Session)
		temps = append(temps, started.Temp)
		if report {
			if err := json.NewEncoder(os.Stderr).Encode(started.Report); err != nil {
				return 9
			}
		}
		printEndpoints([]runEndpoint{started.Endpoint}, "single", jsonOut)

		select {
		case <-ctx.Done():
			return 130
		case <-started.Session.Done():
			fmt.Fprintf(os.Stderr, "warning: %s exited; trying the next node\n", started.Endpoint.Tag)
		case <-timer:
			return 0
		}
	}
	fmt.Fprintln(os.Stderr, "error: all nodes exited; hint: check the cores and the nodes")
	return 5
}

func startNode(ctx context.Context, node ir.Node, to, binary string, strict bool) (startedNode, error) {
	lease, err := supervisor.AcquireRetry(3)
	if err != nil {
		return startedNode{}, fmt.Errorf("lease: %w", err)
	}
	data, rep, err := emitOne(to, node, lease.Port, strict)
	if err != nil {
		return startedNode{}, err
	}
	path, err := writeTemp(data, to)
	if err != nil {
		return startedNode{}, err
	}
	session, err := supervisor.Start(ctx, supervisor.Config{
		Command:      coreCommand(binary, to, path),
		Dir:          filepath.Dir(path),
		Lease:        &lease,
		ReadyTimeout: 10 * time.Second,
		StopTimeout:  5 * time.Second,
	})
	if err != nil {
		_ = os.Remove(path)
		return startedNode{}, fmt.Errorf("core did not start")
	}
	return startedNode{
		Session:  session,
		Endpoint: runEndpoint{Tag: node.Tag, Core: to, URL: session.Endpoint()},
		Temp:     path,
		Report:   rep,
	}, nil
}

func dryRunOutput(profile ir.Profile, to, mode string, strict, report bool) int {
	if mode == "balance" {
		data, rep, err := emitProfile(to, profile, 1080, strict)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 4
		}
		if report {
			if err := json.NewEncoder(os.Stderr).Encode(rep); err != nil {
				return 9
			}
		}
		if _, err := os.Stdout.Write(append(data, '\n')); err != nil {
			return 9
		}
		return 0
	}
	nodes := profile.Nodes
	if mode != "pool" && len(nodes) > 1 {
		nodes = nodes[:1]
	}
	for i, node := range nodes {
		data, rep, err := emitOne(to, node, uint16(1080+i), strict)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v; hint: remove unsupported nodes\n", err)
			return 4
		}
		if report {
			if err := json.NewEncoder(os.Stderr).Encode(rep); err != nil {
				return 9
			}
		}
		if mode == "pool" {
			fmt.Printf("# %s\n", node.Tag)
		}
		if _, err := os.Stdout.Write(append(data, '\n')); err != nil {
			return 9
		}
	}
	return 0
}

type runEndpoint struct {
	Tag  string `json:"tag"`
	Core string `json:"core"`
	URL  string `json:"url"`
}

type runResult struct {
	SchemaVersion int           `json:"schema_version"`
	Endpoints     []runEndpoint `json:"endpoints"`
}

func printEndpoints(endpoints []runEndpoint, mode string, asJSON bool) {
	if asJSON {
		_ = json.NewEncoder(os.Stdout).Encode(runResult{SchemaVersion: 1, Endpoints: endpoints})
		return
	}
	for _, endpoint := range endpoints {
		if mode == "pool" {
			fmt.Printf("%s\t%s\n", endpoint.Tag, endpoint.URL)
			continue
		}
		fmt.Println(endpoint.URL)
	}
}

func emitOne(to string, node ir.Node, port uint16, strict bool) ([]byte, ir.CapabilityReport, error) {
	profile := ir.Profile{Nodes: []ir.Node{node}}
	switch to {
	case "mihomo":
		return mihomo.Emit(profile, mihomo.Options{Port: port, Strict: strict})
	case "xray":
		return xray.Emit(profile, xray.Options{Port: port, Strict: strict})
	default:
		return singbox.EmitWithOptions(profile, singbox.Options{Strict: strict, SocksPort: port})
	}
}

func writeTemp(data []byte, to string) (string, error) {
	pattern := "any2proxy-*.json"
	if to == "mihomo" {
		pattern = "any2proxy-*.yaml"
	}
	config, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	if _, err := config.Write(append(data, '\n')); err != nil {
		_ = config.Close()
		return "", err
	}
	if err := config.Close(); err != nil {
		return "", err
	}
	return config.Name(), nil
}

func coreCommand(binary, to, path string) []string {
	if to == "mihomo" {
		return []string{binary, "-d", filepath.Dir(path), "-f", path}
	}
	return []string{binary, "run", "-c", path}
}

func coresCommand(args []string) int {
	if len(args) == 0 {
		printCoresHelp()
		return 0
	}

	store, err := cores.Open()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error: cannot read the core registry; hint: check A2P_HOME")
		return 3
	}

	switch args[0] {
	case "install", "update":
		fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		version := fs.String("version", "latest", "release tag")
		goos := fs.String("os", "", "target OS")
		arch := fs.String("arch", "", "target architecture")
		repo := fs.String("repo", "", "custom GitHub repository")
		link := fs.String("url", "", "direct download URL")
		assetFilter := fs.String("asset", "", "asset name filter")
		binName := fs.String("bin-name", "", "binary name inside the archive")
		sha := fs.String("sha256", "", "expected sha256 for --url")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if fs.NArg() != 1 {
			fmt.Fprintln(os.Stderr, "error: cores install needs an id; hint: a2p cores install sing-box")
			return 2
		}
		if *repo != "" && *link != "" {
			fmt.Fprintln(os.Stderr, "error: use either --repo or --url")
			return 2
		}
		entry, err := store.Install(context.Background(), fs.Arg(0), cores.InstallOptions{
			Version: *version,
			OS:      *goos,
			Arch:    *arch,
			Repo:    *repo,
			URL:     *link,
			Asset:   *assetFilter,
			Binary:  *binName,
			SHA256:  *sha,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 3
		}
		fmt.Printf("installed %s %s %s\n", entry.ID, entry.Version, entry.Path)
		return 0
	case "list":
		entries := store.List()
		if len(entries) == 0 {
			fmt.Println("no cores")
			return 0
		}
		fmt.Printf("%-12s %-12s %-16s %s\n", "ID", "VERSION", "SHA256", "PATH")
		for _, entry := range entries {
			short := entry.SHA256
			if len(short) > 16 {
				short = short[:16]
			}
			fmt.Printf("%-12s %-12s %-16s %s\n", entry.ID, entry.Version, short, entry.Path)
		}
		return 0
	case "import":
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "error: cores import needs an id and a path; hint: a2p cores import sing-box ./sing-box")
			return 2
		}
		entry, err := store.Import(args[1], args[2], "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 3
		}
		fmt.Printf("imported %s %s\n", entry.ID, entry.SHA256)
		return 0
	case "verify":
		if len(args) > 2 {
			fmt.Fprintln(os.Stderr, "error: cores verify takes one id or nothing")
			return 2
		}
		if len(args) == 2 {
			if err := store.Verify(args[1]); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				return 1
			}
			fmt.Printf("ok %s\n", args[1])
			return 0
		}
		entries := store.List()
		for _, entry := range entries {
			if err := store.Verify(entry.ID); err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				return 1
			}
			fmt.Printf("ok %s\n", entry.ID)
		}
		return 0
	case "remove":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "error: cores remove needs an id; hint: a2p cores remove sing-box")
			return 2
		}
		if err := store.Remove(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 3
		}
		fmt.Printf("removed %s\n", args[1])
		return 0
	case "help", "--help", "-h":
		printCoresHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "error: unknown cores command %q; hint: a2p cores help\n", args[0])
		return 2
	}
}

func printCoresHelp() {
	fmt.Print(`a2p cores - manage core binaries

Usage:
  a2p cores install <id> [flags]
  a2p cores update <id>
  a2p cores list
  a2p cores import <id> <path>
  a2p cores verify [id]
  a2p cores remove <id>

Install flags:
  --version <tag>
  --repo <owner/repo>
  --url <download-url>
  --asset <substring>
  --bin-name <name>
  --sha256 <hex>
  --os <goos> --arch <goarch>

Examples:
  a2p cores install sing-box
  a2p cores install sing-box --version v1.14.4
  a2p cores install fork --repo owner/sing-box --bin-name sing-box
  a2p cores install custom --url https://example.com/core.zip --sha256 <hex> --bin-name core
  a2p cores import sing-box ./sing-box
  a2p run --to sing-box --core fork ./links.txt
`)
}

func doctorCommand(args []string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	core := fs.String("core", os.Getenv("A2P_CORE"), "core binary")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	results := doctor.Run(doctor.Options{Core: *core})
	if *jsonOut {
		if err := json.NewEncoder(os.Stdout).Encode(struct {
			Checks []doctor.Result `json:"checks"`
		}{Checks: results}); err != nil {
			return 9
		}
	} else {
		for _, result := range results {
			fmt.Printf("%s\t%s\t%s\n", result.Status, result.Name, result.Detail)
		}
	}
	for _, result := range results {
		if result.Status == doctor.Fail {
			return 1
		}
	}
	return 0
}

func checkCommand(args []string) int {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	timeout := fs.Duration("timeout", 5*time.Second, "dial timeout")
	targetURL := fs.String("url", "", "fetch this URL through the proxy")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "error: check needs one endpoint; hint: a2p check socks5://127.0.0.1:1080")
		return 2
	}

	endpoint := fs.Arg(0)
	if *targetURL != "" {
		status, latency, err := health.ThroughProxy(context.Background(), endpoint, *targetURL, *timeout)
		if err != nil {
			if *jsonOut {
				_ = json.NewEncoder(os.Stdout).Encode(struct {
					Endpoint string `json:"endpoint"`
					URL      string `json:"url"`
					OK       bool   `json:"ok"`
					Error    string `json:"error"`
				}{Endpoint: endpoint, URL: *targetURL, OK: false, Error: err.Error()})
			} else {
				fmt.Fprintf(os.Stderr, "error: %s through %s failed; hint: check the node and the URL\n", *targetURL, endpoint)
			}
			return 1
		}
		if *jsonOut {
			_ = json.NewEncoder(os.Stdout).Encode(struct {
				Endpoint  string `json:"endpoint"`
				URL       string `json:"url"`
				OK        bool   `json:"ok"`
				Status    int    `json:"status"`
				LatencyMS int64  `json:"latency_ms"`
			}{Endpoint: endpoint, URL: *targetURL, OK: true, Status: status, LatencyMS: latency.Milliseconds()})
			return 0
		}
		fmt.Printf("ok\t%s\t%s\t%d\t%s\n", endpoint, *targetURL, status, latency.Round(time.Millisecond))
		return 0
	}

	addr := endpoint
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Host != "" {
		addr = parsed.Host
	}
	latency, err := health.TCP(context.Background(), addr, *timeout)
	if err != nil {
		if *jsonOut {
			_ = json.NewEncoder(os.Stdout).Encode(struct {
				Endpoint string `json:"endpoint"`
				OK       bool   `json:"ok"`
				Error    string `json:"error"`
			}{Endpoint: endpoint, OK: false, Error: err.Error()})
		} else {
			fmt.Fprintf(os.Stderr, "error: %s unreachable; hint: check the endpoint\n", endpoint)
		}
		return 1
	}
	if *jsonOut {
		_ = json.NewEncoder(os.Stdout).Encode(struct {
			Endpoint  string `json:"endpoint"`
			OK        bool   `json:"ok"`
			LatencyMS int64  `json:"latency_ms"`
		}{Endpoint: endpoint, OK: true, LatencyMS: latency.Milliseconds()})
		return 0
	}
	fmt.Printf("ok\t%s\t%s\n", endpoint, latency.Round(time.Millisecond))
	return 0
}

func subCommand(args []string) int {
	if len(args) == 0 {
		printSubHelp()
		return 0
	}

	switch args[0] {
	case "add":
		if len(args) != 3 {
			fmt.Fprintln(os.Stderr, "error: sub add needs a name and a URL; hint: a2p sub add main https://example.com/sub")
			return 2
		}
		store, err := subscription.Load("")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: cannot read subscriptions; hint: check A2P_HOME")
			return 3
		}
		if err := store.Add(args[1], args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 3
		}
		fmt.Printf("added %s\n", args[1])
		return 0
	case "list":
		store, err := subscription.Load("")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: cannot read subscriptions; hint: check A2P_HOME")
			return 3
		}
		entries := store.List()
		if len(entries) == 0 {
			fmt.Println("no subscriptions")
			return 0
		}
		fmt.Printf("%-16s %-24s %6s %s\n", "NAME", "HOST", "NODES", "LAST")
		for _, entry := range entries {
			last := "-"
			if !entry.LastFetched.IsZero() {
				last = entry.LastFetched.Format(time.RFC3339)
			}
			fmt.Printf("%-16s %-24s %6d %s\n", entry.Name, subscription.Host(entry.URL), entry.Nodes, last)
		}
		return 0
	case "remove":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "error: sub remove needs a name; hint: a2p sub remove main")
			return 2
		}
		store, err := subscription.Load("")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: cannot read subscriptions; hint: check A2P_HOME")
			return 3
		}
		if err := store.Remove(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 3
		}
		fmt.Printf("removed %s\n", args[1])
		return 0
	case "refresh":
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "error: sub refresh needs a name; hint: a2p sub refresh main")
			return 2
		}
		store, err := subscription.Load("")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error: cannot read subscriptions; hint: check A2P_HOME")
			return 3
		}
		if _, err := store.Get(args[1]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 3
		}
		profile, err := readProfile("sub:"+args[1], fetch.Options{})
		if err != nil {
			_ = store.Update(args[1], 0, "error")
			fmt.Fprintf(os.Stderr, "error: refresh failed for %s; hint: check the URL and network\n", args[1])
			return 3
		}
		if err := store.Update(args[1], len(profile.Nodes), "ok"); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 3
		}
		fmt.Printf("%s: %d nodes\n", args[1], len(profile.Nodes))
		return 0
	case "help", "--help", "-h":
		printSubHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "error: unknown sub command %q; hint: a2p sub help\n", args[0])
		return 2
	}
}

func printSubHelp() {
	fmt.Print(`a2p sub - manage subscription URLs

Usage:
  a2p sub add <name> <url>
  a2p sub list
  a2p sub refresh <name>
  a2p sub remove <name>

Examples:
  a2p sub add main https://example.com/sub
  a2p sub refresh main
  a2p run --binary ./sing-box sub:main
`)
}

func readProfile(path string, opts fetch.Options) (ir.Profile, error) {
	target := path
	if strings.HasPrefix(path, "sub:") {
		name := strings.TrimPrefix(path, "sub:")
		store, err := subscription.Load("")
		if err != nil {
			return ir.Profile{}, err
		}
		entry, err := store.Get(name)
		if err != nil {
			return ir.Profile{}, err
		}
		target = entry.URL
	}

	if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
		body, _, err := fetch.New(opts).Get(context.Background(), target)
		if err != nil {
			return ir.Profile{}, err
		}
		parsed, _ := url.Parse(target)
		return parseReader(bytes.NewReader(body), parsed.Path)
	}

	var (
		reader io.Reader
		file   *os.File
		err    error
	)
	if target == "-" {
		reader = os.Stdin
	} else {
		file, err = os.Open(target)
		if err != nil {
			return ir.Profile{}, err
		}
		defer file.Close()
		reader = file
	}
	return parseReader(reader, target)
}

func parseReader(reader io.Reader, path string) (ir.Profile, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return clash.Parse(reader)
	default:
		return plain.Parse(reader)
	}
}

func displaySource(source string) string {
	parsed, err := url.Parse(source)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return source
	}
	return parsed.Scheme + "://" + parsed.Host
}

func printHelp() {
	fmt.Print(`a2p - any link or config to a working local proxy

Usage:
  a2p <command> [arguments]

Commands:
  parse <file|url|->          parse a link list, Clash YAML or subscription URL
  compile [flags] <file|url|-> compile to a core config
  links <file|url|->          export proxy links
  test [flags] <file|url|->   test every proxy through a core
  run [flags] <file|url|sub:name|-> compile, start the core and print the endpoint
  sub <command>               manage subscription URLs
  cores <command>             manage core binaries
  doctor                      check the environment
  check [--url URL] <endpoint> dial, or fetch a URL through the proxy
  version                     print the version
  help                        print this help

Compile flags:
  --to sing-box|mihomo|xray   target format (default sing-box)
  --port <n>                  local socks port for mihomo and xray (default 1080)
  --strict                    fail on unsupported nodes
  --report                    print the fidelity report to stderr

Run flags:
  --binary <path>             core binary (or A2P_CORE)
  --core <id>                 installed core id (fork)
  --to sing-box|mihomo|xray   target format (default sing-box)
  --mode single|auto|pool|balance one node, failover, port per node, or one balancer
  --dry-run                   print the core config and exit
  --json                      print endpoints as JSON
  --strict                    fail on unsupported nodes
  --duration <duration>       stop after this duration

Fetch flags:
  --no-cache                  skip conditional requests
  --offline                   use the cache only
  --allow-private             allow private subscription hosts

Examples:
  a2p parse ./links.txt
  a2p parse https://example.com/sub
  a2p links ./config.yaml
  a2p test --to mihomo ./config.yaml
  a2p compile ./links.txt
  a2p compile --to mihomo ./links.txt
  a2p compile --to xray ./links.txt
  a2p run --binary ./sing-box ./links.txt
  a2p run --binary ./mihomo --to mihomo ./links.txt
  a2p run --binary ./xray --to xray ./links.txt
  a2p sub add main https://example.com/sub
  a2p run --binary ./sing-box sub:main
  a2p cores import sing-box ./sing-box
  a2p cores install sing-box
  a2p cores list
  a2p doctor
  a2p check socks5://127.0.0.1:1080
  a2p check --url https://example.com socks5://127.0.0.1:1080
`)
}
