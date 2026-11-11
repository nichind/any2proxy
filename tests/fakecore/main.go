package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:0", "listen address")
	mode := flag.String("mode", "ok", "ok, crash or hang")
	flag.Parse()

	switch *mode {
	case "crash":
		fmt.Fprintln(os.Stderr, "fakecore: crash requested")
		os.Exit(1)
	case "hang":
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
		<-signals
		return
	}

	listener, err := net.Listen("tcp", *addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "fakecore:", err)
		os.Exit(1)
	}
	fmt.Println("ready", listener.Addr().String())

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-signals
		_ = listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		_ = conn.Close()
	}
}
