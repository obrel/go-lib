// Package requestlog ships one structured log line per finished request
// (one WebSocket session, or one rejected start_session) to VictoriaLogs
// via its ndjson JSON-stream ingestion API.
//
// This is the shared implementation used by aira-tts and aira-stt via
// go-lib, decoupled from their internal/pkg/logging – it uses go-lib/pkg/log
// internally, and exposes a Logger interface for custom loggers if needed.
//
// Fire-and-forget: Emit never blocks and never errors.
package requestlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/obrel/go-lib/pkg/log"
)

// outcome values.
const (
	OutcomeOK       = "ok"
	OutcomeRejected = "rejected"
	OutcomeError    = "error"
	OutcomeTimeout  = "timeout"
)

// Config is the whole surface Configure needs. Enabled=false makes every
// other field meaningless and Emit/Shutdown no-ops.
type Config struct {
	Enabled       bool
	URL           string
	StreamFields  string
	FlushInterval time.Duration
	BatchSize     int
	Timeout       time.Duration
	// Logger optional, defaults to go-lib/pkg/log For("requestlog","shipper")
	Logger log.Logger
}

// Record is one finished request. Zero-valued fields omitted via omitempty.
type Record struct {
	Service    string `json:"service"`
	Event      string `json:"event"`
	Method     string `json:"method"`
	Path       string `json:"path"`
	RemoteAddr string `json:"remote_addr,omitempty"`

	Language string `json:"language,omitempty"`
	Model    string `json:"model,omitempty"`
	Codec    string `json:"codec,omitempty"`

	Code        int    `json:"code,omitempty"`
	CloseReason string `json:"close_reason,omitempty"`
	Outcome     string `json:"outcome"`

	StartedAt  time.Time `json:"started_at"`
	DurationMs float64   `json:"duration_ms"`

	SessionID string `json:"session_id,omitempty"`
	OpaqueID  string `json:"opaque_id,omitempty"`

	// Common timing breakdown (helps explain high duration_ms)
	IdleMs            float64 `json:"idle_ms,omitempty"`             // conn start -> session start
	SessionDurationMs float64 `json:"session_duration_ms,omitempty"` // session start -> connection close

	// STT fields
	TTFPMs        float64 `json:"ttfp_ms,omitempty"`         // now: first audio -> first partial (was conn -> partial)
	FirstFinalMs  float64 `json:"first_final_ms,omitempty"`  // first audio -> first final
	FirstAudioMs  float64 `json:"first_audio_ms,omitempty"`  // session start -> first audio frame
	Partials      int     `json:"partials_emitted,omitempty"`
	Finals        int     `json:"finals_emitted,omitempty"`
	DecodeMsP50   float64 `json:"decode_ms_p50,omitempty"`
	DecodeMsP95   float64 `json:"decode_ms_p95,omitempty"`
	RtfMean       float64 `json:"rtf_mean,omitempty"`
	ChunksDropped int     `json:"chunks_dropped,omitempty"`
	AudioBytes    int64   `json:"audio_bytes,omitempty"`

	// TTS fields
	TTFAMs            float64 `json:"ttfa_ms,omitempty"`       // first synthesize -> first audio chunk
	TTFAMeanMs        float64 `json:"ttfa_ms_mean,omitempty"`  // deprecated, not used
	SynthMsTotal      float64 `json:"synth_ms_total,omitempty"`
	Segments          int     `json:"segments,omitempty"`
	CancelledSegments int     `json:"cancelled_segments,omitempty"`
	CharsTotal        int     `json:"chars_total,omitempty"`
	QueueWaitMs       float64 `json:"queue_wait_ms,omitempty"` // time job waited in worker queue
}

const eventName = "request"

var (
	logger     log.Logger
	cfg        Config
	enabled    atomic.Bool
	ch         chan Record
	done       chan struct{}
	wg         sync.WaitGroup
	client     *http.Client
	configured atomic.Bool
	dropped    atomic.Int64
	flushed    atomic.Int64
	flushErr   atomic.Int64
)

func init() {
	logger = log.For("requestlog", "shipper")
}

// Configure switches the package on and starts the flush goroutine. Call once at startup.
func Configure(c Config) error {
	if !configured.CompareAndSwap(false, true) {
		if c.Logger != nil {
			logger = c.Logger
		}
		if l, ok := logger.(interface{ Warn(args ...interface{}) }); ok {
			_ = l
		}
		// use go-lib log for warning
		log.For("requestlog", "shipper").Warn("requestlog already configured; ignoring second Configure call")
		return nil
	}
	if c.Logger != nil {
		logger = c.Logger
	} else {
		logger = log.For("requestlog", "shipper")
	}
	cfg = c
	if !c.Enabled {
		enabled.Store(false)
		logger.Info("request log disabled")
		return nil
	}
	if c.FlushInterval <= 0 {
		c.FlushInterval = time.Second
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 50
	}
	if c.Timeout <= 0 {
		c.Timeout = 2 * time.Second
	}
	if _, err := url.ParseRequestURI(strings.TrimSuffix(c.URL, "/")); err != nil {
		enabled.Store(false)
		logger.WithField("url", c.URL).WithError(err).Warn("request log disabled: invalid telemetry.url")
		return err
	}
	cfg = c
	ch = make(chan Record, c.BatchSize*4)
	done = make(chan struct{})
	client = &http.Client{Timeout: c.Timeout}
	enabled.Store(true)
	wg.Add(1)
	go flushLoop()
	logger.WithField("url", c.URL).WithField("stream_fields", c.StreamFields).Info("request log enabled")
	return nil
}

