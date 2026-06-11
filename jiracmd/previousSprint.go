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
	"time"
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

type datedSprint struct {
	sprint    jiradata.Sprint
	startDate time.Time
}

// previousSprints returns closed sprints whose start date is before now,
// ordered most recently started first. The active sprint is excluded so
// that offset 0 is the sprint before the current one.
func previousSprints(sprints []jiradata.Sprint, now time.Time) []datedSprint {
	previous := []datedSprint{}
	for _, sprint := range sprints {
		if sprint.State != "closed" {
			continue
		}
		startDate, err := time.Parse(time.RFC3339, sprint.StartDate)
		if err != nil {
			continue
		}
		if startDate.After(now) {
			continue
		}
		previous = append(previous, datedSprint{sprint: sprint, startDate: startDate})
	}
	sort.Slice(previous, func(i, j int) bool {
		return previous[i].startDate.After(previous[j].startDate)
	})
	return previous
}

func CmdPreviousSprint(o *oreo.Client, globals *jiracli.GlobalOptions, opts *PreviousSprintOptions) error {
	data, err := jira.Sprints(o, globals.Endpoint.Value, globals.DefaultBoard.Value, nil)
	if err != nil {
		return err
	}
	previous := previousSprints(data.Values, time.Now())
	if len(previous) == 0 {
		return errors.New("There are no sprints that started before now")
	}
	offset := int(opts.Offset.Value)
	if offset < 0 || offset >= len(previous) {
		return errors.Errorf("Offset %d is out of range, there are only %d previous sprints", offset, len(previous))
	}
	sprint := previous[offset].sprint
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
