package jiracmd

import (
	"bytes"
	"github.com/coryb/figtree"
	"github.com/coryb/oreo"
	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	"golang.org/x/net/html"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
	"strings"
	"time"
)

type ActivityOptions struct {
	jiracli.CommonOptions `yaml:",inline" json:",inline" figtree:",inline"`
	MaxResults            int    `yaml:"max_results,omitempty" json:"max_results,omitempty"`
	Streams               string `json:"streams,omitempty" yaml:"streams,omitempty"`
	Issues                string `json:"issues,omitempty" yaml:"issues,omitempty"`
	Providers             string `json:"providers,omitempty" yaml:"providers,omitempty"`
}

func CmdActivityRegistry() *jiracli.CommandRegistryEntry {
	opts := ActivityOptions{
		CommonOptions: jiracli.CommonOptions{
			Template: figtree.NewStringOption("activity"),
		},
	}

	return &jiracli.CommandRegistryEntry{
		"Prints issue details",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			return CmdActivityUsage(cmd, &opts)
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			return CmdActivity(o, globals, &opts)
		},
	}
}

func CmdActivityUsage(cmd *kingpin.CmdClause, opts *ActivityOptions) error {
	jiracli.TemplateUsage(cmd, &opts.CommonOptions)
	cmd.Flag("max-results", "max results").IntVar(&opts.MaxResults)
	cmd.Flag("streams", "filters for the streams").StringVar(&opts.Streams)
	cmd.Flag("issues", "filters for the issues").StringVar(&opts.Issues)
	cmd.Flag("providers", "the providers").StringVar(&opts.Providers)
	return nil
}

func CmdActivity(o *oreo.Client, globals *jiracli.GlobalOptions, opts *ActivityOptions) error {
	data, err := jira.Activity(o, globals.Endpoint.Value, opts.MaxResults, opts.Streams, opts.Issues, opts.Providers)
	if err != nil {
		return err
	}
	for i := range data.Entries {
		entry := data.Entries[i]
		content := html.UnescapeString(entry.Title.Value)
		doc, err := html.Parse(strings.NewReader(content))
		if err != nil {
			return err
		}
		var buffer bytes.Buffer
		collectTextNodes(doc, &buffer)
		text := buffer.String()
		data.Entries[i].Title.Value = text

		utcTime, err := time.Parse(time.RFC3339, entry.Published)
		if err != nil {
			return err
		}

		data.Entries[i].Published = utcTime.Local().Format("Mon, Jan 2 2006 3:04pm")
	}
	if err := opts.PrintTemplate(data); err != nil {
		return err
	}
	return nil
}

func collectTextNodes(n *html.Node, buf *bytes.Buffer) {
	if n.Type == html.TextNode {
		buf.WriteString(strings.TrimSpace(n.Data) + " ")
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectTextNodes(c, buf)
	}
}
