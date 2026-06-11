package jira

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/go-jira/jira/jiradata"
)

// Sprints fetches all sprints for a board, following pagination. An empty
// states list returns sprints in any state.
func Sprints(ua HttpClient, endpoint string, board string, states []string) (*jiradata.SprintResults, error) {
	results := &jiradata.SprintResults{IsLast: true}
	startAt := 0
	for {
		uri := URLJoin(endpoint, "rest/agile/1.0/board", board, "sprint")
		uri += "?startAt=" + strconv.Itoa(startAt)
		if len(states) > 0 {
			uri += "&state=" + strings.Join(states, ",")
		}
		resp, err := ua.GetJSON(uri)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			err := responseError(resp)
			resp.Body.Close()
			return nil, err
		}

		page := &jiradata.SprintResults{}
		err = json.NewDecoder(resp.Body).Decode(page)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		results.MaxResults = page.MaxResults
		results.Values = append(results.Values, page.Values...)
		if page.IsLast || len(page.Values) == 0 {
			break
		}
		startAt += len(page.Values)
	}
	return results, nil
}
