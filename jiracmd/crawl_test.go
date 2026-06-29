package jiracmd

import (
	"fmt"
	"sort"
	"testing"

	"github.com/go-jira/jira/jiradata"
)

// issueWith builds an issue whose fields contain the given issuelinks,
// subtasks, and parent, mirroring the untyped JSON shape the real API returns.
func issueWith(key string, links []string, subtasks []string, parent string) *jiradata.Issue {
	fields := map[string]interface{}{}

	linkList := []interface{}{}
	for _, l := range links {
		// alternate the direction to exercise both inwardIssue and outwardIssue
		dir := "outwardIssue"
		if len(linkList)%2 == 1 {
			dir = "inwardIssue"
		}
		linkList = append(linkList, map[string]interface{}{
			dir: map[string]interface{}{"key": l},
		})
	}
	fields["issuelinks"] = linkList

	subList := []interface{}{}
	for _, s := range subtasks {
		subList = append(subList, map[string]interface{}{"key": s})
	}
	fields["subtasks"] = subList

	if parent != "" {
		fields["parent"] = map[string]interface{}{"key": parent}
	}

	return &jiradata.Issue{Key: key, Fields: fields}
}

// runCrawl drives crawlGraph against an in-memory graph, returning the fetch
// count per key and the download order.
func runCrawl(t *testing.T, graph map[string]*jiradata.Issue, seeds []string, opts *CrawlOptions) (map[string]int, []string) {
	t.Helper()
	fetches := map[string]int{}
	fetch := func(key string) (*jiradata.Issue, error) {
		fetches[key]++
		issue, ok := graph[key]
		if !ok {
			return nil, fmt.Errorf("not found: %s", key)
		}
		return issue, nil
	}
	saved := []string{}
	save := func(key string, _ *jiradata.Issue) error {
		saved = append(saved, key)
		return nil
	}
	// nothing is cached locally, so every reachable issue is fetched
	noCache := func(string) (*jiradata.Issue, error) { return nil, nil }
	order, _, err := crawlGraph(seeds, opts, noCache, fetch, save)
	if err != nil {
		t.Fatalf("crawlGraph returned error: %s", err)
	}
	return fetches, order
}

// TestCrawlHandlesCycles is the core requirement: a graph with cycles
// (A<->B, plus C self-loop) must terminate and fetch each ticket exactly once.
func TestCrawlHandlesCycles(t *testing.T) {
	graph := map[string]*jiradata.Issue{
		"A": issueWith("A", []string{"B", "C"}, nil, ""),
		"B": issueWith("B", []string{"A"}, nil, ""),      // B links back to A
		"C": issueWith("C", []string{"C", "A"}, nil, ""), // C self-loops and back to A
	}
	opts := &CrawlOptions{Links: true}

	fetches, order := runCrawl(t, graph, []string{"A"}, opts)

	if len(order) != 3 {
		t.Fatalf("expected 3 issues downloaded, got %d (%v)", len(order), order)
	}
	for _, key := range []string{"A", "B", "C"} {
		if fetches[key] != 1 {
			t.Errorf("expected %s fetched exactly once, got %d", key, fetches[key])
		}
	}
}

// TestCrawlFollowsAllRelations confirms issuelinks, subtasks, and parents are
// all traversed when enabled.
func TestCrawlFollowsAllRelations(t *testing.T) {
	graph := map[string]*jiradata.Issue{
		"A": issueWith("A", []string{"L"}, []string{"S"}, "P"),
		"L": issueWith("L", nil, nil, ""),
		"S": issueWith("S", nil, nil, ""),
		"P": issueWith("P", nil, nil, ""),
	}
	opts := &CrawlOptions{Links: true, Subtasks: true, Parents: true}

	_, order := runCrawl(t, graph, []string{"A"}, opts)

	sort.Strings(order)
	want := []string{"A", "L", "P", "S"}
	if fmt.Sprint(order) != fmt.Sprint(want) {
		t.Errorf("expected %v, got %v", want, order)
	}
}

