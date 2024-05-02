package jira

import (
	"encoding/json"
	"github.com/go-jira/jira/jiradata"
)

func Sprints(ua HttpClient, endpoint string, board string, states []string) (*jiradata.SprintResults, error) {
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
}
