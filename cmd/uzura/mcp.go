package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/nyasuto/uzura/internal/mcp"
)

func runMCP() error {
	fs := flag.NewFlagSet("mcp", flag.ExitOnError)
	logLevel := fs.String("log-level", "info", "log level (debug, info, warn, error)")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}

	srv := mcp.NewServer()
	mcp.RegisterBrowseTool(srv)
	mcp.RegisterEvaluateTool(srv)
	mcp.RegisterQueryTool(srv)
	mcp.RegisterInteractTool(srv)

	tr := mcp.NewTransport(os.Stdin, os.Stdout, os.Stderr)
	tr.Log("uzura mcp server starting (log-level=%s)", *logLevel)

	// Graceful shutdown on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(tr)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("serve: %w", err)
		}
		tr.Log("uzura mcp server stopped (EOF)")
		return nil
	case sig := <-sigCh:
		_ = sig
		log.SetOutput(os.Stderr)
		tr.Log("uzura mcp server shutting down")
		return nil
	}
}
