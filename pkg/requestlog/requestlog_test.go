package requestlog

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func resetForTest(t *testing.T) {
	t.Helper()
	if enabled.Swap(false) {
		close(done)
		wg.Wait()
	}
	configured.Store(false)
	cfg = Config{}
	ch = nil
	done = nil
	client = nil
	dropped.Store(0)
	flushed.Store(0)
	flushErr.Store(0)
}

func TestEmitDisabledIsNoop(t *testing.T) {
	resetForTest(t)
	Emit(Record{Service: "x", Path: "/p", Outcome: OutcomeOK})
	if _, d, _ := Stats(); d != 0 {
		t.Fatalf("dropped = %d, want 0", d)
	}
}

func TestConfigureBadURLDisables(t *testing.T) {
	resetForTest(t)
	if err := Configure(Config{Enabled: true, URL: "http://[bad-url", FlushInterval: time.Millisecond, BatchSize: 1, Timeout: time.Second}); err == nil {
		t.Fatal("expected error for bad URL")
	}
	emitted, dropped, _ := Stats()
	if emitted != 0 || dropped != 0 {
		t.Fatalf("emitted/dropped = %d/%d, want 0/0", emitted, dropped)
	}
}

func TestFlushDeliversNdjson(t *testing.T) {
	resetForTest(t)

	var hits, lines atomic.Int64
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err == nil {
			body = append(body, b...)
			lines.Add(int64(strings.Count(string(b), "\n")))
		}
		hits.Add(1)
		w.WriteHeader(200)
	}))
	defer srv.Close()

	if err := Configure(Config{Enabled: true, URL: srv.URL, StreamFields: "service", FlushInterval: 10 * time.Millisecond, BatchSize: 4, Timeout: 5 * time.Second}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	for i := 0; i < 6; i++ {
		Emit(Record{Service: "test", Method: "GET", Path: "/v1/realtime", RemoteAddr: "1.2.3.4:1234", Code: 1000, Outcome: OutcomeOK, StartedAt: now, DurationMs: 5 + float64(i), Language: "id", Model: "piper", Codec: "pcm16", SessionID: "00000000-0000-0000-0000-000000000000"})
	}
	time.Sleep(300 * time.Millisecond)

	shutdownDone := make(chan struct{})
	go func() { Shutdown(shutdownDone); close(shutdownDone) }()
	select {
	case <-shutdownDone:
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown hung")
	}

	if hits.Load() == 0 {
		t.Fatal("no HTTP requests sent")
	}
	got := string(body)
	if !strings.Contains(got, `"_msg":"GET /v1/realtime 1000`) {
		t.Fatalf("expected _msg in body, got:\n%s", got)
	}
	if !strings.Contains(got, `"duration_ms"`) {
		t.Fatalf("expected duration_ms in body")
	}
	first := strings.Split(got, "\n")[0]
	var anyRec map[string]any
	if err := json.Unmarshal([]byte(first), &anyRec); err != nil {
		t.Fatalf("line not valid json: %v", err)
	}
	if anyRec["model"] != "piper" {
		t.Fatalf("expected model field, got %v", anyRec)
	}
}

func TestPercentileAndMean(t *testing.T) {
	if got := Percentile(nil, 50); got != 0 {
		t.Fatalf("p50 nil = %v", got)
	}
	sorted := []float64{10, 20, 30, 40}
	if got := Percentile(sorted, 50); got != 25 {
		t.Fatalf("p50 = %v", got)
	}
	if got := Percentile(sorted, 95); got != 38.5 {
		t.Fatalf("p95 = %v, want 38.5", got)
	}
	if got := Mean([]float64{2, 4, 6}); got != 4 {
		t.Fatalf("mean = %v", got)
	}
}
