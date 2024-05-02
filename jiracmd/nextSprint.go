package jiracmd

import (
	"github.com/coryb/figtree"
	"github.com/coryb/oreo"
	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	"github.com/go-jira/jira/jiradata"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
	"gopkg.in/alecthomas/kingpin.v2"
	"sort"
	"strconv"
)

type NextSprintOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
}

type Next struct {
	SearchResults *jiradata.SearchResults `yaml:"results,inline" json:"results,inline" figtree:"results,inline"`
	Sprint        *jiradata.Sprint        `yaml:"sprint,inline" json:"sprint,inline" figtree:"sprint,inline"`
}

func CmdNextSprintRegistry() *jiracli.CommandRegistryEntry {
	opts := NextSprintOptions{
		CommonOptions: jiracli.CommonOptions{
			Template: figtree.NewStringOption("sprint"),
		},
	}

	return &jiracli.CommandRegistryEntry{
		"Gets the Next sprint",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdNextSprintUsage(cmd, &opts, fig)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			return CmdNextSprint(o, globals, &opts)
		},
	}
}

func CmdNextSprintUsage(cmd *kingpin.CmdClause, opts *NextSprintOptions, fig *figtree.FigTree) error {
	jiracli.TemplateUsage(cmd, &opts.CommonOptions)
	jiracli.GJsonQueryUsage(cmd, &opts.CommonOptions)
	return nil
}

func CmdNextSprint(o *oreo.Client, globals *jiracli.GlobalOptions, opts *NextSprintOptions) error {
	data, err := jira.Sprints(o, globals.Endpoint.Value, globals.DefaultBoard.Value, []string{"future"})
	if err != nil {
		return err
	}
	if len(data.Values) == 0 {
		return errors.New("There is no future sprints")
	}
	values := slices.DeleteFunc(data.Values, func(sprint jiradata.Sprint) bool {
		return sprint.StartDate == ""
	})
	values = slices.DeleteFunc(values, func(sprint jiradata.Sprint) bool {
		return strconv.Itoa(sprint.OriginBoardId) != globals.DefaultBoard.Value
	})
	sort.Slice(values, func(i, j int) bool {
		return data.Values[i].StartDate < data.Values[j].StartDate
	})
	sprint := values[0]
	issues, err := jira.Search(o, globals.Endpoint.Value, &jira.SearchOptions{
		Query:       "sprint = " + strconv.Itoa(sprint.Id),
		QueryFields: "assignee,created,customfield_10006,priority,reporter,status,summary,updated,issuetype",
	})
	sort.Slice(issues.Issues, func(i, j int) bool {
		return issues.Issues[i].Fields["status"].(map[string]interface{})["name"].(string) > issues.Issues[j].Fields["status"].(map[string]interface{})["name"].(string)
	})
	return opts.PrintTemplate(Next{
		Sprint:        &sprint,
		SearchResults: issues,
	})
}
