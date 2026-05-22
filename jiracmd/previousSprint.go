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

type PreviousSprintOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
	Offset                figtree.Int8Option `yaml:"offset,omitempty" json:"offset,omitempty" figtree:"offset,omitempty"`
}

type Previous struct {
	SearchResults     *jiradata.SearchResults `yaml:"results,inline" json:"results,inline" figtree:"results,inline"`
	Sprint            *jiradata.Sprint        `yaml:"sprint,inline" json:"sprint,inline" figtree:"sprint,inline"`
	PointDistribution []PointAllocation       `yaml:"point_distribution" json:"point_distribution"`
}

func CmdPreviousSprintRegistry() *jiracli.CommandRegistryEntry {
	opts := PreviousSprintOptions{
		CommonOptions: jiracli.CommonOptions{
			Template: figtree.NewStringOption("sprint"),
		},
		Offset: figtree.NewInt8Option(0),
	}

	return &jiracli.CommandRegistryEntry{
		"Gets the previous sprint",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdPreviousSprintUsage(cmd, &opts, fig)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			return CmdPreviousSprint(o, globals, &opts)
		},
	}
}

func CmdPreviousSprintUsage(cmd *kingpin.CmdClause, opts *PreviousSprintOptions, fig *figtree.FigTree) error {
	jiracli.TemplateUsage(cmd, &opts.CommonOptions)
	jiracli.GJsonQueryUsage(cmd, &opts.CommonOptions)
	cmd.Flag("offset", "Select OFFSET sprint in the past").SetValue(&opts.Offset)
	return nil
}

func CmdPreviousSprint(o *oreo.Client, globals *jiracli.GlobalOptions, opts *PreviousSprintOptions) error {
	data, err := jira.Sprints(o, globals.Endpoint.Value, globals.DefaultBoard.Value, []string{"closed"})
	if err != nil {
		return err
	}
	if len(data.Values) == 0 {
		return errors.New("There are no closed sprints")
	}
	sort.Slice(data.Values, func(i, j int) bool {
		return data.Values[i].CompleteDate > data.Values[j].CompleteDate
	})
	sprint := data.Values[opts.Offset.Value]
	issues, err := jira.Search(o, globals.Endpoint.Value, &jira.SearchOptions{
		Query:       "sprint = " + strconv.Itoa(sprint.Id),
		QueryFields: "assignee,created,priority,customfield_10105,customfield_10106,reporter,status,summary,updated,issuetype,fixVersions",
	}, jira.WithExpand("changelog"))
	if err != nil {
		return err
	}
	sort.Slice(issues.Issues, func(i, j int) bool {
		return issues.Issues[i].Fields["status"].(map[string]interface{})["name"].(string) > issues.Issues[j].Fields["status"].(map[string]interface{})["name"].(string)
	})
	if globals.Download.Value {
		return downloadSearchResults(o, globals, issues)
	}
	return opts.PrintTemplate(Previous{
		Sprint:            &sprint,
		SearchResults:     issues,
		PointDistribution: computePointDistribution(issues.Issues),
	})
}
