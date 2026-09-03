package jiradata

// CreateMetaIssueTypesPage is the paginated response from
// GET /rest/api/2/issue/createmeta/{projectIdOrKey}/issuetypes
// Jira Server/DC returns the page items under "values", Jira Cloud under "issueTypes".
type CreateMetaIssueTypesPage struct {
	MaxResults int        `json:"maxResults,omitempty" yaml:"maxResults,omitempty"`
	StartAt    int        `json:"startAt,omitempty" yaml:"startAt,omitempty"`
	Total      int        `json:"total,omitempty" yaml:"total,omitempty"`
	Values     IssueTypes `json:"values,omitempty" yaml:"values,omitempty"`
	IssueTypes IssueTypes `json:"issueTypes,omitempty" yaml:"issueTypes,omitempty"`
}

// PageValues returns the page items regardless of which key the server used.
func (p *CreateMetaIssueTypesPage) PageValues() IssueTypes {
	if len(p.Values) > 0 {
		return p.Values
	}
	return p.IssueTypes
}
