package jiracmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/coryb/figtree"
	"github.com/coryb/oreo"
	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	"github.com/go-jira/jira/jiradata"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type CrawlOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
	Project               string   `yaml:"project,omitempty" json:"project,omitempty"`
	Issues                []string `yaml:"issues,omitempty" json:"issues,omitempty"`
	Directory             string   `yaml:"directory,omitempty" json:"directory,omitempty"`
	Depth                 int      `yaml:"depth,omitempty" json:"depth,omitempty"`
	Links                 bool     `yaml:"links,omitempty" json:"links,omitempty"`
	Subtasks              bool     `yaml:"subtasks,omitempty" json:"subtasks,omitempty"`
	Parents               bool     `yaml:"parents,omitempty" json:"parents,omitempty"`
	Refresh               bool     `yaml:"refresh,omitempty" json:"refresh,omitempty"`
}

func CmdCrawlRegistry() *jiracli.CommandRegistryEntry {
	opts := CrawlOptions{}

	return &jiracli.CommandRegistryEntry{
		"Follow issue links from seed issues and download every reachable ticket",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdCrawlUsage(cmd, &opts)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			for i, issue := range opts.Issues {
				opts.Issues[i] = jiracli.FormatIssue(issue, opts.Project)
			}
			return CmdCrawl(o, globals, &opts)
		},
	}
}

func CmdCrawlUsage(cmd *kingpin.CmdClause, opts *CrawlOptions) error {
	cmd.Flag("project", "Project used to fully qualify bare issue ids").Short('p').StringVar(&opts.Project)
	cmd.Flag("directory", "Directory to write downloaded issues into").Short('d').Default("jira-crawl").StringVar(&opts.Directory)
	cmd.Flag("depth", "Maximum link hops from the seed issues (0 = unlimited)").IntVar(&opts.Depth)
	cmd.Flag("links", "Follow issuelinks (blockers/depends/relates)").Default("true").BoolVar(&opts.Links)
	cmd.Flag("subtasks", "Follow subtask links").Default("true").BoolVar(&opts.Subtasks)
	cmd.Flag("parents", "Follow parent links").Default("true").BoolVar(&opts.Parents)
	cmd.Flag("refresh", "Re-download issues even if a local copy already exists").BoolVar(&opts.Refresh)
	cmd.Arg("ISSUE", "Seed issue id(s) to crawl from").Required().StringsVar(&opts.Issues)
	return nil
}

// crawlNode is one entry in the breadth-first traversal queue.
type crawlNode struct {
	key   string
	depth int
}

