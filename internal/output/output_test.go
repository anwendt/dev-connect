package output

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestWriteJSONAddsAPIVersion(t *testing.T) {
	var buf bytes.Buffer

	if err := WriteJSON(&buf, Response{Status: "connected", Server: "dev01"}); err != nil {
		t.Fatalf("write JSON: %v", err)
	}

	var got Response
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal JSON: %v", err)
	}

	if got.APIVersion != APIVersion {
		t.Fatalf("APIVersion = %q, want %q", got.APIVersion, APIVersion)
	}
}
