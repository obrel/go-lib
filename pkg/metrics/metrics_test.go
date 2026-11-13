package metrics

import "testing"

func TestAccumulator(t *testing.T) {
	var a Accumulator
	a.ObserveDecode(10, 0.1)
	a.ObserveDecode(20, 0.2)
	s := a.Snapshot()
	if s.Calls != 2 {
		t.Fatalf("calls %d want 2", s.Calls)
	}
	if s.RtfMean < 0.14 || s.RtfMean > 0.16 {
		t.Fatalf("rtf mean %v", s.RtfMean)
	}
	a.ObserveSynth(100, 5, false)
	a.ObserveSynth(200, 10, true)
	s = a.Snapshot()
	if s.SynthCalls != 2 || s.Chars != 15 || s.Cancelled != 1 {
		t.Fatalf("synth snap %+v", s)
	}
}

func TestAppendCapped(t *testing.T) {
	var dst []float64
	for i := 0; i < 600; i++ {
		dst = appendCapped(dst, float64(i))
	}
	if len(dst) != capSamples {
		t.Fatalf("len %d want %d", len(dst), capSamples)
	}
	if dst[0] != float64(600-capSamples) {
		t.Fatalf("first %v", dst[0])
	}
}
