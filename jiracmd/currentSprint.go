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

const firstAssigneeField = "_first_assignee"

func cacheFirstAssignees(issues jiradata.Issues) {
	for _, issue := range issues {
		issue.Fields[firstAssigneeField] = resolveFirstAssignee(issue)
	}
}

func resolveFirstAssignee(issue *jiradata.Issue) string {
	if issue.Changelog != nil {
		for _, history := range issue.Changelog.Histories {
			for _, item := range history.Items {
				if item.Field == "assignee" && item.FromString == "" && item.ToString != "" {
					return item.ToString
				}
			}
		}
	}
	if a, ok := issue.Fields["assignee"]; ok && a != nil {
		if m, ok := a.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok && name != "" {
				return name
			}
		}
	}
	return "Unassigned"
}

func hasMultipleSprints(issue *jiradata.Issue) bool {
	sprints, ok := issue.Fields["customfield_10105"]
	if !ok || sprints == nil {
		return false
	}
	if arr, ok := sprints.([]interface{}); ok {
		return len(arr) > 1
	}
	return false
}

func computePointDistribution(issues jiradata.Issues) []PointAllocation {
	cacheFirstAssignees(issues)
	totals := map[string]int{}
	for _, issue := range issues {
		if hasMultipleSprints(issue) {
			continue
		}
		assignee := issue.Fields[firstAssigneeField].(string)
		if pts, ok := issue.Fields["customfield_10106"]; ok && pts != nil {
			switch v := pts.(type) {
			case float64:
				totals[assignee] += int(v)
			case int:
				totals[assignee] += v
			}
		}
	}
	result := make([]PointAllocation, 0, len(totals))
	for name, points := range totals {
		result = append(result, PointAllocation{Name: name, Points: points})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Points > result[j].Points
	})
	return result
}

type CurrentSprintOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
}

type PointAllocation struct {
	Name   string `yaml:"name" json:"name"`
	Points int    `yaml:"points" json:"points"`
}

type Current struct {
	SearchResults     *jiradata.SearchResults `yaml:"results,inline" json:"results,inline" figtree:"results,inline"`
	Sprint            *jiradata.Sprint        `yaml:"sprint,inline" json:"sprint,inline" figtree:"sprint,inline"`
	PointDistribution []PointAllocation       `yaml:"point_distribution" json:"point_distribution"`
}

func CmdCurrentSprintRegistry() *jiracli.CommandRegistryEntry {
	opts := CurrentSprintOptions{
		CommonOptions: jiracli.CommonOptions{
			Template: figtree.NewStringOption("sprint"),
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
		QueryFields: "assignee,created,priority,customfield_10105,customfield_10106,reporter,status,summary,updated,issuetype,fixVersions",
	}, jira.WithExpand("changelog"))
	sort.Slice(issues.Issues, func(i, j int) bool {
		return issues.Issues[i].Fields["status"].(map[string]interface{})["name"].(string) > issues.Issues[j].Fields["status"].(map[string]interface{})["name"].(string)
	})
	if globals.Download.Value {
		return downloadSearchResults(o, globals, issues)
	}
	return opts.PrintTemplate(Current{
		Sprint:            sprint,
		SearchResults:     issues,
		PointDistribution: computePointDistribution(issues.Issues),
	})
}
