package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/toprakgureli/sizinti/internal/cli"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	return cli.Run(ctx, os.Args[1:], os.Stdout, os.Stderr)
}
