package main

import (
	"encoding/json"
	"strings"
)

// MarshalJSON keeps the private Git relay useful to ChatGPT without making
// runner filesystem layout part of the model-facing protocol. The request
// already identifies the project by its opaque Workbench project reference, so
// absolute runner paths returned by the local MCP are redundant metadata.
func (out privateControlOutbox) MarshalJSON() ([]byte, error) {
	type wire privateControlOutbox
	clean := wire(out)
	clean.Result = redactPrivateControlHostPaths(out.Result)
	return json.Marshal(clean)
}

func redactPrivateControlHostPaths(result map[string]any) map[string]any {
	if result == nil {
		return nil
	}
	clean, _ := redactPrivateControlValue(result).(map[string]any)
	return clean
}

func redactPrivateControlValue(value any) any {
	switch v := value.(type) {
	case map[string]any:
		clean := make(map[string]any, len(v))
		for key, child := range v {
			lower := strings.ToLower(strings.TrimSpace(key))
			if lower == "project_path" {
				continue
			}
			if lower == "project" {
				if text, ok := child.(string); ok && looksLikeAbsoluteHostPath(text) {
					continue
				}
			}
			clean[key] = redactPrivateControlValue(child)
		}
		return clean
	case []any:
		clean := make([]any, len(v))
		for i := range v {
			clean[i] = redactPrivateControlValue(v[i])
		}
		return clean
	default:
		return value
	}
}

func looksLikeAbsoluteHostPath(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, `\\`) {
		return true
	}
	if len(value) >= 3 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':' && (value[2] == '/' || value[2] == '\\') {
		return true
	}
	return false
}
