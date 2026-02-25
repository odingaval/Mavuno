package api

import (
	"encoding/json"
	"net/http"
)

// decodeJSONBody decodes a JSON request body into v. Kept simple for tests.
func decodeJSONBody(r *http.Request, v interface{}) error {
	decoder := json.NewDecoder(r.Body)
	return decoder.Decode(v)
}