// CmdCrawl walks the graph of linked issues breadth-first starting from the
// seed issues, downloading the full JSON for every ticket it reaches.
func CmdCrawl(o *oreo.Client, globals *jiracli.GlobalOptions, opts *CrawlOptions) error {
	dir := opts.Directory
	if dir == "" {
		dir = "jira-crawl"
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// load returns the locally cached copy of an issue if it was already
	// downloaded, so we can skip re-fetching it. A (nil, nil) result means
	// there is no local copy and the issue should be fetched fresh.
	load := func(key string) (*jiradata.Issue, error) {
		if opts.Refresh {
			return nil, nil
		}
		buf, err := os.ReadFile(crawlIssuePath(dir, key))
		if os.IsNotExist(err) {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		issue := &jiradata.Issue{}
		if err := json.Unmarshal(buf, issue); err != nil {
			return nil, err
		}
		return issue, nil
	}
	// Fetch the full issue (nil query) so the link/subtask/parent fields are
	// always present for traversal regardless of any field config.
	fetch := func(key string) (*jiradata.Issue, error) {
		return jira.GetIssue(o, globals.Endpoint.Value, key, nil)
	}
	save := func(key string, issue *jiradata.Issue) error {
		return jiracli.DownloadToFile(filepath.Join(dir, key), issue)
	}

	downloaded, skipped, err := crawlGraph(opts.Issues, opts, load, fetch, save)
	if err != nil {
		return err
	}

	if !globals.Quiet.Value {
		fmt.Fprintf(os.Stderr, "Crawled %d issue(s) into %s/ (%d already present)\n", len(downloaded), dir, len(skipped))
	}
	return nil
}

// crawlIssuePath is the path DownloadToFile writes an issue's JSON to.
func crawlIssuePath(dir, key string) string {
	return filepath.Join(dir, key) + ".json"
}

// crawlGraph performs the breadth-first traversal over the issue-link graph,
// independent of any network or filesystem concerns so it can be unit tested.
// A "visited" set tracks each key the moment it is enqueued, so link cycles
// (A links B, B links back to A) terminate instead of looping forever, and so
// each ticket is processed at most once.
//
// For each issue, load is consulted first: if it returns an already-downloaded
// copy, the issue is skipped (not re-fetched or re-saved) but its links are
// still walked so the crawl can discover tickets that have not been downloaded
// yet. Otherwise the issue is fetched and saved. It returns the keys that were
// downloaded and the keys that were skipped, in the order they were processed.
func crawlGraph(seeds []string, opts *CrawlOptions, load func(string) (*jiradata.Issue, error), fetch func(string) (*jiradata.Issue, error), save func(string, *jiradata.Issue) error) (downloaded []string, skipped []string, err error) {
	visited := map[string]bool{}
	queue := []crawlNode{}
	enqueue := func(key string, depth int) {
		if key == "" || visited[key] {
			return
		}
		visited[key] = true
		queue = append(queue, crawlNode{key: key, depth: depth})
	}

	for _, key := range seeds {
		enqueue(key, 0)
	}

	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]

		// Prefer an already-downloaded local copy so we skip re-downloading,
		// but still follow its links below to find new tickets.
		data, loadErr := load(node.key)
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not read cached %s: %s; re-downloading\n", node.key, loadErr)
			data = nil
		}
		if data != nil {
			skipped = append(skipped, node.key)
		} else {
			fetched, fetchErr := fetch(node.key)
			if fetchErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to fetch %s: %s\n", node.key, fetchErr)
				continue
			}
			if saveErr := save(node.key, fetched); saveErr != nil {
				return downloaded, skipped, saveErr
			}
			downloaded = append(downloaded, node.key)
			data = fetched
		}

		// Stop descending once we hit the depth limit, but the current issue
		// has already been handled above.
		if opts.Depth > 0 && node.depth >= opts.Depth {
			continue
		}
		for _, key := range crawlLinkedKeys(data, opts) {
			enqueue(key, node.depth+1)
		}
	}

	return downloaded, skipped, nil
}

// crawlLinkedKeys pulls the keys of every issue that the given issue links to.
// Issue fields come back as untyped JSON (map[string]interface{}), so we walk
// the maps defensively, skipping anything that is missing or not shaped the way
// we expect.
func crawlLinkedKeys(issue *jiradata.Issue, opts *CrawlOptions) []string {
	if issue == nil || issue.Fields == nil {
		return nil
	}
	keys := []string{}

	if opts.Links {
		if links, ok := issue.Fields["issuelinks"].([]interface{}); ok {
			for _, raw := range links {
				link, ok := raw.(map[string]interface{})
				if !ok {
					continue
				}
				// A link populates exactly one of inwardIssue/outwardIssue
				// depending on its direction relative to this issue.
				for _, dir := range []string{"inwardIssue", "outwardIssue"} {
					if key := crawlRefKey(link[dir]); key != "" {
						keys = append(keys, key)
					}
				}
			}
		}
	}

	if opts.Subtasks {
		if subtasks, ok := issue.Fields["subtasks"].([]interface{}); ok {
			for _, raw := range subtasks {
				if key := crawlRefKey(raw); key != "" {
					keys = append(keys, key)
				}
			}
		}
	}

	if opts.Parents {
		if key := crawlRefKey(issue.Fields["parent"]); key != "" {
			keys = append(keys, key)
		}
	}

	return keys
}

// crawlRefKey extracts the "key" string from an issue-reference-shaped value.
func crawlRefKey(raw interface{}) string {
	ref, ok := raw.(map[string]interface{})
	if !ok {
		return ""
	}
	key, _ := ref["key"].(string)
	return key
}
