package log

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
	sentrylogrus "github.com/getsentry/sentry-go/logrus"
	"github.com/sirupsen/logrus"
)

// ---------------------------------------------------------------------
// global state
// ---------------------------------------------------------------------

var (
	std        *logrus.Logger
	mu         sync.RWMutex
	currentOut io.Writer = os.Stderr
	disabled   bool
)

func init() {
	std = logrus.New()
	std.SetOutput(currentOut)
	std.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		DisableColors:   false,
		TimestampFormat: time.RFC3339,
	})
	std.SetLevel(logrus.InfoLevel)

	// Make Fatal testable / allow flush via ExitHandler
	// Default ExitFunc = os.Exit; keep it, but register flush handler later if needed.
	logrus.RegisterExitHandler(func() {
		// Try flush sentry if initialized
		sentry.Flush(2 * time.Second)
	})
}

// ---------------------------------------------------------------------
// Types / Levels
// ---------------------------------------------------------------------

// Level is an alias to logrus.Level with safe mapping.
type Level logrus.Level

// Hook alias.
type Hook logrus.Hook

// Format for output.
type Format uint8

// Fields alias to avoid exposing logrus directly in signature.
type Fields = logrus.Fields

const (
	PanicLevel Level = Level(logrus.PanicLevel)
	FatalLevel Level = Level(logrus.FatalLevel)
	ErrorLevel Level = Level(logrus.ErrorLevel)
	WarnLevel  Level = Level(logrus.WarnLevel)
	InfoLevel  Level = Level(logrus.InfoLevel)
	DebugLevel Level = Level(logrus.DebugLevel)
	TraceLevel Level = Level(logrus.TraceLevel)

	// OFF disables all logging. Internally it discards output.
	OFF Level = Level(255)
)

const (
	_ Format = iota
	JSONFormat
	TextFormat
)

// Canonical aliases
const (
	FormatJSON = JSONFormat
	FormatText = TextFormat
)

// ---------------------------------------------------------------------
// Logger interface - production grade with structured methods
// ---------------------------------------------------------------------

// Logger is general logging interface.
// Kept backward compatible with old Print/Debug/Info... plus structured helpers.
type Logger interface {
	// legacy unformatted compat
	Print(args ...interface{})
	Printf(format string, args ...interface{})

	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Trace(args ...interface{})
	Tracef(format string, args ...interface{})

	// structured (production requirement)
	WithField(key string, value interface{}) Logger
	WithFields(fields Fields) Logger
	WithError(err error) Logger
	WithContext(ctx context.Context) Logger

	IsLevelEnabled(Level) bool
}

// Context is kept for backward compat with old For() API, but file/line now captured correctly.
type Context struct {
	Package string
	Scope   string
	Line    int
	File    string
}

// ---------------------------------------------------------------------
// internal wrapper around *logrus.Entry
// ---------------------------------------------------------------------

type entryWrapper struct {
	e *logrus.Entry
}

func newEntry(e *logrus.Entry) *entryWrapper {
	return &entryWrapper{e: e}
}

func (l *entryWrapper) isOff() bool {
	mu.RLock()
	defer mu.RUnlock()
	return disabled
}

func (l *entryWrapper) Print(args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Print(args...)
}
func (l *entryWrapper) Printf(format string, args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Printf(format, args...)
}
func (l *entryWrapper) Debug(args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Debug(args...)
}
func (l *entryWrapper) Debugf(format string, args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Debugf(format, args...)
}
func (l *entryWrapper) Info(args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Info(args...)
}
func (l *entryWrapper) Infof(format string, args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Infof(format, args...)
}
func (l *entryWrapper) Warn(args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Warn(args...)
}
func (l *entryWrapper) Warnf(format string, args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Warnf(format, args...)
}
func (l *entryWrapper) Error(args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Error(args...)
}
func (l *entryWrapper) Errorf(format string, args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Errorf(format, args...)
}
func (l *entryWrapper) Fatal(args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Fatal(args...)
}
func (l *entryWrapper) Fatalf(format string, args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Fatalf(format, args...)
}
func (l *entryWrapper) Trace(args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Trace(args...)
}
func (l *entryWrapper) Tracef(format string, args ...interface{}) {
	if l.isOff() {
		return
	}
	l.e.Tracef(format, args...)
}

func (l *entryWrapper) WithField(key string, value interface{}) Logger {
	return newEntry(l.e.WithField(key, value))
}
func (l *entryWrapper) WithFields(fields Fields) Logger {
	return newEntry(l.e.WithFields(fields))
}
func (l *entryWrapper) WithError(err error) Logger {
	return newEntry(l.e.WithError(err))
}
func (l *entryWrapper) WithContext(ctx context.Context) Logger {
	return newEntry(l.e.WithContext(ctx))
}
func (l *entryWrapper) IsLevelEnabled(level Level) bool {
	if l.isOff() {
		return false
	}
	return l.e.Logger.IsLevelEnabled(logrus.Level(level))
}

