package performance_test

import (
	"io"
	"testing"

	"github.com/anwendt/dev-connect/internal/output"
	"github.com/anwendt/dev-connect/internal/port"
)

func BenchmarkJSONStatusOutput(b *testing.B) {
	response := output.Response{
		Status:    "connected",
		Server:    "dev01",
		SessionID: "session-1",
		LocalPort: 55221,
	}

	b.ReportAllocs()
	for b.Loop() {
		if err := output.WriteJSON(io.Discard, response); err != nil {
			b.Fatalf("write JSON: %v", err)
		}
	}
}

func BenchmarkLoopbackPortAllocation(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		allocation, err := port.AllocateLoopback()
		if err != nil {
			b.Fatalf("allocate loopback port: %v", err)
		}
		if allocation.Port <= 0 {
			b.Fatalf("invalid port allocation: %#v", allocation)
		}
	}
}
