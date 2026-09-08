package jiracmd

import (
	"github.com/coryb/figtree"
	"github.com/coryb/oreo"
	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type StructureOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
	Project               string   `yaml:"project,omitempty" json:"project,omitempty"`
	Issue                 string   `yaml:"issue,omitempty" json:"issue,omitempty"`
	LinkTypes             []string `yaml:"link-types,omitempty" json:"link-types,omitempty"`
	FilterProjects        []string `yaml:"filter-projects,omitempty" json:"filter-projects,omitempty"`
	Depth                 int      `yaml:"depth,omitempty" json:"depth,omitempty"`
}

func CmdStructureRegistry() *jiracli.CommandRegistryEntry {
	opts := StructureOptions{
		CommonOptions: jiracli.CommonOptions{
			Template: figtree.NewStringOption("structure"),
		},
		LinkTypes:      []string{"Issue split", "subtask"},
		FilterProjects: []string{"ADS", "SIMBA", "TRP"},
	}

	return &jiracli.CommandRegistryEntry{
		"Display issue split tree from a root ticket",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdStructureUsage(cmd, &opts)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			opts.Issue = jiracli.FormatIssue(opts.Issue, opts.Project)
			return CmdStructure(o, globals, &opts)
		},
	}
}

func CmdStructureUsage(cmd *kingpin.CmdClause, opts *StructureOptions) error {
	jiracli.TemplateUsage(cmd, &opts.CommonOptions)
	jiracli.GJsonQueryUsage(cmd, &opts.CommonOptions)
	cmd.Flag("project", "project to use for bare issue ids").Short('p').StringVar(&opts.Project)
	cmd.Flag("link-type", "issue link type names to follow (repeatable; 'subtask' follows subtask children)").Default("Issue split", "subtask").StringsVar(&opts.LinkTypes)
	cmd.Flag("filter-project", "only traverse into issues belonging to these projects (repeatable)").Default("ADS", "SIMBA", "TRP").StringsVar(&opts.FilterProjects)
	cmd.Flag("depth", "maximum tree depth (0 = unlimited)").IntVar(&opts.Depth)
	cmd.Arg("ISSUE", "root issue to build tree from").Required().StringVar(&opts.Issue)
	return nil
}

func CmdStructure(o *oreo.Client, globals *jiracli.GlobalOptions, opts *StructureOptions) error {
	data, err := jira.BuildStructureTree(o, globals.Endpoint.Value, opts.Issue, opts.LinkTypes, opts.FilterProjects, opts.Depth)
	if err != nil {
		return err
	}
	return opts.PrintTemplate(data)
}
