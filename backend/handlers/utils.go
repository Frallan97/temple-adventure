package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

func DecodeJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}

// DecodeJSONStrict rejects fields the target type doesn't declare.
//
// Used for story specs, where silently dropping an unrecognised field is
// expensive: a misspelled key looks like it was accepted, and a field the
// running binary predates (because the spec format moved on) vanishes without
// a word. Both produce a story that validates clean and behaves wrongly.
//
// Not applied to gameplay requests, where tolerating extra fields keeps older
// and newer clients interoperable.
func DecodeJSONStrict(r *http.Request, v interface{}) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
