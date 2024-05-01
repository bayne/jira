package jiradata

type SprintResults struct {
	MaxResults int      `json:"maxResults,omitempty" yaml:"maxResults,omitempty"`
	StartAt    int      `json:"startAt,omitempty" yaml:"startAt,omitempty"`
	IsLast     bool     `json:"isLast,omitempty" yaml:"isLast,omitempty"`
	Values     []Sprint `json:"values,omitempty" yaml:"values,omitempty"`
}

type Sprint struct {
	Id            int    `json:"id,omitempty" yaml:"id,omitempty"`
	Name          string `json:"name,omitempty" yaml:"name,omitempty"`
	Self          string `json:"self,omitempty" yaml:"self,omitempty"`
	State         string `json:"state,omitempty" yaml:"state,omitempty"`
	StartDate     string `json:"startDate,omitempty" yaml:"startDate,omitempty"`
	EndDate       string `json:"endDate,omitempty" yaml:"endDate,omitempty"`
	CompleteDate  string `json:"completeDate,omitempty" yaml:"completeDate,omitempty"`
	ActivatedDate string `json:"activatedDate,omitempty" yaml:"activatedDate,omitempty"`
	OriginBoardId int    `json:"originBoardId,omitempty" yaml:"originBoardId,omitempty"`
	Goal          string `json:"goal,omitempty" yaml:"goal,omitempty"`
	Synced        bool   `json:"synced,omitempty" yaml:"synced,omitempty"`
}
