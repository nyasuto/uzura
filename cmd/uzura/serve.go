package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/nyasuto/uzura/internal/cdp"
)

func runServe() error {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 9222, "CDP server port")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}

	addr := fmt.Sprintf(":%d", *port)
	s := cdp.NewServer(
		cdp.WithAddr(addr),
		cdp.WithBrowserVersion(fmt.Sprintf("Uzura/%s", Version)),
	)
	cdp.Setup(s)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := s.Start(ctx); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "CDP server listening on %s\n", s.Addr())
	fmt.Fprintf(os.Stderr, "DevTools WebSocket: ws://%s/devtools/page/default\n", s.Addr())

	<-ctx.Done()
	return s.Shutdown(context.Background())
}
