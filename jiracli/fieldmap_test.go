package jiracli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Sprint", "sprint"},
		{"Story Points", "story_points"},
		{"Epic Link", "epic_link"},
		{"% Complete", "complete"},
		{"Affects Builds", "affects_builds"},
		{"T-Shirt Size", "t_shirt_size"},
		{"Target start", "target_start"},
		{"Fix Version/s", "fix_version_s"},
		{"", ""},
		{"simple", "simple"},
		{"Already_Snake", "already_snake"},
		{"multiple   spaces", "multiple_spaces"},
		{"ALLCAPS", "allcaps"},
		{"leading---dashes", "leading_dashes"},
		{"trailing!!!", "trailing"},
		{"mixed123numbers", "mixed123numbers"},
		{"with (parens)", "with_parens"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, ToKey(tt.input))
		})
	}
}

func testFieldMap() *FieldMap {
	return &FieldMap{
		Fields: map[string]*FieldMapping{
			"summary": {
				Key:   "summary",
				Label: "Summary",
			},
			"customfield_10105": {
				Key:   "sprint",
				Label: "Sprint",
			},
			"customfield_10106": {
				Key:   "story_points",
				Label: "Story Points",
			},
			"customfield_10200": {
				Key:   "severity",
				Label: "Severity",
				Options: map[string]*OptionMapping{
					"10001": {Key: "critical", Label: "Critical"},
					"10002": {Key: "major", Label: "Major"},
					"10003": {Key: "minor", Label: "Minor"},
				},
			},
			"priority": {
				Key:   "priority",
				Label: "Priority",
				Options: map[string]*OptionMapping{
					"1": {Key: "highest", Label: "Highest"},
					"2": {Key: "high", Label: "High"},
					"3": {Key: "medium", Label: "Medium"},
					"4": {Key: "low", Label: "Low"},
				},
			},
		},
	}
}

func TestFieldIDForKey(t *testing.T) {
	fm := testFieldMap()

	tests := []struct {
		key      string
		expected string
	}{
		{"sprint", "customfield_10105"},
		{"story_points", "customfield_10106"},
		{"summary", "summary"},
		{"severity", "customfield_10200"},
		// Unknown key passes through
		{"unknown_field", "unknown_field"},
		// API ID that isn't a friendly key passes through
		{"customfield_99999", "customfield_99999"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.expected, fm.FieldIDForKey(tt.key))
		})
	}
}

func TestFieldLabelForID(t *testing.T) {
	fm := testFieldMap()

	tests := []struct {
		id       string
		expected string
	}{
		{"customfield_10105", "Sprint"},
		{"customfield_10106", "Story Points"},
		{"summary", "Summary"},
		// Unknown ID returns itself
		{"customfield_99999", "customfield_99999"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			assert.Equal(t, tt.expected, fm.FieldLabelForID(tt.id))
		})
	}
}

func TestFieldKeyForID(t *testing.T) {
	fm := testFieldMap()

	tests := []struct {
		id       string
		expected string
	}{
		{"customfield_10105", "sprint"},
		{"customfield_10106", "story_points"},
		{"summary", "summary"},
		// Unknown ID returns itself
		{"customfield_99999", "customfield_99999"},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			assert.Equal(t, tt.expected, fm.FieldKeyForID(tt.id))
		})
	}
}

func TestOptionIDForKey(t *testing.T) {
	fm := testFieldMap()

	tests := []struct {
		fieldID    string
		optionKey  string
		expected   string
	}{
		{"customfield_10200", "critical", "10001"},
		{"customfield_10200", "major", "10002"},
		{"customfield_10200", "minor", "10003"},
		{"priority", "highest", "1"},
		{"priority", "low", "4"},
		// Unknown option passes through
		{"customfield_10200", "unknown_option", "unknown_option"},
		// Field without options passes through
		{"customfield_10105", "anything", "anything"},
		// Unknown field passes through
		{"unknown_field", "anything", "anything"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldID+"/"+tt.optionKey, func(t *testing.T) {
			assert.Equal(t, tt.expected, fm.OptionIDForKey(tt.fieldID, tt.optionKey))
		})
	}
}

func TestOptionLabelForID(t *testing.T) {
	fm := testFieldMap()

	tests := []struct {
		fieldID   string
		optionID  string
		expected  string
	}{
		{"customfield_10200", "10001", "Critical"},
		{"customfield_10200", "10002", "Major"},
		{"priority", "1", "Highest"},
		// Unknown option ID passes through
		{"customfield_10200", "99999", "99999"},
		// Field without options passes through
		{"customfield_10105", "10001", "10001"},
	}

	for _, tt := range tests {
		t.Run(tt.fieldID+"/"+tt.optionID, func(t *testing.T) {
			assert.Equal(t, tt.expected, fm.OptionLabelForID(tt.fieldID, tt.optionID))
		})
	}
}

func TestTransformFieldKeys_BasicMapping(t *testing.T) {
	fm := testFieldMap()

	input := map[string]interface{}{
		"sprint":       42,
		"story_points": 5,
		"summary":      "Test issue",
	}

	result := fm.TransformFieldKeys(input)

	assert.Equal(t, 42, result["customfield_10105"])
	assert.Equal(t, 5, result["customfield_10106"])
	assert.Equal(t, "Test issue", result["summary"])
	// Friendly keys should not appear in output
	_, hasSprint := result["sprint"]
	assert.False(t, hasSprint, "friendly key 'sprint' should not be in result")
}

func TestTransformFieldKeys_OptionValueInMap(t *testing.T) {
	fm := testFieldMap()

	input := map[string]interface{}{
		"severity": map[string]interface{}{
			"id": "critical",
		},
	}

	result := fm.TransformFieldKeys(input)

	severity := result["customfield_10200"].(map[string]interface{})
	assert.Equal(t, "10001", severity["id"])
}

