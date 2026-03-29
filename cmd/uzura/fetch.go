package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/nyasuto/uzura/internal/dom"
	"github.com/nyasuto/uzura/internal/network"
	"github.com/nyasuto/uzura/internal/page"
)

func runFetch() error {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	format := fs.String("format", "text", "output format: text, json, html, markdown")
	verbose := fs.Bool("verbose", false, "show token estimate on stderr (markdown only)")
	timeout := fs.Duration("timeout", network.DefaultTimeout, "request timeout")
	userAgent := fs.String("user-agent", network.DefaultUserAgent, "User-Agent header")
	obeyRobots := fs.Bool("obey-robots", false, "obey robots.txt rules")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return fmt.Errorf("usage: uzura fetch [options] <url>")
	}
	url := fs.Arg(0)

	// Validate timeout
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}

	f := network.NewFetcher(&network.FetcherOptions{
		UserAgent:     *userAgent,
		Timeout:       *timeout,
		EnableCookies: true,
		ObeyRobots:    *obeyRobots,
	})

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	pg := page.New(&page.Options{Fetcher: f})
	if err := pg.Navigate(ctx, url); err != nil {
		return err
	}
	doc := pg.Document()

	switch *format {
	case "text":
		printTree(os.Stdout, doc, 0)
	case "json":
		obj := nodeToMap(doc)
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(obj)
	case "html":
		_, _ = fmt.Fprint(os.Stdout, dom.Serialize(doc))
	case "markdown":
		md := renderDocMarkdown(doc, url)
		_, _ = fmt.Fprint(os.Stdout, md)
		if *verbose {
			fmt.Fprintf(os.Stderr, "estimated tokens: ~%d\n", estimateTokens(md))
		}
	default:
		return fmt.Errorf("unknown format: %s", *format)
	}

	return nil
}
