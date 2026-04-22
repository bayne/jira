package jiracmd

import (
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

type ListOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
	jira.SearchOptions    `yaml:",inline" json:",inline" figtree:",inline"`
	Queries               map[string]string `yaml:"queries,omitempty" json:"queries,omitempty"`
}

func CmdListRegistry() *jiracli.CommandRegistryEntry {
	opts := ListOptions{
		CommonOptions: jiracli.CommonOptions{
			Template: figtree.NewStringOption("list"),
		},
	}

	return &jiracli.CommandRegistryEntry{
		"Prints list of issues for given search criteria",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdListUsage(cmd, &opts, fig)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			if opts.QueryFields == "" {
				opts.QueryFields = "assignee,created,priority,reporter,status,summary,updated,issuetype"
			}
			if opts.Sort == "" {
				opts.Sort = "priority asc, key"
			}
			return CmdList(o, globals, &opts)
		},
	}
}

func CmdListUsage(cmd *kingpin.CmdClause, opts *ListOptions, fig *figtree.FigTree) error {
	jiracli.TemplateUsage(cmd, &opts.CommonOptions)
	jiracli.GJsonQueryUsage(cmd, &opts.CommonOptions)
	cmd.Flag("assignee", "User assigned the issue").Short('a').StringVar(&opts.Assignee)
	cmd.Flag("component", "Component to search for").Short('c').StringVar(&opts.Component)
	cmd.Flag("issuetype", "Issue type to search for").Short('i').StringVar(&opts.IssueType)
	cmd.Flag("limit", "Maximum number of results to return in search").Short('l').IntVar(&opts.MaxResults)
	cmd.Flag("project", "Project to search for").Short('p').StringVar(&opts.Project)
	cmd.Flag("named-query", "The name of a query in the `queries` configuration").Short('n').PreAction(func(ctx *kingpin.ParseContext) error {
		name := jiracli.FlagValue(ctx, "named-query")
		if query, ok := opts.Queries[name]; ok && query != "" {
			var err error
			opts.Query, err = jiracli.ConfigTemplate(fig, query, cmd.FullCommand(), opts)
			return err
		}
		return fmt.Errorf("A valid named-query %q not found in `queries` configuration", name)
	}).String()
	cmd.Flag("query", "Jira Query Language (JQL) expression for the search").Short('q').StringVar(&opts.Query)
	cmd.Flag("queryfields", "Fields that are used in \"list\" template").Short('f').StringVar(&opts.QueryFields)
	cmd.Flag("reporter", "Reporter to search for").Short('r').StringVar(&opts.Reporter)
	cmd.Flag("status", "Filter on issue status").Short('S').StringVar(&opts.Status)
	cmd.Flag("sort", "Sort order to return").Short('s').StringVar(&opts.Sort)
	cmd.Flag("watcher", "Watcher to search for").Short('w').StringVar(&opts.Watcher)
	return nil
}

// List will query jira and send data to "list" template
func CmdList(o *oreo.Client, globals *jiracli.GlobalOptions, opts *ListOptions) error {
	data, err := jira.Search(o, globals.Endpoint.Value, opts, jira.WithAutoPagination())
	if err != nil {
		return err
	}
	if globals.Download.Value {
		return downloadSearchResults(o, globals, data)
	}
	return opts.PrintTemplate(data)
}

// downloadSearchResults fetches the full issue for each result and saves
// each one to a file inside a "jira-download" subdirectory.
func downloadSearchResults(o *oreo.Client, globals *jiracli.GlobalOptions, data *jiradata.SearchResults) error {
	if len(data.Issues) == 0 {
		fmt.Fprintln(os.Stderr, "No issues to download")
		return nil
	}

	dir := "jira-download"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	for _, issue := range data.Issues {
		if issue.Key == "" {
			continue
		}
		// Fetch the full issue
		fullIssue, err := jira.GetIssue(o, globals.Endpoint.Value, issue.Key, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to fetch %s: %s\n", issue.Key, err)
			continue
		}
		filename := filepath.Join(dir, issue.Key)
		if err := jiracli.DownloadToFile(filename, "view", fullIssue); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "Downloaded %d issues to %s/\n", len(data.Issues), dir)
	return nil
}
