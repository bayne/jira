package jira

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-jira/jira/jiradata"
)

type SearchProvider interface {
	ProvideSearchRequest() *jiradata.SearchRequest
}

type SearchOptions struct {
	Assignee    string   `yaml:"assignee,omitempty" json:"assignee,omitempty"`
	Query       string   `yaml:"query,omitempty" json:"query,omitempty"`
	QueryFields string   `yaml:"query-fields,omitempty" json:"query-fields,omitempty"`
	Project     string   `yaml:"project,omitempty" json:"project,omitempty"`
	Component   string   `yaml:"component,omitempty" json:"component,omitempty"`
	IssueType   string   `yaml:"issue-type,omitempty" json:"issue-type,omitempty"`
	Watcher     string   `yaml:"watcher,omitempty" json:"watcher,omitempty"`
	Reporter    string   `yaml:"reporter,omitempty" json:"reporter,omitempty"`
	Status      string   `yaml:"status,omitempty" json:"status,omitempty"`
	Sort        string   `yaml:"sort,omitempty" json:"sort,omitempty"`
	MaxResults int `yaml:"max-results,omitempty" json:"max-results,omitempty"`
}

func (o *SearchOptions) ProvideSearchRequest() *jiradata.SearchRequest {
	req := &jiradata.SearchRequest{}

	if o.Query == "" {
		qbuff := bytes.NewBufferString("resolution = unresolved")
		if o.Project != "" {
			qbuff.WriteString(fmt.Sprintf(" AND project = '%s'", o.Project))
		}
		if o.Component != "" {
			qbuff.WriteString(fmt.Sprintf(" AND component = '%s'", o.Component))
		}
		if o.Assignee != "" {
			qbuff.WriteString(fmt.Sprintf(" AND assignee = '%s'", o.Assignee))
		}
		if o.IssueType != "" {
			qbuff.WriteString(fmt.Sprintf(" AND issuetype = '%s'", o.IssueType))
		}
		if o.Watcher != "" {
			qbuff.WriteString(fmt.Sprintf(" AND watcher = '%s'", o.Watcher))
		}
		if o.Reporter != "" {
			qbuff.WriteString(fmt.Sprintf(" AND reporter = '%s'", o.Reporter))
		}
		if o.Status != "" {
			qbuff.WriteString(fmt.Sprintf(" AND status = '%s'", o.Status))
		}
		if o.Sort != "" {
			qbuff.WriteString(fmt.Sprintf(" ORDER BY %s", o.Sort))
		}
		req.JQL = qbuff.String()
	} else {
		req.JQL = o.Query
	}

	req.Fields = append(req.Fields, "summary")
	if o.QueryFields != "" {
		fields := strings.Split(o.QueryFields, ",")
		req.Fields = append(req.Fields, fields...)
	}
	req.StartAt = 0
	req.MaxResults = o.MaxResults

	return req
}

// https://developer.atlassian.com/cloud/jira/platform/rest/v3/api-group-issue-search/#api-rest-api-3-search-jql-post
func (j *Jira) Search(sp SearchProvider, opts ...SearchOpt) (*jiradata.SearchResults, error) {
	return Search(j.UA, j.Endpoint, sp, opts...)
}

type searchConfig struct {
	autoPaginate bool
	expand       []string
}

type SearchOpt func(*searchConfig)

func WithAutoPagination() SearchOpt {
	return func(c *searchConfig) {
		c.autoPaginate = true
	}
}

func WithExpand(fields ...string) SearchOpt {
	return func(c *searchConfig) {
		c.expand = append(c.expand, fields...)
	}
}

// jqlSearchRequest is the request body for the enhanced JQL search endpoint
// (POST /rest/api/3/search/jql). Unlike the deprecated /rest/api/2/search
// endpoint it uses cursor-based pagination via nextPageToken (rather than
// startAt), and expand is passed in the body as a comma-separated string
// rather than as a query parameter.
type jqlSearchRequest struct {
	JQL           string   `json:"jql,omitempty"`
	Fields        []string `json:"fields,omitempty"`
	Expand        string   `json:"expand,omitempty"`
	MaxResults    int      `json:"maxResults,omitempty"`
	NextPageToken string   `json:"nextPageToken,omitempty"`
	FieldsByKeys  bool     `json:"fieldsByKeys,omitempty"`
}

func Search(ua HttpClient, endpoint string, sp SearchProvider, opts ...SearchOpt) (*jiradata.SearchResults, error) {
	c := &searchConfig{}
	for _, opt := range opts {
		opt(c)
	}

	req := sp.ProvideSearchRequest()
	limit := req.MaxResults

	body := &jqlSearchRequest{
		JQL:          req.JQL,
		Fields:       req.Fields,
		FieldsByKeys: req.FieldsByKeys,
		Expand:       strings.Join(c.expand, ","),
		// max page size for the enhanced search endpoint is 5000; 100 keeps
		// the per-request payload reasonable while paginating.
		MaxResults: 100,
	}
	if limit > 0 && limit < body.MaxResults {
		body.MaxResults = limit
	}

	issues := jiradata.Issues{}
	for {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		uri := URLJoin(endpoint, "rest/api/3/search/jql")
		resp, err := ua.Post(uri, "application/json", bytes.NewBuffer(encoded))
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return nil, responseError(resp)
		}

		page := &jiradata.SearchResults{}
		err = json.NewDecoder(resp.Body).Decode(page)
		if err != nil {
			return nil, err
		}
		if !c.autoPaginate {
			return page, nil
		}

		issues = append(issues, page.Issues...)
		// The enhanced search endpoint reports the end of results via isLast
		// (or an empty nextPageToken); it does not return a total count.
		if (limit > 0 && len(issues) >= limit) || page.IsLast || page.NextPageToken == "" {
			page.Issues = issues
			return page, nil
		}
		body.NextPageToken = page.NextPageToken
		if limit > 0 && limit-len(issues) < body.MaxResults {
			body.MaxResults = limit - len(issues)
		}
	}
}
