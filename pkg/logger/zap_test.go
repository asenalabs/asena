package logger

import (
	"log"
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

func TestMustZapToStdLoggerAtLevel(t *testing.T) {
	tests := []struct {
		name        string
		level       zapcore.Level
		shouldPanic bool
	}{
		{"valid debug", zapcore.DebugLevel, false},
		{"valid info", zapcore.InfoLevel, false},
		{"invalid level", zapcore.Level(100), true}, // intentionally invalid
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z, _ := zap.NewDevelopment()

			defer func() {
				if r := recover(); r != nil {
					if !tt.shouldPanic {
						t.Errorf("unexpected panic: %v", r)
					}
				} else {
					if tt.shouldPanic {
						t.Errorf("expected panic but got none")
					}
				}
			}()

			l := MustZapToStdLoggerAtLevel(z, tt.level)
			if !tt.shouldPanic && l == nil {
				t.Error("expected *log.Logger, got nil")
			}
			if !tt.shouldPanic {
				_, ok := interface{}(l).(*log.Logger)
				if !ok {
					t.Errorf("expected *log.Logger type, got %T", l)
				}
			}
		})
	}
}
