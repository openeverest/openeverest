package api

import (
	"encoding/json"
	"unicode"
)

// MarshalJSON capitalizes Error.Message and marshals it to byte array.
func (e Error) MarshalJSON() ([]byte, error) {
	if e.Message != nil && *e.Message != "" {
		r := []rune(*e.Message)
		r[0] = unicode.ToUpper(r[0])
		*e.Message = string(r)
	}
	return json.Marshal(&struct {
		Message *string `json:"message,omitempty"`
	}{
		Message: e.Message,
	})
}
