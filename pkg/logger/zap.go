package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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

func Get() *zap.Logger {
	if logg == nil {
		panic("Logger not initialized")
	}
	return logg
}

func Sync() {
	if logg != nil {
		if err := logg.Sync(); err != nil {
			panic("Failed to sync logger: " + err.Error())
		}
	}
}
