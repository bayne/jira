package jira

import (
	"fmt"
	"strings"

	"github.com/go-jira/jira/jiradata"
)

var structureFields = jiradata.Fields{
	"summary", "status", "issuetype", "assignee", "priority",
	"issuelinks", "subtasks",
}

const structureBatchSize = 100

func BuildStructureTree(ua HttpClient, endpoint string, rootKey string, linkTypes, filterProjects []string, maxDepth int) (*jiradata.StructureTree, error) {
	allowedProjects := make(map[string]bool, len(filterProjects))
	for _, p := range filterProjects {
		allowedProjects[strings.ToUpper(p)] = true
	}

	wantLinks, followSubtasks := parseLinkTypes(linkTypes)

	issueMap := map[string]*jiradata.Issue{}
	childrenMap := map[string][]string{}
	visited := map[string]bool{}

	frontier := []string{rootKey}
	depth := 0

	for len(frontier) > 0 {
		var toFetch []string
		for _, key := range frontier {
			if !visited[key] {
				toFetch = append(toFetch, key)
				visited[key] = true
			}
		}
		if len(toFetch) == 0 {
			break
		}

		issues, err := batchFetchIssues(ua, endpoint, toFetch)
		if err != nil {
			return nil, err
		}

		var nextFrontier []string
		for _, issue := range issues {
			key := issue.Key
			issueMap[key] = issue
			children := issueChildren(issue, wantLinks, followSubtasks, allowedProjects)
			childrenMap[key] = children
			if maxDepth == 0 || depth < maxDepth {
				for _, childKey := range children {
					if !visited[childKey] {
						nextFrontier = append(nextFrontier, childKey)
					}
				}
			}
		}

		frontier = nextFrontier
		depth++
	}

	var nodes []jiradata.StructureNode
	rendered := map[string]bool{}
	renderTree(issueMap, childrenMap, rootKey, 0, nil, true, rendered, &nodes)

	return &jiradata.StructureTree{Root: rootKey, Nodes: nodes}, nil
}

func batchFetchIssues(ua HttpClient, endpoint string, keys []string) (map[string]*jiradata.Issue, error) {
	result := make(map[string]*jiradata.Issue, len(keys))
	for i := 0; i < len(keys); i += structureBatchSize {
		end := i + structureBatchSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]

		sp := &structureSearchProvider{keys: batch}
		page, err := Search(ua, endpoint, sp, WithAutoPagination())
		if err != nil {
			return nil, fmt.Errorf("batch fetch: %w", err)
		}
		for _, issue := range page.Issues {
			result[issue.Key] = issue
		}
	}
	return result, nil
}

type structureSearchProvider struct {
	keys []string
}

func (p *structureSearchProvider) ProvideSearchRequest() *jiradata.SearchRequest {
	return &jiradata.SearchRequest{
		JQL:        fmt.Sprintf("key in (%s)", strings.Join(p.keys, ",")),
		Fields:     structureFields,
		MaxResults: len(p.keys),
	}
}

func renderTree(issueMap map[string]*jiradata.Issue, childrenMap map[string][]string, key string, depth int, continuations []bool, isLast bool, visited map[string]bool, nodes *[]jiradata.StructureNode) {
	if visited[key] {
		return
	}
	visited[key] = true

	node := jiradata.StructureNode{
		Prefix: structurePrefix(depth, continuations, isLast),
		Key:    key,
	}
	if issue, ok := issueMap[key]; ok {
		extractStructureFields(issue, &node)
	}
	*nodes = append(*nodes, node)

	children := childrenMap[key]
	for i, childKey := range children {
		childIsLast := i == len(children)-1
		var childConts []bool
		if depth > 0 {
			childConts = make([]bool, len(continuations)+1)
			copy(childConts, continuations)
			childConts[len(continuations)] = !isLast
		}
		renderTree(issueMap, childrenMap, childKey, depth+1, childConts, childIsLast, visited, nodes)
	}
}

func parseLinkTypes(linkTypes []string) (wantLinks map[string]bool, followSubtasks bool) {
	wantLinks = make(map[string]bool)
	for _, lt := range linkTypes {
		lower := strings.ToLower(lt)
		if lower == "subtask" || lower == "subtasks" {
			followSubtasks = true
		} else {
			wantLinks[lower] = true
		}
	}
	return
}

func issueChildren(issue *jiradata.Issue, wantLinks map[string]bool, followSubtasks bool, allowedProjects map[string]bool) []string {
	if issue == nil || issue.Fields == nil {
		return nil
	}

	var children []string

	if links, ok := issue.Fields["issuelinks"].([]interface{}); ok {
		for _, raw := range links {
			link, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			lt, ok := link["type"].(map[string]interface{})
			if !ok {
				continue
			}
			name, _ := lt["name"].(string)
			if !wantLinks[strings.ToLower(name)] {
				continue
			}
			if outward, ok := link["outwardIssue"].(map[string]interface{}); ok {
				if key, ok := outward["key"].(string); ok && key != "" && projectAllowed(key, allowedProjects) {
					children = append(children, key)
				}
			}
		}
	}

	if followSubtasks {
		if subtasks, ok := issue.Fields["subtasks"].([]interface{}); ok {
			for _, raw := range subtasks {
				if st, ok := raw.(map[string]interface{}); ok {
					if key, ok := st["key"].(string); ok && key != "" && projectAllowed(key, allowedProjects) {
						children = append(children, key)
					}
				}
			}
		}
	}

	return children
}

func structurePrefix(depth int, continuations []bool, isLast bool) string {
	if depth == 0 {
		return ""
	}
	var b strings.Builder
	for _, cont := range continuations {
		if cont {
			b.WriteString("│  ")
		} else {
			b.WriteString("   ")
		}
	}
	if isLast {
		b.WriteString("└─ ")
	} else {
		b.WriteString("├─ ")
	}
	return b.String()
}

func extractStructureFields(issue *jiradata.Issue, node *jiradata.StructureNode) {
	if issue == nil || issue.Fields == nil {
		return
	}
	node.Summary, _ = issue.Fields["summary"].(string)
	if m, ok := issue.Fields["status"].(map[string]interface{}); ok {
		node.Status, _ = m["name"].(string)
	}
	if m, ok := issue.Fields["issuetype"].(map[string]interface{}); ok {
		node.Type, _ = m["name"].(string)
	}
	if m, ok := issue.Fields["assignee"].(map[string]interface{}); ok {
		node.Assignee, _ = m["displayName"].(string)
	}
	if m, ok := issue.Fields["priority"].(map[string]interface{}); ok {
		node.Priority, _ = m["name"].(string)
	}
}

func projectAllowed(key string, allowed map[string]bool) bool {
	if len(allowed) == 0 {
		return true
	}
	if i := strings.IndexByte(key, '-'); i > 0 {
		return allowed[strings.ToUpper(key[:i])]
	}
	return false
}
