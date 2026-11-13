package requestlog

import (
	"testing"
	"time"
)

func BenchmarkEmitDisabled(b *testing.B) {
	reset := func() {
		enabled.Store(false)
		configured.Store(false)
	}
	reset()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Emit(Record{Service: "x", Path: "/p", Outcome: OutcomeOK})
	}
}

func BenchmarkEncodeNdjson50(b *testing.B) {
	now := time.Now().UTC()
	buf := make([]Record, 50)
	for i := range buf {
		buf[i] = Record{
			Service:     "aira-stt",
			Event:       "request",
			Method:      "GET",
			Path:        "/v1/realtime",
			RemoteAddr:  "10.0.0.1:12345",
			Language:    "id",
			Model:       "piper",
			Codec:       "pcm16",
			Code:        1000,
			Outcome:     OutcomeOK,
			StartedAt:   now,
			DurationMs:  15234.5,
			SessionID:   "00000000-0000-0000-0000-000000000001",
			AudioBytes:  9400000,
			Partials:    1400,
			Finals:      12,
			DecodeMsP50: 2.3,
			DecodeMsP95: 24.1,
			RtfMean:     0.024,
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := encodeNdjson(buf); err != nil {
			b.Fatal(err)
		}
	}
}