// TestCrawlRelationToggles confirms the follow flags gate traversal.
func TestCrawlRelationToggles(t *testing.T) {
	graph := map[string]*jiradata.Issue{
		"A": issueWith("A", []string{"L"}, []string{"S"}, "P"),
		"L": issueWith("L", nil, nil, ""),
		"S": issueWith("S", nil, nil, ""),
		"P": issueWith("P", nil, nil, ""),
	}
	// only follow issuelinks; subtasks and parent must be ignored
	opts := &CrawlOptions{Links: true, Subtasks: false, Parents: false}

	fetches, order := runCrawl(t, graph, []string{"A"}, opts)

	if len(order) != 2 {
		t.Fatalf("expected 2 issues (A, L), got %d (%v)", len(order), order)
	}
	if fetches["S"] != 0 || fetches["P"] != 0 {
		t.Errorf("subtask/parent should not be fetched when disabled: S=%d P=%d", fetches["S"], fetches["P"])
	}
}

// TestCrawlSkipsDownloaded confirms already-downloaded tickets are not
// re-fetched, yet their links are still followed so newly-linked tickets get
// downloaded. Graph: A -> {B, C}, C -> D, with A and C already cached locally.
func TestCrawlSkipsDownloaded(t *testing.T) {
	graph := map[string]*jiradata.Issue{
		"A": issueWith("A", []string{"B", "C"}, nil, ""),
		"B": issueWith("B", nil, nil, ""),
		"C": issueWith("C", []string{"D"}, nil, ""),
		"D": issueWith("D", nil, nil, ""),
	}
	cached := map[string]bool{"A": true, "C": true}

	load := func(key string) (*jiradata.Issue, error) {
		if cached[key] {
			return graph[key], nil
		}
		return nil, nil // not present locally -> must be fetched
	}
	fetches := map[string]int{}
	fetch := func(key string) (*jiradata.Issue, error) {
		fetches[key]++
		return graph[key], nil
	}
	save := func(string, *jiradata.Issue) error { return nil }

	opts := &CrawlOptions{Links: true}
	downloaded, skipped, err := crawlGraph([]string{"A"}, opts, load, fetch, save)
	if err != nil {
		t.Fatalf("crawlGraph returned error: %s", err)
	}

	// cached tickets are never fetched...
	if fetches["A"] != 0 || fetches["C"] != 0 {
		t.Errorf("cached tickets should not be fetched: A=%d C=%d", fetches["A"], fetches["C"])
	}
	// ...but the not-yet-downloaded tickets reachable through them are
	if fetches["B"] != 1 || fetches["D"] != 1 {
		t.Errorf("expected B and D fetched once: B=%d D=%d", fetches["B"], fetches["D"])
	}

	sort.Strings(downloaded)
	if want := []string{"B", "D"}; fmt.Sprint(downloaded) != fmt.Sprint(want) {
		t.Errorf("expected downloaded %v, got %v", want, downloaded)
	}
	sort.Strings(skipped)
	if want := []string{"A", "C"}; fmt.Sprint(skipped) != fmt.Sprint(want) {
		t.Errorf("expected skipped %v, got %v", want, skipped)
	}
}

// TestCrawlDepthLimit confirms the crawl stops descending past the depth limit
// while still saving the boundary issue.
func TestCrawlDepthLimit(t *testing.T) {
	// chain: A -> B -> C -> D
	graph := map[string]*jiradata.Issue{
		"A": issueWith("A", []string{"B"}, nil, ""),
		"B": issueWith("B", []string{"C"}, nil, ""),
		"C": issueWith("C", []string{"D"}, nil, ""),
		"D": issueWith("D", nil, nil, ""),
	}
	opts := &CrawlOptions{Links: true, Depth: 1}

	_, order := runCrawl(t, graph, []string{"A"}, opts)

	// depth 1 = seed (A, depth 0) plus its direct links (B, depth 1); C/D are
	// beyond the limit
	if len(order) != 2 {
		t.Fatalf("expected 2 issues at depth 1, got %d (%v)", len(order), order)
	}
}
