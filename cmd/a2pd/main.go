package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nichind/any2proxy/internal/service"
	"github.com/nichind/any2proxy/internal/transport/rpc"
)

func main() {
	svc := service.New()
	defer svc.Close()

	if err := rpc.Serve(context.Background(), os.Stdin, os.Stdout, svc); err != nil {
		fmt.Fprintln(os.Stderr, "a2pd:", err)
		os.Exit(1)
	}
}
