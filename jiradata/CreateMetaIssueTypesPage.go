package jiradata

// CreateMetaIssueTypesPage is the paginated response from
// GET /rest/api/2/issue/createmeta/{projectIdOrKey}/issuetypes
type CreateMetaIssueTypesPage struct {
	MaxResults int        `json:"maxResults,omitempty" yaml:"maxResults,omitempty"`
	StartAt    int        `json:"startAt,omitempty" yaml:"startAt,omitempty"`
	Total      int        `json:"total,omitempty" yaml:"total,omitempty"`
	Values     IssueTypes `json:"values,omitempty" yaml:"values,omitempty"`
}
