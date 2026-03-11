package jiracli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// FieldMapping represents a single field's mapping between API ID and user-friendly names.
type FieldMapping struct {
	Key     string                    `json:"key" yaml:"key"`
	Label   string                    `json:"label" yaml:"label"`
	Options map[string]*OptionMapping `json:"options,omitempty" yaml:"options,omitempty"`
}

// OptionMapping represents a single option value's mapping.
type OptionMapping struct {
	Key   string `json:"key" yaml:"key"`
	Label string `json:"label" yaml:"label"`
}

// FieldMap holds the full mapping of all fields.
type FieldMap struct {
	Fields map[string]*FieldMapping `json:"fields" yaml:"fields"`
}

// ToKey converts a human-readable label to a snake_case key suitable for use in templates/editors.
func ToKey(label string) string {
	// Replace non-alphanumeric characters with underscores
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	key := re.ReplaceAllString(label, "_")
	// Convert to lowercase
	key = strings.ToLower(key)
	// Convert camelCase to snake_case
	var result []rune
	for i, r := range key {
		if unicode.IsUpper(r) && i > 0 {
			prev := rune(key[i-1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) {
				result = append(result, '_')
			}
		}
		result = append(result, unicode.ToLower(r))
	}
	key = string(result)
	// Trim leading/trailing underscores
	key = strings.Trim(key, "_")
	// Collapse multiple underscores
	re2 := regexp.MustCompile(`_+`)
	key = re2.ReplaceAllString(key, "_")
	return key
}

// LoadFieldMap loads the field mapping from .jira.d/fields.json, searching
// up the directory hierarchy.
func LoadFieldMap() (*FieldMap, error) {
	file, err := findClosestParentPath(filepath.Join(".jira.d", "fields.json"))
	if err != nil {
		return nil, fmt.Errorf("field mapping not found, run 'jira fields-map' to generate it: %w", err)
	}
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	fm := &FieldMap{}
	if err := json.Unmarshal(data, fm); err != nil {
		return nil, err
	}
	return fm, nil
}

// SaveFieldMap writes the field mapping to .jira.d/fields.json.
func SaveFieldMap(fm *FieldMap) error {
	dir := ".jira.d"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(fm, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(dir, "fields.json"), data, 0644)
}

// FieldIDForKey returns the API field ID for a given friendly key.
// Returns the key itself if no mapping is found (passthrough for standard fields).
func (fm *FieldMap) FieldIDForKey(key string) string {
	for id, mapping := range fm.Fields {
		if mapping.Key == key {
			return id
		}
	}
	return key
}

// FieldLabelForID returns the human-readable label for a field ID.
func (fm *FieldMap) FieldLabelForID(id string) string {
	if mapping, ok := fm.Fields[id]; ok {
		return mapping.Label
	}
	return id
}

// FieldKeyForID returns the friendly key for a field ID.
func (fm *FieldMap) FieldKeyForID(id string) string {
	if mapping, ok := fm.Fields[id]; ok {
		return mapping.Key
	}
	return id
}

// OptionIDForKey returns the option API value for a given field ID and friendly option key.
func (fm *FieldMap) OptionIDForKey(fieldID, optionKey string) string {
	if mapping, ok := fm.Fields[fieldID]; ok && mapping.Options != nil {
		for id, opt := range mapping.Options {
			if opt.Key == optionKey {
				return id
			}
		}
	}
	return optionKey
}

// OptionLabelForID returns the human-readable label for a field's option value.
func (fm *FieldMap) OptionLabelForID(fieldID, optionID string) string {
	if mapping, ok := fm.Fields[fieldID]; ok && mapping.Options != nil {
		if opt, ok := mapping.Options[optionID]; ok {
			return opt.Label
		}
	}
	return optionID
}

// TransformFieldKeys takes a map of field key->value (using friendly keys) and
// returns a new map with API field IDs as keys. Option values are also reverse-mapped.
func (fm *FieldMap) TransformFieldKeys(fields map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(fields))
	for key, value := range fields {
		apiID := fm.FieldIDForKey(key)
		result[apiID] = fm.transformOptionValues(apiID, value)
	}
	return result
}

// transformOptionValues converts friendly option keys back to API option IDs
// for a specific field.
func (fm *FieldMap) transformOptionValues(fieldID string, value interface{}) interface{} {
	mapping, ok := fm.Fields[fieldID]
	if !ok || mapping.Options == nil {
		return value
	}

	switch v := value.(type) {
	case map[string]interface{}:
		// Handle objects like { "name": "friendly_key" } or { "id": "friendly_key" }
		if name, ok := v["name"].(string); ok {
			if optID := fm.OptionIDForKey(fieldID, name); optID != name {
				v["name"] = optID
			}
		}
		if id, ok := v["id"].(string); ok {
			if optID := fm.OptionIDForKey(fieldID, id); optID != id {
				v["id"] = optID
			}
		}
		if val, ok := v["value"].(string); ok {
			if optID := fm.OptionIDForKey(fieldID, val); optID != val {
				v["value"] = optID
			}
		}
		return v
	case string:
		return fm.OptionIDForKey(fieldID, v)
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = fm.transformOptionValues(fieldID, item)
		}
		return result
	}
	return value
}
