package jiracmd

import (
	"fmt"
	"os"

	"github.com/coryb/figtree"
	"github.com/coryb/oreo"
	"github.com/go-jira/jira"
	"github.com/go-jira/jira/jiracli"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type FieldsMapOptions struct {
	Project   string `yaml:"project,omitempty" json:"project,omitempty"`
	IssueType string `yaml:"issuetype,omitempty" json:"issuetype,omitempty"`
}

func CmdFieldsMapRegistry() *jiracli.CommandRegistryEntry {
	opts := FieldsMapOptions{}

	return &jiracli.CommandRegistryEntry{
		"Generate field mappings for custom fields and options",
		func(fig *figtree.FigTree, cmd *kingpin.CmdClause) error {
			jiracli.LoadConfigs(cmd, fig, &opts)
			cmd.Flag("project", "Project to fetch create metadata for").Short('p').StringVar(&opts.Project)
			cmd.Flag("issuetype", "Issue type to fetch create metadata for").Short('i').StringVar(&opts.IssueType)
			return nil
		},
		func(o *oreo.Client, globals *jiracli.GlobalOptions) error {
			return CmdFieldsMap(o, globals, &opts)
		},
	}
}

func CmdFieldsMap(o *oreo.Client, globals *jiracli.GlobalOptions, opts *FieldsMapOptions) error {
	fm := &jiracli.FieldMap{
		Fields: make(map[string]*jiracli.FieldMapping),
	}

	// Fetch all fields from the API
	fields, err := jira.GetFields(o, globals.Endpoint.Value)
	if err != nil {
		return fmt.Errorf("failed to fetch fields: %w", err)
	}

	for _, field := range fields {
		key := field.ID
		// For custom fields, generate a friendly key from the name
		if field.Custom && field.Name != "" {
			key = jiracli.ToKey(field.Name)
		}
		fm.Fields[field.ID] = &jiracli.FieldMapping{
			Key:   key,
			Label: field.Name,
		}
	}

	// If project and issuetype are provided, also fetch create metadata
	// to get allowed values for option fields
	if opts.Project != "" {
		issueType := opts.IssueType
		if issueType == "" {
			issueType = "Task"
		}

		createMeta, err := jira.GetIssueCreateMetaIssueType(o, globals.Endpoint.Value, opts.Project, issueType)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not fetch create metadata for %s/%s: %v\n", opts.Project, issueType, err)
		} else {
			for fieldID, fieldMeta := range createMeta.Fields {
				if fieldMeta.AllowedValues == nil || len(fieldMeta.AllowedValues) == 0 {
					continue
				}

				mapping, ok := fm.Fields[fieldID]
				if !ok {
					mapping = &jiracli.FieldMapping{
						Key:   jiracli.ToKey(fieldMeta.Name),
						Label: fieldMeta.Name,
					}
					fm.Fields[fieldID] = mapping
				}

				mapping.Options = make(map[string]*jiracli.OptionMapping)
				for _, av := range fieldMeta.AllowedValues {
					avMap, ok := av.(map[string]interface{})
					if !ok {
						continue
					}

					// Try to extract id and name/value from the allowed value
					optionID := ""
					optionLabel := ""

					if id, ok := avMap["id"].(string); ok {
						optionID = id
					}
					if name, ok := avMap["name"].(string); ok {
						optionLabel = name
					} else if val, ok := avMap["value"].(string); ok {
						optionLabel = val
					}

					if optionID == "" && optionLabel == "" {
						continue
					}
					// Use the label as the ID key if no numeric ID
					if optionID == "" {
						optionID = optionLabel
					}
					if optionLabel == "" {
						optionLabel = optionID
					}

					mapping.Options[optionID] = &jiracli.OptionMapping{
						Key:   jiracli.ToKey(optionLabel),
						Label: optionLabel,
					}
				}
			}
		}
	}

	if err := jiracli.SaveFieldMap(fm); err != nil {
		return fmt.Errorf("failed to save field mapping: %w", err)
	}

	count := len(fm.Fields)
	optCount := 0
	for _, f := range fm.Fields {
		optCount += len(f.Options)
	}
	fmt.Printf("Generated field mapping: %d fields, %d option values\n", count, optCount)
	fmt.Printf("Saved to .jira.d/fields.json\n")
	return nil
}
