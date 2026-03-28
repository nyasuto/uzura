package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nyasuto/uzura/internal/wpt"
)

func runWPT() error {
	fs := flag.NewFlagSet("wpt", flag.ExitOnError)
	wptDir := fs.String("wpt-dir", "testdata/wpt", "path to WPT checkout")
	skipFile := fs.String("skip", "", "path to skip list file")
	jsonOut := fs.Bool("json", false, "output results as JSON")

	if err := fs.Parse(os.Args[2:]); err != nil {
		return err
	}

	dirs := fs.Args()
	if len(dirs) == 0 {
		dirs = []string{"dom/nodes"}
	}

	runner := &wpt.Runner{
		WPTDir: *wptDir,
		Output: os.Stderr,
	}

	if *skipFile != "" {
		sl, err := wpt.LoadSkipFile(*skipFile)
		if err != nil {
			return fmt.Errorf("loading skip file: %w", err)
		}
		runner.SkipList = sl
	}

	summary := &wpt.Summary{}
	for _, dir := range dirs {
		s, err := runner.RunDir(dir)
		if err != nil {
			return fmt.Errorf("running %s: %w", dir, err)
		}
		for _, r := range s.Results {
			summary.Add(r)
		}
	}

	if *jsonOut {
		return summary.WriteJSON(os.Stdout)
	}

	// Print summary to stdout.
	fmt.Printf("\n=== WPT Results ===\n")
	fmt.Printf("Total: %d  Pass: %d  Fail: %d  Skip: %d  Timeout: %d\n",
		summary.Total, summary.Pass, summary.Fail, summary.Skip, summary.Timeout)
	fmt.Printf("Pass rate: %.1f%%\n", summary.PassRate())
	return nil
}