// Shutdown flushes remaining buffered records and stops the flush goroutine.
func Shutdown(ctxDone <-chan struct{}) {
	if !enabled.Swap(false) {
		return
	}
	close(done)
	wait := make(chan struct{})
	go func() { wg.Wait(); close(wait) }()
	select {
	case <-wait:
	case <-ctxDone:
	case <-time.After(cfg.Timeout + time.Second):
	}
}

// Emit queues r for asynchronous delivery. Never blocks.
func Emit(r Record) {
	if !enabled.Load() {
		return
	}
	if r.Event == "" {
		r.Event = eventName
	}
	select {
	case ch <- r:
	default:
		dropped.Add(1)
	}
}

// IsEnabled reports whether telemetry was successfully configured.
func IsEnabled() bool {
	return enabled.Load()
}

// Stats returns (emitted, dropped, flushErrors).
func Stats() (emitted, droppedCount, flushErrors int64) {
	return flushed.Load(), dropped.Load(), flushErr.Load()
}

func flushLoop() {
	defer wg.Done()
	tick := time.NewTicker(cfg.FlushInterval)
	defer tick.Stop()
	buf := make([]Record, 0, cfg.BatchSize)

	for {
		select {
		case <-done:
			drain(ch, &buf, 0)
			for len(buf) > 0 {
				n := len(buf)
				if n > cfg.BatchSize {
					n = cfg.BatchSize
				}
				send(buf[:n])
				buf = buf[n:]
			}
			return

		case r := <-ch:
			buf = append(buf, r)
			if len(buf) >= cfg.BatchSize {
				send(buf)
				buf = buf[:0]
			}

		case <-tick.C:
			drain(ch, &buf, cfg.BatchSize)
			if len(buf) > 0 {
				for len(buf) > 0 {
					n := len(buf)
					if n > cfg.BatchSize {
						n = cfg.BatchSize
					}
					send(buf[:n])
					buf = buf[n:]
				}
				buf = buf[:0]
			}
		}
	}
}

func drain(ch <-chan Record, buf *[]Record, limit int) {
	if limit <= 0 {
		for {
			select {
			case r := <-ch:
				*buf = append(*buf, r)
			default:
				return
			}
		}
	}
	for len(*buf) < limit {
		select {
		case r := <-ch:
			*buf = append(*buf, r)
		default:
			return
		}
	}
}

func send(buf []Record) {
	if len(buf) == 0 {
		return
	}
	body, err := encodeNdjson(buf)
	if err != nil {
		flushErr.Add(int64(len(buf)))
		logger.WithField("records", len(buf)).WithError(err).Error("request log encode failed")
		return
	}
	u := strings.TrimSuffix(cfg.URL, "/") + "/insert/jsonline?_msg_field=_msg"
	if cfg.StreamFields != "" {
		u += "&_stream_fields=" + url.QueryEscape(cfg.StreamFields)
	}
	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		flushErr.Add(int64(len(buf)))
		logger.WithError(err).Error("request log build request failed")
		return
	}
	req.Header.Set("Content-Type", "application/stream+json")
	resp, err := client.Do(req)
	if err != nil {
		flushErr.Add(int64(len(buf)))
		logger.WithField("records", len(buf)).WithError(err).Warn("request log ship failed (records dropped)")
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		flushErr.Add(int64(len(buf)))
		logger.WithField("status", resp.StatusCode).WithField("records", len(buf)).Warn("request log ship rejected (records dropped)")
		return
	}
	flushed.Add(int64(len(buf)))
}

func encodeNdjson(buf []Record) ([]byte, error) {
	var b bytes.Buffer
	b.Grow(len(buf) * 256)
	nowStr := time.Now().UTC().Format(time.RFC3339Nano)
	for _, r := range buf {
		if r.Event == "" {
			r.Event = eventName
		}
		enriched := struct {
			Record
			Msg  string `json:"_msg"`
			Time string `json:"_time"`
		}{
			Record: r,
			Msg:    fmt.Sprintf("%s %s %d %.0fms %s", r.Method, r.Path, r.Code, r.DurationMs, r.Outcome),
			Time:   nowStr,
		}
		line, err := json.Marshal(enriched)
		if err != nil {
			return nil, err
		}
		b.Write(line)
		b.WriteByte('\n')
	}
	return b.Bytes(), nil
}

// Percentile returns the n-th percentile.
func Percentile(v []float64, n float64) float64 {
	if len(v) == 0 {
		return 0
	}
	sorted := append([]float64(nil), v...)
	slices.Sort(sorted)
	if n <= 0 {
		return sorted[0]
	}
	if n >= 100 {
		return sorted[len(sorted)-1]
	}
	pos := float64(len(sorted)-1) * (n / 100)
	lo := int(pos)
	frac := pos - float64(lo)
	if lo+1 < len(sorted) {
		return sorted[lo]*(1-frac) + sorted[lo+1]*frac
	}
	return sorted[lo]
}

// Mean returns arithmetic mean.
func Mean(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	var s float64
	for _, x := range v {
		s += x
	}
	return s / float64(len(v))
}
