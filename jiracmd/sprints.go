package jiracmd

import (
	"github.com/coryb/figtree"
	"github.com/coryb/oreo"
	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	"gopkg.in/alecthomas/kingpin.v2"
	"strings"
)

type SprintsOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
	State                 figtree.StringOption `yaml:"state,omitempty" json:"state,omitempty" figtree:"state,omitempty"`
}

func CmdSprintsRegistry() *jiracli.CommandRegistryEntry {
	opts := SprintsOptions{
		CommonOptions: jiracli.CommonOptions{
			Template: figtree.NewStringOption("sprints"),
		},
	}

	return &jiracli.CommandRegistryEntry{
		"Gets the sprints",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdSprintsUsage(cmd, &opts, fig)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			return CmdSprints(o, globals, &opts)
		},
	}
}

func CmdSprintsUsage(cmd *kingpin.CmdClause, opts *SprintsOptions, fig *figtree.FigTree) error {
	jiracli.TemplateUsage(cmd, &opts.CommonOptions)
	jiracli.GJsonQueryUsage(cmd, &opts.CommonOptions)
	cmd.Flag("state", "Filter sprints by state (active, closed, future), comma separated. Default is all states").SetValue(&opts.State)
	return nil
}

func CmdSprints(o *oreo.Client, globals *jiracli.GlobalOptions, opts *SprintsOptions) error {
	states := []string{}
	if opts.State.Value != "" {
		states = strings.Split(opts.State.Value, ",")
	}
	data, err := jira.Sprints(o, globals.Endpoint.Value, globals.DefaultBoard.Value, states)
	if err != nil {
		return err
	}
	return opts.PrintTemplate(data)
}
