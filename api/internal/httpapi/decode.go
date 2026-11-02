package httpapi

import (
	"encoding/json"
	"net/http"
)

// maxBodyBytes bounds request bodies; every JSON payload in this API is small.
const maxBodyBytes = 1 << 20

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "validation", "request body is not valid JSON")
		return false
	}
	return true
}
