package log

import (
	"bytes"
	"context"
	"sync"
	"testing"
)

func TestLevelOFF(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	SetLevel(OFF)
	Info("hidden")
	if buf.Len() != 0 {
		t.Fatalf("OFF should discard, got %q", buf.String())
	}
	SetLevel(InfoLevel)
	buf.Reset()
	Info("visible")
	if buf.Len() == 0 {
		t.Fatalf("after OFF, Info should appear")
	}
	// cleanup
	SetOutput(&bytes.Buffer{})
}

func TestStructured(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	SetLevel(DebugLevel)
	SetFormat(JSONFormat)

	GetLogger().WithField("user", "bob").Info("test")
	if !bytes.Contains(buf.Bytes(), []byte("bob")) {
		t.Fatalf("WithField not in output: %s", buf.String())
	}
	buf.Reset()
	GetLogger().WithFields(Fields{"a": 1}).Info("f")
	if !bytes.Contains(buf.Bytes(), []byte("\"a\":1")) {
		t.Fatalf("WithFields not in output: %s", buf.String())
	}
}

func TestForCaller(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	SetLevel(InfoLevel)
	SetFormat(TextFormat)

	For("pkg", "scope").Info("hello")
	s := buf.String()
	if !bytes.Contains([]byte(s), []byte("pkg")) || !bytes.Contains([]byte(s), []byte("scope")) {
		t.Fatalf("For context missing: %s", s)
	}
}

func TestConcurrency(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	SetLevel(InfoLevel)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			For("c", "s").WithField("n", n).Info("conc")
		}(i)
	}
	wg.Wait()
}

func TestSentryHookErrorHandling(t *testing.T) {
	// invalid DSN should error, not panic
	_, err := NewSentryHook("https://invalid", "test", "v1")
	if err == nil {
		t.Fatalf("expected error for invalid DSN")
	}
	// empty DSN is allowed by sentry-go (disabled client) - should not panic
	_, err = NewSentryHook("", "test", "v1")
	if err != nil {
		// acceptable to error, but ensure no panic
		t.Logf("empty DSN error (ok): %v", err)
	}
}

func TestWithContext(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	SetLevel(InfoLevel)
	ctx := context.WithValue(context.Background(), "req", "123")
	GetLogger().WithContext(ctx).Info("ctx test")
}

func TestPanicLevel(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	SetLevel(TraceLevel)
	logger := GetLogger()
	if !logger.IsLevelEnabled(TraceLevel) {
		t.Fatalf("Trace should be enabled")
	}
	if !logger.IsLevelEnabled(InfoLevel) {
		t.Fatalf("Info should be enabled when Trace")
	}
}

func TestNewIsolated(t *testing.T) {
	l1 := New()
	var b2 bytes.Buffer
	SetOutput(&b2)
	// isolated logger should not affect global output
	l1.Info("isolated")
	if b2.Len() != 0 {
		t.Fatalf("global buf should be empty for isolated logger")
	}
}
