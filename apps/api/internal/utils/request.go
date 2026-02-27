package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid request body")
	}
	return nil
}
