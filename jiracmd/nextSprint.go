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
	Offset                figtree.Int8Option `yaml:"offset,omitempty" json:"offset,omitempty" figtree:"offset,omitempty"`
}

type Next struct {
	SearchResults     *jiradata.SearchResults `yaml:"results,inline" json:"results,inline" figtree:"results,inline"`
	Sprint            *jiradata.Sprint        `yaml:"sprint,inline" json:"sprint,inline" figtree:"sprint,inline"`
	PointDistribution []PointAllocation       `yaml:"point_distribution" json:"point_distribution"`
}

func CmdNextSprintRegistry() *jiracli.CommandRegistryEntry {
	opts := NextSprintOptions{
		CommonOptions: jiracli.CommonOptions{
			Template: figtree.NewStringOption("sprint"),
		},
		Offset: figtree.NewInt8Option(0),
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
	cmd.Flag("offset", "Select OFFSET sprint in the future").SetValue(&opts.Offset)
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
	sort.Slice(values, func(i, j int) bool {
		return data.Values[i].StartDate < data.Values[j].StartDate
	})
	sprint := values[opts.Offset.Value]
	issues, err := jira.Search(o, globals.Endpoint.Value, &jira.SearchOptions{
		Query:       "sprint = " + strconv.Itoa(sprint.Id),
		QueryFields: "assignee,created,Rank,priority,customfield_10105,customfield_10106,reporter,status,summary,updated,issuetype,customfield_10100,labels",
	}, jira.WithExpand("changelog"))
	sort.Slice(issues.Issues, func(i, j int) bool {
		return issues.Issues[i].Fields["customfield_10100"].(string) < issues.Issues[j].Fields["customfield_10100"].(string)
	})
	if globals.Download.Value {
		return downloadSearchResults(o, globals, issues)
	}
	return opts.PrintTemplate(Next{
		Sprint:            &sprint,
		SearchResults:     issues,
		PointDistribution: computePointDistribution(issues.Issues),
	})
}
