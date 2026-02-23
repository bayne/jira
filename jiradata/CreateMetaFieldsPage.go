package jiradata

// CreateMetaField represents a single field entry returned by
// GET /rest/api/2/issue/createmeta/{projectIdOrKey}/issuetypes/{issueTypeId}
type CreateMetaField struct {
	FieldId         string        `json:"fieldId,omitempty" yaml:"fieldId,omitempty"`
	Name            string        `json:"name,omitempty" yaml:"name,omitempty"`
	Required        bool          `json:"required,omitempty" yaml:"required,omitempty"`
	Schema          *JSONType     `json:"schema,omitempty" yaml:"schema,omitempty"`
	AutoCompleteURL string        `json:"autoCompleteUrl,omitempty" yaml:"autoCompleteUrl,omitempty"`
	HasDefaultValue bool          `json:"hasDefaultValue,omitempty" yaml:"hasDefaultValue,omitempty"`
	DefaultValue    interface{}   `json:"defaultValue,omitempty" yaml:"defaultValue,omitempty"`
	Operations      Operations    `json:"operations,omitempty" yaml:"operations,omitempty"`
	AllowedValues   AllowedValues `json:"allowedValues,omitempty" yaml:"allowedValues,omitempty"`
}

// CreateMetaFieldsPage is the paginated response from
// GET /rest/api/2/issue/createmeta/{projectIdOrKey}/issuetypes/{issueTypeId}
type CreateMetaFieldsPage struct {
	MaxResults int                `json:"maxResults,omitempty" yaml:"maxResults,omitempty"`
	StartAt    int                `json:"startAt,omitempty" yaml:"startAt,omitempty"`
	Total      int                `json:"total,omitempty" yaml:"total,omitempty"`
	Values     []*CreateMetaField `json:"values,omitempty" yaml:"values,omitempty"`
}
