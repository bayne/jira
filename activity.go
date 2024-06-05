package jira

import (
	"encoding/xml"
	"github.com/coryb/oreo"
	"github.com/go-jira/jira/jiradata"
	"net/url"
	"strconv"
)

func Activity(ua HttpClient, endpoint string, maxResults int, streams []string, issues string, providers string) (*jiradata.Feed, error) {
	uri, err := url.Parse(URLJoin(endpoint, "activity"))
	if err != nil {
		return nil, err
	}
	params := url.Values{}
	params.Set("maxResults", strconv.Itoa(maxResults))
	for _, stream := range streams {
		params.Add("streams", stream)
	}
	params.Set("issues", issues)
	params.Set("providers", providers)
	uri.RawQuery = params.Encode()

	resp, err := ua.Do(oreo.RequestBuilder(uri).WithHeader("Accept", "application/xml").Build())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, responseError(resp)
	}

	page := &jiradata.Feed{}
	err = xml.NewDecoder(resp.Body).Decode(page)
	if err != nil {
		return nil, err
	}
	return page, nil
}