func TestTransformFieldKeys_OptionNameInMap(t *testing.T) {
	fm := testFieldMap()

	input := map[string]interface{}{
		"priority": map[string]interface{}{
			"name": "highest",
		},
	}

	result := fm.TransformFieldKeys(input)

	priority := result["priority"].(map[string]interface{})
	assert.Equal(t, "1", priority["name"])
}

func TestTransformFieldKeys_OptionValueField(t *testing.T) {
	fm := testFieldMap()

	input := map[string]interface{}{
		"severity": map[string]interface{}{
			"value": "major",
		},
	}

	result := fm.TransformFieldKeys(input)

	severity := result["customfield_10200"].(map[string]interface{})
	assert.Equal(t, "10002", severity["value"])
}

func TestTransformFieldKeys_StringOptionValue(t *testing.T) {
	fm := testFieldMap()

	input := map[string]interface{}{
		"severity": "minor",
	}

	result := fm.TransformFieldKeys(input)
	assert.Equal(t, "10003", result["customfield_10200"])
}

func TestTransformFieldKeys_ArrayOptionValues(t *testing.T) {
	fm := testFieldMap()

	input := map[string]interface{}{
		"severity": []interface{}{
			map[string]interface{}{"id": "critical"},
			map[string]interface{}{"id": "major"},
		},
	}

	result := fm.TransformFieldKeys(input)

	items := result["customfield_10200"].([]interface{})
	assert.Len(t, items, 2)
	assert.Equal(t, "10001", items[0].(map[string]interface{})["id"])
	assert.Equal(t, "10002", items[1].(map[string]interface{})["id"])
}

func TestTransformFieldKeys_NoMappingPassthrough(t *testing.T) {
	fm := testFieldMap()

	input := map[string]interface{}{
		"unknown_field": "some value",
	}

	result := fm.TransformFieldKeys(input)
	assert.Equal(t, "some value", result["unknown_field"])
}

func TestTransformFieldKeys_FieldWithoutOptionsNoTransform(t *testing.T) {
	fm := testFieldMap()

	input := map[string]interface{}{
		"sprint": map[string]interface{}{
			"name": "Sprint 42",
		},
	}

	result := fm.TransformFieldKeys(input)

	sprint := result["customfield_10105"].(map[string]interface{})
	assert.Equal(t, "Sprint 42", sprint["name"])
}

func TestTransformFieldKeys_UnknownOptionPassthrough(t *testing.T) {
	fm := testFieldMap()

	input := map[string]interface{}{
		"severity": map[string]interface{}{
			"name": "not_a_real_option",
		},
	}

	result := fm.TransformFieldKeys(input)

	severity := result["customfield_10200"].(map[string]interface{})
	assert.Equal(t, "not_a_real_option", severity["name"])
}

func TestTransformFieldKeys_NumericValuePassthrough(t *testing.T) {
	fm := testFieldMap()

	input := map[string]interface{}{
		"severity": 42,
	}

	result := fm.TransformFieldKeys(input)
	assert.Equal(t, 42, result["customfield_10200"])
}

func TestTemplateFunctions_FieldLabel(t *testing.T) {
	cachedFieldMap = testFieldMap()
	defer func() { cachedFieldMap = nil }()

	tmpl, err := TemplateProcessor().Parse(`{{ "customfield_10105" | fieldLabel }}`)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Sprint", buf.String())
}

func TestTemplateFunctions_FieldKey(t *testing.T) {
	cachedFieldMap = testFieldMap()
	defer func() { cachedFieldMap = nil }()

	tmpl, err := TemplateProcessor().Parse(`{{ "customfield_10106" | fieldKey }}`)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	assert.NoError(t, err)
	assert.Equal(t, "story_points", buf.String())
}

func TestTemplateFunctions_FieldID(t *testing.T) {
	cachedFieldMap = testFieldMap()
	defer func() { cachedFieldMap = nil }()

	tmpl, err := TemplateProcessor().Parse(`{{ "sprint" | fieldID }}`)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	assert.NoError(t, err)
	assert.Equal(t, "customfield_10105", buf.String())
}

func TestTemplateFunctions_OptionLabel(t *testing.T) {
	cachedFieldMap = testFieldMap()
	defer func() { cachedFieldMap = nil }()

	tmpl, err := TemplateProcessor().Parse(`{{ optionLabel "customfield_10200" "10001" }}`)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Critical", buf.String())
}

func TestTemplateFunctions_OptionKey(t *testing.T) {
	cachedFieldMap = testFieldMap()
	defer func() { cachedFieldMap = nil }()

	tmpl, err := TemplateProcessor().Parse(`{{ optionKey "priority" "2" }}`)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	assert.NoError(t, err)
	assert.Equal(t, "high", buf.String())
}

func TestTemplateFunctions_FallbackWhenNoMapping(t *testing.T) {
	cachedFieldMap = nil

	tmpl, err := TemplateProcessor().Parse(`{{ "customfield_10105" | fieldLabel }}`)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	assert.NoError(t, err)
	assert.Equal(t, "customfield_10105", buf.String())
}

func TestTemplateFunctions_UnknownFieldFallback(t *testing.T) {
	cachedFieldMap = testFieldMap()
	defer func() { cachedFieldMap = nil }()

	tmpl, err := TemplateProcessor().Parse(`{{ "customfield_99999" | fieldLabel }}`)
	assert.NoError(t, err)

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, nil)
	assert.NoError(t, err)
	assert.Equal(t, "customfield_99999", buf.String())
}
