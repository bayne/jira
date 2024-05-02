package jiracmd

import (
	"github.com/coryb/figtree"
	"github.com/coryb/oreo"
	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	"github.com/go-jira/jira/jiradata"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
	"sort"
	"strconv"
)

type CurrentSprintOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
}

type Current struct {
	SearchResults *jiradata.SearchResults `yaml:"results,inline" json:"results,inline" figtree:"results,inline"`
	Sprint        *jiradata.Sprint        `yaml:"sprint,inline" json:"sprint,inline" figtree:"sprint,inline"`
}

func CmdCurrentSprintRegistry() *jiracli.CommandRegistryEntry {
	opts := CurrentSprintOptions{
		CommonOptions: jiracli.CommonOptions{
			Template: figtree.NewStringOption("current"),
		},
	}

	return &jiracli.CommandRegistryEntry{
		"Gets the current sprint",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdCurrentSprintUsage(cmd, &opts, fig)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			return CmdCurrentSprint(o, globals, &opts)
		},
	}
}

func CmdCurrentSprintUsage(cmd *kingpin.CmdClause, opts *CurrentSprintOptions, fig *figtree.FigTree) error {
	jiracli.TemplateUsage(cmd, &opts.CommonOptions)
	jiracli.GJsonQueryUsage(cmd, &opts.CommonOptions)
	return nil
}

func CmdCurrentSprint(o *oreo.Client, globals *jiracli.GlobalOptions, opts *CurrentSprintOptions) error {
	data, err := jira.Sprints(o, globals.Endpoint.Value, globals.DefaultBoard.Value, []string{"active"})
	if err != nil {
		return err
	}
	if len(data.Values) == 0 {
		return errors.New("There is no active sprints")
	}
	sprint := &data.Values[0]
	issues, err := jira.Search(o, globals.Endpoint.Value, &jira.SearchOptions{
		Query:       "sprint = " + strconv.Itoa(sprint.Id),
		QueryFields: "assignee,created,priority,reporter,status,summary,updated,issuetype",
	})
	sort.Slice(issues.Issues, func(i, j int) bool {
		return issues.Issues[i].Fields["status"].(map[string]interface{})["name"].(string) > issues.Issues[j].Fields["status"].(map[string]interface{})["name"].(string)
	})
	return opts.PrintTemplate(Current{
		Sprint:        sprint,
		SearchResults: issues,
	})
}