// Ensure compile check
var _ Logger = (*entryWrapper)(nil)

// ---------------------------------------------------------------------
// Global configuration (thread-safe)
// ---------------------------------------------------------------------

// GetLogger returns default logger as Logger interface.
func GetLogger() Logger {
	return newEntry(logrus.NewEntry(std))
}

// SetOutput changes logger output, thread-safe.
func SetOutput(w io.Writer) {
	mu.Lock()
	defer mu.Unlock()
	currentOut = w
	if !disabled {
		std.SetOutput(w)
	}
}

// SetLevel sets log level, handles OFF correctly.
func SetLevel(level Level) {
	mu.Lock()
	defer mu.Unlock()
	if level == OFF {
		if !disabled {
			disabled = true
			std.SetOutput(io.Discard)
		}
		return
	}
	if disabled {
		disabled = false
		std.SetOutput(currentOut)
	}
	std.SetLevel(logrus.Level(level))
}

// SetFormat for the logger, thread-safe.
func SetFormat(format Format) {
	mu.Lock()
	defer mu.Unlock()
	switch format {
	case JSONFormat:
		std.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	default:
		std.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: time.RFC3339,
		})
	}
}

// SetReportCaller enables/disables caller reporting (production useful for text logs).
func SetReportCaller(enabled bool) {
	mu.Lock()
	defer mu.Unlock()
	std.SetReportCaller(enabled)
}

// isDisabled internal helper (R) for global functions
func isDisabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return disabled
}

// ---------------------------------------------------------------------
// Global convenience functions (backward compat)
// ---------------------------------------------------------------------

func Print(args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Print(args...)
}
func Printf(format string, args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Printf(format, args...)
}
func Debug(args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Debug(args...)
}
func Debugf(format string, args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Debugf(format, args...)
}
func Info(args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Info(args...)
}
func Infof(format string, args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Infof(format, args...)
}
func Warn(args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Warn(args...)
}
func Warnf(format string, args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Warnf(format, args...)
}
func Error(args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Error(args...)
}
func Errorf(format string, args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Errorf(format, args...)
}
func Fatal(args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Fatal(args...)
}
func Fatalf(format string, args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Fatalf(format, args...)
}
func Trace(args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Trace(args...)
}
func Tracef(format string, args ...interface{}) {
	if isDisabled() {
		return
	}
	std.Tracef(format, args...)
}

// AddHook thread-safe.
func AddHook(h Hook) {
	mu.Lock()
	defer mu.Unlock()
	std.AddHook(h)
}

// Standard returns underlying *logrus.Logger (escape hatch).
func Standard() *logrus.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return std
}

// ---------------------------------------------------------------------
// Contextual logger For(pkg, scope)
// ---------------------------------------------------------------------

// For creates a contextual logger with package/scope and file:line captured at call site.
// Note: file:line is from creation site; prefer WithField for per-log location, or enable ReportCaller.
func For(pkg, scope string) Logger {
	_, file, line, _ := runtime.Caller(1)
	file = filepath.Base(file)
	return withContext(Context{
		Package: pkg,
		Scope:   scope,
		Line:    line,
		File:    file,
	})
}

func withContext(c Context) Logger {
	scope := c.Scope + "[" + c.File + ":" + strconv.Itoa(c.Line) + "]"
	field := logrus.Fields{
		"package": c.Package,
		"scope":   scope,
	}
	return newEntry(std.WithFields(field))
}

// ---------------------------------------------------------------------
// Sentry integration (modern, non-deprecated)
// ---------------------------------------------------------------------

// NewSentryHook creates a sentry logrus hook using maintained sentry-go SDK.
// Returns hook and ensures error is checked before use (fixes old nil deref bug).
func NewSentryHook(dsn, environment, release string) (Hook, error) {
	levels := []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
	}
	hook, err := sentrylogrus.NewLogHook(levels, sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      environment,
		Release:          release,
		AttachStacktrace: true,
	})
	if err != nil {
		return nil, err
	}
	return hook, nil
}

// AddSentryHook adds a sentry hook to standard logger, thread-safe.
func AddSentryHook(dsn, environment, version string) error {
	hook, err := NewSentryHook(dsn, environment, version)
	if err != nil {
		return err
	}
	AddHook(hook)
	return nil
}

// Flush flushes sentry with timeout, useful for graceful shutdown.
func Flush(timeout time.Duration) bool {
	return sentry.Flush(timeout)
}

// FlushWithContext flushes with context (newer Sentry API pattern via Hub).
func FlushWithContext(ctx context.Context) bool {
	if hub := sentry.GetHubFromContext(ctx); hub != nil {
		return hub.FlushWithContext(ctx)
	}
	return sentry.Flush(5 * time.Second)
}
