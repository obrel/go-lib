// Package metrics holds a per-session accumulator for engine timings. It
// intentionally carries no shipping, no logging, and no knowledge of the
// HTTP layer: engines observe decode/synthesis timings, handlers snapshot
// them at session end and forward them to the telemetry layer.
//
// This package is shared between aira-stt and aira-tts via go-lib, so it
// must stay dependency-free (stdlib only) and race-safe.
package metrics

import (
	"slices"
	"sync"
)

// cap bounds samples retained per session; sums/means always use all
// observations via sum counters, and percentile uses a capped sample list
// that prefers recent values.
const capSamples = 512

type Accumulator struct {
	mu           sync.Mutex
	decodeMs   []float64
	decodeSum  float64
	calls        int64
	rtfSum       float64
	rtfCalls     int64
	synthMs      []float64
	synthSum     float64
	synthN       int64
	chars        int64
	cancelled    int64
	queueWaitMs  []float64
	queueWaitSum float64
}

type Snapshot struct {
	DecodeMsP50 float64
	DecodeMsP95 float64
	RtfMean     float64
	Calls       int64

	SynthMsP50   float64
	SynthMsP95   float64
	SynthMsTotal float64
	SynthCalls   int64
	Chars        int64
	Cancelled    int64

	QueueWaitP50   float64
	QueueWaitP95   float64
	QueueWaitTotal float64
	QueueWaitMean  float64
}

// ObserveDecode records one decode call: ms is wall time, rtf = decode time / audio seconds.
func (a *Accumulator) ObserveDecode(ms, rtf float64) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.decodeMs = appendCapped(a.decodeMs, ms)
	a.decodeSum += ms
	a.calls++
	a.rtfSum += rtf
	a.rtfCalls++
}

// ObserveSynth records one synthesis request outcome.
func (a *Accumulator) ObserveSynth(ms float64, chars int, cancelled bool) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.synthMs = appendCapped(a.synthMs, ms)
	a.synthSum += ms
	a.synthN++
	a.chars += int64(chars)
	if cancelled {
		a.cancelled++
	}
}

// ObserveQueueWait records time a job waited in queue before processing.
func (a *Accumulator) ObserveQueueWait(ms float64) {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.queueWaitMs = appendCapped(a.queueWaitMs, ms)
	a.queueWaitSum += ms
}

func appendCapped(dst []float64, v float64) []float64 {
	if len(dst) >= capSamples {
		copy(dst, dst[1:])
		dst = dst[:len(dst)-1]
	}
	return append(dst, v)
}

func (a *Accumulator) Snapshot() Snapshot {
	a.mu.Lock()
	calls := a.calls
	rtfCalls := a.rtfCalls
	rtfSum := a.rtfSum
	synthSum := a.synthSum
	synthN := a.synthN
	chars := a.chars
	cancelled := a.cancelled
	queueWaitSum := a.queueWaitSum
	decodeCopy := append([]float64(nil), a.decodeMs...)
	synthCopy := append([]float64(nil), a.synthMs...)
	queueCopy := append([]float64(nil), a.queueWaitMs...)
	a.mu.Unlock()

	var s Snapshot
	s.Calls = calls
	if rtfCalls > 0 {
		s.RtfMean = rtfSum / float64(rtfCalls)
	}
	s.DecodeMsP50 = percentile(decodeCopy, 50)
	s.DecodeMsP95 = percentile(decodeCopy, 95)
	s.SynthMsP50 = percentile(synthCopy, 50)
	s.SynthMsP95 = percentile(synthCopy, 95)
	s.SynthMsTotal = synthSum
	s.SynthCalls = synthN
	s.Chars = chars
	s.Cancelled = cancelled
	s.QueueWaitP50 = percentile(queueCopy, 50)
	s.QueueWaitP95 = percentile(queueCopy, 95)
	s.QueueWaitTotal = queueWaitSum
	if len(queueCopy) > 0 {
		s.QueueWaitMean = queueWaitSum / float64(len(queueCopy))
	}
	return s
}

func percentile(v []float64, n float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	slices.Sort(s)
	if n <= 0 {
		return s[0]
	}
	if n >= 100 {
		return s[len(s)-1]
	}
	pos := float64(len(s)-1) * (n / 100)
	lo := int(pos)
	frac := pos - float64(lo)
	if lo+1 < len(s) {
		return s[lo]*(1-frac) + s[lo+1]*frac
	}
	return s[lo]
}
