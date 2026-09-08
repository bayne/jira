package jiradata

type StructureNode struct {
	Prefix   string `json:"prefix" yaml:"prefix"`
	Key      string `json:"key" yaml:"key"`
	Summary  string `json:"summary" yaml:"summary"`
	Status   string `json:"status" yaml:"status"`
	Type     string `json:"type" yaml:"type"`
	Assignee string `json:"assignee" yaml:"assignee"`
	Priority string `json:"priority" yaml:"priority"`
}

type StructureTree struct {
	Root  string          `json:"root" yaml:"root"`
	Nodes []StructureNode `json:"nodes" yaml:"nodes"`
}
