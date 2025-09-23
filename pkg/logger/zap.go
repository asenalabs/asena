package logger

import (
	"os"
	"strings"

	"github.com/asenalabs/asena/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type noSyncWriter struct {
	*os.File
}

var logg *zap.Logger

func IntiFallbackZapLogger() {
	var zapCfg zap.Config
	var encoder zapcore.Encoder

	zapCfg = zap.NewDevelopmentConfig()
	encoder = zapcore.NewConsoleEncoder(zapCfg.EncoderConfig)

	var writer zapcore.WriteSyncer
	writer = zapcore.AddSync(os.Stdout)

	core := zapcore.NewCore(encoder, writer, zapcore.DebugLevel)

	logg = zap.New(core, zap.AddCaller())
}

func InitProductionZapLogger(env string, cfg *config.LogCfg) {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder

	var encoder zapcore.Encoder
	switch strings.ToLower(env) {
	case "dev", "development":
		encoder = zapcore.NewConsoleEncoder(encoderCfg)
	default:
		encoder = zapcore.NewJSONEncoder(encoderCfg)
	}

	newWriter := func(cfg *config.LumberjackCfg) zapcore.WriteSyncer {
		return zapcore.AddSync(&lumberjack.Logger{
			Filename:   *cfg.Path,
			MaxSize:    *cfg.MaxSize,
			MaxBackups: *cfg.MaxBackups,
			MaxAge:     *cfg.MaxAge,
			Compress:   *cfg.Compress,
		})
	}

	accessWriter := newWriter(cfg.Lumberjack)

	infoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.InfoLevel
	})

	accessCore := zapcore.NewCore(encoder, accessWriter, infoLevel)

	var combinedCore zapcore.Core
	switch strings.ToLower(env) {
	case "dev", "development":
		consoleWriter := zapcore.AddSync(noSyncWriter{os.Stdout})
		consoleCore := zapcore.NewCore(zapcore.NewConsoleEncoder(encoderCfg), consoleWriter, zapcore.DebugLevel)
		combinedCore = zapcore.NewTee(accessCore, consoleCore)
	default:
		combinedCore = zapcore.NewTee(accessCore)
	}

	logg = zap.New(combinedCore, zap.AddCaller())
}

func Get() *zap.Logger {
	if logg == nil {
		panic("Logger not initialized")
	}
	return logg
}

func Sync() {
	if logg != nil {
		_ = logg.Sync()
	}
}
