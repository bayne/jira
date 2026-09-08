package jira

import (
	"fmt"
	"strings"

	"github.com/go-jira/jira/jiradata"
)

func BuildStructureTree(ua HttpClient, endpoint string, rootKey string, linkTypes, filterProjects []string, maxDepth int) (*jiradata.StructureTree, error) {
	visited := map[string]bool{}
	allowedProjects := make(map[string]bool, len(filterProjects))
	for _, p := range filterProjects {
		allowedProjects[strings.ToUpper(p)] = true
	}
	var nodes []jiradata.StructureNode
	if err := walkStructure(ua, endpoint, rootKey, linkTypes, allowedProjects, maxDepth, 0, nil, true, visited, &nodes); err != nil {
		return nil, err
	}
	return &jiradata.StructureTree{Root: rootKey, Nodes: nodes}, nil
}

func walkStructure(ua HttpClient, endpoint, key string, linkTypes []string, allowedProjects map[string]bool, maxDepth, depth int, continuations []bool, isLast bool, visited map[string]bool, nodes *[]jiradata.StructureNode) error {
	if visited[key] {
		return nil
	}
	visited[key] = true

	issue, err := GetIssue(ua, endpoint, key, nil)
	if err != nil {
		return fmt.Errorf("%s: %w", key, err)
	}

	node := jiradata.StructureNode{
		Prefix: structurePrefix(depth, continuations, isLast),
		Key:    key,
	}
	extractStructureFields(issue, &node)
	*nodes = append(*nodes, node)

	if maxDepth > 0 && depth >= maxDepth {
		return nil
	}

	children := structureChildren(issue, linkTypes, allowedProjects)
	for i, childKey := range children {
		childIsLast := i == len(children)-1
		var childConts []bool
		if depth > 0 {
			childConts = make([]bool, len(continuations)+1)
			copy(childConts, continuations)
			childConts[len(continuations)] = !isLast
		}
		if err := walkStructure(ua, endpoint, childKey, linkTypes, allowedProjects, maxDepth, depth+1, childConts, childIsLast, visited, nodes); err != nil {
			return err
		}
	}
	return nil
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

func structureChildren(issue *jiradata.Issue, linkTypes []string, allowedProjects map[string]bool) []string {
	if issue == nil || issue.Fields == nil {
		return nil
	}

	wantLinks := make(map[string]bool)
	followSubtasks := false
	for _, lt := range linkTypes {
		lower := strings.ToLower(lt)
		if lower == "subtask" || lower == "subtasks" {
			followSubtasks = true
		} else {
			wantLinks[lower] = true
		}
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

func projectAllowed(key string, allowed map[string]bool) bool {
	if len(allowed) == 0 {
		return true
	}
	if i := strings.IndexByte(key, '-'); i > 0 {
		return allowed[strings.ToUpper(key[:i])]
	}
	return false
}
