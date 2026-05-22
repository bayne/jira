package jiracmd

import (
	"fmt"

	"github.com/coryb/figtree"
	"github.com/coryb/oreo"

	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	"github.com/go-jira/jira/jiradata"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type FixVersionSetOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
	Project               string   `yaml:"project,omitempty" json:"project,omitempty"`
	Issue                 string   `yaml:"issue,omitempty" json:"issue,omitempty"`
	Versions              []string `yaml:"versions,omitempty" json:"versions,omitempty"`
}

func CmdFixVersionSetRegistry() *jiracli.CommandRegistryEntry {
	opts := FixVersionSetOptions{}
	return &jiracli.CommandRegistryEntry{
		"Set fix version on an issue",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdFixVersionSetUsage(cmd, &opts)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			opts.Issue = jiracli.FormatIssue(opts.Issue, opts.Project)
			return CmdFixVersionSet(o, globals, &opts)
		},
	}
}

func CmdFixVersionSetUsage(cmd *kingpin.CmdClause, opts *FixVersionSetOptions) error {
	jiracli.BrowseUsage(cmd, &opts.CommonOptions)
	cmd.Arg("ISSUE", "issue id to set fix version").Required().StringVar(&opts.Issue)
	cmd.Arg("VERSION", "version to set on issue").Required().StringsVar(&opts.Versions)
	return nil
}

func CmdFixVersionSet(o *oreo.Client, globals *jiracli.GlobalOptions, opts *FixVersionSetOptions) error {
	versions := []jiradata.Version{}
	for _, v := range opts.Versions {
		versions = append(versions, jiradata.Version{Name: v})
	}

	issueUpdate := jiradata.IssueUpdate{
		Update: jiradata.FieldOperationsMap{
			"fixVersions": jiradata.FieldOperations{
				jiradata.FieldOperation{
					"set": versions,
				},
			},
		},
	}

	if err := jira.EditIssue(o, globals.Endpoint.Value, opts.Issue, &issueUpdate); err != nil {
		return err
	}
	if !globals.Quiet.Value {
		fmt.Printf("OK %s %s\n", opts.Issue, jira.URLJoin(globals.Endpoint.Value, "browse", opts.Issue))
	}
	if opts.Browse.Value {
		return CmdBrowse(globals, opts.Issue)
	}
	return nil
}
