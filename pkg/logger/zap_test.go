package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/asenalabs/asena/internal/config"
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

func TestInitProductionZapLogger(t *testing.T) {
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "access.log")

	path := logFile
	maxSize := 1 // MB
	maxBackups := 1
	maxAge := 1 // days
	compress := false

	cfg := &config.LogCfg{
		Lumberjack: &config.LumberjackCfg{
			Path:       &path,
			MaxSize:    &maxSize,
			MaxBackups: &maxBackups,
			MaxAge:     &maxAge,
			Compress:   &compress,
		},
	}

	InitProductionZapLogger("production", cfg)
	log := Get()

	wantMsg := "production zap logger initialized"
	log.Info(wantMsg)
	Sync()

	time.Sleep(100 * time.Millisecond)
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}

	if !strings.Contains(string(data), wantMsg) {
		t.Errorf("log file does not contain expected message: \nGot: %s", wantMsg)
	}

	if !strings.Contains(string(data), `"level":"INFO"`) && !strings.Contains(string(data), `"level":"INFO"`) {
		t.Errorf("log is not JSON formatted: %s", string(data))
	}
}
