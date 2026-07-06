package output

import (
	"encoding/json"
	"io"
)

// APIVersion is the current public JSON output API version.
const APIVersion = "v1"

// Response is the common versioned command response shape.
type Response struct {
	APIVersion string `json:"apiVersion"`
	Status     string `json:"status,omitempty"`
	Server     string `json:"server,omitempty"`
	SessionID  string `json:"sessionId,omitempty"`
	LocalPort  int    `json:"localPort,omitempty"`
}

// WriteJSON writes a versioned command response as JSON.
func WriteJSON(w io.Writer, response Response) error {
	if response.APIVersion == "" {
		response.APIVersion = APIVersion
	}

	encoder := json.NewEncoder(w)
	return encoder.Encode(response)
}
