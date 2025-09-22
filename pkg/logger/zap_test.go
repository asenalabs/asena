package logger

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestLoggingOutput(t *testing.T) {
	core, observedLogs := observer.New(zapcore.DebugLevel)
	logg = zap.New(core)

	Get().Info("test message")

	logs := observedLogs.All()
	if len(logs) != 1 {
		t.Errorf("got %d logs, expected 1", len(logs))
	}
	if logs[0].Message != "test message" {
		t.Errorf("got unexpected message: %s", logs[0].Message)
	}
}

func TestGetLoggerPanics(t *testing.T) {
	logg = nil

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when logger is not initialized")
		}
	}()

	_ = Get()
}

func TestGetLoggerReturnsLogger(t *testing.T) {
	IntiFallbackZapLogger()
	logg := Get()
	if logg == nil {
		t.Fatal("expected non-nil logg")
	}
}

func TestSyncDoesNotPanic(t *testing.T) {
	IntiFallbackZapLogger()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("Sync panicked: %v", r)
		}
	}()
	Sync()
}
