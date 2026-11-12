package ir

import (
	"strconv"
	"time"
)

type Source struct {
	Host      string    `json:"host,omitempty"`
	Hash      string    `json:"hash,omitempty"`
	ETag      string    `json:"etag,omitempty"`
	FetchedAt time.Time `json:"fetched_at,omitempty"`
}

type Warning struct {
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

type Profile struct {
	Source   *Source   `json:"source,omitempty"`
	Nodes    []Node    `json:"nodes"`
	Groups   []Group   `json:"groups,omitempty"`
	Warnings []Warning `json:"warnings,omitempty"`
}

type Group struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Proxies   []string `json:"proxies"`
	URL       string   `json:"url,omitempty"`
	Interval  int      `json:"interval,omitempty"`
	Tolerance int      `json:"tolerance,omitempty"`
}

func NormalizeTags(nodes []Node) ([]Node, []Warning) {
	out := make([]Node, len(nodes))
	copy(out, nodes)

	seen := make(map[string]int, len(nodes))
	warnings := make([]Warning, 0)
	for i := range out {
		if out[i].Tag == "" {
			out[i].Tag = "node"
		}
		seen[out[i].Tag]++
		if seen[out[i].Tag] == 1 {
			continue
		}
		next := out[i].Tag + "#" + strconv.Itoa(seen[out[i].Tag])
		warnings = append(warnings, Warning{Message: "duplicate tag " + out[i].Tag + " renamed to " + next})
		out[i].Tag = next
	}
	return out, warnings
}
