package jira

import (
	"encoding/json"
	"github.com/go-jira/jira/jiradata"
)

func Sprints(ua HttpClient, endpoint string, board string, states []string) (*jiradata.SprintResults, error) {
	//c := &searchConfig{}
	//for _, opt := range opts {
	//	opt(c)
	//}
	//
	//req := sp.ProvideSearchRequest()
	//limit := req.MaxResults
	//if limit == 0 {
	//	// max page size is 100
	//	req.MaxResults = 100
	//}

	//issues := jiradata.Issues{}
	//for {
	//encoded, err := json.Marshal(req)
	//if err != nil {
	//	return nil, err
	//}
	uri := URLJoin(endpoint, "rest/agile/1.0/board", board, "sprint")
	uri += "?state=" + states[0]
	for state := range states[1:] {
		uri += "&state=" + states[state]
	}
	resp, err := ua.GetJSON(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, responseError(resp)
	}

	page := &jiradata.SprintResults{}
	err = json.NewDecoder(resp.Body).Decode(page)
	if err != nil {
		return nil, err
	}
	return page, nil
	//if !c.autoPaginate {
	//	return page, nil
	//}

	//issues = append(issues, page.Issues...)
	//// if we are done paginating just force all issues onto current
	//// response and return
	//if (limit > 0 && len(issues) >= limit) || len(issues) >= page.Total {
	//	page.Issues = issues
	//	return page, nil
	//}
	//req.StartAt = len(issues)
	//if len(issues)+req.MaxResults > limit {
	//	req.MaxResults = limit - len(issues)
	//}
	//}
}
