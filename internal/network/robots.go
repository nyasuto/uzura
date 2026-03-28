package network

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	uzerr "github.com/nyasuto/uzura/internal/errors"
)

// robotsCache caches parsed robots.txt rules per host.
type robotsCache struct {
	mu    sync.Mutex
	rules map[string]*robotsRules
}

// robotsRules holds parsed allow/disallow rules for a host.
type robotsRules struct {
	groups []robotsGroup
}

type robotsGroup struct {
	agents    []string
	disallows []string
	allows    []string
}

// checkRobots fetches and checks robots.txt for the given URL.
// Returns an error if the path is disallowed for the fetcher's user agent.
func (f *Fetcher) checkRobots(targetURL string) error {
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	origin := parsed.Scheme + "://" + parsed.Host
	rules, err := f.getRobotRules(origin)
	if err != nil || rules == nil {
		return nil // allow on error or missing robots.txt
	}

	if rules.isDisallowed(f.userAgent, parsed.Path) {
		return fmt.Errorf("%w: %s", uzerr.ErrRobotsDisallowed, targetURL)
	}
	return nil
}

func (f *Fetcher) getRobotRules(origin string) (*robotsRules, error) {
	f.robots.mu.Lock()
	defer f.robots.mu.Unlock()

	if cached, ok := f.robots.rules[origin]; ok {
		return cached, nil
	}

	robotsURL := origin + "/robots.txt"
	req, err := http.NewRequest(http.MethodGet, robotsURL, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", f.userAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		f.robots.rules[origin] = nil
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		f.robots.rules[origin] = nil
		return nil, nil
	}

	rules := parseRobotsTxt(resp)
	f.robots.rules[origin] = rules
	return rules, nil
}

func parseRobotsTxt(resp *http.Response) *robotsRules {
	rules := &robotsRules{}
	scanner := bufio.NewScanner(resp.Body)

	var current *robotsGroup
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			if current != nil {
				rules.groups = append(rules.groups, *current)
				current = nil
			}
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(strings.ToLower(parts[0]))
		val := strings.TrimSpace(parts[1])

		switch key {
		case "user-agent":
			if current == nil {
				current = &robotsGroup{}
			}
			current.agents = append(current.agents, val)
		case "disallow":
			if current != nil && val != "" {
				current.disallows = append(current.disallows, val)
			}
		case "allow":
			if current != nil && val != "" {
				current.allows = append(current.allows, val)
			}
		}
	}
	if current != nil {
		rules.groups = append(rules.groups, *current)
	}
	return rules
}

func (r *robotsRules) isDisallowed(userAgent, path string) bool {
	// Extract agent name (first word before /)
	agentName := userAgent
	if idx := strings.Index(agentName, "/"); idx > 0 {
		agentName = agentName[:idx]
	}

	// Find best matching group: specific agent > wildcard
	var specificGroup, wildcardGroup *robotsGroup
	for i := range r.groups {
		g := &r.groups[i]
		for _, a := range g.agents {
			if strings.EqualFold(a, agentName) {
				specificGroup = g
			}
			if a == "*" {
				wildcardGroup = g
			}
		}
	}

	group := specificGroup
	if group == nil {
		group = wildcardGroup
	}
	if group == nil {
		return false
	}

	// Check allow first (more specific)
	for _, a := range group.allows {
		if strings.HasPrefix(path, a) {
			return false
		}
	}
	for _, d := range group.disallows {
		if strings.HasPrefix(path, d) {
			return true
		}
	}
	return false
}
