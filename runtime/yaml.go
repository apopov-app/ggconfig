package runtime

import (
	"fmt"
	"math"

	"gopkg.in/yaml.v3"
)

// YAML is a parsed YAML configuration stored as a generic map.
// Expected top-level structure: map[section]map[key]value.
type YAML struct {
	root map[string]any
}

func (y *YAML) ensure() {
	if y.root == nil {
		y.root = map[string]any{}
	}
}

func ParseYAML(data []byte) (*YAML, error) {
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}
	if root == nil {
		root = map[string]any{}
	}
	return &YAML{root: root}, nil
}

func (y *YAML) GetString(section string, keys ...string) (string, bool) {
	y.ensure()
	sec, ok := y.root[section].(map[string]any)
	if !ok {
		return "", false
	}
	for _, k := range keys {
		if k == "" {
			continue
		}
		if v, ok := sec[k]; ok {
			if s, ok := v.(string); ok {
				return s, true
			}
		}
	}
	return "", false
}

func (y *YAML) GetInt(section string, keys ...string) (int, bool) {
	y.ensure()
	sec, ok := y.root[section].(map[string]any)
	if !ok {
		return 0, false
	}
	for _, k := range keys {
		if k == "" {
			continue
		}
		v, ok := sec[k]
		if !ok {
			continue
		}
		switch t := v.(type) {
		case int:
			return t, true
		case int64:
			if t > int64(math.MaxInt) || t < int64(math.MinInt) {
				return 0, false
			}
			return int(t), true
		case float64:
			// YAML иногда может распарсить числа как float64 в зависимости от структуры.
			if math.Trunc(t) != t {
				return 0, false
			}
			if t > float64(math.MaxInt) || t < float64(math.MinInt) {
				return 0, false
			}
			return int(t), true
		default:
			// no conversion
		}
	}
	return 0, false
}

// GetSlice retrieves a slice value from YAML for a given section and keys.
// It returns the slice as []any and a boolean indicating success.
// This is a generic method that can be used for any slice type.
func (y *YAML) GetSlice(section string, keys ...string) ([]any, bool) {
	y.ensure()
	sec, ok := y.root[section].(map[string]any)
	if !ok {
		return nil, false
	}
	for _, k := range keys {
		if k == "" {
			continue
		}
		if v, ok := sec[k]; ok {
			if slice, ok := v.([]any); ok {
				return slice, true
			}
		}
	}
	return nil, false
}


