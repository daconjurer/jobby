package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// parseJSONObject reads a JSON object from an inline flag value or a file path.
// When both flagValue and filePath are empty, it returns nil.
func parseJSONObject(flagValue, filePath, fieldName string) (map[string]any, error) {
	if flagValue != "" && filePath != "" {
		return nil, fmt.Errorf("%s: use only one of --%s or --%s-file", fieldName, fieldName, fieldName)
	}

	var raw []byte
	switch {
	case filePath != "":
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read %s file: %w", fieldName, err)
		}
		raw = data
	case flagValue != "":
		raw = []byte(flagValue)
	default:
		return nil, nil
	}

	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("invalid %s JSON: %w", fieldName, err)
	}
	if obj == nil {
		return nil, errors.New("invalid " + fieldName + " JSON: must be a JSON object")
	}
	return obj, nil
}
