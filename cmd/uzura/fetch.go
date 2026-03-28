package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/nyasuto/uzura/internal/dom"
	"github.com/nyasuto/uzura/internal/network"
)

func runFetch() error {
	fs := flag.NewFlagSet("fetch", flag.ExitOnError)
	format := fs.String("format", "text", "output format: text, json, html")
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

	opts := &network.FetcherOptions{
		UserAgent:     *userAgent,
		Timeout:       *timeout,
		EnableCookies: true,
		ObeyRobots:    *obeyRobots,
	}

	// Validate timeout
	if *timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	_ = time.Second // ensure time is used

	f := network.NewFetcher(opts)
	doc, err := f.LoadDocument(url)
	if err != nil {
		return err
	}

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
	default:
		return fmt.Errorf("unknown format: %s", *format)
	}

	return nil
}
