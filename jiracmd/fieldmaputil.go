package jiracmd

import (
	"github.com/go-jira/jira/jiracli"
	"github.com/go-jira/jira/jiradata"
)

// applyFieldMappings transforms friendly field keys and option values in an
// IssueUpdate back to their Jira API identifiers using the field mapping file.
// If no mapping file exists, this is a no-op.
func applyFieldMappings(issueUpdate *jiradata.IssueUpdate) {
	fm, err := jiracli.LoadFieldMap()
	if err != nil {
		return
	}
	if issueUpdate.Fields != nil {
		issueUpdate.Fields = fm.TransformFieldKeys(issueUpdate.Fields)
	}
}
