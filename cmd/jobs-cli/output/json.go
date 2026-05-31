package output

import (
	"encoding/json"
	"io"
)

// WriteJSON encodes v as JSON to w (one object per line, matching HTTP handlers).
func WriteJSON(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}
